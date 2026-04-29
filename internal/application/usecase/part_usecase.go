package usecase

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

var ErrEstoqueInsuficiente = errors.New("estoque insuficiente")

type PartUseCase struct {
	repo *repository.PecaRepository
}

func NewPartUseCase(repo *repository.PecaRepository) *PartUseCase {
	return &PartUseCase{repo: repo}
}

func (uc *PartUseCase) Create(nome string, codigo string, preco float64, estoqueAtual *int, estoqueMinimo int) (*entity.Peca, error) {
	slog.Info("executando caso de uso: criar peça")

	if strings.TrimSpace(nome) == "" {
		return nil, errors.New("nome é obrigatório")
	}

	if strings.TrimSpace(codigo) == "" {
		return nil, errors.New("código é obrigatório")
	}

	if preco < 0 {
		return nil, errors.New("preço não pode ser negativo")
	}

	if estoqueMinimo < 0 {
		return nil, errors.New("estoque mínimo não pode ser negativo")
	}

	estoque := 0
	if estoqueAtual != nil {
		if *estoqueAtual < 0 {
			return nil, errors.New("estoque atual não pode ser negativo")
		}
		estoque = *estoqueAtual
	}

	peca := &entity.Peca{
		Nome:          strings.TrimSpace(nome),
		Codigo:        strings.TrimSpace(codigo),
		Preco:         preco,
		EstoqueAtual:  estoque,
		EstoqueMinimo: estoqueMinimo,
	}

	return uc.repo.Salvar(peca)
}

func (uc *PartUseCase) List() ([]*entity.Peca, error) {
	slog.Info("executando caso de uso: listar peças")
	return uc.repo.BuscarTodos()
}

func (uc *PartUseCase) FindByID(id string) (*entity.Peca, error) {
	slog.Info("executando caso de uso: buscar peça por ID", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id inválido")
	}

	return uc.repo.BuscarPorID(id)
}

func (uc *PartUseCase) Update(id string, nome *string, preco *float64, estoqueMinimo *int) (*entity.Peca, error) {
	slog.Info("executando caso de uso: atualizar peça", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id inválido")
	}

	pecaAtual, err := uc.repo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}

	if nome != nil {
		if strings.TrimSpace(*nome) == "" {
			return nil, errors.New("nome não pode ser vazio")
		}
		pecaAtual.Nome = strings.TrimSpace(*nome)
	}

	if preco != nil {
		if *preco < 0 {
			return nil, errors.New("preço não pode ser negativo")
		}
		pecaAtual.Preco = *preco
	}

	if estoqueMinimo != nil {
		if *estoqueMinimo < 0 {
			return nil, errors.New("estoque mínimo não pode ser negativo")
		}
		pecaAtual.EstoqueMinimo = *estoqueMinimo
	}

	return uc.repo.Atualizar(pecaAtual)
}

func (uc *PartUseCase) Delete(id string) error {
	slog.Info("executando caso de uso: remover peça", "id", id)

	if id == "" {
		return errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return errors.New("id inválido")
	}

	return uc.repo.Remover(id)
}

func (uc *PartUseCase) AdjustStock(id string, tipo string, quantidade int) (*entity.Peca, error) {
	slog.Info("executando caso de uso: ajustar estoque", "id", id, "tipo", tipo)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id inválido")
	}

	if quantidade < 0 {
		return nil, errors.New("quantidade não pode ser negativa")
	}

	pecaAtual, err := uc.repo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}

	switch tipo {
	case "entrada":
		pecaAtual.EstoqueAtual += quantidade

	case "saida":
		if pecaAtual.EstoqueAtual < quantidade {
			return nil, ErrEstoqueInsuficiente
		}
		pecaAtual.EstoqueAtual -= quantidade

	case "ajuste":
		pecaAtual.EstoqueAtual = quantidade

	default:
		return nil, errors.New("tipo de ajuste inválido")
	}

	return uc.repo.AtualizarEstoque(pecaAtual)
}
