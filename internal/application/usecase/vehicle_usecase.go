package usecase

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/domain/valueobject"
)

// VehicleUseCase centraliza todos os casos de uso relacionados a veiculo.
type VehicleUseCase struct {
	repo        veiculoRepo
	clienteRepo clienteRepo
}

// NewVehicleUseCase cria uma nova instância do use case de veiculos.
func NewVehicleUseCase(repo veiculoRepo, clienteRepo clienteRepo) *VehicleUseCase {
	return &VehicleUseCase{
		repo:        repo,
		clienteRepo: clienteRepo,
	}
}

// Create executa a criação de um novo veiculo.
func (uc *VehicleUseCase) Create(clienteID string, placa string, marca string, modelo string, ano int, cor *string) (*entity.Veiculo, error) {
	slog.Info("executando caso de uso: criar veículo")

	if clienteID == "" {
		return nil, errors.New("cliente_id é obrigatório")
	}

	parsedClienteID, err := uuid.Parse(clienteID)
	if err != nil {
		return nil, errors.New("cliente_id inválido")
	}

	plate, err := valueobject.NewPlate(placa)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(marca) == "" {
		return nil, errors.New("marca é obrigatória")
	}

	if strings.TrimSpace(modelo) == "" {
		return nil, errors.New("modelo é obrigatório")
	}

	if ano <= 0 {
		return nil, errors.New("ano inválido")
	}

	veiculo := &entity.Veiculo{
		ClienteID: parsedClienteID,
		Placa:     plate.Value,
		Marca:     marca,
		Modelo:    modelo,
		Ano:       ano,
		Cor:       cor,
	}

	return uc.repo.Salvar(veiculo)
}

// List retorna todos os veiculos cadastrados.
func (uc *VehicleUseCase) List() ([]*entity.Veiculo, error) {
	slog.Info("executando caso de uso: listar veículos")
	return uc.repo.BuscarTodos()
}

// FindByID retorna o veiculos cadastrado pelo UUID
func (uc *VehicleUseCase) FindByID(id string) (*entity.Veiculo, error) {
	slog.Info("executando caso de uso: buscar veículo por ID", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id inválido")
	}

	return uc.repo.BuscarPorID(id)
}

// Update atualiza o veiculos cadastrado pelo UUID, com os dados fornecidos
func (uc *VehicleUseCase) Update(id string, marca *string, modelo *string, ano *int, cor *string) (*entity.Veiculo, error) {
	slog.Info("executando caso de uso: atualizar veículo", "id", id)

	if id == "" {
		return nil, errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("id inválido")
	}

	veiculoAtual, err := uc.repo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}

	if marca != nil {
		if strings.TrimSpace(*marca) == "" {
			return nil, errors.New("marca não pode ser vazia")
		}
		veiculoAtual.Marca = *marca
	}

	if modelo != nil {
		if strings.TrimSpace(*modelo) == "" {
			return nil, errors.New("modelo não pode ser vazio")
		}
		veiculoAtual.Modelo = *modelo
	}

	if ano != nil {
		if *ano <= 0 {
			return nil, errors.New("ano inválido")
		}
		veiculoAtual.Ano = *ano
	}

	if cor != nil {
		veiculoAtual.Cor = cor
	}

	return uc.repo.Atualizar(veiculoAtual)
}

// Delete exclui o veiculos cadastrado pelo UUID
func (uc *VehicleUseCase) Delete(id string) error {
	slog.Info("executando caso de uso: remover veículo", "id", id)

	if id == "" {
		return errors.New("id é obrigatório")
	}

	if _, err := uuid.Parse(id); err != nil {
		return errors.New("id inválido")
	}

	return uc.repo.Remover(id)
}

// ListByCliente traz veiculos de um determinado cliente por UUID
func (uc *VehicleUseCase) ListByCliente(clienteID string) ([]*entity.Veiculo, error) {
	if clienteID == "" {
		return nil, errors.New("cliente_id é obrigatório")
	}

	// valida se cliente existe
	_, err := uc.clienteRepo.BuscarPorID(clienteID)
	if err != nil {
		return nil, errors.New("cliente não encontrado")
	}

	return uc.repo.BuscarPorClienteID(clienteID)
}
