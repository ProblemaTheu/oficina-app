package usecase

import (
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

type ListClientsUseCase struct {
	repo *repository.ClientRepository
}

func NewListClientsUseCase(repo *repository.ClientRepository) *ListClientsUseCase {
	return &ListClientsUseCase{
		repo: repo,
	}
}

func (uc *ListClientsUseCase) Execute() ([]*entity.Client, error) {
	return uc.repo.FindAll()
}
