package main

import (
	"log"
	"os"

	"github.com/problematheu/tech-challenge-1/internal/infra/database"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("variável de ambiente DATABASE_URL é obrigatória")
	}

	log.Println("executando migrations...")
	if err := database.RunMigrations(databaseURL); err != nil {
		log.Fatalf("falha ao executar migrations: %v", err)
	}
	log.Println("migrations executadas com sucesso")

	log.Println("Tech Challenge 1 - Oficina API iniciada")
}
