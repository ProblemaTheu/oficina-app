package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// APM cria uma transação de request para o New Relic e registra o tempo de
// resposta e a rota. Caso a licença não esteja configurada, o middleware
// devolve a cadeia sem impacto.
func APM(app *newrelic.Application) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if app == nil {
				next.ServeHTTP(w, r)
				return
			}

			nome := r.Method + " " + r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
				nome = r.Method + " " + rc.RoutePattern()
			}

			txn := app.StartTransaction(nome)
			defer txn.End()
			txn.SetWebRequestHTTP(r)
			w = txn.SetWebResponse(w)
			next.ServeHTTP(w, newrelic.RequestWithTransactionContext(r, txn))
		})
	}
}
