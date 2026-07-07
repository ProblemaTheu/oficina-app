package entity

import (
	"time"

	"github.com/google/uuid"
)

type Cliente struct {
	ID           uuid.UUID
	Nome         string
	CpfCnpj      string
	Email        *string
	Telefone     *string
	CriadoEm     time.Time
	AtualizadoEm time.Time
}
