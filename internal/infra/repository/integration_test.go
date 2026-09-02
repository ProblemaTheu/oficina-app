//go:build integration

// Testes de integração do repositório contra um PostgreSQL real.
//
// Requerem a variável TEST_DATABASE_URL apontando para um banco efêmero
// (as migrations são aplicadas automaticamente). Execução local:
//
//	./scripts/test-integration.sh
//
// Em CI, o workflow integration.yml provê o Postgres como service container.
package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/ProblemaTheu/oficina-app/internal/infra/database"
	"github.com/ProblemaTheu/oficina-app/internal/infra/repository"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var db *sql.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Println("TEST_DATABASE_URL não definido — testes de integração ignorados")
		return
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("falha ao abrir conexão: %v\n", err)
		os.Exit(1)
	}

	// O banco pode ainda estar subindo — tentar por até 30s.
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		fmt.Printf("banco indisponível: %v\n", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(db); err != nil {
		fmt.Printf("falha ao aplicar migrations: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// ── fixtures ──────────────────────────────────────────────────────────────────

func criarClienteVeiculo(t *testing.T) (clienteID, veiculoID uuid.UUID) {
	t.Helper()
	clienteID, veiculoID = uuid.New(), uuid.New()

	sufixo := clienteID.String()[:8]
	if _, err := db.Exec(
		`INSERT INTO clientes (id, nome, cpf_cnpj) VALUES ($1, $2, $3)`,
		clienteID, "Cliente Integração "+sufixo, "it-"+sufixo,
	); err != nil {
		t.Fatalf("fixture cliente: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO veiculos (id, cliente_id, placa, marca, modelo, ano) VALUES ($1, $2, $3, 'VW', 'Gol', 2020)`,
		veiculoID, clienteID, "IT"+sufixo[:6],
	); err != nil {
		t.Fatalf("fixture veículo: %v", err)
	}
	return clienteID, veiculoID
}

func criarPeca(t *testing.T, estoque int, preco float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO pecas (id, nome, codigo, preco, estoque_atual, estoque_minimo) VALUES ($1, $2, $3, $4, $5, 0)`,
		id, "Peça Integração "+id.String()[:8], "it-"+id.String()[:8], preco, estoque,
	); err != nil {
		t.Fatalf("fixture peça: %v", err)
	}
	return id
}

func estoqueDaPeca(t *testing.T, pecaID uuid.UUID) int {
	t.Helper()
	var estoque int
	if err := db.QueryRow(`SELECT estoque_atual FROM pecas WHERE id = $1`, pecaID).Scan(&estoque); err != nil {
		t.Fatalf("consulta de estoque: %v", err)
	}
	return estoque
}

// criarOS persiste uma OS via repositório real com o status e criado_em dados.
func criarOS(t *testing.T, repo *repository.OrdemServicoRepository, clienteID, veiculoID uuid.UUID, status entity.Status, criadoEm time.Time, itensPeca []entity.ItemOsPeca) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	statusID, err := repo.BuscarStatusID(ctx, status)
	if err != nil {
		t.Fatalf("BuscarStatusID(%s): %v", status, err)
	}
	numero, err := repo.GerarNumeroOS(ctx)
	if err != nil {
		t.Fatalf("GerarNumeroOS: %v", err)
	}

	os := &entity.OrdemServico{
		Numero:     numero,
		ClienteID:  clienteID,
		VeiculoID:  veiculoID,
		StatusID:   statusID,
		StatusNome: status,
	}
	if _, err := repo.Criar(ctx, os, nil, itensPeca); err != nil {
		t.Fatalf("Criar OS: %v", err)
	}
	if _, err := db.Exec(`UPDATE ordens_servico SET criado_em = $1 WHERE id = $2`, criadoEm, os.ID); err != nil {
		t.Fatalf("ajuste de criado_em: %v", err)
	}
	return os.ID
}

// ── Listar: ordenação por prioridade e exclusão lógica ───────────────────────

func TestIntegracaoListar_OrdenacaoEExclusaoLogica(t *testing.T) {
	repo := repository.NovoOrdemServicoRepository(db)
	clienteID, veiculoID := criarClienteVeiculo(t)
	base := time.Now().Add(-24 * time.Hour)

	// Criadas fora da ordem esperada de retorno, com criado_em controlado:
	recebidaNova := criarOS(t, repo, clienteID, veiculoID, entity.StatusRecebida, base.Add(4*time.Hour), nil)
	emExecucao := criarOS(t, repo, clienteID, veiculoID, entity.StatusEmExecucao, base.Add(3*time.Hour), nil)
	recebidaAntiga := criarOS(t, repo, clienteID, veiculoID, entity.StatusRecebida, base.Add(1*time.Hour), nil)
	aguardando := criarOS(t, repo, clienteID, veiculoID, entity.StatusAguardandoAprovacao, base.Add(2*time.Hour), nil)
	entregue := criarOS(t, repo, clienteID, veiculoID, entity.StatusEntregue, base, nil)

	params := usecase.ListarOSParams{ClienteID: &clienteID, Page: 1, Limit: 50}
	lista, total, err := repo.Listar(context.Background(), params)
	if err != nil {
		t.Fatalf("Listar: %v", err)
	}

	// entregue fica de fora da listagem padrão
	if total != 4 || len(lista) != 4 {
		t.Fatalf("esperava 4 OSs na listagem padrão, obteve total=%d len=%d", total, len(lista))
	}
	esperado := []uuid.UUID{emExecucao, aguardando, recebidaAntiga, recebidaNova}
	for i, os := range lista {
		if os.ID != esperado[i] {
			t.Errorf("posição %d: esperava OS %s (%s), obteve %s (%s)",
				i, esperado[i], "prioridade+antiguidade", os.ID, os.StatusNome)
		}
	}

	// incluir_encerradas traz a entregue por último
	params.IncluirEncerradas = true
	lista, total, err = repo.Listar(context.Background(), params)
	if err != nil {
		t.Fatalf("Listar incluir_encerradas: %v", err)
	}
	if total != 5 || lista[len(lista)-1].ID != entregue {
		t.Errorf("com incluir_encerradas esperava 5 OSs com a entregue por último; total=%d último=%v", total, lista[len(lista)-1].StatusNome)
	}

	// filtro explícito por status ignora a exclusão padrão
	statusEntregue := entity.StatusEntregue
	lista, total, err = repo.Listar(context.Background(), usecase.ListarOSParams{
		ClienteID: &clienteID, Status: &statusEntregue, Page: 1, Limit: 50,
	})
	if err != nil {
		t.Fatalf("Listar por status: %v", err)
	}
	if total != 1 || lista[0].ID != entregue {
		t.Errorf("filtro status=entregue: esperava exatamente a OS entregue, obteve total=%d", total)
	}
}

// ── DeduzirEstoquePecas: transação real de estoque ────────────────────────────

func TestIntegracaoDeduzirEstoque_Sucesso(t *testing.T) {
	repo := repository.NovoOrdemServicoRepository(db)
	clienteID, veiculoID := criarClienteVeiculo(t)
	pecaID := criarPeca(t, 10, 25.0)

	osID := criarOS(t, repo, clienteID, veiculoID, entity.StatusAguardandoAprovacao, time.Now(), []entity.ItemOsPeca{
		{PecaID: pecaID, Quantidade: 3, PrecoUnitario: 25.0},
	})

	if err := repo.DeduzirEstoquePecas(context.Background(), osID); err != nil {
		t.Fatalf("DeduzirEstoquePecas: %v", err)
	}
	if estoque := estoqueDaPeca(t, pecaID); estoque != 7 {
		t.Errorf("esperava estoque 7 após dedução de 3 sobre 10, obteve %d", estoque)
	}
}

func TestIntegracaoDeduzirEstoque_InsuficienteFazRollback(t *testing.T) {
	repo := repository.NovoOrdemServicoRepository(db)
	clienteID, veiculoID := criarClienteVeiculo(t)
	pecaOk := criarPeca(t, 5, 10.0)
	pecaSemEstoque := criarPeca(t, 1, 10.0)

	osID := criarOS(t, repo, clienteID, veiculoID, entity.StatusAguardandoAprovacao, time.Now(), []entity.ItemOsPeca{
		{PecaID: pecaOk, Quantidade: 2, PrecoUnitario: 10.0},
		{PecaID: pecaSemEstoque, Quantidade: 100, PrecoUnitario: 10.0},
	})

	err := repo.DeduzirEstoquePecas(context.Background(), osID)
	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) {
		t.Fatalf("esperava ErrNaoProcessavel de estoque insuficiente, obteve: %v", err)
	}

	// A transação deve ser revertida por inteiro: nenhuma peça deduzida.
	if estoque := estoqueDaPeca(t, pecaOk); estoque != 5 {
		t.Errorf("rollback falhou: peça com estoque suficiente foi deduzida (esperava 5, obteve %d)", estoque)
	}
	if estoque := estoqueDaPeca(t, pecaSemEstoque); estoque != 1 {
		t.Errorf("rollback falhou: peça sem estoque foi alterada (esperava 1, obteve %d)", estoque)
	}
}

// ── AtualizarStatus: transição persistida com timestamps ─────────────────────

func TestIntegracaoAtualizarStatus_TransicaoComDiagnostico(t *testing.T) {
	repo := repository.NovoOrdemServicoRepository(db)
	clienteID, veiculoID := criarClienteVeiculo(t)
	ctx := context.Background()

	osID := criarOS(t, repo, clienteID, veiculoID, entity.StatusEmDiagnostico, time.Now(), nil)

	statusAguardando, err := repo.BuscarStatusID(ctx, entity.StatusAguardandoAprovacao)
	if err != nil {
		t.Fatalf("BuscarStatusID: %v", err)
	}

	diagnostico := "troca do alternador"
	if err := repo.AtualizarStatus(ctx, osID, statusAguardando, &diagnostico, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("AtualizarStatus: %v", err)
	}

	completa, err := repo.BuscarPorID(ctx, osID.String())
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if completa.StatusNome != entity.StatusAguardandoAprovacao {
		t.Errorf("esperava status aguardando_aprovacao, obteve %s", completa.StatusNome)
	}
	if completa.Diagnostico == nil || *completa.Diagnostico != diagnostico {
		t.Errorf("esperava diagnóstico persistido, obteve %v", completa.Diagnostico)
	}
}

func TestIntegracaoAtualizarStatus_EmExecucaoGravaIniciadoEm(t *testing.T) {
	repo := repository.NovoOrdemServicoRepository(db)
	clienteID, veiculoID := criarClienteVeiculo(t)
	ctx := context.Background()

	osID := criarOS(t, repo, clienteID, veiculoID, entity.StatusAguardandoAprovacao, time.Now(), nil)

	statusExecucao, err := repo.BuscarStatusID(ctx, entity.StatusEmExecucao)
	if err != nil {
		t.Fatalf("BuscarStatusID: %v", err)
	}

	now := time.Now()
	if err := repo.AtualizarStatus(ctx, osID, statusExecucao, nil, &now, nil, &now, nil, nil); err != nil {
		t.Fatalf("AtualizarStatus: %v", err)
	}

	completa, err := repo.BuscarPorID(ctx, osID.String())
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if completa.IniciadoEm == nil || completa.AprovadoEm == nil {
		t.Errorf("esperava iniciado_em e aprovado_em preenchidos, obteve %v / %v", completa.IniciadoEm, completa.AprovadoEm)
	}
}

// ── GerarNumeroOS ─────────────────────────────────────────────────────────────

func TestIntegracaoGerarNumeroOS_FormatoESequencia(t *testing.T) {
	repo := repository.NovoOrdemServicoRepository(db)
	ctx := context.Background()

	n1, err := repo.GerarNumeroOS(ctx)
	if err != nil {
		t.Fatalf("GerarNumeroOS: %v", err)
	}
	n2, err := repo.GerarNumeroOS(ctx)
	if err != nil {
		t.Fatalf("GerarNumeroOS: %v", err)
	}

	formato := regexp.MustCompile(`^OS-\d{4}-\d{5}$`)
	if !formato.MatchString(n1) {
		t.Errorf("número fora do formato OS-YYYY-NNNNN: %s", n1)
	}
	if n1 == n2 {
		t.Errorf("números devem ser sequenciais/únicos, obteve %s duas vezes", n1)
	}
}
