package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ProblemaTheu/oficina-app/internal/infra/config"
)

func tokenValido(t *testing.T) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "usuario-teste",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	assinado, err := token.SignedString(config.JWTSecret())
	if err != nil {
		t.Fatalf("falha ao assinar token de teste: %v", err)
	}
	return assinado
}

func requisitar(method, path, authHeader string) *httptest.ResponseRecorder {
	handler := JWT()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestJWT_TokenValidoPassa(t *testing.T) {
	rec := requisitar(http.MethodGet, "/v1/clients", "Bearer "+tokenValido(t))
	if rec.Code != http.StatusOK {
		t.Errorf("esperava 200 com token válido, obteve %d", rec.Code)
	}
}

func TestJWT_SemTokenRetorna401(t *testing.T) {
	rec := requisitar(http.MethodGet, "/v1/clients", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 sem token, obteve %d", rec.Code)
	}
}

func TestJWT_TokenMalformadoRetorna401(t *testing.T) {
	rec := requisitar(http.MethodGet, "/v1/clients", "Bearer não-é-um-jwt")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 com token malformado, obteve %d", rec.Code)
	}
}

func TestJWT_TokenExpiradoRetorna401(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "usuario-teste",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	assinado, _ := token.SignedString(config.JWTSecret())

	rec := requisitar(http.MethodGet, "/v1/clients", "Bearer "+assinado)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 com token expirado, obteve %d", rec.Code)
	}
}

func TestJWT_AssinaturaDeOutroSegredoRetorna401(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "usuario-teste",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	assinado, _ := token.SignedString([]byte("segredo-errado"))

	rec := requisitar(http.MethodGet, "/v1/clients", "Bearer "+assinado)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 com assinatura de outro segredo, obteve %d", rec.Code)
	}
}

func TestJWT_RotasPublicasNaoExigemToken(t *testing.T) {
	casos := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/auth/login"},
		{http.MethodPost, "/v1/auth/register"},
		{http.MethodGet, "/v1/work-orders/123e4567-e89b-12d3-a456-426614174000/status"},
		{http.MethodPost, "/v1/webhooks/budget-response"},
	}
	for _, c := range casos {
		if rec := requisitar(c.method, c.path, ""); rec.Code != http.StatusOK {
			t.Errorf("%s %s: rota pública deveria passar sem token, obteve %d", c.method, c.path, rec.Code)
		}
	}
}

func TestJWT_MetodoErradoEmRotaPublicaExigeToken(t *testing.T) {
	// GET no webhook (rota pública só para POST) deve exigir token
	rec := requisitar(http.MethodGet, "/v1/webhooks/budget-response", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401 para método fora da rota pública, obteve %d", rec.Code)
	}
}
