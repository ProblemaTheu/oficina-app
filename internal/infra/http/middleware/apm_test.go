package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func TestAPM_SemAgenteDevolveACadeiaIntacta(t *testing.T) {
	handler := APM(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, esperado 418 (handler original)", rec.Code)
	}
}

// Agente desabilitado (sem licença): StartTransaction funciona e não envia
// nada — suficiente para exercitar o caminho instrumentado.
func agenteDesabilitado(t *testing.T) *newrelic.Application {
	t.Helper()
	app, err := newrelic.NewApplication(newrelic.ConfigEnabled(false))
	if err != nil {
		t.Fatalf("agente desabilitado: %v", err)
	}
	return app
}

func TestAPM_ComAgenteColocaTransacaoNoContexto(t *testing.T) {
	var txn *newrelic.Transaction
	handler := APM(agenteDesabilitado(t))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txn = newrelic.FromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/v1/x", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, esperado 204", rec.Code)
	}
	if txn == nil {
		t.Fatal("handler não recebeu a transação no contexto")
	}
}

// Com o chi, o nome da transação usa o padrão da rota ({id}), não o path
// concreto — senão cada OS viraria uma transação diferente no APM.
func TestAPM_NomeiaPeloPadraoDaRota(t *testing.T) {
	r := chi.NewRouter()
	r.Use(APM(agenteDesabilitado(t)))
	var padrao string
	r.Get("/v1/work-orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		padrao = chi.RouteContext(r.Context()).RoutePattern()
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/work-orders/abc", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if padrao != "/v1/work-orders/{id}" {
		t.Fatalf("padrão da rota = %q", padrao)
	}
}
