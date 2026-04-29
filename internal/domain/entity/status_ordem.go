package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusRecebida            = "recebida"
	StatusEmDiagnostico       = "em_diagnostico"
	StatusAguardandoAprovacao = "aguardando_aprovacao"
	StatusEmExecucao          = "em_execucao"
	StatusFinalizada          = "finalizada"
	StatusEntregue            = "entregue"
)

type StatusOrdem struct {
	ID           uuid.UUID
	NomeStatus   string
	Descricao    *string
	CriadoEm    time.Time
	AtualizadoEm time.Time
}
