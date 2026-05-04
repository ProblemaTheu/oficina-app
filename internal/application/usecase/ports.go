package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

type clienteRepo interface {
	Salvar(cliente *entity.Cliente) (*entity.Cliente, error)
	BuscarTodos() ([]*entity.Cliente, error)
	BuscarPorID(id string) (*entity.Cliente, error)
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

type osRepo interface {
	BuscarStatusID(ctx context.Context, nome entity.Status) (uuid.UUID, error)
	GerarNumeroOS(ctx context.Context) (string, error)
	Criar(ctx context.Context, os *entity.OrdemServico, itensServico []entity.ItemOsServico, itensPeca []entity.ItemOsPeca) (*entity.OrdemServico, error)
	Listar(ctx context.Context, params repository.ListarOSParams) ([]*entity.OrdemServico, int, error)
	BuscarPorID(ctx context.Context, id string) (*entity.OrdemServicoCompleta, error)
	AtualizarStatus(ctx context.Context, osID uuid.UUID, novoStatusID uuid.UUID, diagnostico *string, aprovadoEm *time.Time, reprovadoEm *time.Time, iniciadoEm *time.Time, finalizadoEm *time.Time, entregueEm *time.Time) error
	RegistrarHistorico(ctx context.Context, osID uuid.UUID, statusAnteriorID *uuid.UUID, statusNovoID uuid.UUID, observacao *string) error
	DeduzirEstoquePecas(ctx context.Context, osID uuid.UUID) error
	RelatorioTempoMedio(ctx context.Context, params repository.RelatorioTempoMedioParams) ([]repository.ItemTempoMedio, error)
	PrecarregarStatusCache(ctx context.Context) error
}
