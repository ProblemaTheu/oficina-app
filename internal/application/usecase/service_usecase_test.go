package usecase_test

import (
	"errors"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockServicoRepoUC struct {
	salvarFn    func(s *entity.Servico) (*entity.Servico, error)
	buscarIDFn  func(id string) (*entity.Servico, error)
	atualizarFn func(s *entity.Servico) (*entity.Servico, error)
	removerErr  error
}

func (m *mockServicoRepoUC) Salvar(s *entity.Servico) (*entity.Servico, error) {
	if m.salvarFn != nil {
		return m.salvarFn(s)
	}
	s.ID = uuid.New()
	return s, nil
}
func (m *mockServicoRepoUC) BuscarTodos() ([]*entity.Servico, error) { return nil, nil }
func (m *mockServicoRepoUC) BuscarPorID(id string) (*entity.Servico, error) {
	if m.buscarIDFn != nil {
		return m.buscarIDFn(id)
	}
	return &entity.Servico{ID: uuid.MustParse(id)}, nil
}
func (m *mockServicoRepoUC) Atualizar(s *entity.Servico) (*entity.Servico, error) {
	if m.atualizarFn != nil {
		return m.atualizarFn(s)
	}
	return s, nil
}
func (m *mockServicoRepoUC) Remover(_ string) error { return m.removerErr }

// ── testes ────────────────────────────────────────────────────────────────────

func TestServiceCreate_NomeVazioRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Create("", nil, 100.0, 30)
	if err == nil {
		t.Error("esperava erro para nome vazio")
	}
}

func TestServiceCreate_PrecoNegativoRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Create("Troca de óleo", nil, -1.0, 30)
	if err == nil {
		t.Error("esperava erro para preço negativo")
	}
}

func TestServiceCreate_TempoZeroRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Create("Troca de óleo", nil, 100.0, 0)
	if err == nil {
		t.Error("esperava erro para tempo zero")
	}
}

func TestServiceCreate_Sucesso(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	s, err := uc.Create("Troca de óleo", nil, 100.0, 30)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if s.Nome != "Troca de óleo" {
		t.Errorf("esperava nome 'Troca de óleo', obteve '%s'", s.Nome)
	}
}

func TestServiceCreate_NomeDuplicadoRetornaErrConflito(t *testing.T) {
	repo := &mockServicoRepoUC{
		salvarFn: func(_ *entity.Servico) (*entity.Servico, error) {
			return nil, &domainerros.ErrConflito{Campo: "nome"}
		},
	}
	uc := usecase.NewServiceUseCase(repo)
	_, err := uc.Create("Troca de óleo", nil, 100.0, 30)

	var conflito *domainerros.ErrConflito
	if !errors.As(err, &conflito) {
		t.Errorf("esperava ErrConflito, obteve: %v", err)
	}
}

func TestServiceFindByID_InexistenteRetornaErro(t *testing.T) {
	id := uuid.New().String()
	repo := &mockServicoRepoUC{
		buscarIDFn: func(_ string) (*entity.Servico, error) {
			return nil, &domainerros.ErrNaoEncontrado{Recurso: "serviço"}
		},
	}
	uc := usecase.NewServiceUseCase(repo)
	_, err := uc.FindByID(id)

	var naoEncontrado *domainerros.ErrNaoEncontrado
	if !errors.As(err, &naoEncontrado) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func TestServiceFindByID_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.FindByID("")
	if err == nil {
		t.Error("esperava erro para id vazio")
	}
}

func TestServiceUpdate_Sucesso(t *testing.T) {
	id := uuid.New()
	novoNome := "Alinhamento"
	novoPreco := 150.0

	repo := &mockServicoRepoUC{
		buscarIDFn: func(_ string) (*entity.Servico, error) {
			return &entity.Servico{ID: id, Nome: "Antigo", PrecoBase: 100.0, TempoMinutos: 30}, nil
		},
	}
	uc := usecase.NewServiceUseCase(repo)

	s, err := uc.Update(id.String(), &novoNome, nil, &novoPreco, nil)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if s.Nome != novoNome {
		t.Errorf("esperava nome '%s', obteve '%s'", novoNome, s.Nome)
	}
	if s.PrecoBase != novoPreco {
		t.Errorf("esperava preço %.1f, obteve %.1f", novoPreco, s.PrecoBase)
	}
}

func TestServiceUpdate_NomeVazioRetornaErro(t *testing.T) {
	id := uuid.New()
	vazio := ""
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Update(id.String(), &vazio, nil, nil, nil)
	if err == nil {
		t.Error("esperava erro para nome vazio")
	}
}

func TestServiceUpdate_PrecoNegativoRetornaErro(t *testing.T) {
	id := uuid.New()
	precoNeg := -1.0
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Update(id.String(), nil, nil, &precoNeg, nil)
	if err == nil {
		t.Error("esperava erro para preço negativo")
	}
}

func TestServiceUpdate_TempoInvalidoRetornaErro(t *testing.T) {
	id := uuid.New()
	tempoZero := 0
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Update(id.String(), nil, nil, nil, &tempoZero)
	if err == nil {
		t.Error("esperava erro para tempo zero")
	}
}

func TestServiceDelete_Sucesso(t *testing.T) {
	id := uuid.New()
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	if err := uc.Delete(id.String()); err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestServiceList_Sucesso(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.List()
	if err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}
