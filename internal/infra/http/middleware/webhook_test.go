package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func assinar(body string) string {
	mac := hmac.New(sha256.New, webhookSecret())
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func executar(t *testing.T, path, body, signature string) *httptest.ResponseRecorder {
	t.Helper()

	handler := AssinaturaWebhook()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if signature != "" {
		req.Header.Set("X-Signature", signature)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestAssinaturaWebhook_Valida(t *testing.T) {
	body := `{"os_id":"abc","decisao":"aprovado"}`
	rec := executar(t, "/v1/webhooks/budget-response", body, assinar(body))
	if rec.Code != http.StatusOK {
		t.Errorf("esperava 200 com assinatura válida, obteve %d", rec.Code)
	}
}

func TestAssinaturaWebhook_ValidaMaiuscula(t *testing.T) {
	body := `{"decisao":"recusado"}`
	rec := executar(t, "/v1/webhooks/budget-response", body, strings.ToUpper(assinar(body)))
	if rec.Code != http.StatusOK {
		t.Errorf("esperava 200 com assinatura hex maiúscula, obteve %d", rec.Code)
	}
}

func TestAssinaturaWebhook_Invalida(t *testing.T) {
	body := `{"decisao":"aprovado"}`
	rec := executar(t, "/v1/webhooks/budget-response", body, assinar("outro corpo"))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 com assinatura inválida, obteve %d", rec.Code)
	}
}

func TestAssinaturaWebhook_Ausente(t *testing.T) {
	rec := executar(t, "/v1/webhooks/budget-response", `{}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 sem assinatura, obteve %d", rec.Code)
	}
}

func TestAssinaturaWebhook_RotaForaDoPrefixoPassaDireto(t *testing.T) {
	rec := executar(t, "/v1/clients", `{}`, "")
	if rec.Code != http.StatusOK {
		t.Errorf("rota fora de /v1/webhooks/ não deve exigir assinatura; obteve %d", rec.Code)
	}
}

func TestAssinaturaWebhook_CorpoPreservado(t *testing.T) {
	body := `{"decisao":"aprovado"}`
	var recebido string

	handler := AssinaturaWebhook()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, len(body))
		n, _ := r.Body.Read(b)
		recebido = string(b[:n])
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/budget-response", strings.NewReader(body))
	req.Header.Set("X-Signature", assinar(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if recebido != body {
		t.Errorf("corpo deveria ser preservado para o handler; esperava %q, obteve %q", body, recebido)
	}
}
