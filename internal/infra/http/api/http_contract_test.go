package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Testes de contrato HTTP: atravessam o router e os wrappers gerados pelo
// oapi-codegen (api.gen.go) com requisições reais — binding de path/query/body,
// serialização das respostas e mapeamento de erros — usando a mesma montagem
// de main.go (strict handler + chi + TratarErro*).

func novaAPIHTTP(f fixtures, usrR *stubUsuarioRepo, cliR *stubClienteRepo) http.Handler {
	srv := novoServerDeTeste(f, cliR, nil, nil, nil, nil, usrR)
	strict := NewStrictHandlerWithOptions(srv, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  TratarErroRequisicao,
		ResponseErrorHandlerFunc: TratarErroResposta,
	})
	return HandlerFromMuxWithBaseURL(strict, chi.NewRouter(), "/v1")
}

func requisitar(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("X-Signature", "assinatura-validada-pelo-middleware")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestContratoHTTP_TodasAsRotas percorre cada operação do contrato OpenAPI
// pelo caminho feliz, validando status code e que a resposta é JSON quando há corpo.
func TestContratoHTTP_TodasAsRotas(t *testing.T) {
	f := novasFixtures()
	f.osC.StatusNome = entity.StatusAguardandoAprovacao

	hash, err := bcrypt.GenerateFromPassword([]byte("senha123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("falha ao gerar hash: %v", err)
	}
	usrR := &stubUsuarioRepo{
		usuario:   &entity.Usuario{ID: uuid.New(), Nome: "Admin", Email: "admin@oficina.com", SenhaHash: string(hash)},
		nomePapel: "atendente",
	}
	h := novaAPIHTTP(f, usrR, nil)

	clienteID := f.clienteID.String()
	veiculoID := f.veiculoID.String()
	servicoID := f.servico.ID.String()
	pecaID := f.peca.ID.String()
	osID := f.osC.ID.String()

	casos := []struct {
		metodo, rota, corpo string
		statusEsperado      int
	}{
		{http.MethodPost, "/v1/auth/login", `{"email":"admin@oficina.com","senha":"senha123"}`, 200},
		{http.MethodPost, "/v1/auth/register", `{"nome":"Novo","email":"novo@oficina.com","senha":"s3nh4","papel":"atendente"}`, 201},

		{http.MethodGet, "/v1/clients?page=1&limit=10", "", 200},
		{http.MethodGet, "/v1/clients?documento=" + cpfValido, "", 200},
		{http.MethodPost, "/v1/clients", fmt.Sprintf(`{"nome":"Maria","documento":"%s"}`, cpfValido), 201},
		{http.MethodGet, "/v1/clients/" + clienteID, "", 200},
		{http.MethodPut, "/v1/clients/" + clienteID, `{"nome":"João Atualizado"}`, 200},
		{http.MethodDelete, "/v1/clients/" + clienteID, "", 204},
		{http.MethodGet, "/v1/clients/" + clienteID + "/vehicles", "", 200},

		{http.MethodGet, "/v1/vehicles?placa=ABC", "", 200},
		{http.MethodPost, "/v1/vehicles", fmt.Sprintf(`{"cliente_id":"%s","placa":"XYZ9A88","marca":"Fiat","modelo":"Uno","ano":2019}`, clienteID), 201},
		{http.MethodGet, "/v1/vehicles/" + veiculoID, "", 200},
		{http.MethodPut, "/v1/vehicles/" + veiculoID, `{"cor":"azul"}`, 200},
		{http.MethodDelete, "/v1/vehicles/" + veiculoID, "", 204},

		{http.MethodGet, "/v1/services", "", 200},
		{http.MethodPost, "/v1/services", `{"nome":"Alinhamento","preco_base":120,"tempo_minutos":45}`, 201},
		{http.MethodGet, "/v1/services/" + servicoID, "", 200},
		{http.MethodPut, "/v1/services/" + servicoID, `{"preco_base":175}`, 200},
		{http.MethodDelete, "/v1/services/" + servicoID, "", 204},

		{http.MethodGet, "/v1/parts?estoque_baixo=true", "", 200},
		{http.MethodPost, "/v1/parts", `{"nome":"Pastilha","codigo":"PF-010","preco":89.9,"estoque_minimo":4}`, 201},
		{http.MethodGet, "/v1/parts/" + pecaID, "", 200},
		{http.MethodPut, "/v1/parts/" + pecaID, `{"preco":42}`, 200},
		{http.MethodPatch, "/v1/parts/" + pecaID + "/stock", `{"tipo":"entrada","quantidade":5}`, 200},
		{http.MethodDelete, "/v1/parts/" + pecaID, "", 204},

		{http.MethodGet, "/v1/work-orders?status=aguardando_aprovacao&incluir_encerradas=true", "", 200},
		{http.MethodPost, "/v1/work-orders", fmt.Sprintf(
			`{"cliente_id":"%s","veiculo_id":"%s","servicos":[{"servico_id":"%s","quantidade":1}],"pecas":[{"peca_id":"%s"}]}`,
			clienteID, veiculoID, servicoID, pecaID), 201},
		{http.MethodGet, "/v1/work-orders/" + osID, "", 200},
		{http.MethodGet, "/v1/work-orders/" + osID + "/status", "", 200},
		{http.MethodPatch, "/v1/work-orders/" + osID + "/status", `{"status":"em_execucao"}`, 200},
		{http.MethodPost, "/v1/work-orders/" + osID + "/approve", "", 200},
		{http.MethodPost, "/v1/work-orders/" + osID + "/reject", `{"motivo":"caro demais"}`, 200},

		{http.MethodPost, "/v1/webhooks/budget-response", fmt.Sprintf(`{"os_id":"%s","decisao":"aprovado"}`, osID), 200},

		{http.MethodGet, "/v1/reports/avg-execution-time?servico_id=" + servicoID + "&data_inicio=2026-01-01&data_fim=2026-06-30", "", 200},
	}

	for _, c := range casos {
		t.Run(c.metodo+" "+c.rota, func(t *testing.T) {
			rec := requisitar(t, h, c.metodo, c.rota, c.corpo)
			if rec.Code != c.statusEsperado {
				t.Fatalf("esperava %d, obteve %d — corpo: %s", c.statusEsperado, rec.Code, rec.Body.String())
			}
			if rec.Code != http.StatusNoContent && !json.Valid(rec.Body.Bytes()) {
				t.Errorf("resposta deveria ser JSON válido: %s", rec.Body.String())
			}
		})
	}
}

func TestContratoHTTP_UUIDInvalidoNoPathRetorna400(t *testing.T) {
	f := novasFixtures()
	h := novaAPIHTTP(f, nil, nil)

	rec := requisitar(t, h, http.MethodGet, "/v1/clients/nao-e-um-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para UUID inválido, obteve %d", rec.Code)
	}
	// O wrapper gerado responde o erro de binding em texto puro (http.Error),
	// pois HandlerFromMuxWithBaseURL não configura ErrorHandlerFunc.
	if !strings.Contains(rec.Body.String(), "Invalid format for parameter id") {
		t.Errorf("esperava mensagem de formato inválido, obteve: %s", rec.Body.String())
	}
}

func TestContratoHTTP_JSONMalformadoRetorna400(t *testing.T) {
	f := novasFixtures()
	h := novaAPIHTTP(f, nil, nil)

	rec := requisitar(t, h, http.MethodPost, "/v1/clients", `{"nome": sem-aspas}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para JSON malformado, obteve %d", rec.Code)
	}
}

func TestContratoHTTP_ErroDeDominioViraJSONComStatusCorreto(t *testing.T) {
	f := novasFixtures()
	cliR := &stubClienteRepo{err: &domainerros.ErrNaoEncontrado{Recurso: "cliente"}}
	h := novaAPIHTTP(f, nil, cliR)

	rec := requisitar(t, h, http.MethodGet, "/v1/clients/"+uuid.NewString(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperava 404, obteve %d", rec.Code)
	}
	var body Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if body.Code != "NOT_FOUND" {
		t.Errorf("esperava code NOT_FOUND, obteve %q", body.Code)
	}
}

func TestContratoHTTP_RotaDesconhecidaRetorna404(t *testing.T) {
	f := novasFixtures()
	h := novaAPIHTTP(f, nil, nil)

	rec := requisitar(t, h, http.MethodGet, "/v1/rota-inexistente", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("esperava 404 para rota desconhecida, obteve %d", rec.Code)
	}
}
