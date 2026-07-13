package entity

import (
	"time"

	"github.com/google/uuid"
)

type Veiculo struct {
	ID           uuid.UUID
	ClienteID    uuid.UUID
	Placa        string
	Marca        string
	Modelo       string
	Ano          int
	Cor          *string
	CriadoEm     time.Time
	AtualizadoEm time.Time
}
