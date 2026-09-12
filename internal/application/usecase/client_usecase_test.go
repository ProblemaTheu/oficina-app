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

type mockClienteRepoUC struct {
	salvarFn    func(c *entity.Cliente) (*entity.Cliente, error)
	buscarIDFn  func(id string) (*entity.Cliente, error)
	atualizarFn func(c *entity.Cliente) (*entity.Cliente, error)
}

func (m *mockClienteRepoUC) Salvar(c *entity.Cliente) (*entity.Cliente, error) {
	if m.salvarFn != nil {
		return m.salvarFn(c)
	}
	c.ID = uuid.New()
	return c, nil
}
func (m *mockClienteRepoUC) BuscarTodos() ([]*entity.Cliente, error) { return nil, nil }
func (m *mockClienteRepoUC) BuscarPorDocumento(_ string) (*entity.Cliente, error) {
	return &entity.Cliente{}, nil
}
func (m *mockClienteRepoUC) BuscarPorID(id string) (*entity.Cliente, error) {
	if m.buscarIDFn != nil {
		return m.buscarIDFn(id)
	}
	return nil, nil
}
func (m *mockClienteRepoUC) Atualizar(c *entity.Cliente) (*entity.Cliente, error) {
	if m.atualizarFn != nil {
		return m.atualizarFn(c)
	}
	return c, nil
}
func (m *mockClienteRepoUC) Remover(_ string) error { return nil }

// ── testes ────────────────────────────────────────────────────────────────────

func TestClientCreate_CPFDuplicadoRetornaErrConflito(t *testing.T) {
	repo := &mockClienteRepoUC{
		salvarFn: func(_ *entity.Cliente) (*entity.Cliente, error) {
			return nil, &domainerros.ErrConflito{Campo: "cpf_cnpj"}
		},
	}
	uc := usecase.NewClientUseCase(repo)

	_, err := uc.Create("João", "529.982.247-25", nil, nil)

	var conflito *domainerros.ErrConflito
	if !errors.As(err, &conflito) {
		t.Errorf("esperava ErrConflito, obteve: %v", err)
	}
}

func TestClientFindByID_InexistenteRetornaErrNaoEncontrado(t *testing.T) {
	id := uuid.New().String()
	repo := &mockClienteRepoUC{
		buscarIDFn: func(_ string) (*entity.Cliente, error) {
			return nil, &domainerros.ErrNaoEncontrado{Recurso: "cliente"}
		},
	}
	uc := usecase.NewClientUseCase(repo)

	_, err := uc.FindByID(id)

	var naoEncontrado *domainerros.ErrNaoEncontrado
	if !errors.As(err, &naoEncontrado) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func TestClientUpdate_PersisteDados(t *testing.T) {
	id := uuid.New()
	nome := "Novo Nome"

	repo := &mockClienteRepoUC{
		buscarIDFn: func(_ string) (*entity.Cliente, error) {
			return &entity.Cliente{ID: id, Nome: "Nome Antigo"}, nil
		},
		atualizarFn: func(c *entity.Cliente) (*entity.Cliente, error) {
			return c, nil
		},
	}
	uc := usecase.NewClientUseCase(repo)

	updated, err := uc.Update(id.String(), &nome, nil, nil)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if updated.Nome != nome {
		t.Errorf("esperava nome '%s', obteve '%s'", nome, updated.Nome)
	}
}

func TestClientCreate_DocumentoInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})

	_, err := uc.Create("João", "111.111.111-11", nil, nil)
	if err == nil {
		t.Error("esperava erro para documento inválido")
	}
}

func TestClientCreate_NomeVazioRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})

	_, err := uc.Create("", "529.982.247-25", nil, nil)
	if err == nil {
		t.Error("esperava erro para nome vazio")
	}
}

func TestClientFindByID_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	_, err := uc.FindByID("")
	if err == nil {
		t.Error("esperava erro para id vazio")
	}
}

func TestClientDelete_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	if err := uc.Delete("nao-e-uuid"); err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestClientUpdate_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	nome := "Novo"
	_, err := uc.Update("nao-e-uuid", &nome, nil, nil)
	if err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestClientUpdate_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	nome := "Novo"
	_, err := uc.Update("", &nome, nil, nil)
	if err == nil {
		t.Error("esperava erro para id vazio")
	}
}

func TestClientList_Sucesso(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	_, err := uc.List()
	if err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestClientDelete_Sucesso(t *testing.T) {
	id := uuid.New()
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	if err := uc.Delete(id.String()); err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestClientDelete_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	if err := uc.Delete(""); err == nil {
		t.Error("esperava erro para id vazio")
	}
}

func TestClientFindByDocumento_Sucesso(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	if _, err := uc.FindByDocumento("11144477735"); err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestClientFindByDocumento_VazioRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	_, err := uc.FindByDocumento("")
	var ev *domainerros.ErrValidacao
	if !errors.As(err, &ev) {
		t.Errorf("esperava ErrValidacao para documento vazio, obteve: %v", err)
	}
}
