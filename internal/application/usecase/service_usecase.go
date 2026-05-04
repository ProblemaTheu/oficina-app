package usecase

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
)

type ServiceUseCase struct {
	repo servicoRepo
}

func NewServiceUseCase(repo servicoRepo) *ServiceUseCase {
	return &ServiceUseCase{repo: repo}
}

func (uc *ServiceUseCase) Create(nome string, descricao *string, precoBase float64, tempoMinutos int) (*entity.Servico, error) {
	slog.Info("executando caso de uso: criar serviço")

	if strings.TrimSpace(nome) == "" {
		return nil, errors.New("nome é obrigatório")
	}

	if precoBase < 0 {
		return nil, errors.New("preço base não pode ser negativo")
	}

	if tempoMinutos <= 0 {
		return nil, errors.New("tempo em minutos deve ser maior que zero")
	}

	servico := &entity.Servico{
		Nome:         strings.TrimSpace(nome),
		Descricao:    descricao,
		PrecoBase:    precoBase,
		TempoMinutos: tempoMinutos,
	}

	return uc.repo.Salvar(servico)
}

func (uc *ServiceUseCase) List() ([]*entity.Servico, error) {
	slog.Info("executando caso de uso: listar serviços")
	return uc.repo.BuscarTodos()
}

func (uc *ServiceUseCase) FindByID(id string) (*entity.Servico, error) {
	slog.Info("executando caso de uso: buscar serviço por ID", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id inválido")
	}

	return uc.repo.BuscarPorID(id)
}

func (uc *ServiceUseCase) Update(id string, nome *string, descricao *string, precoBase *float64, tempoMinutos *int) (*entity.Servico, error) {
	slog.Info("executando caso de uso: atualizar serviço", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id inválido")
	}

	servicoAtual, err := uc.repo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}

	if nome != nil {
		if strings.TrimSpace(*nome) == "" {
			return nil, errors.New("nome não pode ser vazio")
		}

		servicoAtual.Nome = strings.TrimSpace(*nome)
	}

	if descricao != nil {
		servicoAtual.Descricao = descricao
	}

	if precoBase != nil {
		if *precoBase < 0 {
			return nil, errors.New("preço base não pode ser negativo")
		}

		servicoAtual.PrecoBase = *precoBase
	}

	if tempoMinutos != nil {
		if *tempoMinutos <= 0 {
			return nil, errors.New("tempo em minutos deve ser maior que zero")
		}

		servicoAtual.TempoMinutos = *tempoMinutos
	}

	return uc.repo.Atualizar(servicoAtual)
}

func (uc *ServiceUseCase) Delete(id string) error {
	slog.Info("executando caso de uso: remover serviço", "id", id)

	if id == "" {
		return errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return errors.New("id inválido")
	}

	return uc.repo.Remover(id)
}
