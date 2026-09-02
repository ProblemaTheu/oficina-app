package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// VeiculoRepository implementa o acesso ao banco de dados para clientes.
type VeiculoRepository struct {
	db *sql.DB
}

// NovoVeiculoRepository cria uma nova instância de VeiculoRepository.
func NovoVeiculoRepository(db *sql.DB) *VeiculoRepository {
	return &VeiculoRepository{db: db}
}

// Salvar persiste um novo veiculo no banco de dados.
func (r *VeiculoRepository) Salvar(veiculo *entity.Veiculo) (*entity.Veiculo, error) {
	slog.Info("salvando veículo no banco de dados")

	veiculo.ID = uuid.New()
	veiculo.CriadoEm = time.Now()
	veiculo.AtualizadoEm = time.Now()

	query := `
		INSERT INTO veiculos (id, cliente_id, placa, marca, modelo, ano, cor, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(context.Background(), query,
		veiculo.ID,
		veiculo.ClienteID,
		veiculo.Placa,
		veiculo.Marca,
		veiculo.Modelo,
		veiculo.Ano,
		veiculo.Cor,
		veiculo.CriadoEm,
		veiculo.AtualizadoEm,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, &domainerros.ErrConflito{Campo: "placa"}
		}
		return nil, fmt.Errorf("VeiculoRepository.Salvar: %w", err)
	}

	return veiculo, nil
}

// BuscarTodos retorna todos os veiculos do banco de dados
func (r *VeiculoRepository) BuscarTodos() ([]*entity.Veiculo, error) {
	query := `
		SELECT id, cliente_id, placa, marca, modelo, ano, cor, criado_em, atualizado_em
		FROM veiculos
		ORDER BY criado_em DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("VeiculoRepository.BuscarTodos: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var veiculos []*entity.Veiculo

	for rows.Next() {
		v := &entity.Veiculo{}
		if err := rows.Scan(
			&v.ID,
			&v.ClienteID,
			&v.Placa,
			&v.Marca,
			&v.Modelo,
			&v.Ano,
			&v.Cor,
			&v.CriadoEm,
			&v.AtualizadoEm,
		); err != nil {
			return nil, fmt.Errorf("VeiculoRepository.BuscarTodos scan: %w", err)
		}

		veiculos = append(veiculos, v)
	}

	return veiculos, rows.Err()
}

// BuscarPorID retorna o veiculo com o UUID passado
func (r *VeiculoRepository) BuscarPorID(id string) (*entity.Veiculo, error) {
	var veiculo entity.Veiculo

	err := r.db.QueryRowContext(context.Background(), `
		SELECT id, cliente_id, placa, marca, modelo, ano, cor, criado_em, atualizado_em
		FROM veiculos
		WHERE id = $1
	`, id).Scan(
		&veiculo.ID,
		&veiculo.ClienteID,
		&veiculo.Placa,
		&veiculo.Marca,
		&veiculo.Modelo,
		&veiculo.Ano,
		&veiculo.Cor,
		&veiculo.CriadoEm,
		&veiculo.AtualizadoEm,
	)

	if err == sql.ErrNoRows {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "veículo"}
	}
	if err != nil {
		return nil, fmt.Errorf("VeiculoRepository.BuscarPorID: %w", err)
	}

	return &veiculo, nil
}

// Atualizar atualiza o veiculo com o UUID passado pelas informações inseridas
func (r *VeiculoRepository) Atualizar(veiculo *entity.Veiculo) (*entity.Veiculo, error) {
	err := r.db.QueryRowContext(context.Background(), `
		UPDATE veiculos
		SET marca = $1,
		    modelo = $2,
		    ano = $3,
		    cor = $4,
		    atualizado_em = now()
		WHERE id = $5
		RETURNING id, cliente_id, placa, marca, modelo, ano, cor, criado_em, atualizado_em
	`,
		veiculo.Marca,
		veiculo.Modelo,
		veiculo.Ano,
		veiculo.Cor,
		veiculo.ID,
	).Scan(
		&veiculo.ID,
		&veiculo.ClienteID,
		&veiculo.Placa,
		&veiculo.Marca,
		&veiculo.Modelo,
		&veiculo.Ano,
		&veiculo.Cor,
		&veiculo.CriadoEm,
		&veiculo.AtualizadoEm,
	)

	if err == sql.ErrNoRows {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "veículo"}
	}
	if err != nil {
		return nil, fmt.Errorf("VeiculoRepository.Atualizar: %w", err)
	}

	return veiculo, nil
}

// Remover exclui, do banco de dados, o veiculo cujo UUID corresponde
func (r *VeiculoRepository) Remover(id string) error {
	result, err := r.db.ExecContext(context.Background(), `
		DELETE FROM veiculos
		WHERE id = $1
	`, id)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &domainerros.ErrNaoEncontrado{Recurso: "veículo"}
	}

	return nil
}

// BuscarPorClientID retorna veiculos de um usuário UUID passado.
func (r *VeiculoRepository) BuscarPorClienteID(clienteID string) ([]*entity.Veiculo, error) {
	query := `
		SELECT id, cliente_id, placa, marca, modelo, ano, cor, criado_em, atualizado_em
		FROM veiculos
		WHERE cliente_id = $1
		ORDER BY criado_em DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query, clienteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var veiculos []*entity.Veiculo

	for rows.Next() {
		v := &entity.Veiculo{}
		if err := rows.Scan(
			&v.ID,
			&v.ClienteID,
			&v.Placa,
			&v.Marca,
			&v.Modelo,
			&v.Ano,
			&v.Cor,
			&v.CriadoEm,
			&v.AtualizadoEm,
		); err != nil {
			return nil, err
		}

		veiculos = append(veiculos, v)
	}

	return veiculos, rows.Err()
}
