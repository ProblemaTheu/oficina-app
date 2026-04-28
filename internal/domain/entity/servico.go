package entity

import (
	"time"

	"github.com/google/uuid"
)

type Servico struct {
	ID           uuid.UUID
	Nome         string
	Descricao    *string
	PrecoBase    float64
	TempoMinutos int
	CriadoEm     time.Time
	AtualizadoEm time.Time
}
