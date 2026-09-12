package api

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
)

// A tabela de autorização vive no contrato F3-0.2. O que estes testes
// garantem é a linha mais importante dela: um token de cliente autentica, mas
// não autoriza operação interna nem enxerga OS de terceiros.
//
// Vale lembrar por que isso não é redundante com o authorizer do API Gateway:
// ele valida a assinatura e para por aí. Um token de cliente perfeitamente
// assinado passa por ele e chega aqui.

func TestCliente_NaoCriaOrdemDeServico(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.PostWorkOrders(ctxCliente(f.clienteID.String()), PostWorkOrdersRequestObject{
		Body: &CriarOrdemServicoRequest{ClienteId: f.clienteID, VeiculoId: f.veiculoID},
	})

	var proibido *domainerros.ErrProibido
	if !errors.As(err, &proibido) {
		t.Fatalf("esperava ErrProibido, obteve %v", err)
	}
}

func TestCliente_NaoListaClientes(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.GetClients(ctxCliente(f.clienteID.String()), GetClientsRequestObject{})

	var proibido *domainerros.ErrProibido
	if !errors.As(err, &proibido) {
		t.Fatalf("esperava ErrProibido, obteve %v", err)
	}
}

func TestUsuario_CriaOrdemDeServico(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.PostWorkOrders(ctx(), PostWorkOrdersRequestObject{
		Body: &CriarOrdemServicoRequest{ClienteId: f.clienteID, VeiculoId: f.veiculoID},
	})

	var proibido *domainerros.ErrProibido
	if errors.As(err, &proibido) {
		t.Fatal("funcionário não deveria ser barrado por autorização")
	}
}

// O filtro é IMPOSTO, não aceito do parâmetro: sem isso, bastaria mandar
// cliente_id de outra pessoa para ler as OS dela.
func TestCliente_ListaSomenteAsProprias(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{osCompleta: f.osC}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	outro := uuid.New()
	_, err := srv.GetWorkOrders(ctxCliente(f.clienteID.String()), GetWorkOrdersRequestObject{
		Params: GetWorkOrdersParams{ClienteId: &outro}, // tentativa de bisbilhotar
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}

	if osR.paramsListar.ClienteID == nil {
		t.Fatal("o filtro por cliente deveria ter sido imposto")
	}
	if *osR.paramsListar.ClienteID != f.clienteID {
		t.Fatalf("filtrou por %s, esperava o sub do token (%s)",
			*osR.paramsListar.ClienteID, f.clienteID)
	}
}

func TestUsuario_ListaSemFiltroImposto(t *testing.T) {
	f := novasFixtures()
	osR := &stubOsRepo{osCompleta: f.osC}
	srv := novoServerDeTeste(f, nil, nil, nil, nil, osR, nil)

	if _, err := srv.GetWorkOrders(ctx(), GetWorkOrdersRequestObject{}); err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if osR.paramsListar.ClienteID != nil {
		t.Fatalf("funcionário não deveria ter filtro imposto, veio %s", *osR.paramsListar.ClienteID)
	}
}

// 404 e não 403: um 403 confirmaria que a OS existe, o que já é informação
// demais para quem não deveria vê-la.
func TestCliente_OSDeOutroClienteRespondeNaoEncontrado(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.GetWorkOrdersId(ctxCliente(uuid.NewString()), GetWorkOrdersIdRequestObject{
		Id: f.osC.ID,
	})

	var naoEncontrado *domainerros.ErrNaoEncontrado
	if !errors.As(err, &naoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, obteve %v", err)
	}
}

func TestCliente_LeAPropriaOS(t *testing.T) {
	f := novasFixtures()
	srv := novoServerDeTeste(f, nil, nil, nil, nil, nil, nil)

	_, err := srv.GetWorkOrdersId(ctxCliente(f.clienteID.String()), GetWorkOrdersIdRequestObject{
		Id: f.osC.ID,
	})
	if err != nil {
		t.Fatalf("o dono deveria enxergar a própria OS, obteve %v", err)
	}
}
