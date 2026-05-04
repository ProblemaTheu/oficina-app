package usecase

import (
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/domain/valueobject"
)

// ClientUseCase centraliza todos os casos de uso relacionados a cliente.
type ClientUseCase struct {
	repo clienteRepo
}

// NewClientUseCase cria uma nova instância do use case de cliente.
func NewClientUseCase(repo clienteRepo) *ClientUseCase {
	return &ClientUseCase{repo: repo}
}

// Create executa a criação de um novo cliente.
func (uc *ClientUseCase) Create(nome string, documento string, email *string, telefone *string) (*entity.Cliente, error) {
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

// List retorna todos os clientes cadastrados.
func (uc *ClientUseCase) List() ([]*entity.Cliente, error) {
	slog.Info("executando caso de uso: listar clientes")
	return uc.repo.BuscarTodos()
}

// FindByID retorna o cliente cadastrado pelo UUID
func (uc *ClientUseCase) FindByID(id string) (*entity.Cliente, error) {
	slog.Info("executando caso de uso: buscar cliente por ID", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	return uc.repo.BuscarPorID(id)
}

// Update atualiza o cliente cadastrado pelo UUID, com os dados fornecidos
func (uc *ClientUseCase) Update(id string, nome *string, email *string, telefone *string) (*entity.Cliente, error) {
	slog.Info("executando caso de uso: atualizar cliente", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("id inválido")
	}

	clienteAtual, err := uc.repo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}

	if nome != nil {
		clienteAtual.Nome = *nome
	}
	if email != nil {
		clienteAtual.Email = email
	}
	if telefone != nil {
		clienteAtual.Telefone = telefone
	}

	clienteAtual.ID = parsedID

	return uc.repo.Atualizar(clienteAtual)
}

// Delete exclui o cliente cadastrado pelo UUID
func (uc *ClientUseCase) Delete(id string) error {
	slog.Info("executando caso de uso: remover cliente", "id", id)

	if id == "" {
		return errors.New("id é obrigatório")
	}

	return uc.repo.Remover(id)
}
