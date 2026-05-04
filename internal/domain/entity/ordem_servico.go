package entity

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusRecebida            Status = "recebida"
	StatusEmDiagnostico       Status = "em_diagnostico"
	StatusAguardandoAprovacao Status = "aguardando_aprovacao"
	StatusEmExecucao          Status = "em_execucao"
	StatusFinalizada          Status = "finalizada"
	StatusEntregue            Status = "entregue"
	StatusCancelada           Status = "cancelada"
)

var validTransitions = map[Status][]Status{
	StatusRecebida:            {StatusEmDiagnostico, StatusCancelada},
	StatusEmDiagnostico:       {StatusAguardandoAprovacao, StatusCancelada},
	StatusAguardandoAprovacao: {StatusEmExecucao, StatusFinalizada, StatusCancelada},
	StatusEmExecucao:          {StatusFinalizada, StatusCancelada},
	StatusFinalizada:          {StatusEntregue},
}

func (s Status) CanTransitionTo(next Status) bool {
	allowed := validTransitions[s]
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}

type OrdemServico struct {
	ID                   uuid.UUID
	Numero               string
	ClienteID            uuid.UUID
	VeiculoID            uuid.UUID
	UsuarioResponsavelID *uuid.UUID
	StatusID             uuid.UUID
	StatusNome           Status
	Descricao            *string
	Diagnostico          *string
	ValorTotal           float64
	AprovadoEm           *time.Time
	ReprovadoEm          *time.Time
	IniciadoEm           *time.Time
	FinalizadoEm         *time.Time
	EntregueEm           *time.Time
	CanceladoEm          *time.Time
	CriadoEm             time.Time
	AtualizadoEm         time.Time
}

type ItemOS struct {
	ID            uuid.UUID
	OsID          uuid.UUID
	Tipo          string // "servico" ou "peca"
	ReferenciaID  uuid.UUID
	Nome          string
	Quantidade    int
	PrecoUnitario float64
	Subtotal      float64
}

type OrdemServicoCompleta struct {
	OrdemServico
	Itens   []ItemOS
	Cliente ClienteResumo
	Veiculo VeiculoResumo
}

type ClienteResumo struct {
	ID      uuid.UUID
	Nome    string
	CpfCnpj string
}

type VeiculoResumo struct {
	ID     uuid.UUID
	Placa  string
	Marca  string
	Modelo string
	Ano    int
}
