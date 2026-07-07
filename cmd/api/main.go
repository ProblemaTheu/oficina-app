package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/problematheu/tech-challenge-1/internal/infra/database"
	"github.com/problematheu/tech-challenge-1/internal/infra/http/api"
	apimiddleware "github.com/problematheu/tech-challenge-1/internal/infra/http/middleware"
)

func main() {
	db := database.Connect()
	defer db.Close() //nolint:errcheck

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("falha ao executar migrations: %v", err)
	}
	log.Println("migrations executadas com sucesso")

	server := api.NovoServer(db)

	ctx := context.Background()
	if err := server.InicializarCaches(ctx); err != nil {
		log.Printf("aviso: falha ao pré-carregar caches: %v", err)
	}

	strictHandler := api.NewStrictHandler(server, nil)

	r := chi.NewRouter()

	// ── Rotas de health (sem autenticação) ────────────────────────────────────
	// Liveness: a aplicação está de pé (não verifica dependências externas)
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "UP"}); err != nil {
			log.Printf("health: falha ao escrever resposta: %v", err)
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
	//   - demais rotas         → authenticated (Bearer JWT HS256)
	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.JWT())
		api.HandlerFromMuxWithBaseURL(strictHandler, r, "/v1")
	})

	log.Println("servidor rodando na porta 8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("falha ao iniciar servidor: %v", err)
	}
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
			log.Printf("health: db indisponível: %v", pingErr)
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
