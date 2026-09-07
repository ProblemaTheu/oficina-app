package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ProblemaTheu/oficina-app/internal/application/logging"
	"github.com/google/uuid"
)

// HeaderCorrelacao é o header usado para propagar o identificador de
// correlação entre cliente/API Gateway e a aplicação.
const HeaderCorrelacao = "X-Correlation-Id"

// Correlacao garante que toda requisição tenha um identificador único,
// reaproveitando o que veio do cliente/gateway quando existir, e coloca um
// *slog.Logger já decorado com ele no contexto — lido via logging.Log(ctx)
// pelos handlers e casos de uso downstream.
func Correlacao() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(HeaderCorrelacao)
			if id == "" {
				id = uuid.NewString()
			}
			// Devolver o header permite ao cliente citar o ID ao abrir chamado.
			w.Header().Set(HeaderCorrelacao, id)

			logger := slog.Default().With(
				"correlation_id", id,
				"http.method", r.Method,
				"http.route", r.URL.Path,
			)
			ctx := logging.ComLogger(r.Context(), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
