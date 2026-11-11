//go:build integration

// Integração da Fase 4 (F4-1.2, F4-4.2, F4-4.3): reserva de estoque, outbox e
// inbox contra PostgreSQL real, com a migration 000007 aplicada pelo TestMain.
package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/ProblemaTheu/oficina-app/internal/infra/mensageria"
	"github.com/ProblemaTheu/oficina-app/internal/infra/orquestracao"
	"github.com/ProblemaTheu/oficina-app/internal/infra/repository"
	"github.com/google/uuid"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func emTx(t *testing.T, fn func(tx *sql.Tx) error) error {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		// Devolve em vez de t.Fatalf: emTx também roda em goroutines (teste de
		// concorrência), onde FailNow não é permitido.
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func saldo(t *testing.T, pecaID uuid.UUID) (atual, reservado int) {
	t.Helper()
	if err := db.QueryRow(`SELECT estoque_atual, estoque_reservado FROM pecas WHERE id = $1`, pecaID).
		Scan(&atual, &reservado); err != nil {
		t.Fatalf("saldo: %v", err)
	}
	return atual, reservado
}

func osComPeca(t *testing.T, pecaID uuid.UUID, qtd int) uuid.UUID {
	t.Helper()
	repo := repository.NovoOrdemServicoRepository(db)
	clienteID, veiculoID := criarClienteVeiculo(t)
	return criarOS(t, repo, clienteID, veiculoID, entity.StatusRecebida, time.Now(),
		[]entity.ItemOsPeca{{PecaID: pecaID, Quantidade: qtd, PrecoUnitario: 10}})
}

// ── Migration 000007 ──────────────────────────────────────────────────────────

func TestIntegracaoMigration000007_StatusCancelada(t *testing.T) {
	id, err := repository.NovoOrdemServicoRepository(db).BuscarStatusID(context.Background(), entity.StatusCancelada)
	if err != nil || id == uuid.Nil {
		t.Fatalf("status cancelada deve existir após a 000007: %v", err)
	}
}

func TestIntegracaoMigration000007_CheckBarraReservaMaiorQueEstoque(t *testing.T) {
	pecaID := criarPeca(t, 2, 10)
	_, err := db.Exec(`UPDATE pecas SET estoque_reservado = 3 WHERE id = $1`, pecaID)
	if err == nil {
		t.Fatal("o CHECK do banco deve barrar reserva maior que o estoque mesmo se o código errar")
	}
}

// ── Estoque: reservar, liberar, baixar ───────────────────────────────────────

func TestIntegracaoEstoque_ReservarLiberarIdempotente(t *testing.T) {
	estoque := repository.NovoEstoqueRepository()
	pecaID := criarPeca(t, 5, 10)
	osID := osComPeca(t, pecaID, 2)
	ctx := context.Background()

	for i := 0; i < 2; i++ { // 2ª chamada não pode reservar de novo
		if err := emTx(t, func(tx *sql.Tx) error { return estoque.Reservar(repository.ComTx(ctx, tx), osID) }); err != nil {
			t.Fatalf("Reservar #%d: %v", i+1, err)
		}
	}
	if atual, res := saldo(t, pecaID); atual != 5 || res != 2 {
		t.Fatalf("após reservar: atual=%d reservado=%d, esperava 5 e 2", atual, res)
	}

	for i := 0; i < 2; i++ {
		if err := emTx(t, func(tx *sql.Tx) error { return estoque.Liberar(repository.ComTx(ctx, tx), osID) }); err != nil {
			t.Fatalf("Liberar #%d: %v", i+1, err)
		}
	}
	if atual, res := saldo(t, pecaID); atual != 5 || res != 0 {
		t.Errorf("após liberar 2×: atual=%d reservado=%d, esperava 5 e 0", atual, res)
	}
}

func TestIntegracaoEstoque_BaixarConverteReservaEmSaida(t *testing.T) {
	estoque := repository.NovoEstoqueRepository()
	pecaID := criarPeca(t, 5, 10)
	osID := osComPeca(t, pecaID, 2)
	ctx := context.Background()

	if err := emTx(t, func(tx *sql.Tx) error { return estoque.Reservar(repository.ComTx(ctx, tx), osID) }); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := emTx(t, func(tx *sql.Tx) error { return estoque.Baixar(repository.ComTx(ctx, tx), osID) }); err != nil {
			t.Fatalf("Baixar #%d: %v", i+1, err)
		}
	}
	if atual, res := saldo(t, pecaID); atual != 3 || res != 0 {
		t.Errorf("após baixar 2×: atual=%d reservado=%d, esperava 3 e 0", atual, res)
	}
	// Liberar depois da baixa não devolve nada: a peça já saiu.
	if err := emTx(t, func(tx *sql.Tx) error { return estoque.Liberar(repository.ComTx(ctx, tx), osID) }); err != nil {
		t.Fatal(err)
	}
	if atual, _ := saldo(t, pecaID); atual != 3 {
		t.Errorf("Liberar após Baixar alterou o estoque: %d", atual)
	}
}

func TestIntegracaoEstoque_InsuficienteFazRollbackDeTudo(t *testing.T) {
	estoque := repository.NovoEstoqueRepository()
	pecaID := criarPeca(t, 1, 10)
	osID := osComPeca(t, pecaID, 2)

	err := emTx(t, func(tx *sql.Tx) error { return estoque.Reservar(repository.ComTx(context.Background(), tx), osID) })

	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) || np.Codigo != "INSUFFICIENT_STOCK" {
		t.Fatalf("esperava ErrNaoProcessavel INSUFFICIENT_STOCK, veio %v", err)
	}
	if _, res := saldo(t, pecaID); res != 0 {
		t.Errorf("reserva parcial persistida: reservado=%d", res)
	}
}

// Convivência com o fluxo antigo (até o F4-1.1): deduzir uma peça já reservada
// por outra OS é estoque insuficiente (422), não violação do CHECK (500).
func TestIntegracaoDeduzirEstoque_RespeitaReservaDeOutraOS(t *testing.T) {
	estoque := repository.NovoEstoqueRepository()
	repo := repository.NovoOrdemServicoRepository(db)
	pecaID := criarPeca(t, 1, 10)
	osSaga, osAntiga := osComPeca(t, pecaID, 1), osComPeca(t, pecaID, 1)

	if err := emTx(t, func(tx *sql.Tx) error { return estoque.Reservar(repository.ComTx(context.Background(), tx), osSaga) }); err != nil {
		t.Fatal(err)
	}
	err := repo.DeduzirEstoquePecas(context.Background(), osAntiga)

	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) || np.Codigo != "INSUFFICIENT_STOCK" {
		t.Fatalf("esperava INSUFFICIENT_STOCK (422), veio %v", err)
	}
	if atual, res := saldo(t, pecaID); atual != 1 || res != 1 {
		t.Errorf("saldo alterado: atual=%d reservado=%d", atual, res)
	}
}

// Critério de aceite do F4-1.2: duas OS concorrentes para a última unidade.
func TestIntegracaoEstoque_DuasOSDisputamAUltimaUnidade(t *testing.T) {
	estoque := repository.NovoEstoqueRepository()
	pecaID := criarPeca(t, 1, 10)
	osA, osB := osComPeca(t, pecaID, 1), osComPeca(t, pecaID, 1)

	// As duas transações reservam ao mesmo tempo e seguram a tx aberta até a
	// outra também ter tentado — o cenário que um SELECT-antes-do-UPDATE perderia.
	var (
		wg     sync.WaitGroup
		inicio = make(chan struct{})
		erros  = make([]error, 2)
	)
	for i, osID := range []uuid.UUID{osA, osB} {
		wg.Add(1)
		go func(i int, osID uuid.UUID) {
			defer wg.Done()
			<-inicio
			erros[i] = emTx(t, func(tx *sql.Tx) error { return estoque.Reservar(repository.ComTx(context.Background(), tx), osID) })
		}(i, osID)
	}
	close(inicio)
	wg.Wait()

	ok, insuficiente := 0, 0
	for _, err := range erros {
		var np *domainerros.ErrNaoProcessavel
		switch {
		case err == nil:
			ok++
		case errors.As(err, &np):
			insuficiente++
		default:
			t.Fatalf("erro inesperado: %v", err)
		}
	}
	if ok != 1 || insuficiente != 1 {
		t.Fatalf("esperava 1 sucesso e 1 estoque insuficiente, veio %d e %d", ok, insuficiente)
	}
	if atual, res := saldo(t, pecaID); atual != 1 || res != 1 {
		t.Errorf("saldo final atual=%d reservado=%d, esperava 1 e 1", atual, res)
	}
}

// ── Outbox ────────────────────────────────────────────────────────────────────

func limparOutbox(t *testing.T) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM outbox`); err != nil {
		t.Fatal(err)
	}
}

func envTeste(t *testing.T, tipo string) mensageria.Envelope {
	t.Helper()
	env, err := mensageria.Novo(tipo, uuid.New(), uuid.New(), "req-it", mensageria.OrigemOS, map[string]int{"n": 1}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func TestIntegracaoOutbox_GravarNaTxEPendentesEmOrdem(t *testing.T) {
	limparOutbox(t)
	outbox := repository.NovoOutboxRepository(db)
	ctx := context.Background()
	a, b, c := envTeste(t, "A"), envTeste(t, "B"), envTeste(t, "C")

	// a e b na MESMA transação (mesmo now()): a ordem tem de vir do seq.
	if err := emTx(t, func(tx *sql.Tx) error {
		if err := outbox.Gravar(repository.ComTx(ctx, tx), "execution", a, map[string]string{"traceparent": "00-x-y-01"}); err != nil {
			return err
		}
		return outbox.Gravar(repository.ComTx(ctx, tx), "billing", b, nil)
	}); err != nil {
		t.Fatal(err)
	}
	if err := emTx(t, func(tx *sql.Tx) error { return outbox.Gravar(repository.ComTx(ctx, tx), "execution", c, nil) }); err != nil {
		t.Fatal(err)
	}

	pend, err := outbox.Pendentes(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 3 || pend[0].Envelope.MessageID != a.MessageID || pend[1].Envelope.MessageID != b.MessageID || pend[2].Envelope.MessageID != c.MessageID {
		t.Fatalf("ordem/quantidade errada: %+v", pend)
	}
	if pend[0].Destino != "execution" || pend[0].Trace["traceparent"] != "00-x-y-01" || pend[1].Trace == nil {
		t.Errorf("destino/trace não persistidos: %+v", pend[0])
	}
	if n, _ := outbox.ContarPendentes(ctx); n != 3 {
		t.Errorf("ContarPendentes = %d", n)
	}
}

func TestIntegracaoOutbox_RollbackNaoDeixaMensagem(t *testing.T) {
	limparOutbox(t)
	outbox := repository.NovoOutboxRepository(db)
	_ = emTx(t, func(tx *sql.Tx) error {
		if err := outbox.Gravar(repository.ComTx(context.Background(), tx), "execution", envTeste(t, "A"), nil); err != nil {
			return err
		}
		return errors.New("efeito falhou")
	})
	if n, _ := outbox.ContarPendentes(context.Background()); n != 0 {
		t.Errorf("mensagem sobreviveu ao rollback do efeito: %d", n)
	}
}

func TestIntegracaoOutbox_EnviadaEDesistidaSaemDosPendentes(t *testing.T) {
	limparOutbox(t)
	outbox := repository.NovoOutboxRepository(db)
	ctx := context.Background()
	for _, tipo := range []string{"A", "B", "C"} {
		env := envTeste(t, tipo)
		if err := emTx(t, func(tx *sql.Tx) error { return outbox.Gravar(repository.ComTx(ctx, tx), "execution", env, nil) }); err != nil {
			t.Fatal(err)
		}
	}
	pend, _ := outbox.Pendentes(ctx, 10)

	if err := outbox.MarcarEnviada(ctx, pend[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := outbox.RegistrarFalha(ctx, pend[1].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := outbox.RegistrarFalha(ctx, pend[2].ID, true); err != nil {
		t.Fatal(err)
	}

	resto, _ := outbox.Pendentes(ctx, 10)
	if len(resto) != 1 || resto[0].ID != pend[1].ID || resto[0].Tentativas != 1 {
		t.Fatalf("pendentes após marcar = %+v; esperava só a B, com 1 tentativa", resto)
	}

	if _, err := db.Exec(`UPDATE outbox SET enviado_em = now() - interval '8 days' WHERE id = $1`, pend[0].ID); err != nil {
		t.Fatal(err)
	}
	if n, err := outbox.LimparEnviadas(ctx, 7*24*time.Hour); err != nil || n != 1 {
		t.Errorf("LimparEnviadas = %d, %v", n, err)
	}
}

func TestIntegracaoOutbox_LiderancaExclusiva(t *testing.T) {
	r1, r2 := repository.NovoOutboxRepository(db), repository.NovoOutboxRepository(db)
	ctx := context.Background()

	liberar, lider, err := r1.TentarLideranca(ctx)
	if err != nil || !lider {
		t.Fatalf("1ª réplica deveria liderar: %v %v", lider, err)
	}
	if _, lider2, err := r2.TentarLideranca(ctx); err != nil || lider2 {
		t.Fatalf("2ª réplica não pode liderar ao mesmo tempo: %v %v", lider2, err)
	}
	liberar()
	liberar2, lider2, err := r2.TentarLideranca(ctx)
	if err != nil || !lider2 {
		t.Fatalf("após liberar, a 2ª réplica deveria liderar: %v %v", lider2, err)
	}
	liberar2()
}

// Relay real (pacote mensageria) sobre a outbox Postgres, com SQS falso.
func TestIntegracaoOutbox_RelayPublicaTudoEmOrdem(t *testing.T) {
	limparOutbox(t)
	outbox := repository.NovoOutboxRepository(db)
	ctx := context.Background()
	var gravadas []uuid.UUID
	for i := 0; i < 5; i++ {
		env := envTeste(t, "Cmd")
		gravadas = append(gravadas, env.MessageID)
		if err := emTx(t, func(tx *sql.Tx) error { return outbox.Gravar(repository.ComTx(ctx, tx), "execution", env, nil) }); err != nil {
			t.Fatal(err)
		}
	}
	pub := &publicadorMemoria{}
	relay := mensageria.NovoRelay(outbox, pub, mensageria.ConfigRelay{Destinos: map[string]string{"execution": "fila"}})

	if n, err := relay.Ciclo(ctx); err != nil || n != 5 {
		t.Fatalf("Ciclo = %d, %v", n, err)
	}
	for i, id := range gravadas {
		if pub.ids[i] != id {
			t.Fatalf("posição %d: publicada %s, esperava %s", i, pub.ids[i], id)
		}
	}
	if n, _ := outbox.ContarPendentes(ctx); n != 0 {
		t.Errorf("restaram %d pendentes", n)
	}
}

type publicadorMemoria struct{ ids []uuid.UUID }

func (p *publicadorMemoria) Publicar(_ context.Context, _ string, env mensageria.Envelope, _ map[string]string) error {
	p.ids = append(p.ids, env.MessageID)
	return nil
}

// ── Inbox ─────────────────────────────────────────────────────────────────────

func TestIntegracaoInbox_SegundaEntregaEhDuplicata(t *testing.T) {
	inbox := repository.NovoInboxRepository()
	id := uuid.New()
	ctx := context.Background()

	var primeira, segunda bool
	if err := emTx(t, func(tx *sql.Tx) (err error) {
		primeira, err = inbox.Marcar(repository.ComTx(ctx, tx), id, "IniciarDiagnostico")
		return
	}); err != nil {
		t.Fatal(err)
	}
	if err := emTx(t, func(tx *sql.Tx) (err error) {
		segunda, err = inbox.Marcar(repository.ComTx(ctx, tx), id, "IniciarDiagnostico")
		return
	}); err != nil {
		t.Fatal(err)
	}
	if !primeira || segunda {
		t.Errorf("primeira=%v segunda=%v; esperava true e false", primeira, segunda)
	}
}

func TestIntegracaoInbox_RollbackDoEfeitoPermiteReprocessar(t *testing.T) {
	inbox := repository.NovoInboxRepository()
	id := uuid.New()
	ctx := context.Background()

	_ = emTx(t, func(tx *sql.Tx) error {
		if _, err := inbox.Marcar(repository.ComTx(ctx, tx), id, "T"); err != nil {
			return err
		}
		return errors.New("efeito falhou") // a reentrega tem de processar de novo
	})
	var nova bool
	if err := emTx(t, func(tx *sql.Tx) (err error) { nova, err = inbox.Marcar(repository.ComTx(ctx, tx), id, "T"); return }); err != nil {
		t.Fatal(err)
	}
	if !nova {
		t.Error("marca da inbox não pode sobreviver ao rollback do efeito")
	}
}

// ── CriarOS com saga (F4-1.3) — caso de uso real sobre os repositórios reais ──

type enfileiradorQuebrado struct{}

func (enfileiradorQuebrado) Enfileirar(context.Context, saga.Comando) error {
	return errors.New("outbox indisponível")
}

func ucComSaga(comandos interface {
	Enfileirar(context.Context, saga.Comando) error
}) *usecase.OrdemServicoUseCase {
	return usecase.NewOrdemServicoUseCase(
		repository.NovoOrdemServicoRepository(db), repository.NovoClienteRepository(db),
		repository.NovoVeiculoRepository(db), repository.NovoServicoRepository(db),
		repository.NovoPecaRepository(db), nil,
		usecase.ComSaga(usecase.DependenciasSaga{
			UoW:          repository.NovaUnitOfWork(db),
			Estoque:      repository.NovoEstoqueRepository(),
			Sagas:        repository.NovoSagaRepository(db),
			Comandos:     comandos,
			Inbox:        repository.NovoInboxRepository(),
			Cancelamento: repository.NovoOrdemServicoRepository(db),
		}),
	)
}

func contar(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("contar: %v", err)
	}
	return n
}

func TestIntegracaoCriarOSComSaga_TudoNaMesmaTransacao(t *testing.T) {
	limparOutbox(t)
	clienteID, veiculoID := criarClienteVeiculo(t)
	pecaID := criarPeca(t, 5, 30)
	outbox := repository.NovoOutboxRepository(db)
	uc := ucComSaga(orquestracao.NovoEnfileirador(outbox, nil))

	os, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(), VeiculoID: veiculoID.String(),
		Pecas: []usecase.ItemPecaOSInput{{PecaID: pecaID.String(), Quantidade: 2}},
	})
	if err != nil {
		t.Fatalf("CriarOS: %v", err)
	}

	if _, res := saldo(t, pecaID); res != 2 {
		t.Errorf("reserva = %d, esperava 2", res)
	}
	s, err := repository.NovoSagaRepository(db).BuscarPorOS(context.Background(), os.ID)
	if err != nil || s.Estado != saga.DiagnosticoPendente {
		t.Fatalf("saga = %+v (%v); esperava DIAGNOSTICO_PENDENTE", s, err)
	}
	etapas, _ := repository.NovoSagaRepository(db).Etapas(context.Background(), s.ID)
	if len(etapas) != 1 || etapas[0].Etapa != saga.EtapaAbertura || etapas[0].Resultado != saga.ResultadoOK {
		t.Errorf("etapas = %+v; esperava só T1 ok", etapas)
	}
	if n := contar(t, `SELECT count(*) FROM historicos_status WHERE os_id = $1`, os.ID); n != 1 {
		t.Errorf("histórico inicial = %d linhas (o erro era descartado com `_ =` no fluxo antigo)", n)
	}

	// Critério do F4-1.3: exatamente 1 linha pendente na outbox para a OS.
	pend, _ := outbox.Pendentes(context.Background(), 10)
	if len(pend) != 1 || pend[0].Destino != saga.DestinoExecution ||
		pend[0].Envelope.Tipo != saga.TipoIniciarDiagnostico || pend[0].Envelope.OsID != os.ID || pend[0].Envelope.SagaID != s.ID {
		t.Fatalf("outbox = %+v; esperava 1 IniciarDiagnostico para a OS", pend)
	}
}

// Critério do F4-1.3: falha forçada na outbox → nenhuma OS, reserva ou saga.
func TestIntegracaoCriarOSComSaga_FalhaNaOutboxNaoPersisteNada(t *testing.T) {
	clienteID, veiculoID := criarClienteVeiculo(t)
	pecaID := criarPeca(t, 5, 30)

	_, err := ucComSaga(enfileiradorQuebrado{}).CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(), VeiculoID: veiculoID.String(),
		Pecas: []usecase.ItemPecaOSInput{{PecaID: pecaID.String(), Quantidade: 2}},
	})
	if err == nil {
		t.Fatal("esperava erro")
	}

	if n := contar(t, `SELECT count(*) FROM ordens_servico WHERE cliente_id = $1`, clienteID); n != 0 {
		t.Errorf("OS persistida apesar da falha: %d", n)
	}
	if _, res := saldo(t, pecaID); res != 0 {
		t.Errorf("reserva persistida apesar da falha: %d", res)
	}
	if n := contar(t, `SELECT count(*) FROM sagas_os s JOIN ordens_servico o ON o.id = s.os_id WHERE o.cliente_id = $1`, clienteID); n != 0 {
		t.Errorf("saga persistida apesar da falha: %d", n)
	}
}

func TestIntegracaoCriarOSComSaga_EstoqueInsuficienteEh422(t *testing.T) {
	clienteID, veiculoID := criarClienteVeiculo(t)
	pecaID := criarPeca(t, 1, 30)
	outra := osComPeca(t, pecaID, 1) // a última unidade já reservada por outra OS
	if err := emTx(t, func(tx *sql.Tx) error {
		return repository.NovoEstoqueRepository().Reservar(repository.ComTx(context.Background(), tx), outra)
	}); err != nil {
		t.Fatal(err)
	}

	_, err := ucComSaga(orquestracao.NovoEnfileirador(repository.NovoOutboxRepository(db), nil)).CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(), VeiculoID: veiculoID.String(),
		Pecas: []usecase.ItemPecaOSInput{{PecaID: pecaID.String(), Quantidade: 1}},
	})

	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) || np.Codigo != "INSUFFICIENT_STOCK" {
		t.Fatalf("esperava INSUFFICIENT_STOCK, veio %v", err)
	}
	if n := contar(t, `SELECT count(*) FROM ordens_servico WHERE cliente_id = $1`, clienteID); n != 0 {
		t.Errorf("OS persistida sem reserva: %d", n)
	}
}

func TestIntegracaoUnitOfWork_ReentranteECriarSemTxContinuaFuncionando(t *testing.T) {
	uow := repository.NovaUnitOfWork(db)
	ctx := context.Background()
	chamadas := 0
	err := uow.Executar(ctx, func(ctx context.Context) error {
		return uow.Executar(ctx, func(context.Context) error { chamadas++; return nil }) // reaproveita a tx
	})
	if err != nil || chamadas != 1 {
		t.Fatalf("unidade aninhada: %v, %d chamadas", err, chamadas)
	}

	// Sem transação no contexto, as operações que a exigem falham claramente.
	if err := repository.NovoEstoqueRepository().Reservar(ctx, uuid.New()); !errors.Is(err, repository.ErrSemTransacao) {
		t.Errorf("esperava ErrSemTransacao, veio %v", err)
	}
}

// ── Orquestrador (F4-4.4/F4-4.5) — saga inteira contra Postgres real ──────────

type sagaEmTeste struct {
	t      *testing.T
	uc     *usecase.OrdemServicoUseCase
	outbox *repository.OutboxRepository
	osID   uuid.UUID
	sagaID uuid.UUID
	pecaID uuid.UUID
	email  string
}

func novaSagaEmTeste(t *testing.T) *sagaEmTeste {
	t.Helper()
	limparOutbox(t)
	clienteID, veiculoID := criarClienteVeiculo(t)
	email := "cliente-" + clienteID.String()[:8] + "@exemplo.com"
	if _, err := db.Exec(`UPDATE clientes SET email = $1 WHERE id = $2`, email, clienteID); err != nil {
		t.Fatal(err)
	}
	pecaID := criarPeca(t, 5, 30)
	outbox := repository.NovoOutboxRepository(db)
	uc := ucComSaga(orquestracao.NovoEnfileirador(outbox, nil))
	os, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(), VeiculoID: veiculoID.String(),
		Pecas: []usecase.ItemPecaOSInput{{PecaID: pecaID.String(), Quantidade: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := repository.NovoSagaRepository(db).BuscarPorOS(context.Background(), os.ID)
	return &sagaEmTeste{t: t, uc: uc, outbox: outbox, osID: os.ID, sagaID: s.ID, pecaID: pecaID, email: email}
}

// responder entrega uma resposta ao orquestrador, como se viesse da fila.
func (st *sagaEmTeste) responder(tipo string, payload map[string]string) uuid.UUID {
	st.t.Helper()
	corpo, _ := json.Marshal(payload)
	id := uuid.New()
	if err := st.uc.ProcessarRespostaSaga(context.Background(), usecase.RespostaSaga{
		MessageID: id, Tipo: tipo, SagaID: st.sagaID, OsID: st.osID, Payload: corpo,
	}); err != nil {
		st.t.Fatalf("%s: %v", tipo, err)
	}
	return id
}

func (st *sagaEmTeste) estado() (saga.Estado, string) {
	st.t.Helper()
	s, err := repository.NovoSagaRepository(db).BuscarPorOS(context.Background(), st.osID)
	if err != nil {
		st.t.Fatal(err)
	}
	var status string
	if err := db.QueryRow(`SELECT so.nome_status FROM ordens_servico o JOIN status_ordens so ON so.id = o.status_id WHERE o.id = $1`, st.osID).Scan(&status); err != nil {
		st.t.Fatal(err)
	}
	return s.Estado, status
}

// comandos devolve, em ordem, os tipos enviados pela outbox desta OS.
func (st *sagaEmTeste) comandos() ([]string, map[string]mensageria.Envelope) {
	st.t.Helper()
	pend, err := st.outbox.Pendentes(context.Background(), 100)
	if err != nil {
		st.t.Fatal(err)
	}
	var tipos []string
	porTipo := map[string]mensageria.Envelope{}
	for _, m := range pend {
		if m.Envelope.OsID == st.osID {
			tipos = append(tipos, m.Envelope.Tipo)
			porTipo[m.Envelope.Tipo] = m.Envelope
		}
	}
	return tipos, porTipo
}

func TestIntegracaoOrquestrador_CaminhoFelizAteFinalizada(t *testing.T) {
	st := novaSagaEmTeste(t)
	orc, pag := uuid.NewString(), uuid.NewString()

	st.responder(saga.TipoDiagnosticoIniciado, map[string]string{"tarefa_id": uuid.NewString()})
	if _, status := st.estado(); status != "em_diagnostico" {
		t.Fatalf("DiagnosticoIniciado → OS em_diagnostico, veio %s", status)
	}
	st.responder(saga.TipoDiagnosticoConcluido, map[string]string{"diagnostico": "amortecedor vazando"})
	st.responder(saga.TipoOrcamentoGerado, map[string]string{"orcamento_id": orc})
	st.responder(saga.TipoOrcamentoAprovado, map[string]string{"orcamento_id": orc})
	st.responder(saga.TipoPagamentoConfirmado, map[string]string{"pagamento_id": pag})
	if atual, res := saldo(t, st.pecaID); atual != 5 || res != 2 {
		t.Fatalf("antes do pivô o estoque só está reservado: atual=%d reservado=%d", atual, res)
	}
	st.responder(saga.TipoExecucaoIniciada, nil)
	if atual, res := saldo(t, st.pecaID); atual != 3 || res != 0 {
		t.Fatalf("no início do reparo (pivô) a reserva vira saída: atual=%d reservado=%d", atual, res)
	}
	st.responder(saga.TipoExecucaoConcluida, nil)

	estado, status := st.estado()
	if estado != saga.Concluida || status != "finalizada" {
		t.Fatalf("fim: saga %s, OS %s", estado, status)
	}
	var diag string
	_ = db.QueryRow(`SELECT diagnostico FROM ordens_servico WHERE id = $1`, st.osID).Scan(&diag)
	if diag != "amortecedor vazando" {
		t.Errorf("diagnóstico não gravado na OS: %q", diag)
	}

	tipos, env := st.comandos()
	esperado := []string{saga.TipoIniciarDiagnostico, saga.TipoGerarOrcamento, saga.TipoExecutarReparo}
	if strings.Join(tipos, ",") != strings.Join(esperado, ",") {
		t.Fatalf("comandos = %v, esperava %v", tipos, esperado)
	}
	var orcamento struct {
		Itens []struct {
			Nome  string `json:"nome"`
			Qtd   int    `json:"qtd"`
			Preco string `json:"preco"`
		} `json:"itens"`
		Diagnostico string            `json:"diagnostico"`
		Cliente     map[string]string `json:"cliente"`
	}
	if err := env[saga.TipoGerarOrcamento].LerPayload(&orcamento); err != nil {
		t.Fatal(err)
	}
	if len(orcamento.Itens) != 1 || orcamento.Itens[0].Qtd != 2 || orcamento.Itens[0].Preco != "30.00" ||
		orcamento.Diagnostico != "amortecedor vazando" || orcamento.Cliente["email"] != st.email {
		t.Errorf("GerarOrcamento não é autossuficiente: %+v", orcamento)
	}
	var reparo struct {
		Veiculo saga.Veiculo `json:"veiculo"`
	}
	_ = env[saga.TipoExecutarReparo].LerPayload(&reparo)
	if reparo.Veiculo.Placa == "" {
		t.Error("ExecutarReparo precisa da placa (falha simulada FAL*)")
	}

	etapas, _ := repository.NovoSagaRepository(db).Etapas(context.Background(), st.sagaID)
	var ok []string
	for _, e := range etapas {
		if e.Resultado == saga.ResultadoOK {
			ok = append(ok, string(e.Etapa))
		}
	}
	if strings.Join(ok, ",") != "T1,T2,T3,T4,T5,T6,T7" {
		t.Errorf("etapas ok = %v", ok)
	}
	if n := contar(t, `SELECT count(*) FROM historicos_status WHERE os_id = $1 AND observacao LIKE 'saga: %'`, st.osID); n != 4 {
		t.Errorf("histórico da saga = %d linhas (em_diagnostico, aguardando_aprovacao, em_execucao, finalizada)", n)
	}
}

// A cena do vídeo: falha ao iniciar o reparo depois do pagamento.
func TestIntegracaoOrquestrador_FalhaNoReparoCompensaAteCompensada(t *testing.T) {
	st := novaSagaEmTeste(t)
	orc := uuid.NewString()
	st.responder(saga.TipoDiagnosticoConcluido, map[string]string{"diagnostico": "x"})
	st.responder(saga.TipoOrcamentoGerado, map[string]string{"orcamento_id": orc})
	st.responder(saga.TipoOrcamentoAprovado, nil)
	st.responder(saga.TipoPagamentoConfirmado, map[string]string{"pagamento_id": uuid.NewString()})

	st.responder(saga.TipoExecucaoFalhou, map[string]string{"motivo": "falha_simulada"})

	estado, status := st.estado()
	if estado != saga.Compensando || status != "cancelada" {
		t.Fatalf("após a falha: saga %s, OS %s (OS cancela já no início da compensação)", estado, status)
	}
	var motivo string
	_ = db.QueryRow(`SELECT motivo_cancelamento FROM ordens_servico WHERE id = $1 AND cancelado_em IS NOT NULL`, st.osID).Scan(&motivo)
	if !strings.Contains(motivo, "execucao_falhou") || !strings.Contains(motivo, "falha_simulada") {
		t.Errorf("motivo do cancelamento = %q", motivo)
	}

	for _, r := range []string{saga.TipoPagamentoEstornado, saga.TipoOrcamentoCancelado, saga.TipoTarefasCanceladas} {
		st.responder(r, map[string]string{})
	}

	if estado, status = st.estado(); estado != saga.Compensada || status != "cancelada" {
		t.Fatalf("fim: saga %s, OS %s", estado, status)
	}
	if atual, res := saldo(t, st.pecaID); atual != 5 || res != 0 {
		t.Errorf("estoque deve voltar ao inicial: atual=%d reservado=%d", atual, res)
	}
	tipos, env := st.comandos()
	compensacoes := tipos[len(tipos)-3:]
	if strings.Join(compensacoes, ",") != "EstornarPagamento,CancelarOrcamento,CancelarTarefas" {
		t.Fatalf("compensações fora da ordem inversa: %v", tipos)
	}
	var estorno map[string]string
	_ = env[saga.TipoEstornarPagamento].LerPayload(&estorno)
	if estorno["orcamento_id"] != orc {
		t.Errorf("EstornarPagamento sem o orçamento: %v", estorno)
	}
}

func TestIntegracaoOrquestrador_RespostaRepetidaNaoTemEfeitoDuplo(t *testing.T) {
	st := novaSagaEmTeste(t)
	corpo, _ := json.Marshal(map[string]string{"diagnostico": "x"})
	r := usecase.RespostaSaga{MessageID: uuid.New(), Tipo: saga.TipoDiagnosticoConcluido, SagaID: st.sagaID, OsID: st.osID, Payload: corpo}
	for i := 0; i < 3; i++ {
		if err := st.uc.ProcessarRespostaSaga(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	tipos, _ := st.comandos()
	if strings.Count(strings.Join(tipos, ","), saga.TipoGerarOrcamento) != 1 {
		t.Errorf("reentrega gerou orçamento repetido: %v", tipos)
	}
	s, _ := repository.NovoSagaRepository(db).BuscarPorOS(context.Background(), st.osID)
	if s.Versao != 2 {
		t.Errorf("a saga avançou %d vezes (versão %d)", s.Versao-1, s.Versao)
	}
}

func TestIntegracaoOrquestrador_RespostaSemSaga(t *testing.T) {
	st := novaSagaEmTeste(t)
	err := st.uc.ProcessarRespostaSaga(context.Background(), usecase.RespostaSaga{
		MessageID: uuid.New(), Tipo: saga.TipoOrcamentoGerado, SagaID: uuid.New(), OsID: st.osID,
	})
	if !errors.Is(err, usecase.ErrRespostaSemSaga) {
		t.Fatalf("saga_id divergente: %v", err)
	}
	err = st.uc.ProcessarRespostaSaga(context.Background(), usecase.RespostaSaga{
		MessageID: uuid.New(), Tipo: saga.TipoOrcamentoGerado, SagaID: uuid.New(), OsID: uuid.New(),
	})
	if !errors.Is(err, usecase.ErrRespostaSemSaga) {
		t.Fatalf("OS inexistente: %v", err)
	}
}
