package api

import (
	"os"
	"strings"
	"testing"
)

// A aplicação passou a exigir os segredos na inicialização — sem eles,
// config.JWTSecret entra em pânico de propósito. Ver
// internal/infra/config/segredos.go.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	os.Setenv("WEBHOOK_SECRET", strings.Repeat("y", 32))
	os.Exit(m.Run())
}
