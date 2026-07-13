package api

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
)

// TestNovoServer_ConstroiTodasAsDependencias garante que a montagem do grafo
// de dependências não depende de conexão ativa (os repositórios só usam o *sql.DB
// nas chamadas, não na construção).
func TestNovoServer_ConstroiTodasAsDependencias(t *testing.T) {
	srv := NovoServer(nil)
	if srv == nil {
		t.Fatal("NovoServer não deve retornar nil")
	}
	if srv.clientUseCase == nil || srv.vehicleUseCase == nil || srv.serviceUseCase == nil ||
		srv.partUseCase == nil || srv.osUseCase == nil || srv.authUseCase == nil {
		t.Error("todos os use cases devem ser injetados na construção")
	}
}

func TestGetClientsIdVehicles_ClienteInexistente(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "cliente"}}
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, nil)

	if _, err := srv.GetClientsIdVehicles(ctx(), GetClientsIdVehiclesRequestObject{
		Id: uuid.New(), Params: GetClientsIdVehiclesParams{},
	}); err == nil {
		t.Fatal("esperava erro quando o cliente não existe")
	}
}

func TestGetWorkOrdersIdStatus_NaoEncontrada(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "ordem de serviço"}}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	if _, err := srv.GetWorkOrdersIdStatus(ctx(), GetWorkOrdersIdStatusRequestObject{Id: uuid.New()}); err == nil {
		t.Fatal("esperava erro de não encontrado")
	}
}

func TestPostWorkOrdersIdReject_Erro(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{err: errors.New("falha de banco")}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	if _, err := srv.PostWorkOrdersIdReject(ctx(), PostWorkOrdersIdRejectRequestObject{Id: uuid.New()}); err == nil {
		t.Fatal("esperava propagação do erro")
	}
}
