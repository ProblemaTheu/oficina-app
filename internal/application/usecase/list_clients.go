package usecase

import (
	"log/slog"

	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

// ListarClientesUseCase encapsula a lógica de negócio para listagem de clientes.
type ListarClientesUseCase struct {
	repo *repository.ClienteRepository
}

// NovoListarClientesUseCase cria uma nova instância de ListarClientesUseCase.
func NovoListarClientesUseCase(repo *repository.ClienteRepository) *ListarClientesUseCase {
	return &ListarClientesUseCase{repo: repo}
}

// Executar retorna todos os clientes cadastrados.
func (uc *ListarClientesUseCase) Executar() ([]*entity.Cliente, error) {
	slog.Info("executando caso de uso: listar clientes")
	return uc.repo.BuscarTodos()
}
