package usecase

import (
	"os"
	"strings"
	"testing"
)

// Ver comentário em internal/infra/http/middleware/main_test.go.
func TestMain(m *testing.M) {
	_ = os.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	os.Exit(m.Run())
}
