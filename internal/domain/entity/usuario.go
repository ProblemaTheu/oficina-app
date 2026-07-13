package entity

import (
	"time"

	"github.com/google/uuid"
)

type Usuario struct {
	ID           uuid.UUID
	Nome         string
	Email        string
	SenhaHash    string
	PapelID      uuid.UUID
	CriadoEm     time.Time
	AtualizadoEm time.Time
}
