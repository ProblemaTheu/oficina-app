package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UsuarioRepository implementa o acesso ao banco de dados para usuários.
type UsuarioRepository struct {
	db *sql.DB
}

// NovoUsuarioRepository cria uma nova instância de UsuarioRepository.
func NovoUsuarioRepository(db *sql.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

// BuscarPapelID retorna o UUID do papel pelo nome.
func (r *UsuarioRepository) BuscarPapelID(ctx context.Context, nomePapel string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM papeis_usuario WHERE nome_papel = $1`, nomePapel,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return uuid.Nil, &domainerros.ErrNaoEncontrado{Recurso: "papel '" + nomePapel + "'"}
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("UsuarioRepository.BuscarPapelID: %w", err)
	}
	return id, nil
}

// Salvar persiste um novo usuário no banco.
func (r *UsuarioRepository) Salvar(ctx context.Context, u *entity.Usuario) (*entity.Usuario, error) {
	u.ID = uuid.New()
	u.CriadoEm = time.Now()
	u.AtualizadoEm = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO usuarios (id, nome, email, senha_hash, papel_id, criado_em, atualizado_em)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`,
		u.ID, u.Nome, u.Email, u.SenhaHash, u.PapelID, u.CriadoEm, u.AtualizadoEm,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, &domainerros.ErrConflito{Campo: "email"}
		}
		return nil, fmt.Errorf("UsuarioRepository.Salvar: %w", err)
	}
	return u, nil
}

// BuscarPorEmail retorna o usuário com o e-mail passado.
func (r *UsuarioRepository) BuscarPorEmail(ctx context.Context, email string) (*entity.Usuario, error) {
	u := &entity.Usuario{}
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.nome, u.email, u.senha_hash, u.papel_id, u.criado_em, u.atualizado_em
		FROM usuarios u
		WHERE u.email = $1
	`, email).Scan(&u.ID, &u.Nome, &u.Email, &u.SenhaHash, &u.PapelID, &u.CriadoEm, &u.AtualizadoEm)
	if err == sql.ErrNoRows {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "usuário"}
	}
	if err != nil {
		return nil, fmt.Errorf("UsuarioRepository.BuscarPorEmail: %w", err)
	}
	return u, nil
}

// BuscarNomePapel retorna o nome do papel pelo UUID.
func (r *UsuarioRepository) BuscarNomePapel(ctx context.Context, papelID uuid.UUID) (string, error) {
	var nome string
	err := r.db.QueryRowContext(ctx,
		`SELECT nome_papel FROM papeis_usuario WHERE id = $1`, papelID,
	).Scan(&nome)
	if err != nil {
		return "", fmt.Errorf("UsuarioRepository.BuscarNomePapel: %w", err)
	}
	return nome, nil
}
