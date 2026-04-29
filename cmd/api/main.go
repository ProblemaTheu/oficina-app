package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
	
	"github.com/go-chi/chi/v5"
	"github.com/problematheu/tech-challenge-1/internal/infra/database"
	"github.com/problematheu/tech-challenge-1/internal/infra/http/api"
)

func main() {
	db := database.Connect()
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("falha ao executar migrations: %v", err)
	}
	log.Println("migrations executadas com sucesso")

	server := api.NovoServer(db)
	strictHandler := api.NewStrictHandler(server, nil)

	r := chi.NewRouter()

	// Liveness: a aplicação está de pé (não verifica dependências externas)
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "UP"})
	})

	// Readiness: a aplicação está pronta para receber tráfego (verifica dependências)
	r.Get("/health/ready", healthReadyHandler(db))

	// Alias raiz — retorna o mesmo que /health/ready (compatível com probes simples)
	r.Get("/health", healthReadyHandler(db))

	// Todos os endpoints do OpenAPI montados sob /v1
	api.HandlerFromMuxWithBaseURL(strictHandler, r, "/v1")

	log.Println("servidor rodando na porta 8080")
	http.ListenAndServe(":8080", r)
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
		json.NewEncoder(w).Encode(resp)
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
