package entity

import (
	"time"

	"github.com/google/uuid"
)

type OrdemServico struct {
	ID                    uuid.UUID
	ClienteID             uuid.UUID
	VeiculoID             uuid.UUID
	UsuarioResponsavelID  *uuid.UUID
	StatusID              uuid.UUID
	ValorTotal            float64
	CriadoEm             time.Time
	AtualizadoEm          time.Time
}
