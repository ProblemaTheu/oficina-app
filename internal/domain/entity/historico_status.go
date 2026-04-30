package entity

import (
	"time"

	"github.com/google/uuid"
)

type HistoricoStatus struct {
	ID                   uuid.UUID
	OsID                 uuid.UUID
	StatusAnteriorID     *uuid.UUID
	StatusNovoID         uuid.UUID
	AlteradoEm           time.Time
	AlteradoPorUsuarioID *uuid.UUID
	Observacao           *string
	CriadoEm             time.Time
	AtualizadoEm         time.Time
}
