package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
)

// osRepoSaga embute o mock de OS e registra o que o orquestrador faz com a OS.
type osRepoSaga struct {
	*mockOsRepo
	ids        map[entity.Status]uuid.UUID
	status     []entity.Status
	diagnostic []string
	historicos []string
	os         *entity.OrdemServicoCompleta
}

func novoOsRepoSaga(statusAtual entity.Status, diagnostico *string) *osRepoSaga {
	r := &osRepoSaga{ids: map[entity.Status]uuid.UUID{}}
	for _, s := range []entity.Status{entity.StatusRecebida, entity.StatusEmDiagnostico, entity.StatusAguardandoAprovacao,
		entity.StatusEmExecucao, entity.StatusFinalizada, entity.StatusCancelada} {
		r.ids[s] = uuid.New()
	}
	r.os = &entity.OrdemServicoCompleta{
		OrdemServico: entity.OrdemServico{ID: uuid.New(), Numero: "OS-2026-00007", ClienteID: uuid.New(),
			StatusID: r.ids[statusAtual], StatusNome: statusAtual, Diagnostico: diagnostico},
		Itens:   []entity.ItemOS{{Tipo: "peca", ReferenciaID: uuid.New(), Nome: "Filtro", Quantidade: 2, PrecoUnitario: 89.9}},
		Veiculo: entity.VeiculoResumo{Placa: "FAL1A23", Modelo: "Gol"},
	}
	r.mockOsRepo = &mockOsRepo{
		buscarStatusIDFn: func(_ context.Context, n entity.Status) (uuid.UUID, error) { return r.ids[n], nil },
		buscarPorIDFn:    func(context.Context, string) (*entity.OrdemServicoCompleta, error) { return r.os, nil },
	}
	return r
}

func (r *osRepoSaga) AtualizarStatus(_ context.Context, _ uuid.UUID, id uuid.UUID, diag *string, _, _, _, _, _ *time.Time) error {
	for s, sid := range r.ids {
		if sid == id {
			r.status = append(r.status, s)
		}
	}
	if diag != nil {
		r.diagnostic = append(r.diagnostic, *diag)
	}
	return r.atualizarStatusErr
}

func (r *osRepoSaga) RegistrarHistorico(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ uuid.UUID, obs *string) error {
	r.historicos = append(r.historicos, *obs)
	return r.registrarHistErr
}

type inboxFake struct {
	vistas map[uuid.UUID]bool
	err    error
}

func (i *inboxFake) Marcar(_ context.Context, id uuid.UUID, _ string) (bool, error) {
	if i.err != nil {
		return false, i.err
	}
	if i.vistas[id] {
		return false, nil
	}
	i.vistas[id] = true
	return true, nil
}

type cancelamentoFake struct{ motivos []string }

func (c *cancelamentoFake) CancelarPelaSaga(_ context.Context, _, _ uuid.UUID, motivo string) error {
	c.motivos = append(c.motivos, motivo)
	return nil
}

type cenarioOrquestrador struct {
	uc       *usecase.OrdemServicoUseCase
	os       *osRepoSaga
	sagas    *sagaRepoFake
	comandos *filaComandosFake
	inbox    *inboxFake
	cancel   *cancelamentoFake
	uow      *uowFake
}

func novoOrquestrador(estado saga.Estado, statusOS entity.Status, diagnostico *string) cenarioOrquestrador {
	osR := novoOsRepoSaga(statusOS, diagnostico)
	sg := saga.Nova(osR.os.ID, time.Now())
	sg.Estado = estado
	c := cenarioOrquestrador{
		os: osR, sagas: &sagaRepoFake{criada: sg}, comandos: &filaComandosFake{},
		inbox: &inboxFake{vistas: map[uuid.UUID]bool{}}, cancel: &cancelamentoFake{}, uow: &uowFake{},
	}
	email := "cliente@exemplo.com"
	c.uc = usecase.NewOrdemServicoUseCase(osR, &osCliRepo{cliente: &entity.Cliente{ID: osR.os.ClienteID, Nome: "Maria", Email: &email}},
		&osVeicRepo{}, &osSvcRepo{}, &osPecaRepo{}, nil,
		usecase.ComSaga(usecase.DependenciasSaga{UoW: c.uow, Estoque: &estoqueSagaFake{}, Sagas: c.sagas, Comandos: c.comandos,
			Inbox: c.inbox, Cancelamento: c.cancel}))
	return c
}

func (c cenarioOrquestrador) resposta(tipo string, payload map[string]string) usecase.RespostaSaga {
	corpo, _ := json.Marshal(payload)
	return usecase.RespostaSaga{MessageID: uuid.New(), Tipo: tipo, SagaID: c.sagas.criada.ID, OsID: c.os.os.ID, Payload: corpo}
}

func TestProcessarResposta_DiagnosticoConcluidoGeraOrcamento(t *testing.T) {
	c := novoOrquestrador(saga.DiagnosticoPendente, entity.StatusEmDiagnostico, nil)

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, map[string]string{"diagnostico": " amortecedor "})); err != nil {
		t.Fatal(err)
	}

	if c.sagas.criada.Estado != saga.OrcamentoPendente {
		t.Errorf("saga = %s", c.sagas.criada.Estado)
	}
	if len(c.os.status) != 1 || c.os.status[0] != entity.StatusAguardandoAprovacao || c.os.diagnostic[0] != "amortecedor" {
		t.Errorf("OS: status %v, diagnóstico %v", c.os.status, c.os.diagnostic)
	}
	if len(c.os.historicos) != 1 || c.os.historicos[0] != "saga: DiagnosticoConcluido" {
		t.Errorf("histórico = %v", c.os.historicos)
	}
	cmd := c.comandos.comandos[0]
	p := cmd.Payload.(map[string]any)
	cli := p["cliente"].(map[string]string)
	if cmd.Tipo != saga.TipoGerarOrcamento || p["diagnostico"] != "amortecedor" || cli["email"] != "cliente@exemplo.com" || p["numero_os"] != "OS-2026-00007" {
		t.Fatalf("GerarOrcamento = %+v", p)
	}
	item, _ := json.Marshal(p["itens"])
	if string(item) != `[{"tipo":"peca","ref":"`+c.os.os.Itens[0].ReferenciaID.String()+`","nome":"Filtro","qtd":2,"preco":"89.90"}]` {
		t.Errorf("itens = %s (preço em string decimal, sem float)", item)
	}
}

func TestProcessarResposta_PagamentoConfirmadoGeraReparoComPlacaEDiagnosticoDaOS(t *testing.T) {
	diag := "troca do amortecedor"
	c := novoOrquestrador(saga.AguardandoPagamento, entity.StatusAguardandoAprovacao, &diag)

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoPagamentoConfirmado, map[string]string{"pagamento_id": uuid.NewString()})); err != nil {
		t.Fatal(err)
	}
	p := c.comandos.comandos[0].Payload.(map[string]any)
	if c.comandos.comandos[0].Tipo != saga.TipoExecutarReparo || p["veiculo"].(saga.Veiculo).Placa != "FAL1A23" || p["diagnostico"] != diag {
		t.Fatalf("ExecutarReparo = %+v", p)
	}
	if c.os.status[0] != entity.StatusEmExecucao || c.sagas.criada.PagamentoID == nil {
		t.Errorf("status %v, pagamento %v", c.os.status, c.sagas.criada.PagamentoID)
	}
}

func TestProcessarResposta_ExecucaoIniciadaBaixaSemHistoricoRepetido(t *testing.T) {
	c := novoOrquestrador(saga.ExecucaoPendente, entity.StatusEmExecucao, nil)

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoExecucaoIniciada, nil)); err != nil {
		t.Fatal(err)
	}
	if !contem(c.uow.confirmadas, "baixar") {
		t.Errorf("pivô deve baixar a reserva: %v", c.uow.confirmadas)
	}
	if len(c.os.status) != 1 || len(c.os.historicos) != 0 {
		t.Errorf("status igual (só iniciado_em): não registra histórico — status %v hist %v", c.os.status, c.os.historicos)
	}
}

func TestProcessarResposta_FalhaCancelaOSEIniciaCompensacao(t *testing.T) {
	c := novoOrquestrador(saga.ExecucaoPendente, entity.StatusEmExecucao, nil)
	orc := uuid.New()
	c.sagas.criada.OrcamentoID = &orc

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoExecucaoFalhou, map[string]string{"motivo": "falha_simulada"})); err != nil {
		t.Fatal(err)
	}
	if c.sagas.criada.Estado != saga.Compensando || len(c.cancel.motivos) != 1 || c.cancel.motivos[0] != "execucao_falhou: falha_simulada" {
		t.Fatalf("saga %s, cancelamentos %v", c.sagas.criada.Estado, c.cancel.motivos)
	}
	if len(c.os.status) != 0 {
		t.Error("cancelamento usa CancelarPelaSaga, não AtualizarStatus")
	}
	if len(c.os.historicos) != 1 || c.os.historicos[0] != "saga: ExecucaoFalhou — execucao_falhou: falha_simulada" {
		t.Errorf("histórico = %v", c.os.historicos)
	}
	if c.comandos.comandos[0].Tipo != saga.TipoEstornarPagamento {
		t.Errorf("1ª compensação = %s", c.comandos.comandos[0].Tipo)
	}
}

func TestProcessarResposta_UltimaCompensacaoLiberaReserva(t *testing.T) {
	c := novoOrquestrador(saga.Compensando, entity.StatusCancelada, nil)
	f := saga.EtapaOrcamento
	c.sagas.criada.EtapaFalha, c.sagas.criada.EtapaAtual = &f, saga.EtapaDiagnostico

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoTarefasCanceladas, map[string]string{"quantidade": "1"})); err != nil {
		t.Fatal(err)
	}
	if c.sagas.criada.Estado != saga.Compensada || !contem(c.uow.confirmadas, "liberar") || len(c.comandos.comandos) != 0 {
		t.Errorf("saga %s, passos %v, comandos %d", c.sagas.criada.Estado, c.uow.confirmadas, len(c.comandos.comandos))
	}
}

func TestProcessarResposta_DuplicataEIgnoradaNaoTemEfeito(t *testing.T) {
	c := novoOrquestrador(saga.DiagnosticoPendente, entity.StatusRecebida, nil)
	r := c.resposta(saga.TipoDiagnosticoConcluido, map[string]string{"diagnostico": "x"})
	_ = c.uc.ProcessarRespostaSaga(context.Background(), r)
	versao := c.sagas.criada.Versao

	if err := c.uc.ProcessarRespostaSaga(context.Background(), r); err != nil || c.sagas.criada.Versao != versao || len(c.comandos.comandos) != 1 {
		t.Fatalf("duplicata teve efeito: %v versão %d comandos %d", err, c.sagas.criada.Versao, len(c.comandos.comandos))
	}

	ig := novoOrquestrador(saga.Concluida, entity.StatusFinalizada, nil)
	if err := ig.uc.ProcessarRespostaSaga(context.Background(), ig.resposta(saga.TipoOrcamentoGerado, nil)); err != nil {
		t.Fatal(err)
	}
	if len(ig.os.status) != 0 || len(ig.comandos.comandos) != 0 || ig.sagas.etapas[0] != saga.EtapaDiagnostico {
		t.Errorf("resposta ignorada só registra a etapa: status %v comandos %d etapas %v", ig.os.status, len(ig.comandos.comandos), ig.sagas.etapas)
	}
}

func TestProcessarResposta_SemSaga(t *testing.T) {
	c := novoOrquestrador(saga.DiagnosticoPendente, entity.StatusRecebida, nil)
	r := c.resposta(saga.TipoDiagnosticoConcluido, nil)
	r.SagaID = uuid.New()
	if err := c.uc.ProcessarRespostaSaga(context.Background(), r); !errors.Is(err, usecase.ErrRespostaSemSaga) {
		t.Errorf("saga_id divergente: %v", err)
	}
}

func TestProcessarResposta_Erros(t *testing.T) {
	semSaga := usecase.NewOrdemServicoUseCase(&mockOsRepo{}, nil, nil, nil, nil, nil)
	if err := semSaga.ProcessarRespostaSaga(context.Background(), usecase.RespostaSaga{}); !errors.Is(err, usecase.ErrSagaDesligada) {
		t.Errorf("sem ComSaga: %v", err)
	}

	falha := errors.New("falhou")
	c := novoOrquestrador(saga.DiagnosticoPendente, entity.StatusRecebida, nil)
	c.inbox.err = falha
	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, nil)); !errors.Is(err, falha) {
		t.Errorf("erro da inbox: %v", err)
	}

	c = novoOrquestrador(saga.DiagnosticoPendente, entity.StatusRecebida, nil)
	c.os.atualizarStatusErr = falha
	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, nil)); !errors.Is(err, falha) || len(c.uow.confirmadas) != 0 {
		t.Errorf("erro no status da OS deve desfazer a transação: %v %v", err, c.uow.confirmadas)
	}

	c = novoOrquestrador(saga.DiagnosticoPendente, entity.StatusRecebida, nil)
	c.comandos.err = falha
	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, nil)); !errors.Is(err, falha) {
		t.Errorf("erro na outbox: %v", err)
	}

	c = novoOrquestrador(saga.DiagnosticoPendente, entity.StatusRecebida, nil)
	c.sagas.errEtapa = falha
	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, nil)); !errors.Is(err, falha) {
		t.Errorf("erro ao registrar etapa: %v", err)
	}

	c = novoOrquestrador(saga.DiagnosticoPendente, entity.StatusRecebida, nil)
	c.os.buscarPorIDFn = func(context.Context, string) (*entity.OrdemServicoCompleta, error) {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "ordem de serviço"}
	}
	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, nil)); err == nil {
		t.Error("OS sumiu no meio: erro esperado")
	}
}

func TestProcessarResposta_PayloadInvalidoViraDadosVazios(t *testing.T) {
	c := novoOrquestrador(saga.OrcamentoPendente, entity.StatusAguardandoAprovacao, nil)
	r := c.resposta(saga.TipoOrcamentoGerado, nil)
	r.Payload = []byte(`{quebrado`)
	if err := c.uc.ProcessarRespostaSaga(context.Background(), r); err != nil || c.sagas.criada.Estado != saga.AguardandoAprovacao {
		t.Errorf("a decisão depende do tipo, não do payload: %v %s", err, c.sagas.criada.Estado)
	}
}

func TestProcessarResposta_MotivoAlternativoDoMercadoPago(t *testing.T) {
	c := novoOrquestrador(saga.AguardandoPagamento, entity.StatusAguardandoAprovacao, nil)
	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoPagamentoRecusado, map[string]string{"mp_status_detail": "cc_rejected_other_reason"})); err != nil {
		t.Fatal(err)
	}
	if c.cancel.motivos[0] != "pagamento_recusado: cc_rejected_other_reason" {
		t.Errorf("motivo = %q", c.cancel.motivos[0])
	}
}

func contem(lista []string, v string) bool {
	for _, x := range lista {
		if x == v {
			return true
		}
	}
	return false
}
