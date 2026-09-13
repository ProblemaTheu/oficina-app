package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
)

// OrdemServicoUseCase centraliza todos os casos de uso de Ordens de Serviço.
type OrdemServicoUseCase struct {
	osRepo      osRepo
	clienteRepo clienteRepo
	veiculoRepo veiculoRepo
	servicoRepo servicoRepo
	pecaRepo    pecaRepo
	notifier    notifier
	eventos     eventRecorder
}

// OSUseCaseOption configura dependências opcionais do OrdemServicoUseCase sem
// alterar a assinatura do construtor (os callers existentes seguem válidos).
type OSUseCaseOption func(*OrdemServicoUseCase)

// ComEventRecorder injeta o emissor de eventos de negócio (observabilidade).
// Sem ele, o caso de uso não emite eventos — o comportamento padrão em testes
// e no ambiente local sem New Relic.
func ComEventRecorder(r eventRecorder) OSUseCaseOption {
	return func(uc *OrdemServicoUseCase) { uc.eventos = r }
}

// NewOrdemServicoUseCase cria uma nova instância do use case. O notifier é
// opcional (pode ser nil): quando presente, o cliente é notificado nas
// mudanças de status relevantes. Dependências opcionais (ex.: emissor de
// eventos) são configuradas via OSUseCaseOption.
func NewOrdemServicoUseCase(
	osR osRepo,
	cliR clienteRepo,
	veicR veiculoRepo,
	svcR servicoRepo,
	pecR pecaRepo,
	notif notifier,
	opts ...OSUseCaseOption,
) *OrdemServicoUseCase {
	uc := &OrdemServicoUseCase{
		osRepo:      osR,
		clienteRepo: cliR,
		veiculoRepo: veicR,
		servicoRepo: svcR,
		pecaRepo:    pecR,
		notifier:    notif,
	}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}

// registrarEvento emite um evento de negócio quando há um recorder configurado.
// Nunca bloqueia nem falha o caso de uso: observabilidade é um efeito colateral.
func (uc *OrdemServicoUseCase) registrarEvento(ctx context.Context, e EventoOS) {
	if uc.eventos == nil {
		return
	}
	uc.eventos.RegistrarEventoOS(ctx, e)
}

// statusNotificaveis são as mudanças de status comunicadas ao cliente.
var statusNotificaveis = map[entity.Status]bool{
	entity.StatusAguardandoAprovacao: true,
	entity.StatusEmExecucao:          true,
	entity.StatusFinalizada:          true,
	entity.StatusEntregue:            true,
}

// notificarMudancaStatus dispara a notificação ao cliente de forma assíncrona.
// O envio nunca bloqueia nem falha a transição: erros (cliente sem e-mail,
// provedor indisponível) são apenas logados.
func (uc *OrdemServicoUseCase) notificarMudancaStatus(os *entity.OrdemServico, novoStatus entity.Status, motivo *string) {
	if uc.notifier == nil || !statusNotificaveis[novoStatus] {
		return
	}

	clienteID := os.ClienteID.String()
	numero := os.Numero

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cliente, err := uc.clienteRepo.BuscarPorID(clienteID)
		if err != nil {
			slog.Error("notificação: falha ao buscar cliente", "os", numero, "err", err)
			return
		}
		if cliente.Email == nil || *cliente.Email == "" {
			slog.Warn("notificação: cliente sem e-mail cadastrado", "os", numero, "cliente", cliente.Nome)
			return
		}

		n := NotificacaoStatus{
			DestinatarioEmail: *cliente.Email,
			DestinatarioNome:  cliente.Nome,
			NumeroOS:          numero,
			NovoStatus:        novoStatus,
			Motivo:            motivo,
		}
		if err := uc.notifier.NotificarMudancaStatus(ctx, n); err != nil {
			slog.Error("notificação: falha ao enviar", "os", numero, "status", novoStatus, "err", err)
		}
	}()
}

// CriarOSInput é o payload de criação de OS.
type CriarOSInput struct {
	ClienteID string
	VeiculoID string
	Descricao *string
	Servicos  []ItemServicoOSInput
	Pecas     []ItemPecaOSInput
}

// ItemServicoOSInput representa um item serviço para criação de OS.
type ItemServicoOSInput struct {
	ServicoID  string
	Quantidade int
}

// ItemPecaOSInput representa um item peça para criação de OS.
type ItemPecaOSInput struct {
	PecaID     string
	Quantidade int
}

// CriarOS cria uma nova Ordem de Serviço.
func (uc *OrdemServicoUseCase) CriarOS(ctx context.Context, input CriarOSInput) (*entity.OrdemServicoCompleta, error) {
	slog.Info("executando caso de uso: criar OS")

	if len(input.Servicos) == 0 && len(input.Pecas) == 0 {
		return nil, &domainerros.ErrNaoProcessavel{
			Codigo:   "VALIDATION_ERROR",
			Mensagem: "a OS deve ter ao menos 1 item",
		}
	}

	// Validar cliente
	clienteID, err := uuid.Parse(input.ClienteID)
	if err != nil {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "cliente"}
	}
	_, err = uc.clienteRepo.BuscarPorID(input.ClienteID)
	if err != nil {
		return nil, err
	}

	// Validar veículo e pertencimento ao cliente
	veiculoID, err := uuid.Parse(input.VeiculoID)
	if err != nil {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "veículo"}
	}
	veiculo, err := uc.veiculoRepo.BuscarPorID(input.VeiculoID)
	if err != nil {
		return nil, err
	}
	if veiculo.ClienteID != clienteID {
		return nil, &domainerros.ErrNaoProcessavel{
			Codigo:   "VEHICLE_NOT_OWNED_BY_CLIENT",
			Mensagem: "veículo não pertence ao cliente informado",
		}
	}

	// Resolver itens
	var itensServico []entity.ItemOsServico
	var itensPeca []entity.ItemOsPeca
	var valorTotal float64

	for _, item := range input.Servicos {
		if item.Quantidade <= 0 {
			item.Quantidade = 1
		}

		servico, err := uc.servicoRepo.BuscarPorID(item.ServicoID)
		if err != nil {
			return nil, err
		}

		preco := servico.PrecoBase

		itensServico = append(itensServico, entity.ItemOsServico{
			ServicoID:     servico.ID,
			Quantidade:    item.Quantidade,
			PrecoUnitario: preco,
		})

		valorTotal += float64(item.Quantidade) * preco
	}

	for _, item := range input.Pecas {
		if item.Quantidade <= 0 {
			item.Quantidade = 1
		}

		peca, err := uc.pecaRepo.BuscarPorID(item.PecaID)
		if err != nil {
			return nil, err
		}

		if peca.EstoqueAtual < item.Quantidade {
			return nil, &domainerros.ErrNaoProcessavel{
				Codigo:   "INSUFFICIENT_STOCK",
				Mensagem: fmt.Sprintf("estoque insuficiente para peça '%s': disponível %d, necessário %d", peca.Nome, peca.EstoqueAtual, item.Quantidade),
			}
		}

		itensPeca = append(itensPeca, entity.ItemOsPeca{
			PecaID:        peca.ID,
			Quantidade:    item.Quantidade,
			PrecoUnitario: peca.Preco,
		})

		valorTotal += float64(item.Quantidade) * peca.Preco
	}

	// Status inicial "recebida"
	statusID, err := uc.osRepo.BuscarStatusID(ctx, entity.StatusRecebida)
	if err != nil {
		return nil, err
	}

	// Gerar número sequencial
	numero, err := uc.osRepo.GerarNumeroOS(ctx)
	if err != nil {
		return nil, err
	}

	os := &entity.OrdemServico{
		Numero:     numero,
		ClienteID:  clienteID,
		VeiculoID:  veiculoID,
		StatusID:   statusID,
		StatusNome: entity.StatusRecebida,
		Descricao:  input.Descricao,
		ValorTotal: valorTotal,
	}

	if _, err := uc.osRepo.Criar(ctx, os, itensServico, itensPeca); err != nil {
		uc.registrarEvento(ctx, EventoOS{
			Numero:    numero,
			Status:    string(entity.StatusRecebida),
			Resultado: ResultadoFalha,
			Motivo:    "persistencia_os",
		})
		return nil, err
	}

	// Registrar histórico inicial
	_ = uc.osRepo.RegistrarHistorico(ctx, os.ID, nil, statusID, nil)

	uc.registrarEvento(ctx, EventoOS{
		OsID:      os.ID.String(),
		Numero:    numero,
		Status:    string(entity.StatusRecebida),
		Resultado: ResultadoSucesso,
	})

	return uc.osRepo.BuscarPorID(ctx, os.ID.String())
}

// ListarOSInput são os filtros da listagem.
type ListarOSInput struct {
	Status            *string
	ClienteID         *string
	VeiculoID         *string
	IncluirEncerradas bool
	Page              int
	Limit             int
}

// ListarOS retorna OSs paginadas.
func (uc *OrdemServicoUseCase) ListarOS(ctx context.Context, input ListarOSInput) ([]*entity.OrdemServico, int, error) {
	slog.Info("executando caso de uso: listar OSs")

	params := ListarOSParams{
		IncluirEncerradas: input.IncluirEncerradas,
		Page:              input.Page,
		Limit:             input.Limit,
	}
	if input.Status != nil {
		s := entity.Status(*input.Status)
		params.Status = &s
	}
	if input.ClienteID != nil {
		id, err := uuid.Parse(*input.ClienteID)
		if err == nil {
			params.ClienteID = &id
		}
	}
	if input.VeiculoID != nil {
		id, err := uuid.Parse(*input.VeiculoID)
		if err == nil {
			params.VeiculoID = &id
		}
	}

	return uc.osRepo.Listar(ctx, params)
}

// GetOS retorna a OS completa pelo ID.
func (uc *OrdemServicoUseCase) GetOS(ctx context.Context, id string) (*entity.OrdemServicoCompleta, error) {
	slog.Info("executando caso de uso: buscar OS", "id", id)
	return uc.osRepo.BuscarPorID(ctx, id)
}

// AvancarStatusInput é o payload para transição de status.
type AvancarStatusInput struct {
	OsID        string
	NovoStatus  entity.Status
	Diagnostico *string
	Observacao  *string
}

// AvancarStatus aplica a máquina de estados da OS.
func (uc *OrdemServicoUseCase) AvancarStatus(ctx context.Context, input AvancarStatusInput) (*entity.OrdemServicoCompleta, error) {
	slog.Info("executando caso de uso: avançar status", "id", input.OsID, "novo_status", input.NovoStatus)

	osCompleta, err := uc.osRepo.BuscarPorID(ctx, input.OsID)
	if err != nil {
		return nil, err
	}
	os := &osCompleta.OrdemServico

	if !os.StatusNome.CanTransitionTo(input.NovoStatus) {
		return nil, &domainerros.ErrNaoProcessavel{
			Codigo:   "INVALID_STATUS_TRANSITION",
			Mensagem: fmt.Sprintf("transição de '%s' para '%s' não é permitida", os.StatusNome, input.NovoStatus),
		}
	}

	if input.NovoStatus == entity.StatusAguardandoAprovacao {
		if input.Diagnostico == nil || *input.Diagnostico == "" {
			return nil, &domainerros.ErrValidacao{Mensagem: "campo 'diagnostico' é obrigatório na transição para aguardando_aprovacao"}
		}
	}

	novoStatusID, err := uc.osRepo.BuscarStatusID(ctx, input.NovoStatus)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var iniciadoEm, finalizadoEm, entregueEm *time.Time
	switch input.NovoStatus {
	case entity.StatusEmExecucao:
		iniciadoEm = &now
		if err := uc.osRepo.DeduzirEstoquePecas(ctx, os.ID); err != nil {
			uc.registrarEvento(ctx, eventoFalhaTransicao(os, input.NovoStatus, "deducao_estoque"))
			return nil, err
		}
	case entity.StatusFinalizada:
		finalizadoEm = &now
	case entity.StatusEntregue:
		entregueEm = &now
	}

	if err := uc.osRepo.AtualizarStatus(ctx, os.ID, novoStatusID, input.Diagnostico, nil, nil, iniciadoEm, finalizadoEm, entregueEm); err != nil {
		uc.registrarEvento(ctx, eventoFalhaTransicao(os, input.NovoStatus, "atualizacao_status"))
		return nil, err
	}

	_ = uc.osRepo.RegistrarHistorico(ctx, os.ID, &os.StatusID, novoStatusID, input.Observacao)

	uc.notificarMudancaStatus(os, input.NovoStatus, input.Observacao)

	uc.registrarEvento(ctx, eventoSucessoTransicao(os, input.NovoStatus, now))

	return uc.osRepo.BuscarPorID(ctx, input.OsID)
}

// eventoSucessoTransicao monta o evento de negócio de uma transição bem-sucedida.
// A duração é o tempo que a OS permaneceu no status anterior — a diferença entre
// agora e a última atualização — e alimenta o dashboard de "tempo médio por status".
func eventoSucessoTransicao(os *entity.OrdemServico, novo entity.Status, agora time.Time) EventoOS {
	return EventoOS{
		OsID:                  os.ID.String(),
		Numero:                os.Numero,
		Status:                string(novo),
		StatusAnterior:        string(os.StatusNome),
		Resultado:             ResultadoSucesso,
		DuracaoStatusSegundos: agora.Sub(os.AtualizadoEm).Seconds(),
	}
}

// eventoFalhaTransicao monta o evento de negócio de uma transição que falhou na
// persistência (não em validação de entrada) — a fonte do alerta de falha no
// processamento de OS.
func eventoFalhaTransicao(os *entity.OrdemServico, novo entity.Status, motivo string) EventoOS {
	return EventoOS{
		OsID:           os.ID.String(),
		Numero:         os.Numero,
		Status:         string(novo),
		StatusAnterior: string(os.StatusNome),
		Resultado:      ResultadoFalha,
		Motivo:         motivo,
	}
}

// AprovarOrcamento aprova o orçamento e avança para em_execucao.
func (uc *OrdemServicoUseCase) AprovarOrcamento(ctx context.Context, osID string) (*entity.OrdemServicoCompleta, error) {
	slog.Info("executando caso de uso: aprovar orçamento", "id", osID)

	osCompleta, err := uc.osRepo.BuscarPorID(ctx, osID)
	if err != nil {
		return nil, err
	}
	os := &osCompleta.OrdemServico

	if os.StatusNome != entity.StatusAguardandoAprovacao {
		return nil, &domainerros.ErrNaoProcessavel{
			Codigo:   "INVALID_STATUS_TRANSITION",
			Mensagem: fmt.Sprintf("para aprovar o orçamento, a OS deve estar em 'aguardando_aprovacao', mas está em '%s'", os.StatusNome),
		}
	}

	novoStatusID, err := uc.osRepo.BuscarStatusID(ctx, entity.StatusEmExecucao)
	if err != nil {
		return nil, err
	}

	if err := uc.osRepo.DeduzirEstoquePecas(ctx, os.ID); err != nil {
		uc.registrarEvento(ctx, eventoFalhaTransicao(os, entity.StatusEmExecucao, "deducao_estoque"))
		return nil, err
	}

	now := time.Now()
	if err := uc.osRepo.AtualizarStatus(ctx, os.ID, novoStatusID, nil, &now, nil, &now, nil, nil); err != nil {
		uc.registrarEvento(ctx, eventoFalhaTransicao(os, entity.StatusEmExecucao, "atualizacao_status"))
		return nil, err
	}

	_ = uc.osRepo.RegistrarHistorico(ctx, os.ID, &os.StatusID, novoStatusID, nil)

	uc.notificarMudancaStatus(os, entity.StatusEmExecucao, nil)

	uc.registrarEvento(ctx, eventoSucessoTransicao(os, entity.StatusEmExecucao, now))

	return uc.osRepo.BuscarPorID(ctx, osID)
}

// Decisões aceitas pelo webhook de resposta de orçamento.
const (
	DecisaoAprovado = "aprovado"
	DecisaoRecusado = "recusado"
)

// ProcessarRespostaOrcamento aplica a decisão do cliente recebida via webhook,
// reutilizando AprovarOrcamento/RejeitarOrcamento. É idempotente: se a mesma
// decisão já foi aplicada (notificação reenviada pelo provedor), retorna a OS
// atual sem produzir efeito duplicado (ex.: não deduz estoque novamente).
func (uc *OrdemServicoUseCase) ProcessarRespostaOrcamento(ctx context.Context, osID, decisao string, motivo *string) (*entity.OrdemServicoCompleta, error) {
	slog.Info("executando caso de uso: processar resposta de orçamento", "id", osID, "decisao", decisao)

	if decisao != DecisaoAprovado && decisao != DecisaoRecusado {
		return nil, &domainerros.ErrValidacao{Mensagem: fmt.Sprintf("decisão '%s' inválida: use '%s' ou '%s'", decisao, DecisaoAprovado, DecisaoRecusado)}
	}

	osCompleta, err := uc.osRepo.BuscarPorID(ctx, osID)
	if err != nil {
		return nil, err
	}
	os := &osCompleta.OrdemServico

	// Idempotência: a decisão já foi aplicada anteriormente.
	switch {
	case decisao == DecisaoAprovado && os.StatusNome == entity.StatusEmExecucao && os.AprovadoEm != nil:
		return osCompleta, nil
	case decisao == DecisaoRecusado && os.StatusNome == entity.StatusFinalizada && os.ReprovadoEm != nil:
		return osCompleta, nil
	}

	if decisao == DecisaoAprovado {
		return uc.AprovarOrcamento(ctx, osID)
	}
	return uc.RejeitarOrcamento(ctx, osID, motivo)
}

// RejeitarOrcamento rejeita o orçamento e avança para finalizada.
func (uc *OrdemServicoUseCase) RejeitarOrcamento(ctx context.Context, osID string, motivo *string) (*entity.OrdemServicoCompleta, error) {
	slog.Info("executando caso de uso: rejeitar orçamento", "id", osID)

	osCompleta, err := uc.osRepo.BuscarPorID(ctx, osID)
	if err != nil {
		return nil, err
	}
	os := &osCompleta.OrdemServico

	if os.StatusNome != entity.StatusAguardandoAprovacao {
		return nil, &domainerros.ErrNaoProcessavel{
			Codigo:   "INVALID_STATUS_TRANSITION",
			Mensagem: fmt.Sprintf("para rejeitar o orçamento, a OS deve estar em 'aguardando_aprovacao', mas está em '%s'", os.StatusNome),
		}
	}

	novoStatusID, err := uc.osRepo.BuscarStatusID(ctx, entity.StatusFinalizada)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := uc.osRepo.AtualizarStatus(ctx, os.ID, novoStatusID, nil, nil, &now, nil, &now, nil); err != nil {
		uc.registrarEvento(ctx, eventoFalhaTransicao(os, entity.StatusFinalizada, "atualizacao_status"))
		return nil, err
	}

	_ = uc.osRepo.RegistrarHistorico(ctx, os.ID, &os.StatusID, novoStatusID, motivo)

	uc.notificarMudancaStatus(os, entity.StatusFinalizada, motivo)

	uc.registrarEvento(ctx, eventoSucessoTransicao(os, entity.StatusFinalizada, now))

	return uc.osRepo.BuscarPorID(ctx, osID)
}

// ConsultarStatusPublico retorna apenas os campos públicos da OS.
func (uc *OrdemServicoUseCase) ConsultarStatusPublico(ctx context.Context, osID string) (*entity.OrdemServico, error) {
	slog.Info("executando caso de uso: consultar status público", "id", osID)

	completa, err := uc.osRepo.BuscarPorID(ctx, osID)
	if err != nil {
		return nil, err
	}
	return &completa.OrdemServico, nil
}

// RelatorioTempoMedioInput são os filtros do relatório.
type RelatorioTempoMedioInput struct {
	ServicoID  *string
	DataInicio *string
	DataFim    *string
}

// RelatorioTempoMedio calcula o tempo médio de execução por serviço.
func (uc *OrdemServicoUseCase) RelatorioTempoMedio(ctx context.Context, input RelatorioTempoMedioInput) ([]ItemTempoMedio, error) {
	slog.Info("executando caso de uso: relatório tempo médio")

	params := RelatorioTempoMedioParams{}

	if input.ServicoID != nil {
		id, err := uuid.Parse(*input.ServicoID)
		if err == nil {
			params.ServicoID = &id
		}
	}
	if input.DataInicio != nil {
		t, err := time.Parse("2006-01-02", *input.DataInicio)
		if err == nil {
			params.DataInicio = &t
		}
	}
	if input.DataFim != nil {
		t, err := time.Parse("2006-01-02", *input.DataFim)
		if err == nil {
			// Fim do dia
			fim := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			params.DataFim = &fim
		}
	}

	return uc.osRepo.RelatorioTempoMedio(ctx, params)
}

// inicializarStatusCache pré-carrega o cache de status na inicialização.
func (uc *OrdemServicoUseCase) InicializarStatusCache(ctx context.Context) error {
	return uc.osRepo.PrecarregarStatusCache(ctx)
}
