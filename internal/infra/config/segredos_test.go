package config

import (
	"strings"
	"testing"
)

func esperaPanico(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("esperava pânico: a aplicação não pode subir sem segredo")
		}
	}()
	f()
}

func TestJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("j", tamanhoMinimo))
	if got := string(JWTSecret()); got != strings.Repeat("j", tamanhoMinimo) {
		t.Fatalf("segredo = %q", got)
	}

	t.Setenv("JWT_SECRET", strings.Repeat("j", tamanhoMinimo-1))
	esperaPanico(t, func() { JWTSecret() })

	t.Setenv("JWT_SECRET", "")
	esperaPanico(t, func() { JWTSecret() })
}

func TestWebhookSecret(t *testing.T) {
	t.Setenv("WEBHOOK_SECRET", strings.Repeat("w", tamanhoMinimo))
	if got := string(WebhookSecret()); got != strings.Repeat("w", tamanhoMinimo) {
		t.Fatalf("segredo = %q", got)
	}
	t.Setenv("WEBHOOK_SECRET", "curto")
	esperaPanico(t, func() { WebhookSecret() })
}
