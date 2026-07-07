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

type PecaRepository struct {
	db *sql.DB
}

func NovoPecaRepository(db *sql.DB) *PecaRepository {
	return &PecaRepository{db: db}
}

func (r *PecaRepository) Salvar(peca *entity.Peca) (*entity.Peca, error) {
	slog.Info("salvando peça no banco de dados")

	peca.ID = uuid.New()
	peca.CriadoEm = time.Now()
	peca.AtualizadoEm = time.Now()

	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO pecas (id, nome, codigo, preco, estoque_atual, estoque_minimo, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		peca.ID,
		peca.Nome,
		peca.Codigo,
		peca.Preco,
		peca.EstoqueAtual,
		peca.EstoqueMinimo,
		peca.CriadoEm,
		peca.AtualizadoEm,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, &domainerros.ErrConflito{Campo: "codigo"}
		}
		return nil, fmt.Errorf("PecaRepository.Salvar: %w", err)
	}

	return peca, nil
}

func (r *PecaRepository) BuscarTodos() ([]*entity.Peca, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, nome, codigo, preco, estoque_atual, estoque_minimo, criado_em, atualizado_em
		FROM pecas
		ORDER BY criado_em DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("PecaRepository.BuscarTodos: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var pecas []*entity.Peca

	for rows.Next() {
		p := &entity.Peca{}

		if err := rows.Scan(
			&p.ID,
			&p.Nome,
			&p.Codigo,
			&p.Preco,
			&p.EstoqueAtual,
			&p.EstoqueMinimo,
			&p.CriadoEm,
			&p.AtualizadoEm,
		); err != nil {
			return nil, fmt.Errorf("PecaRepository.BuscarTodos scan: %w", err)
		}

		pecas = append(pecas, p)
	}

	return pecas, rows.Err()
}

func (r *PecaRepository) BuscarPorID(id string) (*entity.Peca, error) {
	var peca entity.Peca

	err := r.db.QueryRowContext(context.Background(), `
		SELECT id, nome, codigo, preco, estoque_atual, estoque_minimo, criado_em, atualizado_em
		FROM pecas
		WHERE id = $1
	`, id).Scan(
		&peca.ID,
		&peca.Nome,
		&peca.Codigo,
		&peca.Preco,
		&peca.EstoqueAtual,
		&peca.EstoqueMinimo,
		&peca.CriadoEm,
		&peca.AtualizadoEm,
	)

	if err != nil {
		return nil, err
	}

	return &peca, nil
}

func (r *PecaRepository) Atualizar(peca *entity.Peca) (*entity.Peca, error) {
	err := r.db.QueryRowContext(context.Background(), `
		UPDATE pecas
		SET nome = $1,
		    preco = $2,
		    estoque_minimo = $3,
		    atualizado_em = now()
		WHERE id = $4
		RETURNING id, nome, codigo, preco, estoque_atual, estoque_minimo, criado_em, atualizado_em
	`,
		peca.Nome,
		peca.Preco,
		peca.EstoqueMinimo,
		peca.ID,
	).Scan(
		&peca.ID,
		&peca.Nome,
		&peca.Codigo,
		&peca.Preco,
		&peca.EstoqueAtual,
		&peca.EstoqueMinimo,
		&peca.CriadoEm,
		&peca.AtualizadoEm,
	)

	if err != nil {
		return nil, err
	}

	return peca, nil
}

func (r *PecaRepository) AtualizarEstoque(peca *entity.Peca) (*entity.Peca, error) {
	err := r.db.QueryRowContext(context.Background(), `
		UPDATE pecas
		SET estoque_atual = $1,
		    atualizado_em = now()
		WHERE id = $2
		RETURNING id, nome, codigo, preco, estoque_atual, estoque_minimo, criado_em, atualizado_em
	`,
		peca.EstoqueAtual,
		peca.ID,
	).Scan(
		&peca.ID,
		&peca.Nome,
		&peca.Codigo,
		&peca.Preco,
		&peca.EstoqueAtual,
		&peca.EstoqueMinimo,
		&peca.CriadoEm,
		&peca.AtualizadoEm,
	)

	if err != nil {
		return nil, err
	}

	return peca, nil
}

func (r *PecaRepository) Remover(id string) error {
	result, err := r.db.ExecContext(context.Background(), `
		DELETE FROM pecas
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
