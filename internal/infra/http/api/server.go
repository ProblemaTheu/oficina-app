// Package api contém a implementação dos handlers HTTP gerados pelo oapi-codegen
// a partir do contrato OpenAPI definido em docs/openapi.yaml.
//
// O padrão adotado é o Strict Server: cada método recebe um RequestObject tipado
// e retorna um ResponseObject tipado, eliminando manipulação manual de
// http.ResponseWriter e json.Encoder nos handlers.
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
	"github.com/problematheu/tech-challenge-1/internal/domain/valueobject"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

// errNaoImplementado é retornado por endpoints ainda não implementados,
// resultando em HTTP 500 até que a lógica seja preenchida.
var errNaoImplementado = errors.New("não implementado")

// Server implementa StrictServerInterface gerado pelo oapi-codegen.
//
// Cada grupo de métodos corresponde a um recurso da API (Clientes, Veículos,
// Serviços, Peças). Novos use cases devem ser adicionados como campos e
// injetados em NovoServer.
type Server struct {
	clientUseCase  *usecase.ClientUseCase
	vehicleUseCase *usecase.VehicleUseCase
	serviceUseCase *usecase.ServiceUseCase
	partUseCase    *usecase.PartUseCase
}

// NovoServer constrói um Server com todas as dependências injetadas.
//
// Parâmetros:
//   - db: conexão ativa com o banco de dados PostgreSQL.
//
// Retorno:
//   - *Server pronto para ser registrado como StrictServerInterface.
func NovoServer(db *sql.DB) *Server {
	clienteRepo := repository.NovoClienteRepository(db)
	veiculoRepo := repository.NovoVeiculoRepository(db)
	servicoRepo := repository.NovoServicoRepository(db)
	pecaRepo := repository.NovoPecaRepository(db)

	return &Server{
		clientUseCase:  usecase.NewClientUseCase(clienteRepo),
		vehicleUseCase: usecase.NewVehicleUseCase(veiculoRepo, clienteRepo),
		serviceUseCase: usecase.NewServiceUseCase(servicoRepo),
		partUseCase:    usecase.NewPartUseCase(pecaRepo),
	}
}

// ── Auth ──────────────────────────────────────────────────────────────────────

// PostAuthLogin autentica um usuário e retorna um JWT Bearer token.
//
// Rota: POST /v1/auth/login
//
// Respostas:
//   - 200: token gerado com sucesso.
//   - 401: e-mail ou senha inválidos.
//
// TODO: implementar autenticação JWT.
func (s *Server) PostAuthLogin(_ context.Context, _ PostAuthLoginRequestObject) (PostAuthLoginResponseObject, error) {
	return nil, errNaoImplementado
}

// ── Clientes ──────────────────────────────────────────────────────────────────

// GetClients retorna a lista paginada de clientes.
//
// Rota: GET /v1/clients
//
// Query params (via request.Params):
//   - page  (int, default 1):  número da página.
//   - limit (int, default 20): itens por página (máx. 100).
//   - nome  (string):          filtro parcial case-insensitive pelo nome.
//   - documento (string):      filtro exato pelo CPF/CNPJ.
//
// Respostas:
//   - 200: ClienteListResponse com data[] e meta de paginação.
//   - 401: token ausente ou inválido.
func (s *Server) GetClients(_ context.Context, request GetClientsRequestObject) (GetClientsResponseObject, error) {
	clientes, err := s.clientUseCase.List()
	if err != nil {
		return nil, err
	}

	page, limit := paginacaoDefaults(request.Params.Page, request.Params.Limit)
	total := len(clientes)
	inicio, fim := calcularFatia(total, page, limit)

	data := make([]ClienteResponse, 0, fim-inicio)
	for _, c := range clientes[inicio:fim] {
		data = append(data, clienteParaResponse(c))
	}

	return GetClients200JSONResponse(ClienteListResponse{
		Data: &data,
		Meta: metaPaginacao(page, limit, total),
	}), nil
}

// PostClients cadastra um novo cliente.
//
// Rota: POST /v1/clients
//
// Body (via request.Body — CriarClienteRequest):
//   - nome      (string, obrigatório): nome completo ou razão social.
//   - documento (string, obrigatório): CPF ou CNPJ, com ou sem formatação.
//   - email     (string, opcional):   e-mail único.
//   - telefone  (string, opcional):   telefone de contato.
//
// Respostas:
//   - 201: cliente criado; header Location aponta para o novo recurso.
//   - 400: payload inválido ou regra de negócio violada (ex: CPF inválido).
//   - 401: token ausente ou inválido.
//   - 409: CPF/CNPJ ou e-mail já cadastrado.
func (s *Server) PostClients(_ context.Context, request PostClientsRequestObject) (PostClientsResponseObject, error) {
	body := request.Body

	cliente, err := s.clientUseCase.Create(
		body.Nome,
		body.Documento,
		(*string)(body.Email),
		body.Telefone,
	)

	if err != nil {
		var errConflito *domainerros.ErrConflito

		switch {

		case errors.As(err, &errConflito):
			return PostClients409JSONResponse{
				ConflictJSONResponse{Code: "CONFLICT", Message: err.Error()},
			}, nil

		case errors.Is(err, valueobject.ErrDocumentoInvalido):
			return PostClients400JSONResponse{
				BadRequestJSONResponse{Code: "INVALID_DOCUMENT", Message: err.Error()},
			}, nil

		default:
			return PostClients400JSONResponse{
				BadRequestJSONResponse{Code: "VALIDATION_ERROR", Message: err.Error()},
			}, nil
		}
	}

	resp := clienteParaResponse(cliente)
	return PostClients201JSONResponse{
		Body: resp,
		Headers: PostClients201ResponseHeaders{
			Location: fmt.Sprintf("/v1/clients/%s", cliente.ID),
		},
	}, nil
}

// GetClientsId busca um cliente pelo seu UUID.
//
// Rota: GET /v1/clients/{id}
//
// Path params (via request.Id):
//   - id (uuid, obrigatório): identificador único do cliente.
//
// Respostas:
//   - 200: ClienteResponse com os dados do cliente.
//   - 401: token ausente ou inválido.
//   - 404: cliente não encontrado.
//
// TODO: implementar busca por ID no repositório.
func (s *Server) GetClientsId(_ context.Context, request GetClientsIdRequestObject) (GetClientsIdResponseObject, error) {
	cliente, err := s.clientUseCase.FindByID(request.Id.String())
	if err != nil {
		return nil, err
	}

	resp := clienteParaResponse(cliente)

	return GetClientsId200JSONResponse(resp), nil
}

// PutClientsId atualiza os dados de um cliente existente.
//
// Rota: PUT /v1/clients/{id}
//
// Path params (via request.Id):
//   - id (uuid, obrigatório): identificador único do cliente.
//
// Body (via request.Body — AtualizarClienteRequest):
//   - nome     (string, opcional): novo nome.
//   - email    (string, opcional): novo e-mail.
//   - telefone (string, opcional): novo telefone.
//
// Nota: CPF/CNPJ é imutável e não pode ser alterado.
//
// Respostas:
//   - 200: ClienteResponse com os dados atualizados.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 404: cliente não encontrado.
//
// TODO: implementar atualização no repositório.
func (s *Server) PutClientsId(_ context.Context, request PutClientsIdRequestObject) (PutClientsIdResponseObject, error) {
	body := request.Body

	var email *string
	if body.Email != nil {
		emailValue := string(*body.Email)
		email = &emailValue
	}

	cliente, err := s.clientUseCase.Update(
		request.Id.String(),
		body.Nome,
		email,
		body.Telefone,
	)
	if err != nil {
		return nil, err
	}

	resp := clienteParaResponse(cliente)

	return PutClientsId200JSONResponse(resp), nil
}

// DeleteClientsId remove um cliente pelo seu UUID.
//
// Rota: DELETE /v1/clients/{id}
//
// Path params (via request.Id):
//   - id (uuid, obrigatório): identificador único do cliente.
//
// Respostas:
//   - 204: cliente removido com sucesso (sem corpo).
//   - 401: token ausente ou inválido.
//   - 404: cliente não encontrado.
//   - 409: cliente possui veículos ou ordens de serviço associados.
//
// TODO: implementar remoção no repositório com validação de dependências.
func (s *Server) DeleteClientsId(_ context.Context, request DeleteClientsIdRequestObject) (DeleteClientsIdResponseObject, error) {
	err := s.clientUseCase.Delete(request.Id.String())
	if err != nil {
		return nil, err
	}

	return DeleteClientsId204Response{}, nil
}

// GetClientsIdVehicles lista os veículos de um cliente específico.
//
// Rota: GET /v1/clients/{id}/vehicles
//
// Path params (via request.Id):
//   - id (uuid, obrigatório): identificador único do cliente.
//
// Query params (via request.Params):
//   - page  (int, default 1):  número da página.
//   - limit (int, default 20): itens por página.
//
// Respostas:
//   - 200: VeiculoListResponse com data[] e meta de paginação.
//   - 401: token ausente ou inválido.
//   - 404: cliente não encontrado.
//
// TODO: implementar listagem de veículos por cliente_id.
func (s *Server) GetClientsIdVehicles(
	ctx context.Context,
	request GetClientsIdVehiclesRequestObject,
) (GetClientsIdVehiclesResponseObject, error) {

	veiculos, err := s.vehicleUseCase.ListByCliente(request.Id.String())
	if err != nil {
		return nil, err
	}

	page, limit := paginacaoDefaults(request.Params.Page, request.Params.Limit)
	total := len(veiculos)
	inicio, fim := calcularFatia(total, page, limit)

	data := make([]VeiculoResponse, 0, fim-inicio)
	for _, v := range veiculos[inicio:fim] {
		data = append(data, veiculoParaResponse(v))
	}

	return GetClientsIdVehicles200JSONResponse(VeiculoListResponse{
		Data: &data,
		Meta: metaPaginacao(page, limit, total),
	}), nil
}

// ── Veículos ──────────────────────────────────────────────────────────────────

// GetVehicles retorna a lista paginada de veículos.
//
// Rota: GET /v1/vehicles
//
// Query params (via request.Params):
//   - page      (int):    número da página.
//   - limit     (int):    itens por página.
//   - placa     (string): busca parcial pela placa.
//   - cliente_id (uuid):  filtra por proprietário.
//   - marca     (string): busca parcial pela marca.
//
// Respostas:
//   - 200: VeiculoListResponse.
//   - 401: token ausente ou inválido.
//
// TODO: implementar listagem de veículos.
func (s *Server) GetVehicles(_ context.Context, request GetVehiclesRequestObject) (GetVehiclesResponseObject, error) {
	veiculos, err := s.vehicleUseCase.List()
	if err != nil {
		return nil, err
	}

	page, limit := paginacaoDefaults(request.Params.Page, request.Params.Limit)
	total := len(veiculos)
	inicio, fim := calcularFatia(total, page, limit)

	data := make([]VeiculoResponse, 0, fim-inicio)
	for _, v := range veiculos[inicio:fim] {
		data = append(data, veiculoParaResponse(v))
	}

	return GetVehicles200JSONResponse(VeiculoListResponse{
		Data: &data,
		Meta: metaPaginacao(page, limit, total),
	}), nil
}

// PostVehicles cadastra um novo veículo vinculado a um cliente.
//
// Rota: POST /v1/vehicles
//
// Body (via request.Body — CriarVeiculoRequest):
//   - cliente_id (uuid, obrigatório):   proprietário do veículo.
//   - placa      (string, obrigatório): placa única (formato antigo ou Mercosul).
//   - marca      (string, obrigatório): marca do veículo.
//   - modelo     (string, obrigatório): modelo do veículo.
//   - ano        (int, obrigatório):    ano de fabricação.
//   - cor        (string, opcional):    cor do veículo.
//
// Respostas:
//   - 201: veículo criado; header Location aponta para o novo recurso.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 404: cliente informado não encontrado.
//   - 409: placa já cadastrada.
//
// TODO: implementar criação de veículo.
func (s *Server) PostVehicles(_ context.Context, request PostVehiclesRequestObject) (PostVehiclesResponseObject, error) {
	body := request.Body

	veiculo, err := s.vehicleUseCase.Create(
		body.ClienteId.String(),
		body.Placa,
		body.Marca,
		body.Modelo,
		body.Ano,
		body.Cor,
	)
	if err != nil {
		return nil, err
	}

	resp := veiculoParaResponse(veiculo)

	return PostVehicles201JSONResponse{
		Body: resp,
		Headers: PostVehicles201ResponseHeaders{
			Location: fmt.Sprintf("/v1/vehicles/%s", veiculo.ID),
		},
	}, nil
}

// GetVehiclesId busca um veículo pelo seu UUID.
//
// Rota: GET /v1/vehicles/{id}
//
// Respostas:
//   - 200: VeiculoResponse.
//   - 401: token ausente ou inválido.
//   - 404: veículo não encontrado.
//
// TODO: implementar busca de veículo por ID.
func (s *Server) GetVehiclesId(_ context.Context, request GetVehiclesIdRequestObject) (GetVehiclesIdResponseObject, error) {
	veiculo, err := s.vehicleUseCase.FindByID(request.Id.String())
	if err != nil {
		return nil, err
	}

	resp := veiculoParaResponse(veiculo)
	return GetVehiclesId200JSONResponse(resp), nil
}

// PutVehiclesId atualiza os dados de um veículo.
//
// Rota: PUT /v1/vehicles/{id}
//
// Body (via request.Body — AtualizarVeiculoRequest):
//   - marca  (string, opcional): nova marca.
//   - modelo (string, opcional): novo modelo.
//   - ano    (int, opcional):    novo ano.
//   - cor    (string, opcional): nova cor.
//
// Nota: placa e cliente_id são imutáveis.
//
// Respostas:
//   - 200: VeiculoResponse atualizado.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 404: veículo não encontrado.
//
// TODO: implementar atualização de veículo.
func (s *Server) PutVehiclesId(_ context.Context, request PutVehiclesIdRequestObject) (PutVehiclesIdResponseObject, error) {
	body := request.Body

	veiculo, err := s.vehicleUseCase.Update(
		request.Id.String(),
		body.Marca,
		body.Modelo,
		body.Ano,
		body.Cor,
	)
	if err != nil {
		return nil, err
	}

	resp := veiculoParaResponse(veiculo)
	return PutVehiclesId200JSONResponse(resp), nil
}

// DeleteVehiclesId remove um veículo pelo seu UUID.
//
// Rota: DELETE /v1/vehicles/{id}
//
// Respostas:
//   - 204: veículo removido com sucesso.
//   - 401: token ausente ou inválido.
//   - 404: veículo não encontrado.
//   - 409: veículo possui ordens de serviço associadas.
//
// TODO: implementar remoção de veículo com validação de dependências.
func (s *Server) DeleteVehiclesId(_ context.Context, request DeleteVehiclesIdRequestObject) (DeleteVehiclesIdResponseObject, error) {
	err := s.vehicleUseCase.Delete(request.Id.String())
	if err != nil {
		return nil, err
	}

	return DeleteVehiclesId204Response{}, nil
}

// ── Serviços ──────────────────────────────────────────────────────────────────

// GetServices retorna o catálogo paginado de serviços.
//
// Rota: GET /v1/services
//
// Query params (via request.Params):
//   - page  (int):    número da página.
//   - limit (int):    itens por página.
//   - nome  (string): busca parcial case-insensitive pelo nome.
//
// Respostas:
//   - 200: ServicoListResponse.
//   - 401: token ausente ou inválido.
//
// TODO: implementar listagem de serviços.
func (s *Server) GetServices(_ context.Context, request GetServicesRequestObject) (GetServicesResponseObject, error) {
	servicos, err := s.serviceUseCase.List()
	if err != nil {
		return nil, err
	}

	page, limit := paginacaoDefaults(request.Params.Page, request.Params.Limit)
	total := len(servicos)
	inicio, fim := calcularFatia(total, page, limit)

	data := make([]ServicoResponse, 0, fim-inicio)
	for _, servico := range servicos[inicio:fim] {
		data = append(data, servicoParaResponse(servico))
	}

	return GetServices200JSONResponse(ServicoListResponse{
		Data: &data,
		Meta: metaPaginacao(page, limit, total),
	}), nil
}

// PostServices cadastra um novo serviço no catálogo.
//
// Rota: POST /v1/services
//
// Body (via request.Body — CriarServicoRequest):
//   - nome          (string, obrigatório):  nome único do serviço.
//   - descricao     (string, opcional):     descrição detalhada.
//   - preco_base    (float, obrigatório):   preço base em reais.
//   - tempo_minutos (int, obrigatório):     tempo estimado em minutos.
//
// Respostas:
//   - 201: serviço criado; header Location aponta para o novo recurso.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 409: já existe um serviço com este nome.
//
// TODO: implementar criação de serviço.
func (s *Server) PostServices(_ context.Context, request PostServicesRequestObject) (PostServicesResponseObject, error) {
	body := request.Body

	servico, err := s.serviceUseCase.Create(
		body.Nome,
		body.Descricao,
		body.PrecoBase,
		body.TempoMinutos,
	)
	if err != nil {
		return nil, err
	}

	resp := servicoParaResponse(servico)

	return PostServices201JSONResponse{
		Body: resp,
		Headers: PostServices201ResponseHeaders{
			Location: fmt.Sprintf("/v1/services/%s", servico.ID),
		},
	}, nil
}

// GetServicesId busca um serviço pelo seu UUID.
//
// Rota: GET /v1/services/{id}
//
// Respostas:
//   - 200: ServicoResponse.
//   - 401: token ausente ou inválido.
//   - 404: serviço não encontrado.
//
// TODO: implementar busca de serviço por ID.
func (s *Server) GetServicesId(_ context.Context, request GetServicesIdRequestObject) (GetServicesIdResponseObject, error) {
	servico, err := s.serviceUseCase.FindByID(request.Id.String())
	if err != nil {
		return nil, err
	}

	resp := servicoParaResponse(servico)
	return GetServicesId200JSONResponse(resp), nil
}

// PutServicesId atualiza um serviço existente.
//
// Rota: PUT /v1/services/{id}
//
// Body (via request.Body — AtualizarServicoRequest):
//   - nome          (string, opcional): novo nome.
//   - descricao     (string, opcional): nova descrição.
//   - preco_base    (float, opcional):  novo preço base.
//   - tempo_minutos (int, opcional):    novo tempo estimado.
//
// Respostas:
//   - 200: ServicoResponse atualizado.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 404: serviço não encontrado.
//   - 409: já existe outro serviço com este nome.
//
// TODO: implementar atualização de serviço.
func (s *Server) PutServicesId(_ context.Context, request PutServicesIdRequestObject) (PutServicesIdResponseObject, error) {
	body := request.Body

	servico, err := s.serviceUseCase.Update(
		request.Id.String(),
		body.Nome,
		body.Descricao,
		body.PrecoBase,
		body.TempoMinutos,
	)
	if err != nil {
		return nil, err
	}

	resp := servicoParaResponse(servico)
	return PutServicesId200JSONResponse(resp), nil
}

// DeleteServicesId remove um serviço do catálogo.
//
// Rota: DELETE /v1/services/{id}
//
// Respostas:
//   - 204: serviço removido com sucesso.
//   - 401: token ausente ou inválido.
//   - 404: serviço não encontrado.
//   - 409: serviço referenciado em ordens de serviço existentes.
//
// TODO: implementar remoção de serviço com validação de dependências.
func (s *Server) DeleteServicesId(_ context.Context, request DeleteServicesIdRequestObject) (DeleteServicesIdResponseObject, error) {
	err := s.serviceUseCase.Delete(request.Id.String())
	if err != nil {
		return nil, err
	}

	return DeleteServicesId204Response{}, nil
}

// ── Peças ─────────────────────────────────────────────────────────────────────

// GetParts retorna o catálogo paginado de peças e insumos.
//
// Rota: GET /v1/parts
//
// Query params (via request.Params):
//   - page         (int):     número da página.
//   - limit        (int):     itens por página.
//   - nome         (string):  busca parcial pelo nome.
//   - codigo       (string):  busca parcial pelo código.
//   - estoque_baixo (bool):   se true, retorna apenas peças com estoque_atual < estoque_minimo.
//
// Respostas:
//   - 200: PecaListResponse.
//   - 401: token ausente ou inválido.
//
// TODO: implementar listagem de peças.
func (s *Server) GetParts(_ context.Context, request GetPartsRequestObject) (GetPartsResponseObject, error) {
	pecas, err := s.partUseCase.List()
	if err != nil {
		return nil, err
	}

	page, limit := paginacaoDefaults(request.Params.Page, request.Params.Limit)
	total := len(pecas)
	inicio, fim := calcularFatia(total, page, limit)

	data := make([]PecaResponse, 0, fim-inicio)
	for _, peca := range pecas[inicio:fim] {
		data = append(data, pecaParaResponse(peca))
	}

	return GetParts200JSONResponse(PecaListResponse{
		Data: &data,
		Meta: metaPaginacao(page, limit, total),
	}), nil
}

// PostParts cadastra uma nova peça no catálogo.
//
// Rota: POST /v1/parts
//
// Body (via request.Body — CriarPecaRequest):
//   - nome           (string, obrigatório): nome da peça.
//   - codigo         (string, obrigatório): código único de identificação.
//   - preco          (float, obrigatório):  preço de venda em reais.
//   - estoque_minimo (int, obrigatório):    quantidade mínima para alerta.
//   - estoque_atual  (int, opcional):       quantidade inicial em estoque (default 0).
//
// Respostas:
//   - 201: peça criada; header Location aponta para o novo recurso.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 409: já existe uma peça com este código.
//
// TODO: implementar criação de peça.
func (s *Server) PostParts(_ context.Context, request PostPartsRequestObject) (PostPartsResponseObject, error) {
	body := request.Body

	peca, err := s.partUseCase.Create(
		body.Nome,
		body.Codigo,
		body.Preco,
		body.EstoqueAtual,
		body.EstoqueMinimo,
	)
	if err != nil {
		return nil, err
	}

	resp := pecaParaResponse(peca)

	return PostParts201JSONResponse{
		Body: resp,
		Headers: PostParts201ResponseHeaders{
			Location: fmt.Sprintf("/v1/parts/%s", peca.ID),
		},
	}, nil
}

// GetPartsId busca uma peça pelo seu UUID.
//
// Rota: GET /v1/parts/{id}
//
// Respostas:
//   - 200: PecaResponse (inclui campo calculado estoque_baixo).
//   - 401: token ausente ou inválido.
//   - 404: peça não encontrada.
//
// TODO: implementar busca de peça por ID.
func (s *Server) GetPartsId(_ context.Context, request GetPartsIdRequestObject) (GetPartsIdResponseObject, error) {
	peca, err := s.partUseCase.FindByID(request.Id.String())
	if err != nil {
		return nil, err
	}

	resp := pecaParaResponse(peca)
	return GetPartsId200JSONResponse(resp), nil
}

// PutPartsId atualiza os dados de uma peça.
//
// Rota: PUT /v1/parts/{id}
//
// Body (via request.Body — AtualizarPecaRequest):
//   - nome           (string, opcional): novo nome.
//   - preco          (float, opcional):  novo preço.
//   - estoque_minimo (int, opcional):    novo mínimo de estoque.
//
// Nota: o código da peça é imutável. Para movimentar estoque use PATCH /parts/{id}/stock.
//
// Respostas:
//   - 200: PecaResponse atualizada.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 404: peça não encontrada.
//
// TODO: implementar atualização de peça.
func (s *Server) PutPartsId(_ context.Context, request PutPartsIdRequestObject) (PutPartsIdResponseObject, error) {
	body := request.Body

	peca, err := s.partUseCase.Update(
		request.Id.String(),
		body.Nome,
		body.Preco,
		body.EstoqueMinimo,
	)
	if err != nil {
		return nil, err
	}

	resp := pecaParaResponse(peca)
	return PutPartsId200JSONResponse(resp), nil
}

// DeletePartsId remove uma peça do catálogo.
//
// Rota: DELETE /v1/parts/{id}
//
// Respostas:
//   - 204: peça removida com sucesso.
//   - 401: token ausente ou inválido.
//   - 404: peça não encontrada.
//   - 409: peça referenciada em ordens de serviço existentes.
//
// TODO: implementar remoção de peça com validação de dependências.
func (s *Server) DeletePartsId(_ context.Context, request DeletePartsIdRequestObject) (DeletePartsIdResponseObject, error) {
	err := s.partUseCase.Delete(request.Id.String())
	if err != nil {
		return nil, err
	}

	return DeletePartsId204Response{}, nil
}

// PatchPartsIdStock registra uma movimentação de estoque para uma peça.
//
// Rota: PATCH /v1/parts/{id}/stock
//
// Body (via request.Body — AjustarEstoqueRequest):
//   - quantidade (int, obrigatório):    quantidade a movimentar (mínimo 1).
//   - tipo       (string, obrigatório): tipo de movimentação:
//   - "entrada" → estoque_atual += quantidade
//   - "saida"   → estoque_atual -= quantidade (422 se insuficiente)
//   - "ajuste"  → estoque_atual  = quantidade (acerto de inventário)
//   - motivo    (string, opcional):     justificativa da movimentação.
//
// Respostas:
//   - 200: PecaResponse com estoque atualizado.
//   - 400: payload inválido.
//   - 401: token ausente ou inválido.
//   - 404: peça não encontrada.
//   - 422: estoque insuficiente para realizar saída.
//
// TODO: implementar movimentação de estoque.
func (s *Server) PatchPartsIdStock(_ context.Context, request PatchPartsIdStockRequestObject) (PatchPartsIdStockResponseObject, error) {
	body := request.Body

	peca, err := s.partUseCase.AdjustStock(
		request.Id.String(),
		string(body.Tipo),
		body.Quantidade,
	)
	if err != nil {
		if errors.Is(err, usecase.ErrEstoqueInsuficiente) {
			return PatchPartsIdStock422JSONResponse{
				Code:    "INSUFFICIENT_STOCK",
				Message: err.Error(),
			}, nil
		}

		return nil, err
	}

	resp := pecaParaResponse(peca)
	return PatchPartsIdStock200JSONResponse(resp), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// clienteParaResponse converte a entidade de domínio Cliente para o DTO de resposta
// ClienteResponse utilizado pela camada HTTP.
func clienteParaResponse(c *entity.Cliente) ClienteResponse {
	id := openapi_types.UUID(c.ID)
	cpfCnpj := c.CpfCnpj
	return ClienteResponse{
		Id:           &id,
		Nome:         &c.Nome,
		CpfCnpj:      &cpfCnpj,
		Email:        (*openapi_types.Email)(c.Email),
		Telefone:     c.Telefone,
		CriadoEm:     &c.CriadoEm,
		AtualizadoEm: &c.AtualizadoEm,
	}
}

// veiculoParaResponse converte a entidade de domínio veiculo para o DTO de resposta
// veiculoResponse utilizado pela camada HTTP.
func veiculoParaResponse(v *entity.Veiculo) VeiculoResponse {
	id := openapi_types.UUID(v.ID)
	clienteID := openapi_types.UUID(v.ClienteID)

	return VeiculoResponse{
		Id:           &id,
		ClienteId:    &clienteID,
		Placa:        &v.Placa,
		Marca:        &v.Marca,
		Modelo:       &v.Modelo,
		Ano:          &v.Ano,
		Cor:          v.Cor,
		CriadoEm:     &v.CriadoEm,
		AtualizadoEm: &v.AtualizadoEm,
	}
}

// servicoParaResponse a entidade de domínio servicos para o DTO de resposta
// servicoResponse utilizado pela camada HTTP.
func servicoParaResponse(s *entity.Servico) ServicoResponse {
	id := openapi_types.UUID(s.ID)

	return ServicoResponse{
		Id:           &id,
		Nome:         &s.Nome,
		Descricao:    s.Descricao,
		PrecoBase:    &s.PrecoBase,
		TempoMinutos: &s.TempoMinutos,
		CriadoEm:     &s.CriadoEm,
		AtualizadoEm: &s.AtualizadoEm,
	}
}

// pecaParaResponse a entidade de domínio servicos para o DTO de resposta
// pecaResponse utilizado pela camada HTTP.
func pecaParaResponse(p *entity.Peca) PecaResponse {
	id := openapi_types.UUID(p.ID)
	estoqueBaixo := p.EstoqueAtual < p.EstoqueMinimo

	return PecaResponse{
		Id:            &id,
		Nome:          &p.Nome,
		Codigo:        &p.Codigo,
		Preco:         &p.Preco,
		EstoqueAtual:  &p.EstoqueAtual,
		EstoqueMinimo: &p.EstoqueMinimo,
		EstoqueBaixo:  &estoqueBaixo,
		CriadoEm:      &p.CriadoEm,
		AtualizadoEm:  &p.AtualizadoEm,
	}
}

// paginacaoDefaults retorna os valores de página e limite com defaults aplicados
// quando os query params não são informados ou são inválidos.
//
// Defaults: page=1, limit=20.
func paginacaoDefaults(page, limit *int) (int, int) {
	p, l := 1, 20
	if page != nil && *page > 0 {
		p = *page
	}
	if limit != nil && *limit > 0 {
		l = *limit
	}
	return p, l
}

// calcularFatia retorna os índices de início e fim para aplicar paginação
// em memória sobre um slice de tamanho total.
//
// Parâmetros:
//   - total: tamanho total do slice.
//   - page:  página solicitada (1-based).
//   - limit: itens por página.
//
// Retorno:
//   - inicio: índice inclusivo de início.
//   - fim:    índice exclusivo de fim.
func calcularFatia(total, page, limit int) (inicio, fim int) {
	inicio = (page - 1) * limit
	if inicio > total {
		inicio = total
	}
	fim = inicio + limit
	if fim > total {
		fim = total
	}
	return
}

// metaPaginacao constrói o objeto PaginationMeta a partir dos parâmetros
// de paginação e do total de registros encontrados.
func metaPaginacao(page, limit, total int) *PaginationMeta {
	totalPages := 1
	if limit > 0 && total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return &PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
