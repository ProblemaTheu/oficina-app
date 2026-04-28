package entity

import (
	"time"

	"github.com/google/uuid"
)

type Peca struct {
	ID            uuid.UUID
	Nome          string
	Codigo        string
	Preco         float64
	EstoqueAtual  int
	EstoqueMinimo int
	CriadoEm      time.Time
	AtualizadoEm  time.Time
}
