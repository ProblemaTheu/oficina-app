package usecase

import (
	"errors"
	"log/slog"

	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/domain/valueobject"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

// CriarClienteUseCase encapsula a lógica de negócio para criação de clientes.
type CriarClienteUseCase struct {
	repo *repository.ClienteRepository
}

// NovoCriarClienteUseCase cria uma nova instância de CriarClienteUseCase.
func NovoCriarClienteUseCase(repo *repository.ClienteRepository) *CriarClienteUseCase {
	return &CriarClienteUseCase{repo: repo}
}

// Executar valida os dados e persiste um novo cliente.
func (uc *CriarClienteUseCase) Executar(nome string, documento string, email *string, telefone *string) (*entity.Cliente, error) {
	slog.Info("executando caso de uso: criar cliente")

	if nome == "" {
		return nil, errors.New("nome é obrigatório")
	}
	if documento == "" {
		return nil, errors.New("documento é obrigatório")
	}

	doc, err := valueobject.NewDocument(documento)
	if err != nil {
		return nil, err
	}

	cliente := &entity.Cliente{
		Nome:     nome,
		CpfCnpj:  doc.Value,
		Email:    email,
		Telefone: telefone,
	}

	resultado, err := uc.repo.Salvar(cliente)
	if err != nil {
		slog.Error("erro ao salvar cliente", "erro", err)
		return nil, err
	}

	slog.Info("cliente criado com sucesso", "id", resultado.ID)
	return resultado, nil
}
