package usecase_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockPecaRepoUC struct {
	peca           *entity.Peca
	buscarErr      error
	atualizarEstFn func(p *entity.Peca) (*entity.Peca, error)
	salvarFn       func(p *entity.Peca) (*entity.Peca, error)
}

func (m *mockPecaRepoUC) Salvar(p *entity.Peca) (*entity.Peca, error) {
	if m.salvarFn != nil {
		return m.salvarFn(p)
	}
	p.ID = uuid.New()
	return p, nil
}
func (m *mockPecaRepoUC) BuscarTodos() ([]*entity.Peca, error) { return nil, nil }
func (m *mockPecaRepoUC) BuscarPorID(_ string) (*entity.Peca, error) {
	return m.peca, m.buscarErr
}
func (m *mockPecaRepoUC) Atualizar(p *entity.Peca) (*entity.Peca, error) { return p, nil }
func (m *mockPecaRepoUC) AtualizarEstoque(p *entity.Peca) (*entity.Peca, error) {
	if m.atualizarEstFn != nil {
		return m.atualizarEstFn(p)
	}
	return p, nil
}
func (m *mockPecaRepoUC) Remover(_ string) error { return nil }

// ── testes ────────────────────────────────────────────────────────────────────

func TestAdjustStock_EntradaIncrementaEstoque(t *testing.T) {
	id := uuid.New()
	repo := &mockPecaRepoUC{
		peca: &entity.Peca{ID: id, EstoqueAtual: 5},
	}
	uc := usecase.NewPartUseCase(repo)

	p, err := uc.AdjustStock(id.String(), "entrada", 3)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if p.EstoqueAtual != 8 {
		t.Errorf("esperava estoque 8, obteve %d", p.EstoqueAtual)
	}
}

func TestAdjustStock_SaidaInsuficienteRetornaErro(t *testing.T) {
	id := uuid.New()
	repo := &mockPecaRepoUC{
		peca: &entity.Peca{ID: id, EstoqueAtual: 2},
	}
	uc := usecase.NewPartUseCase(repo)

	_, err := uc.AdjustStock(id.String(), "saida", 5)
	if !errors.Is(err, usecase.ErrEstoqueInsuficiente) {
		t.Errorf("esperava ErrEstoqueInsuficiente, obteve: %v", err)
	}
}

func TestAdjustStock_AjusteDefineValorAbsoluto(t *testing.T) {
	id := uuid.New()
	repo := &mockPecaRepoUC{
		peca: &entity.Peca{ID: id, EstoqueAtual: 10},
	}
	uc := usecase.NewPartUseCase(repo)

	p, err := uc.AdjustStock(id.String(), "ajuste", 7)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if p.EstoqueAtual != 7 {
		t.Errorf("esperava estoque 7, obteve %d", p.EstoqueAtual)
	}
}

func TestAdjustStock_SaidaExataPermitida(t *testing.T) {
	id := uuid.New()
	repo := &mockPecaRepoUC{
		peca: &entity.Peca{ID: id, EstoqueAtual: 5},
	}
	uc := usecase.NewPartUseCase(repo)

	p, err := uc.AdjustStock(id.String(), "saida", 5)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if p.EstoqueAtual != 0 {
		t.Errorf("esperava estoque 0, obteve %d", p.EstoqueAtual)
	}
}

func TestAdjustStock_TipoInvalidoRetornaErro(t *testing.T) {
	id := uuid.New()
	repo := &mockPecaRepoUC{
		peca: &entity.Peca{ID: id, EstoqueAtual: 5},
	}
	uc := usecase.NewPartUseCase(repo)

	_, err := uc.AdjustStock(id.String(), "invalido", 1)
	if err == nil {
		t.Error("esperava erro para tipo inválido")
	}
}

func TestAdjustStock_PecaNaoEncontradaRetornaErro(t *testing.T) {
	id := uuid.New()
	repo := &mockPecaRepoUC{
		buscarErr: &domainerros.ErrNaoEncontrado{Recurso: "peça"},
	}
	uc := usecase.NewPartUseCase(repo)

	_, err := uc.AdjustStock(id.String(), "entrada", 1)
	var naoEncontrado *domainerros.ErrNaoEncontrado
	if !errors.As(err, &naoEncontrado) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func TestPartCreate_Sucesso(t *testing.T) {
	estoque := 10
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	p, err := uc.Create("Filtro", "F001", 35.0, &estoque, 5)
	if err != nil || p == nil {
		t.Errorf("esperava peça criada, obteve: %v", err)
	}
}

func TestPartCreate_PrecoNegativoRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.Create("Filtro", "F001", -1.0, nil, 5)
	if err == nil {
		t.Error("esperava erro para preço negativo")
	}
}

func TestPartCreate_NomeVazioRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.Create("", "F001", 35.0, nil, 5)
	if err == nil {
		t.Error("esperava erro para nome vazio")
	}
}

func TestPartList_Sucesso(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.List()
	if err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestPartFindByID_Sucesso(t *testing.T) {
	id := uuid.New()
	repo := &mockPecaRepoUC{peca: &entity.Peca{ID: id}}
	uc := usecase.NewPartUseCase(repo)
	p, err := uc.FindByID(id.String())
	if err != nil || p == nil {
		t.Errorf("esperava peça, obteve: %v", err)
	}
}

func TestPartUpdate_Sucesso(t *testing.T) {
	id := uuid.New()
	novoNome := "Filtro Premium"
	repo := &mockPecaRepoUC{peca: &entity.Peca{ID: id, Nome: "Filtro"}}
	uc := usecase.NewPartUseCase(repo)
	p, err := uc.Update(id.String(), &novoNome, nil, nil)
	if err != nil || p.Nome != novoNome {
		t.Errorf("esperava nome '%s', obteve: %v %v", novoNome, p, err)
	}
}

func TestPartUpdate_PrecoNegativoRetornaErro(t *testing.T) {
	id := uuid.New()
	precoNeg := -1.0
	repo := &mockPecaRepoUC{peca: &entity.Peca{ID: id}}
	uc := usecase.NewPartUseCase(repo)
	_, err := uc.Update(id.String(), nil, &precoNeg, nil)
	if err == nil {
		t.Error("esperava erro para preço negativo")
	}
}

func TestPartUpdate_EstoqueMinimoNegativoRetornaErro(t *testing.T) {
	id := uuid.New()
	estoqueNeg := -1
	repo := &mockPecaRepoUC{peca: &entity.Peca{ID: id}}
	uc := usecase.NewPartUseCase(repo)
	_, err := uc.Update(id.String(), nil, nil, &estoqueNeg)
	if err == nil {
		t.Error("esperava erro para estoque mínimo negativo")
	}
}

func TestPartDelete_Sucesso(t *testing.T) {
	id := uuid.New()
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	if err := uc.Delete(id.String()); err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestPartDelete_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	if err := uc.Delete("nao-e-uuid"); err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestPartFindByID_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.FindByID("")
	if err == nil {
		t.Error("esperava erro para id vazio")
	}
}

func TestPartFindByID_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.FindByID("nao-e-uuid")
	if err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestPartUpdate_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.Update("nao-e-uuid", nil, nil, nil)
	if err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestPartCreate_EstoqueAtualNegativoRetornaErro(t *testing.T) {
	estoqueNeg := -1
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.Create("Filtro", "F001", 35.0, &estoqueNeg, 5)
	if err == nil {
		t.Error("esperava erro para estoque atual negativo")
	}
}

func TestAdjustStock_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.AdjustStock("", "entrada", 1)
	if err == nil {
		t.Error("esperava erro para id vazio")
	}
}

func TestAdjustStock_QuantidadeNegativaRetornaErro(t *testing.T) {
	id := uuid.New()
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.AdjustStock(id.String(), "entrada", -1)
	if err == nil {
		t.Error("esperava erro para quantidade negativa")
	}
}
