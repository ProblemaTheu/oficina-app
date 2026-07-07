package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strings"
)

// webhookSecret retorna o segredo compartilhado usado para assinar os corpos
// dos webhooks. Configurável via WEBHOOK_SECRET.
func webhookSecret() []byte {
	if s := os.Getenv("WEBHOOK_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("changeme-insecure-webhook-secret")
}

// AssinaturaWebhook retorna um middleware que valida a assinatura HMAC-SHA256
// do corpo das requisições destinadas a /v1/webhooks/*. O provedor externo
// envia a assinatura (hex) no header X-Signature, calculada com o segredo
// compartilhado. Requisições fora do prefixo de webhooks passam direto.
//
// A comparação usa hmac.Equal (tempo constante). Após a leitura, o corpo é
// restaurado para os handlers downstream.
func AssinaturaWebhook() func(http.Handler) http.Handler {
	secret := webhookSecret()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/v1/webhooks/") {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeUnauthorized(w, "não foi possível ler o corpo da requisição")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			mac := hmac.New(sha256.New, secret)
			mac.Write(body)
			esperada := hex.EncodeToString(mac.Sum(nil))

			recebida := r.Header.Get("X-Signature")
			if recebida == "" || !hmac.Equal([]byte(esperada), []byte(strings.ToLower(recebida))) {
				writeUnauthorized(w, "assinatura do webhook ausente ou inválida")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
