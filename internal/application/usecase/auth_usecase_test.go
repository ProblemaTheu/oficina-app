package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockUsuarioRepo struct {
	papelID      uuid.UUID
	papelErr     error
	salvarErr    error
	usuario      *entity.Usuario
	emailErr     error
	nomePapel    string
	nomePapelErr error
}

func (m *mockUsuarioRepo) BuscarPapelID(_ context.Context, _ string) (uuid.UUID, error) {
	return m.papelID, m.papelErr
}

func (m *mockUsuarioRepo) Salvar(_ context.Context, u *entity.Usuario) (*entity.Usuario, error) {
	if m.salvarErr != nil {
		return nil, m.salvarErr
	}
	u.ID = uuid.New()
	return u, nil
}

func (m *mockUsuarioRepo) BuscarPorEmail(_ context.Context, _ string) (*entity.Usuario, error) {
	return m.usuario, m.emailErr
}

func (m *mockUsuarioRepo) BuscarNomePapel(_ context.Context, _ uuid.UUID) (string, error) {
	return m.nomePapel, m.nomePapelErr
}

// ── testes ────────────────────────────────────────────────────────────────────

func TestCadastrarUsuario_EmailDuplicadoRetornaErrConflito(t *testing.T) {
	repo := &mockUsuarioRepo{
		papelID:   uuid.New(),
		salvarErr: &domainerros.ErrConflito{Campo: "email"},
		nomePapel: "atendente",
	}
	uc := usecase.NewAuthUseCase(repo)

	_, _, err := uc.CadastrarUsuario(context.Background(), usecase.CadastrarUsuarioInput{
		Nome:  "Teste",
		Email: "duplicado@example.com",
		Senha: "senha123",
	})

	var conflito *domainerros.ErrConflito
	if !errors.As(err, &conflito) {
		t.Errorf("esperava ErrConflito, obteve: %v", err)
	}
}

func TestLogin_SenhaErradaRetornaErro(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("senhaCorreta"), bcrypt.MinCost)

	repo := &mockUsuarioRepo{
		usuario: &entity.Usuario{
			ID:        uuid.New(),
			Email:     "user@example.com",
			SenhaHash: string(hash),
		},
	}
	uc := usecase.NewAuthUseCase(repo)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email: "user@example.com",
		Senha: "senhaErrada",
	})

	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) || np.Codigo != "credenciais_invalidas" {
		t.Errorf("esperava ErrNaoProcessavel credenciais_invalidas, obteve: %v", err)
	}
}

func TestLogin_EmailNaoEncontradoRetornaErro(t *testing.T) {
	repo := &mockUsuarioRepo{
		emailErr: errors.New("não encontrado"),
	}
	uc := usecase.NewAuthUseCase(repo)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email: "inexistente@example.com",
		Senha: "qualquer",
	})

	var np *domainerros.ErrNaoProcessavel
	if !errors.As(err, &np) || np.Codigo != "credenciais_invalidas" {
		t.Errorf("esperava ErrNaoProcessavel credenciais_invalidas, obteve: %v", err)
	}
}

func TestCadastrarUsuario_AliasAdminNormalizado(t *testing.T) {
	repo := &mockUsuarioRepo{
		papelID:   uuid.New(),
		nomePapel: "administrador",
	}
	uc := usecase.NewAuthUseCase(repo)

	u, papel, err := uc.CadastrarUsuario(context.Background(), usecase.CadastrarUsuarioInput{
		Nome:  "Admin",
		Email: "admin@example.com",
		Senha: "senha123",
		Papel: "admin",
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if u == nil || papel != "administrador" {
		t.Errorf("esperava papel 'administrador', obteve '%s'", papel)
	}
}

func TestCadastrarUsuario_PapelNaoEncontradoRetornaErro(t *testing.T) {
	repo := &mockUsuarioRepo{
		papelErr: &domainerros.ErrNaoEncontrado{Recurso: "papel"},
	}
	uc := usecase.NewAuthUseCase(repo)

	_, _, err := uc.CadastrarUsuario(context.Background(), usecase.CadastrarUsuarioInput{
		Nome:  "User",
		Email: "user@example.com",
		Senha: "senha123",
		Papel: "inexistente",
	})
	if err == nil {
		t.Error("esperava erro para papel inexistente")
	}
}

func TestLogin_SucessoRetornaToken(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("senha123"), bcrypt.MinCost)

	repo := &mockUsuarioRepo{
		usuario: &entity.Usuario{
			ID:        uuid.New(),
			Nome:      "João",
			Email:     "joao@example.com",
			SenhaHash: string(hash),
		},
	}
	uc := usecase.NewAuthUseCase(repo)

	out, err := uc.Login(context.Background(), usecase.LoginInput{
		Email: "joao@example.com",
		Senha: "senha123",
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	if out.AccessToken == "" {
		t.Error("token não deveria estar vazio")
	}
	if out.TokenType != "Bearer" {
		t.Errorf("esperava TokenType 'Bearer', obteve '%s'", out.TokenType)
	}
	if out.ExpiresIn != 28800 {
		t.Errorf("esperava ExpiresIn 28800, obteve %d", out.ExpiresIn)
	}
}
