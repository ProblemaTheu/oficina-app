package api

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/ProblemaTheu/oficina-app/internal/infra/http/middleware"
)

// Toda operação interna é barrada para token de cliente — esta é a tabela
// completa do contrato F3-0.2, não só a amostra de autorizacao_test.go.
// Cada linha chama o handler direto com um contexto de cliente; a resposta
// tem que ser ErrProibido antes de qualquer acesso a caso de uso.
func TestCliente_TodasAsOperacoesInternasSaoProibidas(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)
	ctx := ctxCliente(f.clienteID.String())
	id := uuid.New()

	handlers := map[string]func(context.Context) error{
		"GetClients":  func(c context.Context) error { _, e := srv.GetClients(c, GetClientsRequestObject{}); return e },
		"PostClients": func(c context.Context) error { _, e := srv.PostClients(c, PostClientsRequestObject{}); return e },
		"GetClientsId": func(c context.Context) error {
			_, e := srv.GetClientsId(c, GetClientsIdRequestObject{Id: id})
			return e
		},
		"PutClientsId": func(c context.Context) error {
			_, e := srv.PutClientsId(c, PutClientsIdRequestObject{Id: id})
			return e
		},
		"DeleteClientsId": func(c context.Context) error {
			_, e := srv.DeleteClientsId(c, DeleteClientsIdRequestObject{Id: id})
			return e
		},
		"GetClientsIdVehicles": func(c context.Context) error {
			_, e := srv.GetClientsIdVehicles(c, GetClientsIdVehiclesRequestObject{Id: id})
			return e
		},
		"GetVehicles":  func(c context.Context) error { _, e := srv.GetVehicles(c, GetVehiclesRequestObject{}); return e },
		"PostVehicles": func(c context.Context) error { _, e := srv.PostVehicles(c, PostVehiclesRequestObject{}); return e },
		"GetVehiclesId": func(c context.Context) error {
			_, e := srv.GetVehiclesId(c, GetVehiclesIdRequestObject{Id: id})
			return e
		},
		"PutVehiclesId": func(c context.Context) error {
			_, e := srv.PutVehiclesId(c, PutVehiclesIdRequestObject{Id: id})
			return e
		},
		"DeleteVehiclesId": func(c context.Context) error {
			_, e := srv.DeleteVehiclesId(c, DeleteVehiclesIdRequestObject{Id: id})
			return e
		},
		"GetServices":  func(c context.Context) error { _, e := srv.GetServices(c, GetServicesRequestObject{}); return e },
		"PostServices": func(c context.Context) error { _, e := srv.PostServices(c, PostServicesRequestObject{}); return e },
		"GetServicesId": func(c context.Context) error {
			_, e := srv.GetServicesId(c, GetServicesIdRequestObject{Id: id})
			return e
		},
		"PutServicesId": func(c context.Context) error {
			_, e := srv.PutServicesId(c, PutServicesIdRequestObject{Id: id})
			return e
		},
		"DeleteServicesId": func(c context.Context) error {
			_, e := srv.DeleteServicesId(c, DeleteServicesIdRequestObject{Id: id})
			return e
		},
		"GetParts":   func(c context.Context) error { _, e := srv.GetParts(c, GetPartsRequestObject{}); return e },
		"PostParts":  func(c context.Context) error { _, e := srv.PostParts(c, PostPartsRequestObject{}); return e },
		"GetPartsId": func(c context.Context) error { _, e := srv.GetPartsId(c, GetPartsIdRequestObject{Id: id}); return e },
		"PutPartsId": func(c context.Context) error { _, e := srv.PutPartsId(c, PutPartsIdRequestObject{Id: id}); return e },
		"DeletePartsId": func(c context.Context) error {
			_, e := srv.DeletePartsId(c, DeletePartsIdRequestObject{Id: id})
			return e
		},
		"PatchPartsIdStock": func(c context.Context) error {
			_, e := srv.PatchPartsIdStock(c, PatchPartsIdStockRequestObject{Id: id})
			return e
		},
		"PostWorkOrders": func(c context.Context) error { _, e := srv.PostWorkOrders(c, PostWorkOrdersRequestObject{}); return e },
		"PatchWorkOrdersIdStatus": func(c context.Context) error {
			_, e := srv.PatchWorkOrdersIdStatus(c, PatchWorkOrdersIdStatusRequestObject{Id: id})
			return e
		},
		"PostWorkOrdersIdApprove": func(c context.Context) error {
			_, e := srv.PostWorkOrdersIdApprove(c, PostWorkOrdersIdApproveRequestObject{Id: id})
			return e
		},
		"PostWorkOrdersIdReject": func(c context.Context) error {
			_, e := srv.PostWorkOrdersIdReject(c, PostWorkOrdersIdRejectRequestObject{Id: id})
			return e
		},
		"GetReportsAvgExecutionTime": func(c context.Context) error {
			_, e := srv.GetReportsAvgExecutionTime(c, GetReportsAvgExecutionTimeRequestObject{})
			return e
		},
		"PostAuthRegister": func(c context.Context) error {
			_, e := srv.PostAuthRegister(c, PostAuthRegisterRequestObject{})
			return e
		},
	}

	for nome, chamar := range handlers {
		t.Run(nome, func(t *testing.T) {
			var proibido *domainerros.ErrProibido
			if err := chamar(ctx); !errors.As(err, &proibido) {
				t.Fatalf("token de cliente deveria receber ErrProibido, obteve %v", err)
			}
		})
	}
}

// Criar usuário aceita o papel desejado no corpo: só administrador pode.
func TestFuncionarioSemPapelDeAdmin_NaoRegistraUsuario(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	atendente := context.WithValue(context.Background(), middleware.ClaimsContextKey,
		jwt.MapClaims{"sub": "func-2", "tipo": middleware.TipoUsuario, "papel": "atendente"})
	_, err := srv.PostAuthRegister(atendente, PostAuthRegisterRequestObject{})

	var proibido *domainerros.ErrProibido
	if !errors.As(err, &proibido) {
		t.Fatalf("funcionário sem papel de administrador deveria receber ErrProibido, obteve %v", err)
	}
	if proibido.Error() != proibido.Mensagem {
		t.Fatalf("Error() = %q, esperado a mensagem", proibido.Error())
	}
}
