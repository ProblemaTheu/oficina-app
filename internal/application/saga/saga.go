// Package saga modela a saga orquestrada da Ordem de Serviço (Fase 4): estados,
// etapas, tipos de mensagem e os comandos enviados aos outros serviços.
//
// É lógica pura, sem infraestrutura. O desenho completo — máquina de estados,
// etapas T1–T7 e compensações — está em
// docs/planejamentos/fase-4/arquitetura.md §3.
package saga

import (
	"time"

	"github.com/google/uuid"
)

// Estado é o estado da saga (persistido em sagas_os.estado).
type Estado string

const (
	DiagnosticoPendente Estado = "DIAGNOSTICO_PENDENTE"
	OrcamentoPendente   Estado = "ORCAMENTO_PENDENTE"
	AguardandoAprovacao Estado = "AGUARDANDO_APROVACAO"
	AguardandoPagamento Estado = "AGUARDANDO_PAGAMENTO"
	ExecucaoPendente    Estado = "EXECUCAO_PENDENTE"
	EmExecucao          Estado = "EM_EXECUCAO"
	Concluida           Estado = "CONCLUIDA"
	Compensando         Estado = "COMPENSANDO"
	Compensada          Estado = "COMPENSADA"
)

// Terminal informa se a saga já terminou (com sucesso ou compensada).
func (e Estado) Terminal() bool { return e == Concluida || e == Compensada }

// Etapa identifica um passo da saga (arquitetura §3.3).
type Etapa string

const (
	EtapaAbertura    Etapa = "T1" // OS aberta + peças reservadas (local)
	EtapaDiagnostico Etapa = "T2" // Execution
	EtapaOrcamento   Etapa = "T3" // Billing
	EtapaAprovacao   Etapa = "T4" // Billing (humano)
	EtapaPagamento   Etapa = "T5" // Billing (Mercado Pago)
	EtapaReparo      Etapa = "T6" // Execution — pivô: depois daqui não se compensa
	EtapaConclusao   Etapa = "T7"
)

// Resultado de uma etapa registrada em sagas_os_etapas.
type Resultado string

const (
	ResultadoOK         Resultado = "ok"
	ResultadoFalha      Resultado = "falha"
	ResultadoCompensada Resultado = "compensada"
	ResultadoIgnorada   Resultado = "ignorada"
)

// Destinos lógicos dos comandos — resolvidos em URL de fila pelo relay.
const (
	DestinoExecution = "execution"
	DestinoBilling   = "billing"
)

// Tipos de mensagem do catálogo (arquitetura §5.2).
const (
	TipoIniciarDiagnostico   = "IniciarDiagnostico"
	TipoDiagnosticoIniciado  = "DiagnosticoIniciado"
	TipoDiagnosticoConcluido = "DiagnosticoConcluido"
	TipoDiagnosticoFalhou    = "DiagnosticoFalhou"
	TipoGerarOrcamento       = "GerarOrcamento"
	TipoOrcamentoGerado      = "OrcamentoGerado"
	TipoOrcamentoFalhou      = "OrcamentoFalhou"
	TipoOrcamentoAprovado    = "OrcamentoAprovado"
	TipoOrcamentoRecusado    = "OrcamentoRecusado"
	TipoPagamentoConfirmado  = "PagamentoConfirmado"
	TipoPagamentoRecusado    = "PagamentoRecusado"
	TipoExecutarReparo       = "ExecutarReparo"
	TipoExecucaoIniciada     = "ExecucaoIniciada"
	TipoExecucaoFalhou       = "ExecucaoFalhou"
	TipoExecucaoConcluida    = "ExecucaoConcluida"
	TipoEstornarPagamento    = "EstornarPagamento"
	TipoPagamentoEstornado   = "PagamentoEstornado"
	TipoCancelarOrcamento    = "CancelarOrcamento"
	TipoOrcamentoCancelado   = "OrcamentoCancelado"
	TipoCancelarTarefas      = "CancelarTarefas"
	TipoTarefasCanceladas    = "TarefasCanceladas"
)

// Saga é o estado da saga de uma OS (tabela sagas_os).
type Saga struct {
	ID           uuid.UUID
	OsID         uuid.UUID
	Estado       Estado
	EtapaAtual   Etapa
	EtapaFalha   *Etapa
	OrcamentoID  *uuid.UUID
	PagamentoID  *uuid.UUID
	PrazoEm      *time.Time
	Versao       int // lock otimista entre réplicas do orquestrador
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

// Nova cria a saga de uma OS recém-aberta. Ela nasce em DIAGNOSTICO_PENDENTE —
// não há estado INICIADA: a abertura (T1) é local e acontece na mesma
// transação que grava a OS, a reserva e o comando IniciarDiagnostico.
func Nova(osID uuid.UUID, agora time.Time) Saga {
	return Saga{
		ID:           uuid.New(),
		OsID:         osID,
		Estado:       DiagnosticoPendente,
		EtapaAtual:   EtapaDiagnostico,
		Versao:       1,
		CriadoEm:     agora.UTC(),
		AtualizadoEm: agora.UTC(),
	}
}

// Comando é uma mensagem que o orquestrador envia a outro serviço. O adaptador
// de infraestrutura transforma em envelope e grava na outbox.
type Comando struct {
	Destino string
	Tipo    string
	SagaID  uuid.UUID
	OsID    uuid.UUID
	Payload any
}

// Veiculo é o recorte do veículo que viaja nos comandos ao Execution.
type Veiculo struct {
	Placa  string `json:"placa"`
	Modelo string `json:"modelo"`
}

// PayloadIniciarDiagnostico é o payload do comando IniciarDiagnostico.
// Autossuficiente: o Execution não consulta o OS para saber a placa.
type PayloadIniciarDiagnostico struct {
	NumeroOS  string  `json:"numero_os"`
	Veiculo   Veiculo `json:"veiculo"`
	Descricao string  `json:"descricao,omitempty"`
}

// ComandoIniciarDiagnostico monta o primeiro comando da saga.
func (s Saga) ComandoIniciarDiagnostico(p PayloadIniciarDiagnostico) Comando {
	return Comando{Destino: DestinoExecution, Tipo: TipoIniciarDiagnostico, SagaID: s.ID, OsID: s.OsID, Payload: p}
}
