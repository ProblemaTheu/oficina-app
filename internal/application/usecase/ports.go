package usecase

import (
	"context"
	"time"

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
