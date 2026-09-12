package middleware

import (
	"log/slog"
	"net/http"
	"time"

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
			inicio := time.Now()
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
			response := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(response, r.WithContext(ctx))
			logging.Log(ctx).Info("http request completed",
				"http.status_code", response.statusCode,
				"http.duration_ms", time.Since(inicio).Milliseconds(),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}
