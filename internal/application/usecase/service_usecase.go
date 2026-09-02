package usecase

import (
	"log/slog"
	"strings"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
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
		return nil, &domainerros.ErrValidacao{Mensagem: "nome é obrigatório"}
	}

	if precoBase < 0 {
		return nil, &domainerros.ErrValidacao{Mensagem: "preço base não pode ser negativo"}
	}

	if tempoMinutos <= 0 {
		return nil, &domainerros.ErrValidacao{Mensagem: "tempo em minutos deve ser maior que zero"}
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
		return nil, &domainerros.ErrValidacao{Mensagem: "id é obrigatório"}
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, &domainerros.ErrValidacao{Mensagem: "id inválido"}
	}

	return uc.repo.BuscarPorID(id)
}

func (uc *ServiceUseCase) Update(id string, nome *string, descricao *string, precoBase *float64, tempoMinutos *int) (*entity.Servico, error) {
	slog.Info("executando caso de uso: atualizar serviço", "id", id)

	if id == "" {
		return nil, &domainerros.ErrValidacao{Mensagem: "id é obrigatório"}
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, &domainerros.ErrValidacao{Mensagem: "id inválido"}
	}

	servicoAtual, err := uc.repo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}

	if nome != nil {
		if strings.TrimSpace(*nome) == "" {
			return nil, &domainerros.ErrValidacao{Mensagem: "nome não pode ser vazio"}
		}

		servicoAtual.Nome = strings.TrimSpace(*nome)
	}

	if descricao != nil {
		servicoAtual.Descricao = descricao
	}

	if precoBase != nil {
		if *precoBase < 0 {
			return nil, &domainerros.ErrValidacao{Mensagem: "preço base não pode ser negativo"}
		}

		servicoAtual.PrecoBase = *precoBase
	}

	if tempoMinutos != nil {
		if *tempoMinutos <= 0 {
			return nil, &domainerros.ErrValidacao{Mensagem: "tempo em minutos deve ser maior que zero"}
		}

		servicoAtual.TempoMinutos = *tempoMinutos
	}

	return uc.repo.Atualizar(servicoAtual)
}

func (uc *ServiceUseCase) Delete(id string) error {
	slog.Info("executando caso de uso: remover serviço", "id", id)

	if id == "" {
		return &domainerros.ErrValidacao{Mensagem: "id é obrigatório"}
	}

	if _, err := uuid.Parse(id); err != nil {
		return &domainerros.ErrValidacao{Mensagem: "id inválido"}
	}

	return uc.repo.Remover(id)
}
