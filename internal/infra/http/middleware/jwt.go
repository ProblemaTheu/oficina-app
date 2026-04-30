// Package middleware contém middlewares HTTP reutilizáveis.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ClaimsContextKey é a chave usada para armazenar os claims JWT no contexto da requisição.
// Use middleware.ClaimsContextKey para recuperar os claims em handlers downstream.
type ctxKey string

const ClaimsContextKey ctxKey = "jwt_claims"

// publicRoute define um endpoint isento de autenticação JWT.
type publicRoute struct {
	method  string
	pattern *regexp.Regexp
}

// publicRoutes lista as rotas que NÃO exigem Bearer token.
var publicRoutes = []publicRoute{
	{"POST", regexp.MustCompile(`^/v1/auth/login$`)},
	{"POST", regexp.MustCompile(`^/v1/auth/register$`)},
	// Consulta pública de status de OS (cliente final, sem conta)
	{"GET", regexp.MustCompile(`^/v1/work-orders/[^/]+/status$`)},
}

func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("changeme-insecure-default-secret")
}

// JWT retorna um middleware chi que valida o Bearer token JWT em todas as
// rotas, exceto aquelas listadas em publicRoutes.
//
// Em caso de sucesso, os claims são armazenados no contexto via ClaimsContextKey.
// Em caso de falha, responde imediatamente com 401 JSON e interrompe a cadeia.
func JWT() func(http.Handler) http.Handler {
	secret := jwtSecret()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verificar se a rota é pública
			for _, route := range publicRoutes {
				if r.Method == route.method && route.pattern.MatchString(r.URL.Path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extrair Bearer token
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeUnauthorized(w, "token ausente ou mal formatado")
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Parsear e validar o token
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			}, jwt.WithValidMethods([]string{"HS256"}))

			if err != nil || !token.Valid {
				writeUnauthorized(w, "token inválido ou expirado")
				return
			}

			// Propagar claims no contexto para uso nos handlers
			ctx := context.WithValue(r.Context(), ClaimsContextKey, token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"code":    "nao_autorizado",
		"message": msg,
	})
}
