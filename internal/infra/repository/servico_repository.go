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

type ServicoRepository struct {
	db *sql.DB
}

func NovoServicoRepository(db *sql.DB) *ServicoRepository {
	return &ServicoRepository{db: db}
}

func (r *ServicoRepository) Salvar(servico *entity.Servico) (*entity.Servico, error) {
	slog.Info("salvando serviço no banco de dados")

	servico.ID = uuid.New()
	servico.CriadoEm = time.Now()
	servico.AtualizadoEm = time.Now()

	query := `
		INSERT INTO servicos (id, nome, descricao, preco_base, tempo_minutos, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		context.Background(),
		query,
		servico.ID,
		servico.Nome,
		servico.Descricao,
		servico.PrecoBase,
		servico.TempoMinutos,
		servico.CriadoEm,
		servico.AtualizadoEm,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, &domainerros.ErrConflito{Campo: "nome"}
		}

		return nil, fmt.Errorf("ServicoRepository.Salvar: %w", err)
	}

	return servico, nil
}

func (r *ServicoRepository) BuscarTodos() ([]*entity.Servico, error) {
	query := `
		SELECT id, nome, descricao, preco_base, tempo_minutos, criado_em, atualizado_em
		FROM servicos
		ORDER BY criado_em DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("ServicoRepository.BuscarTodos: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var servicos []*entity.Servico

	for rows.Next() {
		s := &entity.Servico{}

		if err := rows.Scan(
			&s.ID,
			&s.Nome,
			&s.Descricao,
			&s.PrecoBase,
			&s.TempoMinutos,
			&s.CriadoEm,
			&s.AtualizadoEm,
		); err != nil {
			return nil, fmt.Errorf("ServicoRepository.BuscarTodos scan: %w", err)
		}

		servicos = append(servicos, s)
	}

	return servicos, rows.Err()
}

func (r *ServicoRepository) BuscarPorID(id string) (*entity.Servico, error) {
	var servico entity.Servico

	err := r.db.QueryRowContext(context.Background(), `
		SELECT id, nome, descricao, preco_base, tempo_minutos, criado_em, atualizado_em
		FROM servicos
		WHERE id = $1
	`, id).Scan(
		&servico.ID,
		&servico.Nome,
		&servico.Descricao,
		&servico.PrecoBase,
		&servico.TempoMinutos,
		&servico.CriadoEm,
		&servico.AtualizadoEm,
	)

	if err != nil {
		return nil, err
	}

	return &servico, nil
}

func (r *ServicoRepository) Atualizar(servico *entity.Servico) (*entity.Servico, error) {
	err := r.db.QueryRowContext(context.Background(), `
		UPDATE servicos
		SET nome = $1,
		    descricao = $2,
		    preco_base = $3,
		    tempo_minutos = $4,
		    atualizado_em = now()
		WHERE id = $5
		RETURNING id, nome, descricao, preco_base, tempo_minutos, criado_em, atualizado_em
	`,
		servico.Nome,
		servico.Descricao,
		servico.PrecoBase,
		servico.TempoMinutos,
		servico.ID,
	).Scan(
		&servico.ID,
		&servico.Nome,
		&servico.Descricao,
		&servico.PrecoBase,
		&servico.TempoMinutos,
		&servico.CriadoEm,
		&servico.AtualizadoEm,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, &domainerros.ErrConflito{Campo: "nome"}
		}

		return nil, err
	}

	return servico, nil
}

func (r *ServicoRepository) Remover(id string) error {
	result, err := r.db.ExecContext(context.Background(), `
		DELETE FROM servicos
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
