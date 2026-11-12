package usecase

import (
	"context"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	"github.com/google/uuid"
)

// ListarOSParams contém os filtros para listagem de OSs.
type ListarOSParams struct {
	Status    *entity.Status
	ClienteID *uuid.UUID
	VeiculoID *uuid.UUID
	// IncluirEncerradas inclui OSs finalizadas/entregues na listagem padrão
	// (quando Status é informado, o filtro explícito prevalece).
	IncluirEncerradas bool
	Page              int
	Limit             int
}

// RelatorioTempoMedioParams são os filtros do relatório.
type RelatorioTempoMedioParams struct {
	ServicoID  *uuid.UUID
	DataInicio *time.Time
	DataFim    *time.Time
}

// ItemTempoMedio é o resultado do relatório por serviço.
type ItemTempoMedio struct {
	ServicoID         uuid.UUID
	ServicoNome       string
	TotalExecucoes    int
	TempoMedioMinutos float64
}

type clienteRepo interface {
	Salvar(cliente *entity.Cliente) (*entity.Cliente, error)
	BuscarTodos() ([]*entity.Cliente, error)
	BuscarPorID(id string) (*entity.Cliente, error)
	BuscarPorDocumento(documento string) (*entity.Cliente, error)
	Atualizar(cliente *entity.Cliente) (*entity.Cliente, error)
	Remover(id string) error
}

type veiculoRepo interface {
	Salvar(veiculo *entity.Veiculo) (*entity.Veiculo, error)
	BuscarTodos() ([]*entity.Veiculo, error)
	BuscarPorID(id string) (*entity.Veiculo, error)
	Atualizar(veiculo *entity.Veiculo) (*entity.Veiculo, error)
	Remover(id string) error
	BuscarPorClienteID(clienteID string) ([]*entity.Veiculo, error)
}

type pecaRepo interface {
	Salvar(peca *entity.Peca) (*entity.Peca, error)
	BuscarTodos() ([]*entity.Peca, error)
	BuscarPorID(id string) (*entity.Peca, error)
	Atualizar(peca *entity.Peca) (*entity.Peca, error)
	AtualizarEstoque(peca *entity.Peca) (*entity.Peca, error)
	Remover(id string) error
}

type servicoRepo interface {
	Salvar(servico *entity.Servico) (*entity.Servico, error)
	BuscarTodos() ([]*entity.Servico, error)
	BuscarPorID(id string) (*entity.Servico, error)
	Atualizar(servico *entity.Servico) (*entity.Servico, error)
	Remover(id string) error
}

type usuarioRepo interface {
	BuscarPapelID(ctx context.Context, nomePapel string) (uuid.UUID, error)
	Salvar(ctx context.Context, usuario *entity.Usuario) (*entity.Usuario, error)
	BuscarPorEmail(ctx context.Context, email string) (*entity.Usuario, error)
	BuscarNomePapel(ctx context.Context, papelID uuid.UUID) (string, error)
}

// NotificacaoStatus contém os dados para notificar o cliente sobre a mudança
// de status de sua Ordem de Serviço.
type NotificacaoStatus struct {
	DestinatarioEmail string
	DestinatarioNome  string
	NumeroOS          string
	NovoStatus        entity.Status
	Motivo            *string
}

// notifier é a porta de saída para notificar o cliente (e-mail, SMS, etc.).
// Implementações vivem na infra (ex.: SMTP, log para ambiente local).
type notifier interface {
	NotificarMudancaStatus(ctx context.Context, n NotificacaoStatus) error
}

// Resultados possíveis de um EventoOS.
const (
	ResultadoSucesso = "sucesso"
	ResultadoFalha   = "falha"
)

// EventoOS descreve um evento de negócio de Ordem de Serviço emitido para a
// plataforma de observabilidade. Alimenta os dashboards (volume diário de OS,
// tempo médio por status) e o alerta de falha no processamento de OS. Nenhum
// dado sensível (CPF, e-mail) trafega neste evento — apenas identificadores e
// metadados de status.
type EventoOS struct {
	OsID                  string  // UUID da OS (vazio quando a criação falha antes de persistir)
	Numero                string  // número legível (OS-YYYY-NNNNN)
	Status                string  // status resultante da operação
	StatusAnterior        string  // status de origem (vazio na criação)
	Resultado             string  // ResultadoSucesso | ResultadoFalha
	Motivo                string  // rótulo curto do erro quando Resultado == ResultadoFalha
	DuracaoStatusSegundos float64 // tempo no status anterior; 0 quando não aplicável
}

// eventRecorder é a porta de saída para eventos de negócio (observabilidade).
// Implementações vivem na infra (ex.: custom events do New Relic). É opcional:
// quando ausente, os casos de uso operam normalmente, apenas sem emitir eventos.
type eventRecorder interface {
	RegistrarEventoOS(ctx context.Context, e EventoOS)
}

// EventoSaga descreve uma transição da saga da OS (F4-7.2): um por etapa
// registrada em sagas_os_etapas. Alimenta o dashboard da saga (desfechos, taxa
// de compensação, onde falham, tempo por etapa) e o alerta de compensação.
// Mesma regra de LGPD do EventoOS: sem CPF, e-mail ou valor pago.
type EventoSaga struct {
	SagaID         string
	OsID           string
	EstadoAnterior string  // estado da saga antes da resposta ("" na abertura)
	Estado         string  // estado da saga depois da resposta
	Etapa          string  // T1–T7
	Resultado      string  // ok | falha | compensada | ignorada
	Motivo         string  // falha: motivo; ignorada: tipo da resposta inesperada
	TipoMensagem   string  // resposta que causou a transição ("" na abertura)
	DuracaoEtapa   float64 // segundos desde a transição anterior da saga
	DuracaoTotal   float64 // segundos desde a abertura — só quando Terminou
	// Terminou marca o ÚNICO evento da transição que levou a saga a
	// CONCLUIDA ou COMPENSADA. Contar desfechos por `estado` inflaria: uma
	// transição terminal registra várias etapas, e respostas atrasadas chegam
	// com a saga já terminal (ignoradas).
	Terminou bool
}

// sagaEventRecorder é a extensão opcional do eventRecorder para a saga. É uma
// interface à parte para que emissores só de EventoOS continuem válidos: o
// caso de uso descobre a capacidade por type assertion.
type sagaEventRecorder interface {
	RegistrarEventoSaga(ctx context.Context, e EventoSaga)
}

type osRepo interface {
	BuscarStatusID(ctx context.Context, nome entity.Status) (uuid.UUID, error)
	GerarNumeroOS(ctx context.Context) (string, error)
	Criar(ctx context.Context, os *entity.OrdemServico, itensServico []entity.ItemOsServico, itensPeca []entity.ItemOsPeca) (*entity.OrdemServico, error)
	Listar(ctx context.Context, params ListarOSParams) ([]*entity.OrdemServico, int, error)
	BuscarPorID(ctx context.Context, id string) (*entity.OrdemServicoCompleta, error)
	AtualizarStatus(ctx context.Context, osID uuid.UUID, novoStatusID uuid.UUID, diagnostico *string, aprovadoEm *time.Time, reprovadoEm *time.Time, iniciadoEm *time.Time, finalizadoEm *time.Time, entregueEm *time.Time) error
	RegistrarHistorico(ctx context.Context, osID uuid.UUID, statusAnteriorID *uuid.UUID, statusNovoID uuid.UUID, observacao *string) error
	DeduzirEstoquePecas(ctx context.Context, osID uuid.UUID) error
	RelatorioTempoMedio(ctx context.Context, params RelatorioTempoMedioParams) ([]ItemTempoMedio, error)
	PrecarregarStatusCache(ctx context.Context) error
}

// ── Saga (Fase 4) ─────────────────────────────────────────────────────────────

// unidadeDeTrabalho executa fn numa transação: tudo o que os repositórios
// fizerem com o contexto recebido é confirmado ou desfeito junto (F4-1.3).
type unidadeDeTrabalho interface {
	Executar(ctx context.Context, fn func(ctx context.Context) error) error
}

// estoqueSaga reserva as peças da OS sem retirá-las do estoque (T1), libera a
// reserva (compensação de T1) e a converte em saída (pivô T6).
type estoqueSaga interface {
	Reservar(ctx context.Context, osID uuid.UUID) error
	Liberar(ctx context.Context, osID uuid.UUID) error
	Baixar(ctx context.Context, osID uuid.UUID) error
}

// sagaRepo persiste o estado e o histórico de etapas da saga.
type sagaRepo interface {
	Criar(ctx context.Context, s saga.Saga) error
	RegistrarEtapa(ctx context.Context, sagaID uuid.UUID, etapa saga.Etapa, resultado saga.Resultado, messageID *uuid.UUID, detalhe any) error
	BuscarPorOS(ctx context.Context, osID uuid.UUID) (*saga.Saga, error)
	// Atualizar grava o novo estado com lock otimista (falha se a versão mudou).
	Atualizar(ctx context.Context, s saga.Saga, versaoAnterior int) error
}

// inboxSaga torna o consumo das respostas idempotente: false = já processada.
type inboxSaga interface {
	Marcar(ctx context.Context, messageID uuid.UUID, tipo string) (bool, error)
}

// cancelamentoOS grava o cancelamento da OS pela compensação da saga.
type cancelamentoOS interface {
	CancelarPelaSaga(ctx context.Context, osID, statusCanceladaID uuid.UUID, motivo string) error
}

// filaComandos grava um comando da saga na outbox, na transação do contexto.
type filaComandos interface {
	Enfileirar(ctx context.Context, c saga.Comando) error
}
