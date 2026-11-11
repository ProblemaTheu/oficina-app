package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
)

// ── fakes da saga ─────────────────────────────────────────────────────────────

// uowFake imita a transação: só "confirma" o que foi registrado se fn não falhar.
type uowFake struct {
	confirmadas []string
	execucoes   int
}

type ctxRegistro struct{}

func (u *uowFake) Executar(ctx context.Context, fn func(ctx context.Context) error) error {
	u.execucoes++
	var pend []string
	ctx = context.WithValue(ctx, ctxRegistro{}, &pend)
	if err := fn(ctx); err != nil {
		return err // rollback: nada de pend é confirmado
	}
	u.confirmadas = append(u.confirmadas, pend...)
	return nil
}

func registrar(ctx context.Context, passo string) {
	if p, ok := ctx.Value(ctxRegistro{}).(*[]string); ok {
		*p = append(*p, passo)
	}
}

type estoqueSagaFake struct{ err error }

func (e *estoqueSagaFake) Liberar(ctx context.Context, _ uuid.UUID) error {
	registrar(ctx, "liberar")
	return nil
}
func (e *estoqueSagaFake) Baixar(ctx context.Context, _ uuid.UUID) error {
	registrar(ctx, "baixar")
	return nil
}

func (e *estoqueSagaFake) Reservar(ctx context.Context, _ uuid.UUID) error {
	if e.err != nil {
		return e.err
	}
	registrar(ctx, "reserva")
	return nil
}

type sagaRepoFake struct {
	criada             saga.Saga
	etapas             []saga.Etapa
	errCriar, errEtapa error
}

func (s *sagaRepoFake) Criar(ctx context.Context, sg saga.Saga) error {
	if s.errCriar != nil {
		return s.errCriar
	}
	s.criada = sg
	registrar(ctx, "saga")
	return nil
}
func (s *sagaRepoFake) BuscarPorOS(_ context.Context, _ uuid.UUID) (*saga.Saga, error) {
	return &s.criada, nil
}
func (s *sagaRepoFake) Atualizar(_ context.Context, sg saga.Saga, _ int) error {
	s.criada = sg
	return nil
}

func (s *sagaRepoFake) RegistrarEtapa(ctx context.Context, _ uuid.UUID, e saga.Etapa, _ saga.Resultado, _ *uuid.UUID, _ any) error {
	if s.errEtapa != nil {
		return s.errEtapa
	}
	s.etapas = append(s.etapas, e)
	registrar(ctx, "etapa:"+string(e))
	return nil
}

type filaComandosFake struct {
	comandos []saga.Comando
	err      error
}

func (f *filaComandosFake) Enfileirar(ctx context.Context, c saga.Comando) error {
	if f.err != nil {
		return f.err
	}
	f.comandos = append(f.comandos, c)
	registrar(ctx, "outbox:"+c.Tipo)
	return nil
}

type cenarioSaga struct {
	uc       *usecase.OrdemServicoUseCase
	uow      *uowFake
	sagas    *sagaRepoFake
	comandos *filaComandosFake
	eventos  *fakeEventRecorder
	input    usecase.CriarOSInput
}

func novoCenarioSaga(estoque *estoqueSagaFake, comandos *filaComandosFake) cenarioSaga {
	clienteID, veiculoID := uuid.New(), uuid.New()
	desc := "barulho na suspensão"
	osR := &mockOsRepo{criarFn: func(ctx context.Context, os *entity.OrdemServico, _ []entity.ItemOsServico, _ []entity.ItemOsPeca) (*entity.OrdemServico, error) {
		os.ID = uuid.New()
		registrar(ctx, "os")
		return os, nil
	}}
	c := cenarioSaga{
		uow: &uowFake{}, sagas: &sagaRepoFake{}, comandos: comandos, eventos: &fakeEventRecorder{},
		input: usecase.CriarOSInput{
			ClienteID: clienteID.String(), VeiculoID: veiculoID.String(), Descricao: &desc,
			Pecas: []usecase.ItemPecaOSInput{{PecaID: uuid.New().String(), Quantidade: 1}},
		},
	}
	c.uc = usecase.NewOrdemServicoUseCase(osR,
		&osCliRepo{cliente: &entity.Cliente{ID: clienteID}},
		&osVeicRepo{veiculo: &entity.Veiculo{ID: veiculoID, ClienteID: clienteID, Placa: "ABC1D23", Modelo: "Gol"}},
		&osSvcRepo{}, &osPecaRepo{peca: &entity.Peca{ID: uuid.New(), Nome: "Filtro", Preco: 30, EstoqueAtual: 5}}, nil,
		usecase.ComEventRecorder(c.eventos),
		usecase.ComSaga(usecase.DependenciasSaga{
			UoW: c.uow, Estoque: estoque, Sagas: c.sagas, Comandos: comandos,
			Agora: func() time.Time { return time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC) },
		}),
	)
	return c
}

// ── testes ────────────────────────────────────────────────────────────────────

func TestCriarOSComSaga_GravaTudoNumaTransacao(t *testing.T) {
	c := novoCenarioSaga(&estoqueSagaFake{}, &filaComandosFake{})

	if _, err := c.uc.CriarOS(context.Background(), c.input); err != nil {
		t.Fatalf("CriarOS: %v", err)
	}

	if c.uow.execucoes != 1 {
		t.Fatalf("esperava 1 unidade de trabalho, veio %d", c.uow.execucoes)
	}
	esperado := []string{"os", "reserva", "saga", "etapa:T1", "outbox:IniciarDiagnostico"}
	if len(c.uow.confirmadas) != len(esperado) {
		t.Fatalf("passos confirmados = %v, esperava %v", c.uow.confirmadas, esperado)
	}
	for i := range esperado {
		if c.uow.confirmadas[i] != esperado[i] {
			t.Errorf("passo %d = %s, esperava %s", i, c.uow.confirmadas[i], esperado[i])
		}
	}
	if c.sagas.criada.Estado != saga.DiagnosticoPendente {
		t.Errorf("saga nasce em %s, esperava DIAGNOSTICO_PENDENTE", c.sagas.criada.Estado)
	}

	cmd := c.comandos.comandos[0]
	p := cmd.Payload.(saga.PayloadIniciarDiagnostico)
	if cmd.Destino != saga.DestinoExecution || cmd.SagaID != c.sagas.criada.ID || cmd.OsID != c.sagas.criada.OsID {
		t.Errorf("comando errado: %+v", cmd)
	}
	if p.Veiculo.Placa != "ABC1D23" || p.Veiculo.Modelo != "Gol" || p.NumeroOS == "" || p.Descricao != "barulho na suspensão" {
		t.Errorf("payload incompleto (o Execution não consulta o OS): %+v", p)
	}
	if len(c.eventos.eventos) != 1 || c.eventos.eventos[0].Resultado != usecase.ResultadoSucesso {
		t.Errorf("evento de sucesso não emitido: %+v", c.eventos.eventos)
	}
}

// Critério de aceite do F4-1.3: falha na outbox → nada persistido.
func TestCriarOSComSaga_FalhaNaOutboxDesfazTudo(t *testing.T) {
	c := novoCenarioSaga(&estoqueSagaFake{}, &filaComandosFake{err: errors.New("outbox indisponível")})

	_, err := c.uc.CriarOS(context.Background(), c.input)

	if err == nil {
		t.Fatal("esperava erro")
	}
	if len(c.uow.confirmadas) != 0 {
		t.Errorf("nada pode ser confirmado quando a outbox falha: %v", c.uow.confirmadas)
	}
	if len(c.eventos.eventos) != 1 || c.eventos.eventos[0].Motivo != "abertura_saga" {
		t.Errorf("esperava evento de falha 'abertura_saga': %+v", c.eventos.eventos)
	}
}

func TestCriarOSComSaga_EstoqueInsuficienteNaoCriaSaga(t *testing.T) {
	semEstoque := &domainerros.ErrNaoProcessavel{Codigo: "INSUFFICIENT_STOCK", Mensagem: "sem saldo"}
	c := novoCenarioSaga(&estoqueSagaFake{err: semEstoque}, &filaComandosFake{})

	_, err := c.uc.CriarOS(context.Background(), c.input)

	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) || np.Codigo != "INSUFFICIENT_STOCK" {
		t.Fatalf("esperava 422 INSUFFICIENT_STOCK, veio %v", err)
	}
	if c.sagas.criada.ID != uuid.Nil || len(c.comandos.comandos) != 0 || len(c.uow.confirmadas) != 0 {
		t.Error("sem reserva não nasce saga nem comando (T1)")
	}
}

// Sem relógio injetado (Agora nil → time.Now) e sem descrição: o comando sai
// com descricao vazia e o caso de uso não quebra.
func TestCriarOSComSaga_PadroesSemRelogioESemDescricao(t *testing.T) {
	c := novoCenarioSaga(&estoqueSagaFake{}, &filaComandosFake{})
	clienteID, veiculoID := uuid.New(), uuid.New()
	uc := usecase.NewOrdemServicoUseCase(&mockOsRepo{},
		&osCliRepo{cliente: &entity.Cliente{ID: clienteID}},
		&osVeicRepo{veiculo: &entity.Veiculo{ID: veiculoID, ClienteID: clienteID, Placa: "XYZ9Z99"}},
		&osSvcRepo{servico: &entity.Servico{ID: uuid.New(), PrecoBase: 50}}, &osPecaRepo{}, nil,
		usecase.ComSaga(usecase.DependenciasSaga{UoW: c.uow, Estoque: &estoqueSagaFake{}, Sagas: c.sagas, Comandos: c.comandos}),
	)
	antes := time.Now()

	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(), VeiculoID: veiculoID.String(),
		Servicos: []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.sagas.criada.CriadoEm.Before(antes.Add(-time.Second)) {
		t.Error("sem relógio injetado, a saga deve usar time.Now")
	}
	if p := c.comandos.comandos[0].Payload.(saga.PayloadIniciarDiagnostico); p.Descricao != "" || p.Veiculo.Placa != "XYZ9Z99" {
		t.Errorf("payload = %+v", p)
	}
}

// Qualquer passo da abertura que falhe desfaz a transação inteira.
func TestCriarOSComSaga_FalhaEmCadaPassoDesfazTudo(t *testing.T) {
	falha := errors.New("falhou")
	casos := map[string]func(c *cenarioSaga, os *mockOsRepo){
		"gravar OS": func(_ *cenarioSaga, os *mockOsRepo) {
			os.criarFn = func(context.Context, *entity.OrdemServico, []entity.ItemOsServico, []entity.ItemOsPeca) (*entity.OrdemServico, error) {
				return nil, falha
			}
		},
		"histórico":    func(_ *cenarioSaga, os *mockOsRepo) { os.registrarHistErr = falha },
		"criar saga":   func(c *cenarioSaga, _ *mockOsRepo) { c.sagas.errCriar = falha },
		"registrar T1": func(c *cenarioSaga, _ *mockOsRepo) { c.sagas.errEtapa = falha },
	}
	for nome, quebrar := range casos {
		t.Run(nome, func(t *testing.T) {
			c := cenarioSaga{uow: &uowFake{}, sagas: &sagaRepoFake{}, comandos: &filaComandosFake{}}
			clienteID, veiculoID := uuid.New(), uuid.New()
			osR := &mockOsRepo{}
			quebrar(&c, osR)
			uc := usecase.NewOrdemServicoUseCase(osR,
				&osCliRepo{cliente: &entity.Cliente{ID: clienteID}},
				&osVeicRepo{veiculo: &entity.Veiculo{ID: veiculoID, ClienteID: clienteID}},
				&osSvcRepo{servico: &entity.Servico{ID: uuid.New(), PrecoBase: 50}}, &osPecaRepo{}, nil,
				usecase.ComSaga(usecase.DependenciasSaga{UoW: c.uow, Estoque: &estoqueSagaFake{}, Sagas: c.sagas, Comandos: c.comandos}),
			)
			_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
				ClienteID: clienteID.String(), VeiculoID: veiculoID.String(),
				Servicos: []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
			})
			if !errors.Is(err, falha) {
				t.Fatalf("esperava a falha do passo, veio %v", err)
			}
			if len(c.uow.confirmadas) != 0 || len(c.comandos.comandos) != 0 {
				t.Errorf("nada pode ser confirmado nem enfileirado: %v / %d comandos", c.uow.confirmadas, len(c.comandos.comandos))
			}
		})
	}
}
