package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
)

// ── mock osRepo ───────────────────────────────────────────────────────────────

type mockOsRepo struct {
	buscarStatusIDFn    func(ctx context.Context, nome entity.Status) (uuid.UUID, error)
	gerarNumeroFn       func(ctx context.Context) (string, error)
	criarFn             func(ctx context.Context, os *entity.OrdemServico, sv []entity.ItemOsServico, pc []entity.ItemOsPeca) (*entity.OrdemServico, error)
	listarFn            func(ctx context.Context, p usecase.ListarOSParams) ([]*entity.OrdemServico, int, error)
	buscarPorIDFn       func(ctx context.Context, id string) (*entity.OrdemServicoCompleta, error)
	atualizarStatusErr  error
	registrarHistErr    error
	deduzirEstoqueErr   error
	deduzirChamadas     int
	relatorioFn         func(ctx context.Context, p usecase.RelatorioTempoMedioParams) ([]usecase.ItemTempoMedio, error)
	precarregarCacheErr error
}

func (m *mockOsRepo) BuscarStatusID(ctx context.Context, nome entity.Status) (uuid.UUID, error) {
	if m.buscarStatusIDFn != nil {
		return m.buscarStatusIDFn(ctx, nome)
	}
	return uuid.New(), nil
}
func (m *mockOsRepo) GerarNumeroOS(ctx context.Context) (string, error) {
	if m.gerarNumeroFn != nil {
		return m.gerarNumeroFn(ctx)
	}
	return "OS-2026-00001", nil
}
func (m *mockOsRepo) Criar(ctx context.Context, os *entity.OrdemServico, sv []entity.ItemOsServico, pc []entity.ItemOsPeca) (*entity.OrdemServico, error) {
	if m.criarFn != nil {
		return m.criarFn(ctx, os, sv, pc)
	}
	os.ID = uuid.New()
	return os, nil
}
func (m *mockOsRepo) Listar(ctx context.Context, p usecase.ListarOSParams) ([]*entity.OrdemServico, int, error) {
	if m.listarFn != nil {
		return m.listarFn(ctx, p)
	}
	return nil, 0, nil
}
func (m *mockOsRepo) BuscarPorID(ctx context.Context, id string) (*entity.OrdemServicoCompleta, error) {
	if m.buscarPorIDFn != nil {
		return m.buscarPorIDFn(ctx, id)
	}
	return &entity.OrdemServicoCompleta{OrdemServico: entity.OrdemServico{ID: uuid.New()}}, nil
}
func (m *mockOsRepo) AtualizarStatus(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *string, _, _, _, _, _ *time.Time) error {
	return m.atualizarStatusErr
}
func (m *mockOsRepo) RegistrarHistorico(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ uuid.UUID, _ *string) error {
	return m.registrarHistErr
}
func (m *mockOsRepo) DeduzirEstoquePecas(_ context.Context, _ uuid.UUID) error {
	m.deduzirChamadas++
	return m.deduzirEstoqueErr
}
func (m *mockOsRepo) RelatorioTempoMedio(ctx context.Context, p usecase.RelatorioTempoMedioParams) ([]usecase.ItemTempoMedio, error) {
	if m.relatorioFn != nil {
		return m.relatorioFn(ctx, p)
	}
	return nil, nil
}
func (m *mockOsRepo) PrecarregarStatusCache(_ context.Context) error {
	return m.precarregarCacheErr
}

// ── mocks auxiliares para OS use case ─────────────────────────────────────────

type osCliRepo struct{ cliente *entity.Cliente }

func (r *osCliRepo) Salvar(c *entity.Cliente) (*entity.Cliente, error) { return c, nil }
func (r *osCliRepo) BuscarTodos() ([]*entity.Cliente, error)           { return nil, nil }
func (r *osCliRepo) BuscarPorDocumento(_ string) (*entity.Cliente, error) {
	return &entity.Cliente{}, nil
}
func (r *osCliRepo) Atualizar(c *entity.Cliente) (*entity.Cliente, error) { return c, nil }
func (r *osCliRepo) Remover(_ string) error                               { return nil }
func (r *osCliRepo) BuscarPorID(_ string) (*entity.Cliente, error) {
	if r.cliente != nil {
		return r.cliente, nil
	}
	return &entity.Cliente{ID: uuid.New()}, nil
}

type osVeicRepo struct{ veiculo *entity.Veiculo }

func (r *osVeicRepo) Salvar(v *entity.Veiculo) (*entity.Veiculo, error)      { return v, nil }
func (r *osVeicRepo) BuscarTodos() ([]*entity.Veiculo, error)                { return nil, nil }
func (r *osVeicRepo) Atualizar(v *entity.Veiculo) (*entity.Veiculo, error)   { return v, nil }
func (r *osVeicRepo) Remover(_ string) error                                 { return nil }
func (r *osVeicRepo) BuscarPorClienteID(_ string) ([]*entity.Veiculo, error) { return nil, nil }
func (r *osVeicRepo) BuscarPorID(_ string) (*entity.Veiculo, error) {
	if r.veiculo != nil {
		return r.veiculo, nil
	}
	return &entity.Veiculo{ID: uuid.New()}, nil
}

type osSvcRepo struct{ servico *entity.Servico }

func (r *osSvcRepo) Salvar(s *entity.Servico) (*entity.Servico, error)    { return s, nil }
func (r *osSvcRepo) BuscarTodos() ([]*entity.Servico, error)              { return nil, nil }
func (r *osSvcRepo) Atualizar(s *entity.Servico) (*entity.Servico, error) { return s, nil }
func (r *osSvcRepo) Remover(_ string) error                               { return nil }
func (r *osSvcRepo) BuscarPorID(_ string) (*entity.Servico, error) {
	if r.servico != nil {
		return r.servico, nil
	}
	return &entity.Servico{ID: uuid.New(), PrecoBase: 100.0}, nil
}

type osPecaRepo struct {
	peca      *entity.Peca
	buscarErr error
}

func (r *osPecaRepo) Salvar(p *entity.Peca) (*entity.Peca, error)           { return p, nil }
func (r *osPecaRepo) BuscarTodos() ([]*entity.Peca, error)                  { return nil, nil }
func (r *osPecaRepo) Atualizar(p *entity.Peca) (*entity.Peca, error)        { return p, nil }
func (r *osPecaRepo) AtualizarEstoque(p *entity.Peca) (*entity.Peca, error) { return p, nil }
func (r *osPecaRepo) Remover(_ string) error                                { return nil }
func (r *osPecaRepo) BuscarPorID(_ string) (*entity.Peca, error) {
	if r.buscarErr != nil {
		return nil, r.buscarErr
	}
	if r.peca != nil {
		return r.peca, nil
	}
	return &entity.Peca{ID: uuid.New(), EstoqueAtual: 100}, nil
}

// ── helper ────────────────────────────────────────────────────────────────────

func osUseCase(osR *mockOsRepo) *usecase.OrdemServicoUseCase {
	return usecase.NewOrdemServicoUseCase(osR, &osCliRepo{}, &osVeicRepo{}, &osSvcRepo{}, &osPecaRepo{})
}

func osCompleta(status entity.Status) *entity.OrdemServicoCompleta {
	return &entity.OrdemServicoCompleta{
		OrdemServico: entity.OrdemServico{ID: uuid.New(), StatusNome: status, StatusID: uuid.New()},
	}
}

// ── AprovarOrcamento ──────────────────────────────────────────────────────────

func TestOSAprovarOrcamento_StatusErrado(t *testing.T) {
	for _, status := range []entity.Status{
		entity.StatusRecebida, entity.StatusEmDiagnostico,
		entity.StatusEmExecucao, entity.StatusFinalizada, entity.StatusEntregue,
	} {
		s := status
		osR := &mockOsRepo{
			buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
				return osCompleta(s), nil
			},
		}
		_, err := osUseCase(osR).AprovarOrcamento(context.Background(), uuid.New().String())
		var np *domainerros.ErrNaoProcessavel
		if !errors.As(err, &np) {
			t.Errorf("status '%s': esperava ErrNaoProcessavel, obteve: %v", s, err)
		}
	}
}

func TestOSAprovarOrcamento_Sucesso(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
	}
	_, err := osUseCase(osR).AprovarOrcamento(context.Background(), uuid.New().String())
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

// ── RejeitarOrcamento ─────────────────────────────────────────────────────────

func TestOSRejeitarOrcamento_StatusErrado(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusRecebida), nil
		},
	}
	_, err := osUseCase(osR).RejeitarOrcamento(context.Background(), uuid.New().String(), nil)
	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) {
		t.Errorf("esperava ErrNaoProcessavel, obteve: %v", err)
	}
}

func TestOSRejeitarOrcamento_Sucesso(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
	}
	motivo := "preço alto"
	_, err := osUseCase(osR).RejeitarOrcamento(context.Background(), uuid.New().String(), &motivo)
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

// ── ProcessarRespostaOrcamento (webhook) ──────────────────────────────────────

func TestOSProcessarResposta_DecisaoInvalida(t *testing.T) {
	_, err := osUseCase(&mockOsRepo{}).ProcessarRespostaOrcamento(context.Background(), uuid.New().String(), "talvez", nil)
	var ev *domainerros.ErrValidacao
	if !errors.As(err, &ev) {
		t.Errorf("esperava ErrValidacao para decisão desconhecida, obteve: %v", err)
	}
}

func TestOSProcessarResposta_AprovadoSucesso(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
	}
	_, err := osUseCase(osR).ProcessarRespostaOrcamento(context.Background(), uuid.New().String(), usecase.DecisaoAprovado, nil)
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
	if osR.deduzirChamadas != 1 {
		t.Errorf("esperava 1 dedução de estoque, obteve %d", osR.deduzirChamadas)
	}
}

func TestOSProcessarResposta_RecusadoSucesso(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
	}
	motivo := "muito caro"
	_, err := osUseCase(osR).ProcessarRespostaOrcamento(context.Background(), uuid.New().String(), usecase.DecisaoRecusado, &motivo)
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
	if osR.deduzirChamadas != 0 {
		t.Errorf("recusa não deve deduzir estoque; obteve %d deduções", osR.deduzirChamadas)
	}
}

func TestOSProcessarResposta_IdempotenteAprovado(t *testing.T) {
	// OS já aprovada anteriormente: em_execucao com AprovadoEm preenchido.
	// Reprocessar a mesma notificação não deve deduzir estoque de novo.
	aprovadoEm := time.Now()
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			oc := osCompleta(entity.StatusEmExecucao)
			oc.AprovadoEm = &aprovadoEm
			return oc, nil
		},
	}
	_, err := osUseCase(osR).ProcessarRespostaOrcamento(context.Background(), uuid.New().String(), usecase.DecisaoAprovado, nil)
	if err != nil {
		t.Errorf("esperava idempotência (sem erro), obteve: %v", err)
	}
	if osR.deduzirChamadas != 0 {
		t.Errorf("notificação repetida não deve deduzir estoque; obteve %d deduções", osR.deduzirChamadas)
	}
}

func TestOSProcessarResposta_IdempotenteRecusado(t *testing.T) {
	reprovadoEm := time.Now()
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			oc := osCompleta(entity.StatusFinalizada)
			oc.ReprovadoEm = &reprovadoEm
			return oc, nil
		},
	}
	_, err := osUseCase(osR).ProcessarRespostaOrcamento(context.Background(), uuid.New().String(), usecase.DecisaoRecusado, nil)
	if err != nil {
		t.Errorf("esperava idempotência (sem erro), obteve: %v", err)
	}
}

func TestOSProcessarResposta_EstadoInvalido(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusRecebida), nil
		},
	}
	_, err := osUseCase(osR).ProcessarRespostaOrcamento(context.Background(), uuid.New().String(), usecase.DecisaoAprovado, nil)
	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) {
		t.Errorf("esperava ErrNaoProcessavel para OS fora de aguardando_aprovacao, obteve: %v", err)
	}
}

func TestOSProcessarResposta_OSNaoEncontrada(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return nil, &domainerros.ErrNaoEncontrado{Recurso: "ordem de serviço"}
		},
	}
	_, err := osUseCase(osR).ProcessarRespostaOrcamento(context.Background(), uuid.New().String(), usecase.DecisaoRecusado, nil)
	var ne *domainerros.ErrNaoEncontrado
	if !errors.As(err, &ne) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

// ── AvancarStatus ─────────────────────────────────────────────────────────────

func TestOSAvancarStatus_TransicaoInvalida(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusEntregue), nil
		},
	}
	_, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID:       uuid.New().String(),
		NovoStatus: entity.StatusRecebida,
	})
	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) {
		t.Errorf("esperava ErrNaoProcessavel, obteve: %v", err)
	}
}

func TestOSAvancarStatus_AguardandoAprovacaoSemDiagnostico(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusEmDiagnostico), nil
		},
	}
	_, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID:       uuid.New().String(),
		NovoStatus: entity.StatusAguardandoAprovacao,
	})
	if err == nil {
		t.Error("esperava erro por falta de diagnóstico")
	}
}

func TestOSAvancarStatus_Sucesso(t *testing.T) {
	diag := "motor com desgaste"
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusEmDiagnostico), nil
		},
	}
	_, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID:        uuid.New().String(),
		NovoStatus:  entity.StatusAguardandoAprovacao,
		Diagnostico: &diag,
	})
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

// ── PassThrough: GetOS / ListarOS / ConsultarStatusPublico / Cache / Relatório ─

func TestOSGetOS_Sucesso(t *testing.T) {
	_, err := osUseCase(&mockOsRepo{}).GetOS(context.Background(), uuid.New().String())
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

func TestOSListarOS_Sucesso(t *testing.T) {
	_, _, err := osUseCase(&mockOsRepo{}).ListarOS(context.Background(), usecase.ListarOSInput{Page: 1, Limit: 10})
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

func TestOSConsultarStatusPublico_Sucesso(t *testing.T) {
	_, err := osUseCase(&mockOsRepo{}).ConsultarStatusPublico(context.Background(), uuid.New().String())
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

func TestOSInicializarStatusCache_Sucesso(t *testing.T) {
	if err := osUseCase(&mockOsRepo{}).InicializarStatusCache(context.Background()); err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

func TestOSRelatorioTempoMedio_Sucesso(t *testing.T) {
	_, err := osUseCase(&mockOsRepo{}).RelatorioTempoMedio(context.Background(), usecase.RelatorioTempoMedioInput{})
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

// ── CriarOS ───────────────────────────────────────────────────────────────────

func TestOSListarOS_ComFiltros(t *testing.T) {
	status := "recebida"
	clienteID := uuid.New().String()
	veiculoID := uuid.New().String()

	var recebido usecase.ListarOSParams
	repo := &mockOsRepo{listarFn: func(_ context.Context, p usecase.ListarOSParams) ([]*entity.OrdemServico, int, error) {
		recebido = p
		return nil, 0, nil
	}}

	_, _, err := osUseCase(repo).ListarOS(context.Background(), usecase.ListarOSInput{
		Status:    &status,
		ClienteID: &clienteID,
		VeiculoID: &veiculoID,
		Page:      1, Limit: 10,
	})
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
	if recebido.Status == nil || *recebido.Status != entity.Status(status) {
		t.Errorf("esperava status '%s' repassado ao repositório, obteve %v", status, recebido.Status)
	}
	if recebido.IncluirEncerradas {
		t.Error("IncluirEncerradas deveria ser false por padrão")
	}
}

func TestOSListarOS_IncluirEncerradas(t *testing.T) {
	var recebido usecase.ListarOSParams
	repo := &mockOsRepo{listarFn: func(_ context.Context, p usecase.ListarOSParams) ([]*entity.OrdemServico, int, error) {
		recebido = p
		return nil, 0, nil
	}}

	_, _, err := osUseCase(repo).ListarOS(context.Background(), usecase.ListarOSInput{
		IncluirEncerradas: true,
		Page:              1, Limit: 10,
	})
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
	if !recebido.IncluirEncerradas {
		t.Error("esperava IncluirEncerradas=true repassado ao repositório")
	}
}

func TestOSRelatorioTempoMedio_ComFiltros(t *testing.T) {
	servicoID := uuid.New().String()
	dataInicio := "2026-01-01"
	dataFim := "2026-12-31"
	_, err := osUseCase(&mockOsRepo{}).RelatorioTempoMedio(context.Background(), usecase.RelatorioTempoMedioInput{
		ServicoID:  &servicoID,
		DataInicio: &dataInicio,
		DataFim:    &dataFim,
	})
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}

func TestOSCriarOS_SemItensRetornaErro(t *testing.T) {
	_, err := osUseCase(&mockOsRepo{}).CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: uuid.New().String(),
		VeiculoID: uuid.New().String(),
	})
	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) {
		t.Errorf("esperava ErrNaoProcessavel, obteve: %v", err)
	}
}

func TestOSCriarOS_Sucesso(t *testing.T) {
	clienteID := uuid.New()
	veiculoID := uuid.New()
	servicoID := uuid.New()

	osR := &mockOsRepo{}
	cliR := &osCliRepo{cliente: &entity.Cliente{ID: clienteID}}
	veicR := &osVeicRepo{veiculo: &entity.Veiculo{ID: veiculoID, ClienteID: clienteID}}
	svcR := &osSvcRepo{servico: &entity.Servico{ID: servicoID, PrecoBase: 100.0}}
	pecR := &osPecaRepo{}

	uc := usecase.NewOrdemServicoUseCase(osR, cliR, veicR, svcR, pecR)
	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(),
		VeiculoID: veiculoID.String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: servicoID.String(), Quantidade: 1}},
	})
	if err != nil {
		t.Errorf("esperava sucesso, obteve: %v", err)
	}
}
