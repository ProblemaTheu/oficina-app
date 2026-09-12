package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookSecret_ViaAmbiente(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", "segredo-do-teste-com-32-ou-mais-caracteres")

	body := `{"decisao":"aprovado"}`
	mac := hmac.New(sha256.New, []byte("segredo-do-teste-com-32-ou-mais-caracteres"))
	mac.Write([]byte(body))
	assinatura := hex.EncodeToString(mac.Sum(nil))

	// O middleware captura o segredo na construção — depois do t.Setenv.
	rec := executar(t, "/v1/webhooks/budget-response", body, assinatura)
	if rec.Code != http.StatusOK {
		t.Errorf("esperava 200 com assinatura feita com o segredo do ambiente, obteve %d", rec.Code)
	}
}

// corpoComErro simula um corpo de requisição cujo Read falha no meio.
type corpoComErro struct{}

func (corpoComErro) Read(_ []byte) (int, error) { return 0, errors.New("conexão interrompida") }

func TestAssinaturaWebhook_ErroAoLerCorpoRetorna401(t *testing.T) {
	handler := AssinaturaWebhook()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/budget-response", corpoComErro{})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 quando o corpo não pode ser lido, obteve %d", rec.Code)
	}
}
