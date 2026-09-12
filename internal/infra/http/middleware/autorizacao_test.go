package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
)

func ctxComClaims(claims jwt.MapClaims) context.Context {
	return context.WithValue(context.Background(), ClaimsContextKey, claims)
}

func TestClaimsDoContexto(t *testing.T) {
	if _, ok := ClaimsDoContexto(context.Background()); ok {
		t.Fatal("contexto vazio não deveria ter claims")
	}
	claims, ok := ClaimsDoContexto(ctxComClaims(jwt.MapClaims{"sub": "u1"}))
	if !ok || claims["sub"] != "u1" {
		t.Fatalf("claims = %v, ok = %v", claims, ok)
	}
}

func TestTipoDoContexto(t *testing.T) {
	casos := []struct {
		nome   string
		ctx    context.Context
		espera string
	}{
		{"sem claims", context.Background(), ""},
		// Token anterior ao contrato: tratado como funcionário para não
		// deslogar todo mundo no deploy.
		{"sem claim tipo", ctxComClaims(jwt.MapClaims{"sub": "u1"}), TipoUsuario},
		{"cliente", ctxComClaims(jwt.MapClaims{"tipo": TipoCliente}), TipoCliente},
		{"usuario", ctxComClaims(jwt.MapClaims{"tipo": TipoUsuario}), TipoUsuario},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := TipoDoContexto(c.ctx); got != c.espera {
				t.Fatalf("tipo = %q, esperado %q", got, c.espera)
			}
		})
	}
}

func TestSubEPapelDoContexto(t *testing.T) {
	if SubDoContexto(context.Background()) != "" || PapelDoContexto(context.Background()) != "" {
		t.Fatal("contexto vazio deveria devolver strings vazias")
	}
	ctx := ctxComClaims(jwt.MapClaims{"sub": "770e8400", "papel": "administrador"})
	if got := SubDoContexto(ctx); got != "770e8400" {
		t.Errorf("sub = %q", got)
	}
	if got := PapelDoContexto(ctx); got != "administrador" {
		t.Errorf("papel = %q", got)
	}
}

func proibido(t *testing.T, err error) *domainerros.ErrProibido {
	t.Helper()
	var e *domainerros.ErrProibido
	if !errors.As(err, &e) {
		t.Fatalf("erro = %v, esperado *ErrProibido", err)
	}
	return e
}

func TestExigirUsuario(t *testing.T) {
	if err := ExigirUsuario(ctxComClaims(jwt.MapClaims{"tipo": TipoUsuario})); err != nil {
		t.Fatalf("usuário não deveria ser barrado: %v", err)
	}
	// O authorizer do Gateway aceita este token — é aqui que ele é barrado.
	e := proibido(t, ExigirUsuario(ctxComClaims(jwt.MapClaims{"tipo": TipoCliente})))
	if e.Codigo != "ACCESS_DENIED" {
		t.Fatalf("código = %q", e.Codigo)
	}
}

func TestExigirPapel(t *testing.T) {
	admin := ctxComClaims(jwt.MapClaims{"tipo": TipoUsuario, "papel": "administrador"})
	atendente := ctxComClaims(jwt.MapClaims{"tipo": TipoUsuario, "papel": "atendente"})
	cliente := ctxComClaims(jwt.MapClaims{"tipo": TipoCliente, "papel": "administrador"})

	if err := ExigirPapel(admin, "administrador"); err != nil {
		t.Fatalf("administrador deveria passar: %v", err)
	}
	if err := ExigirPapel(atendente, "administrador", "atendente"); err != nil {
		t.Fatalf("atendente está na lista e deveria passar: %v", err)
	}
	_ = proibido(t, ExigirPapel(atendente, "administrador"))
	// Cliente com papel forjado: barrado antes de olhar o papel.
	_ = proibido(t, ExigirPapel(cliente, "administrador"))
}
