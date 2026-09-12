package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/ProblemaTheu/oficina-app/internal/application/logging"
)

// capturarLogs troca o logger default por um JSON em memória durante o teste
// e devolve o buffer — o middleware deriva o logger de slog.Default().
func capturarLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	anterior := slog.Default()
	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	return buf
}

func ultimoLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	linhas := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	var registro map[string]any
	if err := json.Unmarshal(linhas[len(linhas)-1], &registro); err != nil {
		t.Fatalf("log não é JSON: %v\n%s", err, buf.String())
	}
	return registro
}

func TestCorrelacao_ReaproveitaIDDoCliente(t *testing.T) {
	buf := capturarLogs(t)
	var idNoContexto string
	handler := Correlacao()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// O logger do contexto já carrega o correlation_id.
		logging.Log(r.Context()).Info("dentro do handler")
		idNoContexto = w.Header().Get(HeaderCorrelacao)
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/work-orders", nil)
	req.Header.Set(HeaderCorrelacao, "abc-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderCorrelacao); got != "abc-123" {
		t.Fatalf("header de resposta = %q, esperado abc-123", got)
	}
	if idNoContexto != "abc-123" {
		t.Fatalf("header visível ao handler = %q, esperado abc-123", idNoContexto)
	}

	registro := ultimoLog(t, buf)
	if registro["msg"] != "http request completed" {
		t.Fatalf("último log = %v, esperado o access log", registro["msg"])
	}
	if registro["correlation_id"] != "abc-123" {
		t.Errorf("correlation_id = %v", registro["correlation_id"])
	}
	if registro["http.status_code"] != float64(http.StatusCreated) {
		t.Errorf("status_code = %v, esperado 201", registro["http.status_code"])
	}
	if registro["http.method"] != http.MethodPost || registro["http.route"] != "/v1/work-orders" {
		t.Errorf("método/rota = %v %v", registro["http.method"], registro["http.route"])
	}
	if _, ok := registro["http.duration_ms"]; !ok {
		t.Errorf("access log sem http.duration_ms")
	}
}

func TestCorrelacao_GeraUUIDQuandoAusente(t *testing.T) {
	capturarLogs(t)
	handler := Correlacao()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Escrever sem WriteHeader explícito: o recorder assume 200.
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	id := rec.Header().Get(HeaderCorrelacao)
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("header %q não é UUID: %v", id, err)
	}
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("resposta = %d %q", rec.Code, rec.Body.String())
	}
}

func TestStatusRecorder_StatusImplicitoNoWrite(t *testing.T) {
	buf := capturarLogs(t)
	handler := Correlacao()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("sem WriteHeader"))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if got := ultimoLog(t, buf)["http.status_code"]; got != float64(http.StatusOK) {
		t.Fatalf("status_code registrado = %v, esperado 200 implícito", got)
	}
}
