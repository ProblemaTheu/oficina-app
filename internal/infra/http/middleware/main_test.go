package middleware

import (
	"os"
	"strings"
	"testing"
)

// TestMain garante os segredos que a aplicação passou a exigir na
// inicialização. Sem eles config.JWTSecret e config.WebhookSecret entram em
// pânico — que é justamente o comportamento desejado em produção.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	os.Setenv("WEBHOOK_SECRET", strings.Repeat("y", 32))
	os.Exit(m.Run())
}
