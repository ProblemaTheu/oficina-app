// Package usecase contém os casos de uso da aplicação.
// Cada use case orquestra entidades de domínio e repositórios para
// executar uma operação de negócio específica, sem depender de detalhes
// de transporte HTTP ou de persistência.
package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
	domainerros "github.com/problematheu/tech-challenge-1/internal/domain/erros"
	"golang.org/x/crypto/bcrypt"
)

const jwtExpiresIn = 8 * time.Hour

func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("changeme-insecure-default-secret")
}

type AuthUseCase struct {
	usuarioRepo usuarioRepo
}

func NewAuthUseCase(usuarioRepo usuarioRepo) *AuthUseCase {
	return &AuthUseCase{usuarioRepo: usuarioRepo}
}

// ── Cadastrar usuário ─────────────────────────────────────────────────────────

// CadastrarUsuarioInput é o payload necessário para criar um novo usuário.
//
// Campos:
//   - Nome:  nome completo do usuário (obrigatório).
//   - Email: endereço de e-mail único no sistema (obrigatório).
//   - Senha: senha em texto plano — será hasheada antes de persistir (obrigatório).
//   - Papel: nome do papel atribuído ao usuário. Valores aceitos: "administrador",
//     "mecanico", "atendente". Se omitido, aplica-se "atendente" como padrão.
type CadastrarUsuarioInput struct {
	Nome  string
	Email string
	Senha string
	Papel string
}

// CadastrarUsuarioOutput é o resultado retornado após a criação bem-sucedida
// de um usuário. Expõe apenas os dados seguros (sem hash de senha).
type CadastrarUsuarioOutput struct {
	ID       string
	Nome     string
	Email    string
	Papel    string
	CriadoEm string
}

// CadastrarUsuario cria um novo usuário no sistema.
//
// Fluxo:
//  1. Normaliza o campo Papel (aplica "atendente" como padrão; mapeia alias "admin"
//     para "administrador").
//  2. Resolve o UUID do papel no banco de dados.
//  3. Gera o hash bcrypt da senha (custo padrão ≈ 10).
//  4. Persiste o usuário via repositório.
//  5. Retorna a entidade salva junto com o nome legível do papel.
//
// Erros possíveis:
//   - Papel inexistente no banco → ErrNaoEncontrado.
//   - E-mail duplicado → erro de constraint único do PostgreSQL.
//   - Falha de hash ou IO → erro genérico encapsulado com fmt.Errorf.
func (uc *AuthUseCase) CadastrarUsuario(ctx context.Context, input CadastrarUsuarioInput) (*entity.Usuario, string, error) {
	slog.InfoContext(ctx, "cadastrar usuário: iniciando", "email", input.Email, "papel_solicitado", input.Papel)

	papel := input.Papel
	if papel == "" {
		papel = "atendente"
	}

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
		slog.WarnContext(ctx, "cadastrar usuário: papel não encontrado", "papel", papel, "error", err)
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Senha), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "cadastrar usuário: falha ao gerar hash da senha", "error", err)
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
		slog.ErrorContext(ctx, "cadastrar usuário: falha ao persistir", "email", input.Email, "error", err)
		return nil, "", err
	}

	nomePapel, err := uc.usuarioRepo.BuscarNomePapel(ctx, saved.PapelID)
	if err != nil {
		slog.WarnContext(ctx, "cadastrar usuário: não foi possível resolver nome do papel, usando valor normalizado", "papel_id", saved.PapelID, "error", err)
		nomePapel = papel
	}

	slog.InfoContext(ctx, "cadastrar usuário: concluído com sucesso", "usuario_id", saved.ID, "email", saved.Email, "papel", nomePapel)
	return saved, nomePapel, nil
}

// ── Login ─────────────────────────────────────────────────────────────────────

// LoginInput é o payload necessário para autenticar um usuário.
type LoginInput struct {
	Email string
	Senha string
}

// LoginOutput é o resultado retornado após autenticação bem-sucedida.
//
// Campos:
//   - AccessToken: JWT assinado com HS256, válido por jwtExpiresIn (8h).
//   - TokenType:   sempre "Bearer" — deve ser usado no header Authorization.
//   - ExpiresIn:   tempo de expiração em segundos (28800 para 8h).
type LoginOutput struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
}

// Login autentica um usuário por e-mail e senha e emite um JWT Bearer token.
//
// Fluxo:
//  1. Busca o usuário pelo e-mail. Se não encontrado, retorna credenciais inválidas
//     (sem revelar se o e-mail existe — prevenção de user enumeration).
//  2. Compara a senha fornecida com o hash bcrypt armazenado.
//  3. Se válido, gera um JWT HS256 com os claims abaixo e o assina com jwtSecret().
//
// Claims do JWT:
//   - sub:   UUID do usuário (string).
//   - email: endereço de e-mail do usuário.
//   - nome:  nome completo do usuário.
//   - iat:   timestamp de emissão (Unix).
//   - exp:   timestamp de expiração (Unix, iat + 8h).
//
// Erros possíveis:
//   - E-mail não encontrado ou senha incorreta → ErrNaoProcessavel (código
//     "credenciais_invalidas"). A mesma mensagem é usada em ambos os casos
//     para evitar enumeração de usuários.
//   - Falha ao assinar o token → erro genérico encapsulado com fmt.Errorf.
func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	slog.InfoContext(ctx, "login: tentativa de autenticação", "email", input.Email)

	usuario, err := uc.usuarioRepo.BuscarPorEmail(ctx, input.Email)
	if err != nil {
		// Log com Warn — falha esperada (e-mail inexistente), não é erro de sistema.
		// A mensagem retornada ao cliente é genérica para evitar user enumeration.
		slog.WarnContext(ctx, "login: e-mail não encontrado", "email", input.Email)
		return nil, &domainerros.ErrNaoProcessavel{Codigo: "credenciais_invalidas", Mensagem: "e-mail ou senha inválidos"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(input.Senha)); err != nil {
		slog.WarnContext(ctx, "login: senha incorreta", "usuario_id", usuario.ID, "email", input.Email)
		return nil, &domainerros.ErrNaoProcessavel{Codigo: "credenciais_invalidas", Mensagem: "e-mail ou senha inválidos"}
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   usuario.ID.String(),
		"email": usuario.Email,
		"nome":  usuario.Nome,
		"iat":   now.Unix(),
		"exp":   now.Add(jwtExpiresIn).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret())
	if err != nil {
		slog.ErrorContext(ctx, "login: falha ao assinar token JWT", "usuario_id", usuario.ID, "error", err)
		return nil, fmt.Errorf("auth: gerar token: %w", err)
	}

	slog.InfoContext(ctx, "login: autenticação bem-sucedida", "usuario_id", usuario.ID, "email", usuario.Email, "expira_em_segundos", int(jwtExpiresIn.Seconds()))
	return &LoginOutput{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int(jwtExpiresIn.Seconds()),
	}, nil
}
