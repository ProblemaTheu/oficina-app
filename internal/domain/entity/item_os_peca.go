package entity

import (
	"time"

	"github.com/google/uuid"
)

type ItemOsPeca struct {
	ID            uuid.UUID
	OsID          uuid.UUID
	PecaID        uuid.UUID
	Quantidade    int
	PrecoUnitario float64
	CriadoEm      time.Time
	AtualizadoEm  time.Time
}
