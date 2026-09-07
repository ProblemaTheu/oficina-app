package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/infra/database"
	"github.com/ProblemaTheu/oficina-app/internal/infra/http/api"
	apimiddleware "github.com/ProblemaTheu/oficina-app/internal/infra/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	nrApp := novoAppNewRelic()
	configurarLog(nrApp)

	db := database.Connect()
	defer db.Close() //nolint:errcheck

	if err := database.RunMigrations(db); err != nil {
		slog.Error("falha ao executar migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations executadas com sucesso")

	server := api.NovoServer(db)

	ctx := context.Background()
	if err := server.InicializarCaches(ctx); err != nil {
		slog.Warn("falha ao pré-carregar caches", "error", err)
	}

	strictHandler := api.NewStrictHandlerWithOptions(server, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.TratarErroRequisicao,
		ResponseErrorHandlerFunc: api.TratarErroResposta,
	})

	r := chi.NewRouter()
	// 1º de todos: garante que todo log da requisição (aqui e nos use cases)
	// já sai correlacionado, inclusive nas rotas de health.
	r.Use(apimiddleware.Correlacao())
	r.Use(apimiddleware.APM(nrApp))

	// ── Rotas de health (sem autenticação) ────────────────────────────────────
	// Liveness: a aplicação está de pé (não verifica dependências externas)
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "UP"}); err != nil {
			slog.Error("health: falha ao escrever resposta", "error", err)
		}
	})

	// Readiness: a aplicação está pronta para receber tráfego (verifica dependências)
	r.Get("/health/ready", healthReadyHandler(db))

	// Alias raiz — retorna o mesmo que /health/ready (compatível com probes simples)
	r.Get("/health", healthReadyHandler(db))

	// ── Rotas da API /v1 (protegidas por JWT, exceto rotas públicas) ──────────
	// Equivalente ao SecurityConfig do Spring Security:
	//   - /v1/auth/login       → permitAll
	//   - /v1/auth/register    → permitAll
	//   - /v1/work-orders/{id}/status → permitAll
	//   - /v1/webhooks/*       → assinatura HMAC (X-Signature), sem JWT
	//   - demais rotas         → authenticated (Bearer JWT HS256)
	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.AssinaturaWebhook())
		r.Use(apimiddleware.JWT())
		api.HandlerFromMuxWithBaseURL(strictHandler, r, "/v1")
	})

	slog.Info("servidor rodando na porta 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		slog.Error("falha ao iniciar servidor", "error", err)
		os.Exit(1)
	}
}

// novoAppNewRelic inicializa o APM do New Relic quando a licença está presente.
// Em desenvolvimento local sem a chave, a aplicação continua funcionando e
// apenas ignora o agente.
func novoAppNewRelic() *newrelic.Application {
	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if licenseKey == "" {
		slog.Info("new relic desabilitado: NEW_RELIC_LICENSE_KEY ausente")
		return nil
	}

	appName := os.Getenv("NEW_RELIC_APP_NAME")
	if appName == "" {
		appName = "oficina-api"
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(appName),
		newrelic.ConfigLicense(licenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigEnabled(true),
	)
	if err != nil {
		slog.Warn("new relic: falha ao inicializar", "error", err)
		return nil
	}

	slog.Info("new relic inicializado com sucesso", "app_name", appName)
	return app
}

// configurarLog troca o logger default do slog por um handler JSON — nomes
// de campo que o New Relic reconhece sem configuração extra (F3-4.1) — e
// deve ser a primeira coisa que roda no main, antes de qualquer outro log.
func configurarLog(nrApp *newrelic.Application) {
	nivel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		nivel = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: nivel,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	})
	var handler slog.Handler = h
	if nrApp != nil {
		handler = nrslog.WrapHandler(nrApp, h)
	}
	slog.SetDefault(slog.New(handler).With(
		"service.name", os.Getenv("NEW_RELIC_APP_NAME"),
		"hostname", os.Getenv("HOSTNAME"), // K8s injeta o nome do pod
	))
}

type componentHealth struct {
	Status string `json:"status"`
}

type healthResponse struct {
	Status     string                     `json:"status"`
	Components map[string]componentHealth `json:"components"`
}

// healthReadyHandler verifica a conectividade com o banco de dados e retorna
// HTTP 200 com status "UP" quando saudável, ou HTTP 503 com status "DOWN"
// quando alguma dependência estiver indisponível.
func healthReadyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		dbStatus := "UP"

		if pingErr := pingDB(db); pingErr != nil {
			dbStatus = "DOWN"
			slog.Warn("health: db indisponível", "error", pingErr)
		}

		overall := "UP"
		statusCode := http.StatusOK
		if dbStatus == "DOWN" {
			overall = "DOWN"
			statusCode = http.StatusServiceUnavailable
		}

		resp := healthResponse{
			Status: overall,
			Components: map[string]componentHealth{
				"db": {Status: dbStatus},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("health: falha ao escrever resposta: %v", err)
		}
	}
}

// pingDB executa um ping com timeout curto para não bloquear o health check.
func pingDB(db *sql.DB) error {
	done := make(chan error, 1)
	go func() {
		done <- db.Ping()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		return http.ErrHandlerTimeout
	}
}
