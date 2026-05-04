package usecase_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockVeiculoRepoUC struct {
	salvarFn       func(v *entity.Veiculo) (*entity.Veiculo, error)
	buscarIDFn     func(id string) (*entity.Veiculo, error)
	buscarClienteFn func(clienteID string) ([]*entity.Veiculo, error)
}

func (m *mockVeiculoRepoUC) Salvar(v *entity.Veiculo) (*entity.Veiculo, error) {
	if m.salvarFn != nil {
		return m.salvarFn(v)
	}
	v.ID = uuid.New()
	return v, nil
}
func (m *mockVeiculoRepoUC) BuscarTodos() ([]*entity.Veiculo, error) { return nil, nil }
func (m *mockVeiculoRepoUC) BuscarPorID(id string) (*entity.Veiculo, error) {
	if m.buscarIDFn != nil {
		return m.buscarIDFn(id)
	}
	return nil, nil
}
func (m *mockVeiculoRepoUC) Atualizar(v *entity.Veiculo) (*entity.Veiculo, error) { return v, nil }
func (m *mockVeiculoRepoUC) Remover(_ string) error                               { return nil }
func (m *mockVeiculoRepoUC) BuscarPorClienteID(cid string) ([]*entity.Veiculo, error) {
	if m.buscarClienteFn != nil {
		return m.buscarClienteFn(cid)
	}
	return nil, nil
}

type mockClienteRepoForVehicle struct {
	buscarIDFn func(id string) (*entity.Cliente, error)
}

func (m *mockClienteRepoForVehicle) Salvar(c *entity.Cliente) (*entity.Cliente, error)  { return c, nil }
func (m *mockClienteRepoForVehicle) BuscarTodos() ([]*entity.Cliente, error)            { return nil, nil }
func (m *mockClienteRepoForVehicle) Atualizar(c *entity.Cliente) (*entity.Cliente, error) { return c, nil }
func (m *mockClienteRepoForVehicle) Remover(_ string) error                             { return nil }
func (m *mockClienteRepoForVehicle) BuscarPorID(id string) (*entity.Cliente, error) {
	if m.buscarIDFn != nil {
		return m.buscarIDFn(id)
	}
	return &entity.Cliente{ID: uuid.MustParse(id)}, nil
}

// ── testes ────────────────────────────────────────────────────────────────────

func TestVehicleCreate_PlacaDuplicadaRetornaErrConflito(t *testing.T) {
	veicRepo := &mockVeiculoRepoUC{
		salvarFn: func(_ *entity.Veiculo) (*entity.Veiculo, error) {
			return nil, &domainerros.ErrConflito{Campo: "placa"}
		},
	}
	cliRepo := &mockClienteRepoForVehicle{}
	uc := usecase.NewVehicleUseCase(veicRepo, cliRepo)

	_, err := uc.Create(uuid.New().String(), "ABC1234", "Toyota", "Corolla", 2020, nil)

	var conflito *domainerros.ErrConflito
	if !errors.As(err, &conflito) {
		t.Errorf("esperava ErrConflito, obteve: %v", err)
	}
}

func TestVehicleCreate_PlacaInvalidaRetornaErro(t *testing.T) {
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})

	_, err := uc.Create(uuid.New().String(), "INVALIDA", "Toyota", "Corolla", 2020, nil)
	if err == nil {
		t.Error("esperava erro para placa inválida")
	}
}

func TestVehicleListByCliente_ClienteInexistenteRetornaErro(t *testing.T) {
	cliRepo := &mockClienteRepoForVehicle{
		buscarIDFn: func(_ string) (*entity.Cliente, error) {
			return nil, &domainerros.ErrNaoEncontrado{Recurso: "cliente"}
		},
	}
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, cliRepo)

	_, err := uc.ListByCliente(uuid.New().String())
	if err == nil {
		t.Error("esperava erro para cliente inexistente")
	}
}

func TestVehicleCreate_AnoInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})

	_, err := uc.Create(uuid.New().String(), "ABC1234", "Toyota", "Corolla", 0, nil)
	if err == nil {
		t.Error("esperava erro para ano inválido")
	}
}

func TestVehicleList_Sucesso(t *testing.T) {
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})
	_, err := uc.List()
	if err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestVehicleFindByID_Sucesso(t *testing.T) {
	id := uuid.New()
	veicRepo := &mockVeiculoRepoUC{
		buscarIDFn: func(_ string) (*entity.Veiculo, error) {
			return &entity.Veiculo{ID: id}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(veicRepo, &mockClienteRepoForVehicle{})
	v, err := uc.FindByID(id.String())
	if err != nil || v == nil {
		t.Errorf("esperava veículo, obteve: %v", err)
	}
}

func TestVehicleDelete_Sucesso(t *testing.T) {
	id := uuid.New()
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})
	if err := uc.Delete(id.String()); err != nil {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestVehicleDelete_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})
	if err := uc.Delete("nao-e-uuid"); err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestVehicleFindByID_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})
	_, err := uc.FindByID("nao-e-uuid")
	if err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestVehicleUpdate_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})
	_, err := uc.Update("nao-e-uuid", nil, nil, nil, nil)
	if err == nil {
		t.Error("esperava erro para id inválido")
	}
}

func TestVehicleCreate_ClienteIDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})
	_, err := uc.Create("nao-e-uuid", "ABC1234", "Toyota", "Corolla", 2020, nil)
	if err == nil {
		t.Error("esperava erro para clienteID inválido")
	}
}

func TestVehicleUpdate_MarcaVaziaRetornaErro(t *testing.T) {
	id := uuid.New()
	vazio := ""
	veicRepo := &mockVeiculoRepoUC{
		buscarIDFn: func(_ string) (*entity.Veiculo, error) {
			return &entity.Veiculo{ID: id, Marca: "Toyota"}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(veicRepo, &mockClienteRepoForVehicle{})
	_, err := uc.Update(id.String(), &vazio, nil, nil, nil)
	if err == nil {
		t.Error("esperava erro para marca vazia")
	}
}

func TestVehicleUpdate_Sucesso(t *testing.T) {
	id := uuid.New()
	novoModelo := "Civic"
	novoAno := 2022
	veicRepo := &mockVeiculoRepoUC{
		buscarIDFn: func(_ string) (*entity.Veiculo, error) {
			return &entity.Veiculo{ID: id, Marca: "Honda", Modelo: "Fit", Ano: 2020}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(veicRepo, &mockClienteRepoForVehicle{})
	v, err := uc.Update(id.String(), nil, &novoModelo, &novoAno, nil)
	if err != nil || v.Modelo != novoModelo || v.Ano != novoAno {
		t.Errorf("esperava update, obteve: %v %+v", err, v)
	}
}

func TestVehicleUpdate_ModeloVazioRetornaErro(t *testing.T) {
	id := uuid.New()
	vazio := ""
	veicRepo := &mockVeiculoRepoUC{
		buscarIDFn: func(_ string) (*entity.Veiculo, error) {
			return &entity.Veiculo{ID: id}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(veicRepo, &mockClienteRepoForVehicle{})
	_, err := uc.Update(id.String(), nil, &vazio, nil, nil)
	if err == nil {
		t.Error("esperava erro para modelo vazio")
	}
}

func TestVehicleUpdate_AnoInvalidoRetornaErro(t *testing.T) {
	id := uuid.New()
	anoInvalido := 0
	veicRepo := &mockVeiculoRepoUC{
		buscarIDFn: func(_ string) (*entity.Veiculo, error) {
			return &entity.Veiculo{ID: id}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(veicRepo, &mockClienteRepoForVehicle{})
	_, err := uc.Update(id.String(), nil, nil, &anoInvalido, nil)
	if err == nil {
		t.Error("esperava erro para ano inválido")
	}
}

func TestVehicleListByCliente_Sucesso(t *testing.T) {
	clienteID := uuid.New()
	veicRepo := &mockVeiculoRepoUC{
		buscarClienteFn: func(_ string) ([]*entity.Veiculo, error) {
			return []*entity.Veiculo{{ID: uuid.New(), ClienteID: clienteID}}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(veicRepo, &mockClienteRepoForVehicle{})
	veiculos, err := uc.ListByCliente(clienteID.String())
	if err != nil || len(veiculos) == 0 {
		t.Errorf("esperava veículos, obteve: %v", err)
	}
}
