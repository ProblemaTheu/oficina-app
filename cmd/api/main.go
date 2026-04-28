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
