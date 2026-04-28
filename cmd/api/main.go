package main

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/problematheu/tech-challenge-1/internal/infra/http/handler"
	"github.com/problematheu/tech-challenge-1/internal/infra/database"
)

func main() {
	db := database.Connect()
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("falha ao executar migrations: %v", err)
	}
	log.Println("migrations executadas com sucesso")

	r := chi.NewRouter()

	clientHandler := handler.NewClientHandler()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	r.Post("/clients", clientHandler.CreateClient)

	log.Println("servidor rodando na porta 8080")
	http.ListenAndServe(":8080", r)
}
