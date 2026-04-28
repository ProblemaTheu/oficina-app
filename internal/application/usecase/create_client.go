package usecase

import (
	"errors"

	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/domain/valueobject"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

type CreateClientUseCase struct {
	repo *repository.ClientRepository
}

func NewCreateClientUseCase(repo *repository.ClientRepository) *CreateClientUseCase {
	return &CreateClientUseCase{
		repo: repo,
	}
}

func (uc *CreateClientUseCase) Execute(name string, document string) (*entity.Cliente, error) {
	if name == "" {
		return nil, errors.New("nome é obrigatório")
	}

	doc, err := valueobject.NewDocument(document)
	if err != nil {
		return nil, err
	}

	client := &entity.Cliente{
		Nome:    name,
		CpfCnpj: doc.Value,
	}

	return uc.repo.Save(client)
}