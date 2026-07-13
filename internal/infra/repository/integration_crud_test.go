//go:build integration

// Testes de integração dos repositórios de CRUD (cliente, veículo, serviço,
// peça e usuário) contra PostgreSQL real. Complementam integration_test.go,
// que foca no repositório de ordens de serviço. Mesmos requisitos:
// TEST_DATABASE_URL definido (ver ./scripts/test-integration.sh).
package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

func esperaNaoEncontrado(t *testing.T, err error) {
	t.Helper()
	var ne *domainerros.ErrNaoEncontrado
	if !errors.As(err, &ne) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func esperaConflito(t *testing.T, err error) {
	t.Helper()
	var ec *domainerros.ErrConflito
	if !errors.As(err, &ec) {
		t.Errorf("esperava ErrConflito, obteve: %v", err)
	}
}

// ── ClienteRepository ─────────────────────────────────────────────────────────

func TestIntegracaoClienteRepository_CRUD(t *testing.T) {
	repo := repository.NovoClienteRepository(db)
	sufixo := uuid.NewString()[:8]
	email := "crud-" + sufixo + "@teste.com"

	salvo, err := repo.Salvar(&entity.Cliente{
		Nome: "Cliente CRUD " + sufixo, CpfCnpj: "crud-" + sufixo, Email: &email,
	})
	if err != nil {
		t.Fatalf("Salvar: %v", err)
	}

	buscado, err := repo.BuscarPorID(salvo.ID.String())
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if buscado.Nome != salvo.Nome || buscado.CpfCnpj != salvo.CpfCnpj {
		t.Errorf("dados divergentes após round-trip: %+v vs %+v", buscado, salvo)
	}

	porDoc, err := repo.BuscarPorDocumento(salvo.CpfCnpj)
	if err != nil {
		t.Fatalf("BuscarPorDocumento: %v", err)
	}
	if porDoc.ID != salvo.ID {
		t.Errorf("BuscarPorDocumento retornou cliente errado: %s", porDoc.ID)
	}

	todos, err := repo.BuscarTodos()
	if err != nil {
		t.Fatalf("BuscarTodos: %v", err)
	}
	if len(todos) == 0 {
		t.Error("BuscarTodos deveria incluir o cliente recém-criado")
	}

	buscado.Nome = "Cliente Atualizado " + sufixo
	atualizado, err := repo.Atualizar(buscado)
	if err != nil {
		t.Fatalf("Atualizar: %v", err)
	}
	if atualizado.Nome != buscado.Nome {
		t.Errorf("nome não atualizado: %q", atualizado.Nome)
	}

	if err := repo.Remover(salvo.ID.String()); err != nil {
		t.Fatalf("Remover: %v", err)
	}
	_, err = repo.BuscarPorID(salvo.ID.String())
	esperaNaoEncontrado(t, err)
	esperaNaoEncontrado(t, repo.Remover(salvo.ID.String()))
}

func TestIntegracaoClienteRepository_DocumentoDuplicadoRetornaConflito(t *testing.T) {
	repo := repository.NovoClienteRepository(db)
	sufixo := uuid.NewString()[:8]
	doc := "dup-" + sufixo

	if _, err := repo.Salvar(&entity.Cliente{Nome: "Original", CpfCnpj: doc}); err != nil {
		t.Fatalf("Salvar original: %v", err)
	}
	_, err := repo.Salvar(&entity.Cliente{Nome: "Duplicado", CpfCnpj: doc})
	esperaConflito(t, err)
}

func TestIntegracaoClienteRepository_BuscarPorDocumentoInexistente(t *testing.T) {
	repo := repository.NovoClienteRepository(db)
	_, err := repo.BuscarPorDocumento("documento-que-nao-existe")
	esperaNaoEncontrado(t, err)
}

// ── VeiculoRepository ─────────────────────────────────────────────────────────

func TestIntegracaoVeiculoRepository_CRUD(t *testing.T) {
	clienteID, _ := criarClienteVeiculo(t)
	repo := repository.NovoVeiculoRepository(db)
	placa := "CR" + uuid.NewString()[:5]

	salvo, err := repo.Salvar(&entity.Veiculo{
		ClienteID: clienteID, Placa: placa, Marca: "Fiat", Modelo: "Argo", Ano: 2023,
	})
	if err != nil {
		t.Fatalf("Salvar: %v", err)
	}

	buscado, err := repo.BuscarPorID(salvo.ID.String())
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if buscado.Placa != placa || buscado.ClienteID != clienteID {
		t.Errorf("dados divergentes após round-trip: %+v", buscado)
	}

	doCliente, err := repo.BuscarPorClienteID(clienteID.String())
	if err != nil {
		t.Fatalf("BuscarPorClienteID: %v", err)
	}
	// criarClienteVeiculo já cria um veículo para o cliente; este é o segundo.
	if len(doCliente) < 2 {
		t.Errorf("esperava ao menos 2 veículos do cliente, obteve %d", len(doCliente))
	}

	todos, err := repo.BuscarTodos()
	if err != nil {
		t.Fatalf("BuscarTodos: %v", err)
	}
	if len(todos) == 0 {
		t.Error("BuscarTodos deveria incluir o veículo recém-criado")
	}

	cor := "vermelho"
	buscado.Modelo = "Argo Trekking"
	buscado.Cor = &cor
	atualizado, err := repo.Atualizar(buscado)
	if err != nil {
		t.Fatalf("Atualizar: %v", err)
	}
	if atualizado.Modelo != "Argo Trekking" || atualizado.Cor == nil || *atualizado.Cor != cor {
		t.Errorf("campos não atualizados: %+v", atualizado)
	}

	if err := repo.Remover(salvo.ID.String()); err != nil {
		t.Fatalf("Remover: %v", err)
	}
	_, err = repo.BuscarPorID(salvo.ID.String())
	esperaNaoEncontrado(t, err)
	esperaNaoEncontrado(t, repo.Remover(salvo.ID.String()))
}

func TestIntegracaoVeiculoRepository_PlacaDuplicadaRetornaConflito(t *testing.T) {
	clienteID, veiculoID := criarClienteVeiculo(t)
	repo := repository.NovoVeiculoRepository(db)

	existente, err := repo.BuscarPorID(veiculoID.String())
	if err != nil {
		t.Fatalf("BuscarPorID do veículo da fixture: %v", err)
	}

	_, err = repo.Salvar(&entity.Veiculo{
		ClienteID: clienteID, Placa: existente.Placa, Marca: "VW", Modelo: "Polo", Ano: 2021,
	})
	esperaConflito(t, err)
}

// ── ServicoRepository ─────────────────────────────────────────────────────────

func TestIntegracaoServicoRepository_CRUD(t *testing.T) {
	repo := repository.NovoServicoRepository(db)
	nome := "Serviço CRUD " + uuid.NewString()[:8]

	salvo, err := repo.Salvar(&entity.Servico{Nome: nome, PrecoBase: 150, TempoMinutos: 60})
	if err != nil {
		t.Fatalf("Salvar: %v", err)
	}

	buscado, err := repo.BuscarPorID(salvo.ID.String())
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if buscado.Nome != nome || buscado.PrecoBase != 150 {
		t.Errorf("dados divergentes após round-trip: %+v", buscado)
	}

	todos, err := repo.BuscarTodos()
	if err != nil {
		t.Fatalf("BuscarTodos: %v", err)
	}
	if len(todos) == 0 {
		t.Error("BuscarTodos deveria incluir o serviço recém-criado")
	}

	buscado.PrecoBase = 199.9
	atualizado, err := repo.Atualizar(buscado)
	if err != nil {
		t.Fatalf("Atualizar: %v", err)
	}
	if atualizado.PrecoBase != 199.9 {
		t.Errorf("preço não atualizado: %.2f", atualizado.PrecoBase)
	}

	if err := repo.Remover(salvo.ID.String()); err != nil {
		t.Fatalf("Remover: %v", err)
	}
	_, err = repo.BuscarPorID(salvo.ID.String())
	esperaNaoEncontrado(t, err)
	esperaNaoEncontrado(t, repo.Remover(salvo.ID.String()))
}

func TestIntegracaoServicoRepository_NomeDuplicadoRetornaConflito(t *testing.T) {
	repo := repository.NovoServicoRepository(db)
	nome := "Serviço Dup " + uuid.NewString()[:8]

	if _, err := repo.Salvar(&entity.Servico{Nome: nome, PrecoBase: 100, TempoMinutos: 30}); err != nil {
		t.Fatalf("Salvar original: %v", err)
	}
	_, err := repo.Salvar(&entity.Servico{Nome: nome, PrecoBase: 200, TempoMinutos: 45})
	esperaConflito(t, err)
}

// ── PecaRepository ────────────────────────────────────────────────────────────

func TestIntegracaoPecaRepository_CRUD(t *testing.T) {
	repo := repository.NovoPecaRepository(db)
	codigo := "crud-" + uuid.NewString()[:8]

	salvo, err := repo.Salvar(&entity.Peca{
		Nome: "Peça CRUD " + codigo, Codigo: codigo, Preco: 35.5, EstoqueAtual: 10, EstoqueMinimo: 2,
	})
	if err != nil {
		t.Fatalf("Salvar: %v", err)
	}

	buscado, err := repo.BuscarPorID(salvo.ID.String())
	if err != nil {
		t.Fatalf("BuscarPorID: %v", err)
	}
	if buscado.Codigo != codigo || buscado.EstoqueAtual != 10 {
		t.Errorf("dados divergentes após round-trip: %+v", buscado)
	}

	todos, err := repo.BuscarTodos()
	if err != nil {
		t.Fatalf("BuscarTodos: %v", err)
	}
	if len(todos) == 0 {
		t.Error("BuscarTodos deveria incluir a peça recém-criada")
	}

	buscado.Preco = 42.0
	atualizado, err := repo.Atualizar(buscado)
	if err != nil {
		t.Fatalf("Atualizar: %v", err)
	}
	if atualizado.Preco != 42.0 {
		t.Errorf("preço não atualizado: %.2f", atualizado.Preco)
	}

	buscado.EstoqueAtual = 25
	comEstoque, err := repo.AtualizarEstoque(buscado)
	if err != nil {
		t.Fatalf("AtualizarEstoque: %v", err)
	}
	if comEstoque.EstoqueAtual != 25 {
		t.Errorf("estoque não atualizado: %d", comEstoque.EstoqueAtual)
	}
	if estoqueDaPeca(t, salvo.ID) != 25 {
		t.Error("estoque não persistido no banco")
	}

	if err := repo.Remover(salvo.ID.String()); err != nil {
		t.Fatalf("Remover: %v", err)
	}
	_, err = repo.BuscarPorID(salvo.ID.String())
	esperaNaoEncontrado(t, err)
	esperaNaoEncontrado(t, repo.Remover(salvo.ID.String()))
}

func TestIntegracaoPecaRepository_CodigoDuplicadoRetornaConflito(t *testing.T) {
	repo := repository.NovoPecaRepository(db)
	codigo := "dup-" + uuid.NewString()[:8]

	if _, err := repo.Salvar(&entity.Peca{Nome: "Original", Codigo: codigo, Preco: 10}); err != nil {
		t.Fatalf("Salvar original: %v", err)
	}
	_, err := repo.Salvar(&entity.Peca{Nome: "Duplicada", Codigo: codigo, Preco: 20})
	esperaConflito(t, err)
}

// ── UsuarioRepository ─────────────────────────────────────────────────────────

func TestIntegracaoUsuarioRepository_FluxoCompleto(t *testing.T) {
	repo := repository.NovoUsuarioRepository(db)
	ctx := context.Background()

	papelID, err := repo.BuscarPapelID(ctx, "atendente")
	if err != nil {
		t.Fatalf("BuscarPapelID (papel semeado pelas migrations): %v", err)
	}

	nomePapel, err := repo.BuscarNomePapel(ctx, papelID)
	if err != nil {
		t.Fatalf("BuscarNomePapel: %v", err)
	}
	if nomePapel != "atendente" {
		t.Errorf("esperava papel 'atendente', obteve %q", nomePapel)
	}

	email := "usuario-" + uuid.NewString()[:8] + "@teste.com"
	salvo, err := repo.Salvar(ctx, &entity.Usuario{
		Nome: "Usuário Integração", Email: email, SenhaHash: "hash-qualquer", PapelID: papelID,
	})
	if err != nil {
		t.Fatalf("Salvar: %v", err)
	}

	buscado, err := repo.BuscarPorEmail(ctx, email)
	if err != nil {
		t.Fatalf("BuscarPorEmail: %v", err)
	}
	if buscado.ID != salvo.ID || buscado.SenhaHash != "hash-qualquer" {
		t.Errorf("dados divergentes após round-trip: %+v", buscado)
	}
}

func TestIntegracaoUsuarioRepository_PapelInexistente(t *testing.T) {
	repo := repository.NovoUsuarioRepository(db)
	if _, err := repo.BuscarPapelID(context.Background(), "papel-que-nao-existe"); err == nil {
		t.Error("esperava erro para papel inexistente")
	}
}

func TestIntegracaoUsuarioRepository_EmailInexistente(t *testing.T) {
	repo := repository.NovoUsuarioRepository(db)
	if _, err := repo.BuscarPorEmail(context.Background(), "ninguem@teste.com"); err == nil {
		t.Error("esperava erro para e-mail inexistente")
	}
}
