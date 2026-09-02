package usecase_test

import (
	"errors"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
)

// Este arquivo cobre ramos de validação e de erro dos use cases de CRUD que
// ficaram de fora dos arquivos de teste principais, reutilizando os mocks
// já definidos em cada *_usecase_test.go.

func esperaErrValidacao(t *testing.T, err error) {
	t.Helper()
	var ev *domainerros.ErrValidacao
	if !errors.As(err, &ev) {
		t.Errorf("esperava ErrValidacao, obteve: %v", err)
	}
}

// ── ClientUseCase ─────────────────────────────────────────────────────────────

func TestClientCreate_Sucesso(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	cliente, err := uc.Create("Maria", "529.982.247-25", nil, nil)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if cliente.ID == uuid.Nil {
		t.Error("cliente criado deve ter ID atribuído")
	}
	if cliente.CpfCnpj != "52998224725" {
		t.Errorf("documento deve ser normalizado para apenas dígitos, obteve %q", cliente.CpfCnpj)
	}
}

func TestClientCreate_DocumentoVazioRetornaErro(t *testing.T) {
	uc := usecase.NewClientUseCase(&mockClienteRepoUC{})
	_, err := uc.Create("Maria", "", nil, nil)
	esperaErrValidacao(t, err)
}

func TestClientUpdate_ClienteInexistenteRetornaErro(t *testing.T) {
	repo := &mockClienteRepoUC{
		buscarIDFn: func(_ string) (*entity.Cliente, error) {
			return nil, &domainerros.ErrNaoEncontrado{Recurso: "cliente"}
		},
	}
	uc := usecase.NewClientUseCase(repo)
	_, err := uc.Update(uuid.New().String(), nil, nil, nil)
	var ne *domainerros.ErrNaoEncontrado
	if !errors.As(err, &ne) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func TestClientUpdate_CamposNilNaoAlteram(t *testing.T) {
	id := uuid.New()
	email := "original@teste.com"
	repo := &mockClienteRepoUC{
		buscarIDFn: func(_ string) (*entity.Cliente, error) {
			return &entity.Cliente{ID: id, Nome: "Original", Email: &email}, nil
		},
	}
	uc := usecase.NewClientUseCase(repo)
	cliente, err := uc.Update(id.String(), nil, nil, nil)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if cliente.Nome != "Original" || cliente.Email == nil || *cliente.Email != email {
		t.Errorf("campos nil não devem alterar o cliente: %+v", cliente)
	}
}

// ── VehicleUseCase ────────────────────────────────────────────────────────────

func vehicleUC() *usecase.VehicleUseCase {
	return usecase.NewVehicleUseCase(&mockVeiculoRepoUC{}, &mockClienteRepoForVehicle{})
}

func TestVehicleCreate_ClienteIDVazioRetornaErro(t *testing.T) {
	_, err := vehicleUC().Create("", "ABC1D23", "VW", "Gol", 2020, nil)
	esperaErrValidacao(t, err)
}

func TestVehicleCreate_MarcaVaziaRetornaErro(t *testing.T) {
	_, err := vehicleUC().Create(uuid.New().String(), "ABC1D23", "  ", "Gol", 2020, nil)
	esperaErrValidacao(t, err)
}

func TestVehicleCreate_ModeloVazioRetornaErro(t *testing.T) {
	_, err := vehicleUC().Create(uuid.New().String(), "ABC1D23", "VW", "", 2020, nil)
	esperaErrValidacao(t, err)
}

func TestVehicleFindByID_IDVazioRetornaErro(t *testing.T) {
	_, err := vehicleUC().FindByID("")
	esperaErrValidacao(t, err)
}

func TestVehicleUpdate_IDVazioRetornaErro(t *testing.T) {
	_, err := vehicleUC().Update("", nil, nil, nil, nil)
	esperaErrValidacao(t, err)
}

func TestVehicleUpdate_VeiculoInexistenteRetornaErro(t *testing.T) {
	repo := &mockVeiculoRepoUC{
		buscarIDFn: func(_ string) (*entity.Veiculo, error) {
			return nil, &domainerros.ErrNaoEncontrado{Recurso: "veículo"}
		},
	}
	uc := usecase.NewVehicleUseCase(repo, &mockClienteRepoForVehicle{})
	_, err := uc.Update(uuid.New().String(), nil, nil, nil, nil)
	var ne *domainerros.ErrNaoEncontrado
	if !errors.As(err, &ne) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func TestVehicleUpdate_CorAtualizada(t *testing.T) {
	repo := &mockVeiculoRepoUC{
		buscarIDFn: func(id string) (*entity.Veiculo, error) {
			return &entity.Veiculo{ID: uuid.MustParse(id), Marca: "VW", Modelo: "Gol", Ano: 2020}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(repo, &mockClienteRepoForVehicle{})
	cor := "vermelho"
	veiculo, err := uc.Update(uuid.New().String(), nil, nil, nil, &cor)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if veiculo.Cor == nil || *veiculo.Cor != cor {
		t.Errorf("cor não atualizada: %v", veiculo.Cor)
	}
}

func TestVehicleDelete_IDVazioRetornaErro(t *testing.T) {
	esperaErrValidacao(t, vehicleUC().Delete(""))
}

func TestVehicleListByCliente_ClienteIDVazioRetornaErro(t *testing.T) {
	_, err := vehicleUC().ListByCliente("")
	esperaErrValidacao(t, err)
}

// ── ServiceUseCase ────────────────────────────────────────────────────────────

func TestServiceFindByID_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.FindByID("nao-e-uuid")
	esperaErrValidacao(t, err)
}

func TestServiceUpdate_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Update("", nil, nil, nil, nil)
	esperaErrValidacao(t, err)
}

func TestServiceUpdate_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	_, err := uc.Update("nao-e-uuid", nil, nil, nil, nil)
	esperaErrValidacao(t, err)
}

func TestServiceUpdate_ServicoInexistenteRetornaErro(t *testing.T) {
	repo := &mockServicoRepoUC{
		buscarIDFn: func(_ string) (*entity.Servico, error) {
			return nil, &domainerros.ErrNaoEncontrado{Recurso: "serviço"}
		},
	}
	uc := usecase.NewServiceUseCase(repo)
	_, err := uc.Update(uuid.New().String(), nil, nil, nil, nil)
	var ne *domainerros.ErrNaoEncontrado
	if !errors.As(err, &ne) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func TestServiceUpdate_DescricaoAtualizada(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	descricao := "descrição nova"
	servico, err := uc.Update(uuid.New().String(), nil, &descricao, nil, nil)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if servico.Descricao == nil || *servico.Descricao != descricao {
		t.Errorf("descrição não atualizada: %v", servico.Descricao)
	}
}

func TestServiceDelete_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	esperaErrValidacao(t, uc.Delete(""))
}

func TestServiceDelete_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})
	esperaErrValidacao(t, uc.Delete("nao-e-uuid"))
}

// ── PartUseCase ───────────────────────────────────────────────────────────────

func TestPartCreate_CodigoVazioRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.Create("Peça", "  ", 10.0, nil, 1)
	esperaErrValidacao(t, err)
}

func TestPartCreate_EstoqueMinimoNegativoRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.Create("Peça", "PC-01", 10.0, nil, -1)
	esperaErrValidacao(t, err)
}

func TestPartUpdate_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.Update("", nil, nil, nil)
	esperaErrValidacao(t, err)
}

func TestPartUpdate_PecaInexistenteRetornaErro(t *testing.T) {
	repo := &mockPecaRepoUC{buscarErr: &domainerros.ErrNaoEncontrado{Recurso: "peça"}}
	uc := usecase.NewPartUseCase(repo)
	_, err := uc.Update(uuid.New().String(), nil, nil, nil)
	var ne *domainerros.ErrNaoEncontrado
	if !errors.As(err, &ne) {
		t.Errorf("esperava ErrNaoEncontrado, obteve: %v", err)
	}
}

func TestPartUpdate_NomeVazioRetornaErro(t *testing.T) {
	repo := &mockPecaRepoUC{peca: &entity.Peca{ID: uuid.New(), Nome: "Original"}}
	uc := usecase.NewPartUseCase(repo)
	nome := "   "
	_, err := uc.Update(uuid.New().String(), &nome, nil, nil)
	esperaErrValidacao(t, err)
}

func TestPartUpdate_NomeAtualizadoComTrim(t *testing.T) {
	repo := &mockPecaRepoUC{peca: &entity.Peca{ID: uuid.New(), Nome: "Original"}}
	uc := usecase.NewPartUseCase(repo)
	nome := "  Pastilha nova  "
	peca, err := uc.Update(uuid.New().String(), &nome, nil, nil)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if peca.Nome != "Pastilha nova" {
		t.Errorf("nome deve ser atualizado com trim, obteve %q", peca.Nome)
	}
}

func TestPartDelete_IDVazioRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	esperaErrValidacao(t, uc.Delete(""))
}

func TestAdjustStock_IDInvalidoRetornaErro(t *testing.T) {
	uc := usecase.NewPartUseCase(&mockPecaRepoUC{})
	_, err := uc.AdjustStock("nao-e-uuid", "entrada", 1)
	esperaErrValidacao(t, err)
}

// ── Updates com todos os campos preenchidos ───────────────────────────────────

func TestClientUpdate_TodosOsCampos(t *testing.T) {
	id := uuid.New()
	repo := &mockClienteRepoUC{
		buscarIDFn: func(_ string) (*entity.Cliente, error) {
			return &entity.Cliente{ID: id, Nome: "Original"}, nil
		},
	}
	uc := usecase.NewClientUseCase(repo)

	nome, email, telefone := "Novo Nome", "novo@teste.com", "11777777777"
	cliente, err := uc.Update(id.String(), &nome, &email, &telefone)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if cliente.Nome != nome {
		t.Errorf("nome não atualizado: %q", cliente.Nome)
	}
	if cliente.Email == nil || *cliente.Email != email {
		t.Errorf("email não atualizado: %v", cliente.Email)
	}
	if cliente.Telefone == nil || *cliente.Telefone != telefone {
		t.Errorf("telefone não atualizado: %v", cliente.Telefone)
	}
}

func TestPartUpdate_TodosOsCampos(t *testing.T) {
	repo := &mockPecaRepoUC{peca: &entity.Peca{ID: uuid.New(), Nome: "Original", Preco: 10, EstoqueMinimo: 1}}
	uc := usecase.NewPartUseCase(repo)

	nome, preco, estoqueMinimo := "Peça Nova", 25.5, 3
	peca, err := uc.Update(uuid.New().String(), &nome, &preco, &estoqueMinimo)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if peca.Nome != nome || peca.Preco != preco || peca.EstoqueMinimo != estoqueMinimo {
		t.Errorf("campos não atualizados: %+v", peca)
	}
}

func TestServiceUpdate_TodosOsCampos(t *testing.T) {
	uc := usecase.NewServiceUseCase(&mockServicoRepoUC{})

	nome, descricao, preco, tempo := "Serviço Novo", "descrição", 99.9, 90
	servico, err := uc.Update(uuid.New().String(), &nome, &descricao, &preco, &tempo)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if servico.Nome != nome || servico.PrecoBase != preco || servico.TempoMinutos != tempo {
		t.Errorf("campos não atualizados: %+v", servico)
	}
}

func TestVehicleUpdate_TodosOsCampos(t *testing.T) {
	repo := &mockVeiculoRepoUC{
		buscarIDFn: func(id string) (*entity.Veiculo, error) {
			return &entity.Veiculo{ID: uuid.MustParse(id), Marca: "VW", Modelo: "Gol", Ano: 2020}, nil
		},
	}
	uc := usecase.NewVehicleUseCase(repo, &mockClienteRepoForVehicle{})

	marca, modelo, ano, cor := "Fiat", "Argo", 2023, "branco"
	veiculo, err := uc.Update(uuid.New().String(), &marca, &modelo, &ano, &cor)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if veiculo.Marca != marca || veiculo.Modelo != modelo || veiculo.Ano != ano {
		t.Errorf("campos não atualizados: %+v", veiculo)
	}
}
