package entity

import (
	"time"

	"github.com/google/uuid"
)

type StatusOrdem struct {
	ID           uuid.UUID
	NomeStatus   string
	Descricao    *string
	CriadoEm     time.Time
	AtualizadoEm time.Time
}
