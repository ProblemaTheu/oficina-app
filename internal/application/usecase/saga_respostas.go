package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ProblemaTheu/oficina-app/internal/application/logging"
	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
)

// ErrRespostaSemSaga indica uma resposta que não pertence a nenhuma saga deste
// serviço (OS inexistente ou saga_id divergente). Reprocessar não resolve: o
// consumidor descarta com log de erro.
var ErrRespostaSemSaga = errors.New("resposta sem saga correspondente")

// ErrSagaDesligada indica que o caso de uso foi montado sem ComSaga.
var ErrSagaDesligada = errors.New("saga não configurada")

// RespostaSaga é uma mensagem recebida na fila os-saga-replies.fifo.
type RespostaSaga struct {
	MessageID uuid.UUID
	Tipo      string
	SagaID    uuid.UUID
	OsID      uuid.UUID
	Payload   []byte
}

// ProcessarRespostaSaga é o orquestrador em ação (F4-4.4/F4-4.5): marca a
// resposta na inbox, decide com saga.Decidir e aplica a decisão — saga (com
// lock otimista), etapas, estoque, status e histórico da OS e próximos
// comandos na outbox — numa única transação. Duplicata = sucesso sem efeito.
func (uc *OrdemServicoUseCase) ProcessarRespostaSaga(ctx context.Context, r RespostaSaga) error {
	if uc.saga == nil {
		return ErrSagaDesligada
	}
	dados := lerDados(r.Payload)

	var (
		dec      saga.Decisao
		os       *entity.OrdemServicoCompleta
		duplicat bool
	)
	err := uc.saga.UoW.Executar(ctx, func(ctx context.Context) error {
		nova, err := uc.saga.Inbox.Marcar(ctx, r.MessageID, r.Tipo)
		if err != nil {
			return err
		}
		if !nova {
			duplicat = true
			return nil
		}
		s, err := uc.saga.Sagas.BuscarPorOS(ctx, r.OsID)
		var naoEncontrado *domainerros.ErrNaoEncontrado
		if errors.As(err, &naoEncontrado) || (err == nil && s.ID != r.SagaID) {
			return fmt.Errorf("%w: os %s, saga %s", ErrRespostaSemSaga, r.OsID, r.SagaID)
		}
		if err != nil {
			return err
		}

		dec = saga.Decidir(*s, r.Tipo, dados, uc.saga.Agora())
		if dec.MudouSaga {
			if err := uc.saga.Sagas.Atualizar(ctx, dec.Saga, s.Versao); err != nil {
				return err
			}
		}
		for _, e := range dec.Etapas {
			if err := uc.saga.Sagas.RegistrarEtapa(ctx, s.ID, e.Etapa, e.Resultado, &r.MessageID, detalhe(e.Detalhe)); err != nil {
				return err
			}
		}
		if dec.BaixarReserva {
			if err := uc.saga.Estoque.Baixar(ctx, r.OsID); err != nil {
				return err
			}
		}
		if dec.LiberarReserva {
			if err := uc.saga.Estoque.Liberar(ctx, r.OsID); err != nil {
				return err
			}
		}
		if dec.StatusOS == "" && len(dec.Comandos) == 0 {
			return nil
		}
		if os, err = uc.osRepo.BuscarPorID(ctx, r.OsID.String()); err != nil {
			return err
		}
		if dec.StatusOS != "" {
			if err := uc.aplicarStatusDaSaga(ctx, os, dec, r.Tipo); err != nil {
				return err
			}
		}
		for _, c := range dec.Comandos {
			if c.Payload == nil {
				if c.Payload, err = uc.payloadDoComando(c.Tipo, os, dec); err != nil {
					return err
				}
			}
			if err := uc.saga.Comandos.Enfileirar(ctx, c); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	log := logging.Log(ctx).With("os_id", r.OsID, "saga_id", r.SagaID, "tipo_mensagem", r.Tipo)
	switch {
	case duplicat:
		log.Info("resposta da saga já processada; ignorada")
	case dec.Ignorada:
		log.Warn("resposta inesperada para o estado da saga; registrada como ignorada", "estado", dec.Saga.Estado)
	default:
		log.Info("saga avançou", "estado", dec.Saga.Estado, "etapa_atual", dec.Saga.EtapaAtual, "comandos", len(dec.Comandos))
	}
	// Notificação ao cliente só depois do commit (e-mail não tem rollback).
	if os != nil && dec.StatusOS != "" && dec.StatusOS != os.StatusNome {
		var motivo *string
		if dec.CancelarOS {
			motivo = &dec.MotivoCancelamento
		}
		uc.notificarMudancaStatus(&os.OrdemServico, dec.StatusOS, motivo)
	}
	return nil
}

// aplicarStatusDaSaga muda o status da OS (o orquestrador não passa pelas
// transições manuais) e registra o histórico com a mensagem que causou a
// mudança (F4-1.4).
func (uc *OrdemServicoUseCase) aplicarStatusDaSaga(ctx context.Context, os *entity.OrdemServicoCompleta, dec saga.Decisao, tipo string) error {
	statusID, err := uc.osRepo.BuscarStatusID(ctx, dec.StatusOS)
	if err != nil {
		return err
	}
	if dec.CancelarOS {
		if err := uc.saga.Cancelamento.CancelarPelaSaga(ctx, os.ID, statusID, dec.MotivoCancelamento); err != nil {
			return err
		}
	} else {
		agora := uc.saga.Agora()
		var aprovado, iniciado, finalizado *time.Time
		switch dec.Marco {
		case saga.MarcoAprovado:
			aprovado = &agora
		case saga.MarcoIniciado:
			iniciado = &agora
		case saga.MarcoFinalizado:
			finalizado = &agora
		}
		if err := uc.osRepo.AtualizarStatus(ctx, os.ID, statusID, textoOuNil(dec.Diagnostico), aprovado, nil, iniciado, finalizado, nil); err != nil {
			return err
		}
	}
	if statusID == os.StatusID {
		return nil // só um carimbo de tempo (ex.: iniciado_em); status igual
	}
	obs := "saga: " + tipo
	if dec.CancelarOS {
		obs += " — " + dec.MotivoCancelamento
	}
	return uc.osRepo.RegistrarHistorico(ctx, os.ID, &os.StatusID, statusID, &obs)
}

// itemComando é um item da OS como viaja nos comandos — preço em string
// decimal: o Billing converte para centavos sem passar por float.
type itemComando struct {
	Tipo  string `json:"tipo"`
	Ref   string `json:"ref"`
	Nome  string `json:"nome"`
	Qtd   int    `json:"qtd"`
	Preco string `json:"preco"`
}

// payloadDoComando completa os comandos que levam dados da OS — os payloads são
// autossuficientes: Billing e Execution não consultam o OS (arquitetura §5).
func (uc *OrdemServicoUseCase) payloadDoComando(tipo string, os *entity.OrdemServicoCompleta, dec saga.Decisao) (any, error) {
	itens := make([]itemComando, 0, len(os.Itens))
	for _, i := range os.Itens {
		itens = append(itens, itemComando{Tipo: i.Tipo, Ref: i.ReferenciaID.String(), Nome: i.Nome, Qtd: i.Quantidade,
			Preco: fmt.Sprintf("%.2f", i.PrecoUnitario)})
	}
	diagnostico := dec.Diagnostico // gravado nesta mesma transação: ainda não visível na leitura
	if diagnostico == "" && os.Diagnostico != nil {
		diagnostico = *os.Diagnostico
	}
	switch tipo {
	case saga.TipoGerarOrcamento:
		cli, err := uc.clienteRepo.BuscarPorID(os.ClienteID.String())
		if err != nil {
			return nil, err
		}
		email := ""
		if cli.Email != nil {
			email = *cli.Email
		}
		return map[string]any{
			"numero_os": os.Numero, "itens": itens, "diagnostico": diagnostico,
			"cliente": map[string]string{"id": cli.ID.String(), "nome": cli.Nome, "email": email},
		}, nil
	case saga.TipoExecutarReparo:
		return map[string]any{
			"numero_os": os.Numero, "itens": itens, "diagnostico": diagnostico,
			"veiculo": saga.Veiculo{Placa: os.Veiculo.Placa, Modelo: os.Veiculo.Modelo},
		}, nil
	default:
		return nil, fmt.Errorf("comando %s sem payload", tipo)
	}
}

// lerDados extrai das respostas os campos que o orquestrador usa. Payload
// inválido vira Dados vazio: a decisão depende do tipo, não do payload.
func lerDados(payload []byte) saga.Dados {
	var p map[string]any
	_ = json.Unmarshal(payload, &p)
	texto := func(chave string) string {
		v, _ := p[chave].(string)
		return strings.TrimSpace(v)
	}
	id := func(chave string) *uuid.UUID {
		if u, err := uuid.Parse(texto(chave)); err == nil {
			return &u
		}
		return nil
	}
	motivo := texto("motivo")
	if motivo == "" {
		motivo = texto("mp_status_detail")
	}
	return saga.Dados{OrcamentoID: id("orcamento_id"), PagamentoID: id("pagamento_id"), Diagnostico: texto("diagnostico"), Motivo: motivo}
}

func textoOuNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func detalhe(d map[string]string) any {
	if len(d) == 0 {
		return nil
	}
	return d
}
