package entity

import (
	"time"

	"github.com/google/uuid"
)

type PapelUsuario struct {
	ID           uuid.UUID
	NomePapel    string
	Descricao    *string
	CriadoEm    time.Time
	AtualizadoEm time.Time
}
