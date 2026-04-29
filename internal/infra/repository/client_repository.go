package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
)

// ClienteRepository implementa o acesso ao banco de dados para clientes.
type ClienteRepository struct {
	db *sql.DB
}

// NovoClienteRepository cria uma nova instância de ClienteRepository.
func NovoClienteRepository(db *sql.DB) *ClienteRepository {
	return &ClienteRepository{db: db}
}

// Salvar persiste um novo cliente no banco de dados.
func (r *ClienteRepository) Salvar(cliente *entity.Cliente) (*entity.Cliente, error) {
	slog.Info("salvando cliente no banco de dados")

	cliente.ID = uuid.New()
	cliente.CriadoEm = time.Now()
	cliente.AtualizadoEm = time.Now()

	query := `
		INSERT INTO clientes (id, nome, cpf_cnpj, email, telefone, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(context.Background(), query,
		cliente.ID, cliente.Nome, cliente.CpfCnpj,
		cliente.Email, cliente.Telefone,
		cliente.CriadoEm, cliente.AtualizadoEm,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, &domainerros.ErrConflito{Campo: campoDoConstraint(pqErr.Constraint)}
		}
		return nil, fmt.Errorf("ClienteRepository.Salvar: %w", err)
	}

	slog.Info("cliente salvo com sucesso", "id", cliente.ID)
	return cliente, nil
}

// BuscarTodos retorna todos os clientes cadastrados no banco de dados.
func (r *ClienteRepository) BuscarTodos() ([]*entity.Cliente, error) {
	slog.Info("buscando todos os clientes no banco de dados")

	query := `
		SELECT id, nome, cpf_cnpj, email, telefone, criado_em, atualizado_em
		FROM clientes
		ORDER BY criado_em DESC
	`
	rows, err := r.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("ClienteRepository.BuscarTodos: %w", err)
	}
	defer rows.Close()

	var clientes []*entity.Cliente
	for rows.Next() {
		c := &entity.Cliente{}
		if err := rows.Scan(&c.ID, &c.Nome, &c.CpfCnpj, &c.Email, &c.Telefone, &c.CriadoEm, &c.AtualizadoEm); err != nil {
			return nil, fmt.Errorf("ClienteRepository.BuscarTodos scan: %w", err)
		}
		clientes = append(clientes, c)
	}

	slog.Info("clientes encontrados", "quantidade", len(clientes))
	return clientes, rows.Err()
}

// BuscarPorID retorna o cliente com o UUID passado
func (r *ClienteRepository) BuscarPorID(id string) (*entity.Cliente, error) {
	var cliente entity.Cliente

	err := r.db.QueryRow(`
		SELECT id, nome, cpf_cnpj, email, telefone
		FROM clientes
		WHERE id = $1
	`, id).Scan(
		&cliente.ID,
		&cliente.Nome,
		&cliente.CpfCnpj,
		&cliente.Email,
		&cliente.Telefone,
	)

	if err != nil {
		return nil, err
	}

	return &cliente, nil
}

// Atualizar atualiza o cliente com o UUID passado pelas informações inseridas
func (r *ClienteRepository) Atualizar(cliente *entity.Cliente) (*entity.Cliente, error) {
	err := r.db.QueryRow(`
		UPDATE clientes
		SET nome = $1,
		    cpf_cnpj = $2,
		    email = $3,
		    telefone = $4
		WHERE id = $5
		RETURNING id, nome, cpf_cnpj, email, telefone
	`,
		cliente.Nome,
		cliente.CpfCnpj,
		cliente.Email,
		cliente.Telefone,
		cliente.ID,
	).Scan(
		&cliente.ID,
		&cliente.Nome,
		&cliente.CpfCnpj,
		&cliente.Email,
		&cliente.Telefone,
	)

	if err != nil {
		return nil, err
	}

	return cliente, nil
}

// Remover exclui, do banco de dados, o cliente cujo UUID corresponde
func (r *ClienteRepository) Remover(id string) error {
	result, err := r.db.Exec(`
		DELETE FROM clientes
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
		return sql.ErrNoRows
	}

	return nil
}

// campoDoConstraint mapeia o nome do constraint PostgreSQL para o nome legível
// do campo, usado para compor a mensagem de ErrConflito.
func campoDoConstraint(constraint string) string {
	switch constraint {
	case "clientes_cpf_cnpj_key":
		return "cpf_cnpj"
	case "clientes_email_key":
		return "email"
	default:
		return constraint
	}
}
