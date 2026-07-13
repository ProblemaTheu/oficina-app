package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
	"golang.org/x/crypto/bcrypt"
)

// cpfValido é um CPF sintático e matematicamente válido usado nas fixtures.
const cpfValido = "52998224725"

// ── stubs de repositório ──────────────────────────────────────────────────────

type stubClienteRepo struct {
	cliente *entity.Cliente
	lista   []*entity.Cliente
	err     error
	docErr  error
}

func (r *stubClienteRepo) Salvar(c *entity.Cliente) (*entity.Cliente, error) {
	if r.err != nil {
		return nil, r.err
	}
	c.ID = uuid.New()
	return c, nil
}
func (r *stubClienteRepo) BuscarTodos() ([]*entity.Cliente, error) { return r.lista, r.err }
func (r *stubClienteRepo) BuscarPorID(_ string) (*entity.Cliente, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.cliente, nil
}
func (r *stubClienteRepo) BuscarPorDocumento(_ string) (*entity.Cliente, error) {
	if r.docErr != nil {
		return nil, r.docErr
	}
	return r.cliente, nil
}
func (r *stubClienteRepo) Atualizar(c *entity.Cliente) (*entity.Cliente, error) {
	if r.err != nil {
		return nil, r.err
	}
	return c, nil
}
func (r *stubClienteRepo) Remover(_ string) error { return r.err }

type stubVeiculoRepo struct {
	veiculo *entity.Veiculo
	lista   []*entity.Veiculo
	err     error
}

func (r *stubVeiculoRepo) Salvar(v *entity.Veiculo) (*entity.Veiculo, error) {
	if r.err != nil {
		return nil, r.err
	}
	v.ID = uuid.New()
	return v, nil
}
func (r *stubVeiculoRepo) BuscarTodos() ([]*entity.Veiculo, error) { return r.lista, r.err }
func (r *stubVeiculoRepo) BuscarPorID(_ string) (*entity.Veiculo, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.veiculo, nil
}
func (r *stubVeiculoRepo) Atualizar(v *entity.Veiculo) (*entity.Veiculo, error) {
	if r.err != nil {
		return nil, r.err
	}
	return v, nil
}
func (r *stubVeiculoRepo) Remover(_ string) error { return r.err }
func (r *stubVeiculoRepo) BuscarPorClienteID(_ string) ([]*entity.Veiculo, error) {
	return r.lista, r.err
}

type stubServicoRepo struct {
	servico *entity.Servico
	lista   []*entity.Servico
	err     error
}

func (r *stubServicoRepo) Salvar(s *entity.Servico) (*entity.Servico, error) {
	if r.err != nil {
		return nil, r.err
	}
	s.ID = uuid.New()
	return s, nil
}
func (r *stubServicoRepo) BuscarTodos() ([]*entity.Servico, error) { return r.lista, r.err }
func (r *stubServicoRepo) BuscarPorID(_ string) (*entity.Servico, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.servico, nil
}
func (r *stubServicoRepo) Atualizar(s *entity.Servico) (*entity.Servico, error) {
	if r.err != nil {
		return nil, r.err
	}
	return s, nil
}
func (r *stubServicoRepo) Remover(_ string) error { return r.err }

type stubPecaRepo struct {
	peca  *entity.Peca
	lista []*entity.Peca
	err   error
}

func (r *stubPecaRepo) Salvar(p *entity.Peca) (*entity.Peca, error) {
	if r.err != nil {
		return nil, r.err
	}
	p.ID = uuid.New()
	return p, nil
}
func (r *stubPecaRepo) BuscarTodos() ([]*entity.Peca, error) { return r.lista, r.err }
func (r *stubPecaRepo) BuscarPorID(_ string) (*entity.Peca, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.peca, nil
}
func (r *stubPecaRepo) Atualizar(p *entity.Peca) (*entity.Peca, error) {
	if r.err != nil {
		return nil, r.err
	}
	return p, nil
}
func (r *stubPecaRepo) AtualizarEstoque(p *entity.Peca) (*entity.Peca, error) {
	if r.err != nil {
		return nil, r.err
	}
	return p, nil
}
func (r *stubPecaRepo) Remover(_ string) error { return r.err }

type stubOsRepo struct {
	osCompleta *entity.OrdemServicoCompleta
	lista      []*entity.OrdemServico
	total      int
	relatorio  []usecase.ItemTempoMedio
	err        error
}

func (r *stubOsRepo) BuscarStatusID(_ context.Context, _ entity.Status) (uuid.UUID, error) {
	return uuid.New(), r.err
}
func (r *stubOsRepo) GerarNumeroOS(_ context.Context) (string, error) {
	return "OS-2026-00042", r.err
}
func (r *stubOsRepo) Criar(_ context.Context, os *entity.OrdemServico, _ []entity.ItemOsServico, _ []entity.ItemOsPeca) (*entity.OrdemServico, error) {
	if r.err != nil {
		return nil, r.err
	}
	os.ID = uuid.New()
	return os, nil
}
func (r *stubOsRepo) Listar(_ context.Context, _ usecase.ListarOSParams) ([]*entity.OrdemServico, int, error) {
	return r.lista, r.total, r.err
}
func (r *stubOsRepo) BuscarPorID(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.osCompleta, nil
}
func (r *stubOsRepo) AtualizarStatus(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *string, _, _, _, _, _ *time.Time) error {
	return r.err
}
func (r *stubOsRepo) RegistrarHistorico(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ uuid.UUID, _ *string) error {
	return r.err
}
func (r *stubOsRepo) DeduzirEstoquePecas(_ context.Context, _ uuid.UUID) error { return r.err }
func (r *stubOsRepo) RelatorioTempoMedio(_ context.Context, _ usecase.RelatorioTempoMedioParams) ([]usecase.ItemTempoMedio, error) {
	return r.relatorio, r.err
}
func (r *stubOsRepo) PrecarregarStatusCache(_ context.Context) error { return r.err }

type stubUsuarioRepo struct {
	usuario   *entity.Usuario
	nomePapel string
	err       error
}

func (r *stubUsuarioRepo) BuscarPapelID(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.New(), r.err
}
func (r *stubUsuarioRepo) Salvar(_ context.Context, u *entity.Usuario) (*entity.Usuario, error) {
	if r.err != nil {
		return nil, r.err
	}
	u.ID = uuid.New()
	u.CriadoEm = time.Now()
	return u, nil
}
func (r *stubUsuarioRepo) BuscarPorEmail(_ context.Context, _ string) (*entity.Usuario, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.usuario, nil
}
func (r *stubUsuarioRepo) BuscarNomePapel(_ context.Context, _ uuid.UUID) (string, error) {
	return r.nomePapel, r.err
}

// ── fixtures ──────────────────────────────────────────────────────────────────

type fixtures struct {
	clienteID uuid.UUID
	veiculoID uuid.UUID
	cliente   *entity.Cliente
	veiculo   *entity.Veiculo
	servico   *entity.Servico
	peca      *entity.Peca
	osC       *entity.OrdemServicoCompleta
}

func novasFixtures() fixtures {
	clienteID := uuid.New()
	veiculoID := uuid.New()
	email := "cliente@teste.com"
	telefone := "11999999999"
	cor := "prata"
	descricao := "revisão completa"

	cliente := &entity.Cliente{
		ID: clienteID, Nome: "João da Silva", CpfCnpj: cpfValido,
		Email: &email, Telefone: &telefone,
	}
	veiculo := &entity.Veiculo{
		ID: veiculoID, ClienteID: clienteID, Placa: "ABC1D23",
		Marca: "VW", Modelo: "Gol", Ano: 2020, Cor: &cor,
	}
	servico := &entity.Servico{
		ID: uuid.New(), Nome: "Troca de óleo", Descricao: &descricao,
		PrecoBase: 150.0, TempoMinutos: 60,
	}
	peca := &entity.Peca{
		ID: uuid.New(), Nome: "Filtro de óleo", Codigo: "FLT-001",
		Preco: 35.5, EstoqueAtual: 10, EstoqueMinimo: 2,
	}
	osC := &entity.OrdemServicoCompleta{
		OrdemServico: entity.OrdemServico{
			ID: uuid.New(), Numero: "OS-2026-00042",
			ClienteID: clienteID, VeiculoID: veiculoID,
			StatusID: uuid.New(), StatusNome: entity.StatusRecebida,
			ValorTotal: 185.5,
		},
		Itens: []entity.ItemOS{
			{ID: uuid.New(), Tipo: "servico", ReferenciaID: servico.ID, Nome: servico.Nome, Quantidade: 1, PrecoUnitario: 150.0, Subtotal: 150.0},
			{ID: uuid.New(), Tipo: "peca", ReferenciaID: peca.ID, Nome: peca.Nome, Quantidade: 1, PrecoUnitario: 35.5, Subtotal: 35.5},
		},
		Cliente: entity.ClienteResumo{ID: clienteID, Nome: cliente.Nome, CpfCnpj: cliente.CpfCnpj},
		Veiculo: entity.VeiculoResumo{ID: veiculoID, Placa: veiculo.Placa, Marca: veiculo.Marca, Modelo: veiculo.Modelo, Ano: veiculo.Ano},
	}

	return fixtures{clienteID, veiculoID, cliente, veiculo, servico, peca, osC}
}

// novoServerDeTeste monta o Server injetando os stubs nos use cases reais,
// exercitando o caminho completo handler → use case → repositório.
func novoServerDeTeste(f fixtures, cliR *stubClienteRepo, veicR *stubVeiculoRepo, svcR *stubServicoRepo, pecaR *stubPecaRepo, osR *stubOsRepo, usrR *stubUsuarioRepo) *Server {
	if cliR == nil {
		cliR = &stubClienteRepo{cliente: f.cliente, lista: []*entity.Cliente{f.cliente}}
	}
	if veicR == nil {
		veicR = &stubVeiculoRepo{veiculo: f.veiculo, lista: []*entity.Veiculo{f.veiculo}}
	}
	if svcR == nil {
		svcR = &stubServicoRepo{servico: f.servico, lista: []*entity.Servico{f.servico}}
	}
	if pecaR == nil {
		pecaR = &stubPecaRepo{peca: f.peca, lista: []*entity.Peca{f.peca}}
	}
	if osR == nil {
		osR = &stubOsRepo{osCompleta: f.osC, lista: []*entity.OrdemServico{&f.osC.OrdemServico}, total: 1}
	}
	if usrR == nil {
		usrR = &stubUsuarioRepo{nomePapel: "atendente"}
	}
	return &Server{
		clientUseCase:  usecase.NewClientUseCase(cliR),
		vehicleUseCase: usecase.NewVehicleUseCase(veicR, cliR),
		serviceUseCase: usecase.NewServiceUseCase(svcR),
		partUseCase:    usecase.NewPartUseCase(pecaR),
		osUseCase:      usecase.NewOrdemServicoUseCase(osR, cliR, veicR, svcR, pecaR, nil),
		authUseCase:    usecase.NewAuthUseCase(usrR),
	}
}

func ctx() context.Context { return context.Background() }

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestPostAuthLogin_Sucesso(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("senha123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("falha ao gerar hash: %v", err)
	}
	usr := &entity.Usuario{ID: uuid.New(), Nome: "Admin", Email: "admin@oficina.com", SenhaHash: string(hash)}
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, &stubUsuarioRepo{usuario: usr})

	resp, err := srv.PostAuthLogin(ctx(), PostAuthLoginRequestObject{
		Body: &PostAuthLoginJSONRequestBody{Email: "admin@oficina.com", Senha: "senha123"},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	ok, isOk := resp.(PostAuthLogin200JSONResponse)
	if !isOk {
		t.Fatalf("esperava PostAuthLogin200JSONResponse, obteve %T", resp)
	}
	if ok.AccessToken == nil || *ok.AccessToken == "" {
		t.Error("access_token não deve ser vazio")
	}
	if ok.TokenType == nil || *ok.TokenType != "Bearer" {
		t.Error("token_type deve ser Bearer")
	}
}

func TestPostAuthLogin_CredenciaisInvalidas(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, &stubUsuarioRepo{err: errors.New("não encontrado")})

	resp, err := srv.PostAuthLogin(ctx(), PostAuthLoginRequestObject{
		Body: &PostAuthLoginJSONRequestBody{Email: "x@y.com", Senha: "errada"},
	})
	if err != nil {
		t.Fatalf("login inválido deve retornar 401, não erro: %v", err)
	}
	if _, is401 := resp.(PostAuthLogin401JSONResponse); !is401 {
		t.Fatalf("esperava PostAuthLogin401JSONResponse, obteve %T", resp)
	}
}

func TestPostAuthRegister_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	papel := CadastrarUsuarioRequestPapel("atendente")
	resp, err := srv.PostAuthRegister(ctx(), PostAuthRegisterRequestObject{
		Body: &PostAuthRegisterJSONRequestBody{
			Nome: "Novo Usuário", Email: "novo@oficina.com", Senha: "s3nh4", Papel: &papel,
		},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	created, isCreated := resp.(PostAuthRegister201JSONResponse)
	if !isCreated {
		t.Fatalf("esperava PostAuthRegister201JSONResponse, obteve %T", resp)
	}
	if created.Papel == nil || *created.Papel != "atendente" {
		t.Errorf("papel inesperado: %v", created.Papel)
	}
}

func TestPostAuthRegister_PapelInexistente(t *testing.T) {
	f := novasFixtures()
	usrR := &stubUsuarioRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "papel"}}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, usrR)

	_, err := srv.PostAuthRegister(ctx(), PostAuthRegisterRequestObject{
		Body: &PostAuthRegisterJSONRequestBody{Nome: "X", Email: "x@y.com", Senha: "s"},
	})
	if err == nil {
		t.Fatal("esperava erro de papel inexistente")
	}
}

// ── Clientes ──────────────────────────────────────────────────────────────────

func TestGetClients_ListaPaginada(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetClients(ctx(), GetClientsRequestObject{Params: GetClientsParams{}})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	lista, isOk := resp.(GetClients200JSONResponse)
	if !isOk {
		t.Fatalf("esperava GetClients200JSONResponse, obteve %T", resp)
	}
	if lista.Data == nil || len(*lista.Data) != 1 {
		t.Fatalf("esperava 1 cliente, obteve %v", lista.Data)
	}
	if lista.Meta == nil || lista.Meta.Total != 1 || lista.Meta.Page != 1 || lista.Meta.Limit != 20 {
		t.Errorf("meta de paginação inesperada: %+v", lista.Meta)
	}
}

func TestGetClients_FiltroPorDocumento(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	doc := cpfValido
	resp, err := srv.GetClients(ctx(), GetClientsRequestObject{Params: GetClientsParams{Documento: &doc}})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	lista := resp.(GetClients200JSONResponse)
	if lista.Data == nil || len(*lista.Data) != 1 {
		t.Fatalf("esperava 1 cliente no filtro por documento, obteve %v", lista.Data)
	}
}

func TestGetClients_DocumentoNaoEncontradoRetornaListaVazia(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{docErr: &domainerros.ErrNaoEncontrado{Recurso: "cliente"}}
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, nil)

	doc := "00000000000"
	resp, err := srv.GetClients(ctx(), GetClientsRequestObject{Params: GetClientsParams{Documento: &doc}})
	if err != nil {
		t.Fatalf("documento não encontrado deve retornar lista vazia, não erro: %v", err)
	}
	lista := resp.(GetClients200JSONResponse)
	if lista.Data == nil || len(*lista.Data) != 0 {
		t.Errorf("esperava lista vazia, obteve %v", lista.Data)
	}
	if lista.Meta == nil || lista.Meta.Total != 0 {
		t.Errorf("esperava total 0, obteve %+v", lista.Meta)
	}
}

func TestGetClients_ErroDeBancoNoFiltroPorDocumento(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{docErr: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, nil)

	doc := cpfValido
	if _, err := srv.GetClients(ctx(), GetClientsRequestObject{Params: GetClientsParams{Documento: &doc}}); err == nil {
		t.Fatal("esperava propagação do erro de banco")
	}
}

func TestGetClients_ErroAoListar(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{err: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, nil)

	if _, err := srv.GetClients(ctx(), GetClientsRequestObject{Params: GetClientsParams{}}); err == nil {
		t.Fatal("esperava propagação do erro de banco")
	}
}

func TestPostClients_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, &stubClienteRepo{}, nil, nil, nil, nil, nil)

	email := openapi_types.Email("novo@teste.com")
	resp, err := srv.PostClients(ctx(), PostClientsRequestObject{
		Body: &PostClientsJSONRequestBody{Nome: "Maria", Documento: cpfValido, Email: &email},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	created, isCreated := resp.(PostClients201JSONResponse)
	if !isCreated {
		t.Fatalf("esperava PostClients201JSONResponse, obteve %T", resp)
	}
	if created.Headers.Location == "" {
		t.Error("header Location deve apontar para o novo recurso")
	}
	if created.Body.Nome == nil || *created.Body.Nome != "Maria" {
		t.Errorf("nome inesperado: %v", created.Body.Nome)
	}
}

func TestPostClients_DocumentoInvalido(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	if _, err := srv.PostClients(ctx(), PostClientsRequestObject{
		Body: &PostClientsJSONRequestBody{Nome: "Maria", Documento: "123"},
	}); err == nil {
		t.Fatal("esperava erro de documento inválido")
	}
}

func TestGetClientsId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetClientsId(ctx(), GetClientsIdRequestObject{Id: f.clienteID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	cli := resp.(GetClientsId200JSONResponse)
	if cli.Id == nil || *cli.Id != f.clienteID {
		t.Errorf("id inesperado: %v", cli.Id)
	}
}

func TestGetClientsId_NaoEncontrado(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "cliente"}}
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, nil)

	if _, err := srv.GetClientsId(ctx(), GetClientsIdRequestObject{Id: uuid.New()}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestPutClientsId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	nome := "João Atualizado"
	email := openapi_types.Email("atualizado@teste.com")
	telefone := "11888888888"
	resp, err := srv.PutClientsId(ctx(), PutClientsIdRequestObject{
		Id:   f.clienteID,
		Body: &PutClientsIdJSONRequestBody{Nome: &nome, Email: &email, Telefone: &telefone},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	cli := resp.(PutClientsId200JSONResponse)
	if cli.Nome == nil || *cli.Nome != nome {
		t.Errorf("nome não atualizado: %v", cli.Nome)
	}
}

func TestPutClientsId_NaoEncontrado(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "cliente"}}
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, nil)

	if _, err := srv.PutClientsId(ctx(), PutClientsIdRequestObject{
		Id: uuid.New(), Body: &PutClientsIdJSONRequestBody{},
	}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestDeleteClientsId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.DeleteClientsId(ctx(), DeleteClientsIdRequestObject{Id: f.clienteID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, is204 := resp.(DeleteClientsId204Response); !is204 {
		t.Fatalf("esperava DeleteClientsId204Response, obteve %T", resp)
	}
}

func TestDeleteClientsId_ComVinculos(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{err: &domainerros.ErrConflito{Campo: "cliente"}}
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, nil)

	if _, err := srv.DeleteClientsId(ctx(), DeleteClientsIdRequestObject{Id: f.clienteID}); err == nil {
		t.Fatal("esperava erro de conflito")
	}
}

func TestGetClientsIdVehicles_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetClientsIdVehicles(ctx(), GetClientsIdVehiclesRequestObject{
		Id: f.clienteID, Params: GetClientsIdVehiclesParams{},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	lista := resp.(GetClientsIdVehicles200JSONResponse)
	if lista.Data == nil || len(*lista.Data) != 1 {
		t.Fatalf("esperava 1 veículo, obteve %v", lista.Data)
	}
}

// ── Veículos ──────────────────────────────────────────────────────────────────

func TestGetVehicles_ListaPaginada(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetVehicles(ctx(), GetVehiclesRequestObject{Params: GetVehiclesParams{}})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	lista := resp.(GetVehicles200JSONResponse)
	if lista.Data == nil || len(*lista.Data) != 1 {
		t.Fatalf("esperava 1 veículo, obteve %v", lista.Data)
	}
}

func TestGetVehicles_Erro(t *testing.T) {
	f := novasFixtures()
	veicR := &stubVeiculoRepo{err: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, nil, veicR, nil, nil, nil, nil)

	if _, err := srv.GetVehicles(ctx(), GetVehiclesRequestObject{Params: GetVehiclesParams{}}); err == nil {
		t.Fatal("esperava propagação do erro")
	}
}

func TestPostVehicles_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, &stubVeiculoRepo{}, nil, nil, nil, nil)

	cor := "preto"
	resp, err := srv.PostVehicles(ctx(), PostVehiclesRequestObject{
		Body: &PostVehiclesJSONRequestBody{
			ClienteId: f.clienteID, Placa: "ABC1D23", Marca: "Fiat", Modelo: "Uno", Ano: 2018, Cor: &cor,
		},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	created, isCreated := resp.(PostVehicles201JSONResponse)
	if !isCreated {
		t.Fatalf("esperava PostVehicles201JSONResponse, obteve %T", resp)
	}
	if created.Headers.Location == "" {
		t.Error("header Location deve apontar para o novo recurso")
	}
}

func TestPostVehicles_PlacaInvalida(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	if _, err := srv.PostVehicles(ctx(), PostVehiclesRequestObject{
		Body: &PostVehiclesJSONRequestBody{
			ClienteId: f.clienteID, Placa: "INVALIDA!", Marca: "Fiat", Modelo: "Uno", Ano: 2018,
		},
	}); err == nil {
		t.Fatal("esperava erro de placa inválida")
	}
}

func TestGetVehiclesId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetVehiclesId(ctx(), GetVehiclesIdRequestObject{Id: f.veiculoID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	v := resp.(GetVehiclesId200JSONResponse)
	if v.Placa == nil || *v.Placa != "ABC1D23" {
		t.Errorf("placa inesperada: %v", v.Placa)
	}
}

func TestGetVehiclesId_NaoEncontrado(t *testing.T) {
	f := novasFixtures()
	veicR := &stubVeiculoRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "veículo"}}
	srv := novoServerDeTeste(f, nil, veicR, nil, nil, nil, nil)

	if _, err := srv.GetVehiclesId(ctx(), GetVehiclesIdRequestObject{Id: uuid.New()}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestPutVehiclesId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	marca := "Chevrolet"
	ano := 2022
	resp, err := srv.PutVehiclesId(ctx(), PutVehiclesIdRequestObject{
		Id:   f.veiculoID,
		Body: &PutVehiclesIdJSONRequestBody{Marca: &marca, Ano: &ano},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	v := resp.(PutVehiclesId200JSONResponse)
	if v.Marca == nil || *v.Marca != marca {
		t.Errorf("marca não atualizada: %v", v.Marca)
	}
}

func TestPutVehiclesId_Erro(t *testing.T) {
	f := novasFixtures()
	veicR := &stubVeiculoRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "veículo"}}
	srv := novoServerDeTeste(f, nil, veicR, nil, nil, nil, nil)

	if _, err := srv.PutVehiclesId(ctx(), PutVehiclesIdRequestObject{
		Id: uuid.New(), Body: &PutVehiclesIdJSONRequestBody{},
	}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestDeleteVehiclesId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.DeleteVehiclesId(ctx(), DeleteVehiclesIdRequestObject{Id: f.veiculoID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, is204 := resp.(DeleteVehiclesId204Response); !is204 {
		t.Fatalf("esperava DeleteVehiclesId204Response, obteve %T", resp)
	}
}

func TestDeleteVehiclesId_Erro(t *testing.T) {
	f := novasFixtures()
	veicR := &stubVeiculoRepo{err: &domainerros.ErrConflito{Campo: "veículo"}}
	srv := novoServerDeTeste(f, nil, veicR, nil, nil, nil, nil)

	if _, err := srv.DeleteVehiclesId(ctx(), DeleteVehiclesIdRequestObject{Id: f.veiculoID}); err == nil {
		t.Fatal("esperava erro de conflito")
	}
}

// ── Serviços ──────────────────────────────────────────────────────────────────

func TestGetServices_ListaPaginada(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetServices(ctx(), GetServicesRequestObject{Params: GetServicesParams{}})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	lista := resp.(GetServices200JSONResponse)
	if lista.Data == nil || len(*lista.Data) != 1 {
		t.Fatalf("esperava 1 serviço, obteve %v", lista.Data)
	}
}

func TestGetServices_Erro(t *testing.T) {
	f := novasFixtures()
	svcR := &stubServicoRepo{err: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, nil, nil, svcR, nil, nil, nil)

	if _, err := srv.GetServices(ctx(), GetServicesRequestObject{Params: GetServicesParams{}}); err == nil {
		t.Fatal("esperava propagação do erro")
	}
}

func TestPostServices_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, &stubServicoRepo{}, nil, nil, nil)

	descricao := "alinhamento e balanceamento"
	resp, err := srv.PostServices(ctx(), PostServicesRequestObject{
		Body: &PostServicesJSONRequestBody{
			Nome: "Alinhamento", Descricao: &descricao, PrecoBase: 120.0, TempoMinutos: 45,
		},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	created, isCreated := resp.(PostServices201JSONResponse)
	if !isCreated {
		t.Fatalf("esperava PostServices201JSONResponse, obteve %T", resp)
	}
	if created.Headers.Location == "" {
		t.Error("header Location deve apontar para o novo recurso")
	}
}

func TestPostServices_NomeVazio(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	if _, err := srv.PostServices(ctx(), PostServicesRequestObject{
		Body: &PostServicesJSONRequestBody{Nome: "", PrecoBase: 10, TempoMinutos: 30},
	}); err == nil {
		t.Fatal("esperava erro de validação")
	}
}

func TestGetServicesId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetServicesId(ctx(), GetServicesIdRequestObject{Id: f.servico.ID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	s := resp.(GetServicesId200JSONResponse)
	if s.Nome == nil || *s.Nome != f.servico.Nome {
		t.Errorf("nome inesperado: %v", s.Nome)
	}
}

func TestGetServicesId_NaoEncontrado(t *testing.T) {
	f := novasFixtures()
	svcR := &stubServicoRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "serviço"}}
	srv := novoServerDeTeste(f, nil, nil, svcR, nil, nil, nil)

	if _, err := srv.GetServicesId(ctx(), GetServicesIdRequestObject{Id: uuid.New()}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestPutServicesId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	preco := 175.0
	resp, err := srv.PutServicesId(ctx(), PutServicesIdRequestObject{
		Id:   f.servico.ID,
		Body: &PutServicesIdJSONRequestBody{PrecoBase: &preco},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	s := resp.(PutServicesId200JSONResponse)
	if s.PrecoBase == nil || *s.PrecoBase != preco {
		t.Errorf("preço não atualizado: %v", s.PrecoBase)
	}
}

func TestPutServicesId_Erro(t *testing.T) {
	f := novasFixtures()
	svcR := &stubServicoRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "serviço"}}
	srv := novoServerDeTeste(f, nil, nil, svcR, nil, nil, nil)

	if _, err := srv.PutServicesId(ctx(), PutServicesIdRequestObject{
		Id: uuid.New(), Body: &PutServicesIdJSONRequestBody{},
	}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestDeleteServicesId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.DeleteServicesId(ctx(), DeleteServicesIdRequestObject{Id: f.servico.ID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, is204 := resp.(DeleteServicesId204Response); !is204 {
		t.Fatalf("esperava DeleteServicesId204Response, obteve %T", resp)
	}
}

func TestDeleteServicesId_Erro(t *testing.T) {
	f := novasFixtures()
	svcR := &stubServicoRepo{err: &domainerros.ErrConflito{Campo: "serviço"}}
	srv := novoServerDeTeste(f, nil, nil, svcR, nil, nil, nil)

	if _, err := srv.DeleteServicesId(ctx(), DeleteServicesIdRequestObject{Id: f.servico.ID}); err == nil {
		t.Fatal("esperava erro de conflito")
	}
}

// ── Peças ─────────────────────────────────────────────────────────────────────

func TestGetParts_ListaPaginada(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetParts(ctx(), GetPartsRequestObject{Params: GetPartsParams{}})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	lista := resp.(GetParts200JSONResponse)
	if lista.Data == nil || len(*lista.Data) != 1 {
		t.Fatalf("esperava 1 peça, obteve %v", lista.Data)
	}
}

func TestGetParts_Erro(t *testing.T) {
	f := novasFixtures()
	pecaR := &stubPecaRepo{err: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, nil, nil, nil, pecaR, nil, nil)

	if _, err := srv.GetParts(ctx(), GetPartsRequestObject{Params: GetPartsParams{}}); err == nil {
		t.Fatal("esperava propagação do erro")
	}
}

func TestPostParts_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, &stubPecaRepo{}, nil, nil)

	estoqueAtual := 15
	resp, err := srv.PostParts(ctx(), PostPartsRequestObject{
		Body: &PostPartsJSONRequestBody{
			Nome: "Pastilha de freio", Codigo: "PF-010", Preco: 89.9,
			EstoqueAtual: &estoqueAtual, EstoqueMinimo: 4,
		},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	created, isCreated := resp.(PostParts201JSONResponse)
	if !isCreated {
		t.Fatalf("esperava PostParts201JSONResponse, obteve %T", resp)
	}
	if created.Headers.Location == "" {
		t.Error("header Location deve apontar para o novo recurso")
	}
}

func TestPostParts_CodigoVazio(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	if _, err := srv.PostParts(ctx(), PostPartsRequestObject{
		Body: &PostPartsJSONRequestBody{Nome: "Peça", Codigo: "", Preco: 10, EstoqueMinimo: 1},
	}); err == nil {
		t.Fatal("esperava erro de validação")
	}
}

func TestGetPartsId_Sucesso_ComEstoqueBaixoCalculado(t *testing.T) {
	f := novasFixtures()
	f.peca.EstoqueAtual = 1 // abaixo do mínimo (2)
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetPartsId(ctx(), GetPartsIdRequestObject{Id: f.peca.ID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	p := resp.(GetPartsId200JSONResponse)
	if p.EstoqueBaixo == nil || !*p.EstoqueBaixo {
		t.Error("estoque_baixo deve ser true quando estoque_atual < estoque_minimo")
	}
}

func TestGetPartsId_NaoEncontrado(t *testing.T) {
	f := novasFixtures()
	pecaR := &stubPecaRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "peça"}}
	srv := novoServerDeTeste(f, nil, nil, nil, pecaR, nil, nil)

	if _, err := srv.GetPartsId(ctx(), GetPartsIdRequestObject{Id: uuid.New()}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestPutPartsId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	preco := 42.0
	resp, err := srv.PutPartsId(ctx(), PutPartsIdRequestObject{
		Id:   f.peca.ID,
		Body: &PutPartsIdJSONRequestBody{Preco: &preco},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	p := resp.(PutPartsId200JSONResponse)
	if p.Preco == nil || *p.Preco != preco {
		t.Errorf("preço não atualizado: %v", p.Preco)
	}
}

func TestPutPartsId_Erro(t *testing.T) {
	f := novasFixtures()
	pecaR := &stubPecaRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "peça"}}
	srv := novoServerDeTeste(f, nil, nil, nil, pecaR, nil, nil)

	if _, err := srv.PutPartsId(ctx(), PutPartsIdRequestObject{
		Id: uuid.New(), Body: &PutPartsIdJSONRequestBody{},
	}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestDeletePartsId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.DeletePartsId(ctx(), DeletePartsIdRequestObject{Id: f.peca.ID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, is204 := resp.(DeletePartsId204Response); !is204 {
		t.Fatalf("esperava DeletePartsId204Response, obteve %T", resp)
	}
}

func TestDeletePartsId_Erro(t *testing.T) {
	f := novasFixtures()
	pecaR := &stubPecaRepo{err: &domainerros.ErrConflito{Campo: "peça"}}
	srv := novoServerDeTeste(f, nil, nil, nil, pecaR, nil, nil)

	if _, err := srv.DeletePartsId(ctx(), DeletePartsIdRequestObject{Id: f.peca.ID}); err == nil {
		t.Fatal("esperava erro de conflito")
	}
}

func TestPatchPartsIdStock_Entrada(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.PatchPartsIdStock(ctx(), PatchPartsIdStockRequestObject{
		Id:   f.peca.ID,
		Body: &PatchPartsIdStockJSONRequestBody{Tipo: "entrada", Quantidade: 5},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	p := resp.(PatchPartsIdStock200JSONResponse)
	if p.EstoqueAtual == nil || *p.EstoqueAtual != 15 {
		t.Errorf("esperava estoque 15 após entrada de 5, obteve %v", p.EstoqueAtual)
	}
}

func TestPatchPartsIdStock_SaidaInsuficiente(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.PatchPartsIdStock(ctx(), PatchPartsIdStockRequestObject{
		Id:   f.peca.ID,
		Body: &PatchPartsIdStockJSONRequestBody{Tipo: "saida", Quantidade: 999},
	})
	if !errors.Is(err, usecase.ErrEstoqueInsuficiente) {
		t.Fatalf("esperava ErrEstoqueInsuficiente, obteve: %v", err)
	}
}

// ── Ordens de Serviço ─────────────────────────────────────────────────────────

func TestGetWorkOrders_ComFiltros(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	status := "recebida"
	incluir := true
	resp, err := srv.GetWorkOrders(ctx(), GetWorkOrdersRequestObject{
		Params: GetWorkOrdersParams{
			Status:            &status,
			IncluirEncerradas: &incluir,
			ClienteId:         &f.clienteID,
			VeiculoId:         &f.veiculoID,
		},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	lista := resp.(GetWorkOrders200JSONResponse)
	if lista.Data == nil || len(*lista.Data) != 1 {
		t.Fatalf("esperava 1 OS, obteve %v", lista.Data)
	}
	if lista.Meta == nil || lista.Meta.Total != 1 {
		t.Errorf("meta inesperada: %+v", lista.Meta)
	}
}

func TestGetWorkOrders_Erro(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{err: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	if _, err := srv.GetWorkOrders(ctx(), GetWorkOrdersRequestObject{Params: GetWorkOrdersParams{}}); err == nil {
		t.Fatal("esperava propagação do erro")
	}
}

func TestPostWorkOrders_ComServicosEPecas(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	qtd := 2
	descricao := "barulho no motor"
	resp, err := srv.PostWorkOrders(ctx(), PostWorkOrdersRequestObject{
		Body: &PostWorkOrdersJSONRequestBody{
			ClienteId: f.clienteID,
			VeiculoId: f.veiculoID,
			Descricao: &descricao,
			Servicos:  &[]ItemServicoOSRequest{{ServicoId: f.servico.ID, Quantidade: &qtd}},
			Pecas:     &[]ItemPecaOSRequest{{PecaId: f.peca.ID}}, // quantidade omitida → default 1
		},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	created, isCreated := resp.(PostWorkOrders201JSONResponse)
	if !isCreated {
		t.Fatalf("esperava PostWorkOrders201JSONResponse, obteve %T", resp)
	}
	if created.Servicos == nil || len(*created.Servicos) != 1 {
		t.Errorf("esperava 1 item de serviço na resposta, obteve %v", created.Servicos)
	}
	if created.Pecas == nil || len(*created.Pecas) != 1 {
		t.Errorf("esperava 1 item de peça na resposta, obteve %v", created.Pecas)
	}
	if created.Cliente == nil || created.Veiculo == nil {
		t.Error("resposta deve embutir resumos de cliente e veículo")
	}
}

func TestPostWorkOrders_SemItens(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.PostWorkOrders(ctx(), PostWorkOrdersRequestObject{
		Body: &PostWorkOrdersJSONRequestBody{ClienteId: f.clienteID, VeiculoId: f.veiculoID},
	})
	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) {
		t.Fatalf("esperava ErrNaoProcessavel para OS sem itens, obteve: %v", err)
	}
}

func TestGetWorkOrdersId_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetWorkOrdersId(ctx(), GetWorkOrdersIdRequestObject{Id: f.osC.ID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	os := resp.(GetWorkOrdersId200JSONResponse)
	if os.Numero == nil || *os.Numero != "OS-2026-00042" {
		t.Errorf("número inesperado: %v", os.Numero)
	}
}

func TestGetWorkOrdersId_NaoEncontrada(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "ordem de serviço"}}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	if _, err := srv.GetWorkOrdersId(ctx(), GetWorkOrdersIdRequestObject{Id: uuid.New()}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestGetWorkOrdersIdStatus_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.GetWorkOrdersIdStatus(ctx(), GetWorkOrdersIdStatusRequestObject{Id: f.osC.ID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	st := resp.(GetWorkOrdersIdStatus200JSONResponse)
	if st.Status == nil || *st.Status != string(entity.StatusRecebida) {
		t.Errorf("status inesperado: %v", st.Status)
	}
}

func TestPatchWorkOrdersIdStatus_Sucesso(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.PatchWorkOrdersIdStatus(ctx(), PatchWorkOrdersIdStatusRequestObject{
		Id:   f.osC.ID,
		Body: &PatchWorkOrdersIdStatusJSONRequestBody{Status: EmDiagnostico},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, isOk := resp.(PatchWorkOrdersIdStatus200JSONResponse); !isOk {
		t.Fatalf("esperava PatchWorkOrdersIdStatus200JSONResponse, obteve %T", resp)
	}
}

func TestPatchWorkOrdersIdStatus_TransicaoInvalida(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.PatchWorkOrdersIdStatus(ctx(), PatchWorkOrdersIdStatusRequestObject{
		Id:   f.osC.ID,
		Body: &PatchWorkOrdersIdStatusJSONRequestBody{Status: Entregue}, // recebida → entregue é inválido
	})
	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) || np.Codigo != "INVALID_STATUS_TRANSITION" {
		t.Fatalf("esperava INVALID_STATUS_TRANSITION, obteve: %v", err)
	}
}

func TestPostWorkOrdersIdApprove_Sucesso(t *testing.T) {
	f := novasFixtures()
	f.osC.StatusNome = entity.StatusAguardandoAprovacao
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.PostWorkOrdersIdApprove(ctx(), PostWorkOrdersIdApproveRequestObject{Id: f.osC.ID})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, isOk := resp.(PostWorkOrdersIdApprove200JSONResponse); !isOk {
		t.Fatalf("esperava PostWorkOrdersIdApprove200JSONResponse, obteve %T", resp)
	}
}

func TestPostWorkOrdersIdApprove_StatusErrado(t *testing.T) {
	f := novasFixtures() // status recebida
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	if _, err := srv.PostWorkOrdersIdApprove(ctx(), PostWorkOrdersIdApproveRequestObject{Id: f.osC.ID}); err == nil {
		t.Fatal("esperava erro de transição inválida")
	}
}

func TestPostWorkOrdersIdReject_ComMotivo(t *testing.T) {
	f := novasFixtures()
	f.osC.StatusNome = entity.StatusAguardandoAprovacao
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	motivo := "orçamento muito alto"
	resp, err := srv.PostWorkOrdersIdReject(ctx(), PostWorkOrdersIdRejectRequestObject{
		Id:   f.osC.ID,
		Body: &PostWorkOrdersIdRejectJSONRequestBody{Motivo: &motivo},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, isOk := resp.(PostWorkOrdersIdReject200JSONResponse); !isOk {
		t.Fatalf("esperava PostWorkOrdersIdReject200JSONResponse, obteve %T", resp)
	}
}

func TestPostWorkOrdersIdReject_SemBody(t *testing.T) {
	f := novasFixtures()
	f.osC.StatusNome = entity.StatusAguardandoAprovacao
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	if _, err := srv.PostWorkOrdersIdReject(ctx(), PostWorkOrdersIdRejectRequestObject{Id: f.osC.ID}); err != nil {
		t.Fatalf("rejeição sem body deve ser aceita, obteve: %v", err)
	}
}

func TestPostWebhooksBudgetResponse_Aprovado(t *testing.T) {
	f := novasFixtures()
	f.osC.StatusNome = entity.StatusAguardandoAprovacao
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	resp, err := srv.PostWebhooksBudgetResponse(ctx(), PostWebhooksBudgetResponseRequestObject{
		Body: &PostWebhooksBudgetResponseJSONRequestBody{OsId: f.osC.ID, Decisao: Aprovado},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if _, isOk := resp.(PostWebhooksBudgetResponse200JSONResponse); !isOk {
		t.Fatalf("esperava PostWebhooksBudgetResponse200JSONResponse, obteve %T", resp)
	}
}

func TestPostWebhooksBudgetResponse_DecisaoInvalida(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.PostWebhooksBudgetResponse(ctx(), PostWebhooksBudgetResponseRequestObject{
		Body: &PostWebhooksBudgetResponseJSONRequestBody{OsId: f.osC.ID, Decisao: "talvez"},
	})
	var ev *domainerros.ErrValidacao
	if !errors.As(err, &ev) {
		t.Fatalf("esperava ErrValidacao, obteve: %v", err)
	}
}

// ── Relatórios ────────────────────────────────────────────────────────────────

func TestGetReportsAvgExecutionTime_ComFiltros(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{
		osCompleta: f.osC,
		relatorio: []usecase.ItemTempoMedio{
			{ServicoID: f.servico.ID, ServicoNome: f.servico.Nome, TotalExecucoes: 3, TempoMedioMinutos: 62.5},
		},
	}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	dataInicio := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	dataFim := openapi_types.Date{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)}
	resp, err := srv.GetReportsAvgExecutionTime(ctx(), GetReportsAvgExecutionTimeRequestObject{
		Params: GetReportsAvgExecutionTimeParams{
			ServicoId:  &f.servico.ID,
			DataInicio: &dataInicio,
			DataFim:    &dataFim,
		},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	itens := resp.(GetReportsAvgExecutionTime200JSONResponse)
	if len(itens) != 1 {
		t.Fatalf("esperava 1 item no relatório, obteve %d", len(itens))
	}
	if itens[0].TempoMedioMinutos == nil || *itens[0].TempoMedioMinutos != 62.5 {
		t.Errorf("tempo médio inesperado: %v", itens[0].TempoMedioMinutos)
	}
}

func TestGetReportsAvgExecutionTime_Erro(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{err: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	if _, err := srv.GetReportsAvgExecutionTime(ctx(), GetReportsAvgExecutionTimeRequestObject{
		Params: GetReportsAvgExecutionTimeParams{},
	}); err == nil {
		t.Fatal("esperava propagação do erro")
	}
}

// ── Inicialização ─────────────────────────────────────────────────────────────

func TestInicializarCaches(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	if err := srv.InicializarCaches(ctx()); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func TestPaginacaoDefaults(t *testing.T) {
	page5, limit50 := 5, 50
	zero, negativo := 0, -1

	casos := []struct {
		nome        string
		page, limit *int
		espP, espL  int
	}{
		{"nil aplica defaults", nil, nil, 1, 20},
		{"valores válidos preservados", &page5, &limit50, 5, 50},
		{"zero aplica defaults", &zero, &zero, 1, 20},
		{"negativo aplica defaults", &negativo, &negativo, 1, 20},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			p, l := paginacaoDefaults(c.page, c.limit)
			if p != c.espP || l != c.espL {
				t.Errorf("esperava (%d, %d), obteve (%d, %d)", c.espP, c.espL, p, l)
			}
		})
	}
}

func TestCalcularFatia(t *testing.T) {
	casos := []struct {
		nome               string
		total, page, limit int
		espIni, espFim     int
	}{
		{"primeira página", 50, 1, 20, 0, 20},
		{"última página parcial", 50, 3, 20, 40, 50},
		{"página além do total", 10, 5, 20, 10, 10},
		{"total zero", 0, 1, 20, 0, 0},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			ini, fim := calcularFatia(c.total, c.page, c.limit)
			if ini != c.espIni || fim != c.espFim {
				t.Errorf("esperava (%d, %d), obteve (%d, %d)", c.espIni, c.espFim, ini, fim)
			}
		})
	}
}

func TestMetaPaginacao(t *testing.T) {
	casos := []struct {
		nome               string
		page, limit, total int
		espTotalPages      int
	}{
		{"divisão exata", 1, 20, 40, 2},
		{"divisão com resto arredonda para cima", 1, 20, 41, 3},
		{"total zero mantém 1 página", 1, 20, 0, 1},
		{"limit zero mantém 1 página", 1, 0, 10, 1},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			meta := metaPaginacao(c.page, c.limit, c.total)
			if meta.TotalPages != c.espTotalPages {
				t.Errorf("esperava %d páginas, obteve %d", c.espTotalPages, meta.TotalPages)
			}
			if meta.Page != c.page || meta.Limit != c.limit || meta.Total != c.total {
				t.Errorf("meta inconsistente: %+v", meta)
			}
		})
	}
}

func TestQuantidadeOuPadrao(t *testing.T) {
	dois, zero, negativo := 2, 0, -3
	casos := []struct {
		nome string
		qtd  *int
		esp  int
	}{
		{"nil vira 1", nil, 1},
		{"zero vira 1", &zero, 1},
		{"negativo vira 1", &negativo, 1},
		{"positivo preservado", &dois, 2},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := quantidadeOuPadrao(c.qtd); got != c.esp {
				t.Errorf("esperava %d, obteve %d", c.esp, got)
			}
		})
	}
}
