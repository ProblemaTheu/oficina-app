package usecase

import (
	"context"
	"log/slog"

	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthUseCase centraliza os casos de uso de autenticação e cadastro de usuários.
type AuthUseCase struct {
	usuarioRepo *repository.UsuarioRepository
}

// NewAuthUseCase cria uma nova instância do use case de autenticação.
func NewAuthUseCase(usuarioRepo *repository.UsuarioRepository) *AuthUseCase {
	return &AuthUseCase{usuarioRepo: usuarioRepo}
}

// CadastrarUsuarioInput é o payload de criação de usuário.
type CadastrarUsuarioInput struct {
	Nome  string
	Email string
	Senha string
	Papel string
}

// CadastrarUsuarioOutput é o resultado após criação.
type CadastrarUsuarioOutput struct {
	ID       string
	Nome     string
	Email    string
	Papel    string
	CriadoEm string
}

// CadastrarUsuario cria um novo usuário com senha hasheada (bcrypt custo 12).
func (uc *AuthUseCase) CadastrarUsuario(ctx context.Context, input CadastrarUsuarioInput) (*entity.Usuario, string, error) {
	slog.Info("executando caso de uso: cadastrar usuário")

	papel := input.Papel
	if papel == "" {
		papel = "atendente"
	}

	// Mapear aliases para nome canônico do banco
	switch papel {
	case "admin":
		papel = "administrador"
	case "mecanico":
		papel = "mecanico"
	case "atendente":
		papel = "atendente"
	}

	papelID, err := uc.usuarioRepo.BuscarPapelID(ctx, papel)
	if err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Senha), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	usuario := &entity.Usuario{
		Nome:      input.Nome,
		Email:     input.Email,
		SenhaHash: string(hash),
		PapelID:   papelID,
	}

	saved, err := uc.usuarioRepo.Salvar(ctx, usuario)
	if err != nil {
		return nil, "", err
	}

	nomePapel, err := uc.usuarioRepo.BuscarNomePapel(ctx, saved.PapelID)
	if err != nil {
		nomePapel = papel
	}

	return saved, nomePapel, nil
}
