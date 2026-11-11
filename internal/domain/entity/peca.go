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
	// EstoqueReservado: unidades seguras por OS em andamento (saga da Fase 4).
	// Disponível = EstoqueAtual - EstoqueReservado.
	EstoqueReservado int
	CriadoEm         time.Time
	AtualizadoEm     time.Time
}
