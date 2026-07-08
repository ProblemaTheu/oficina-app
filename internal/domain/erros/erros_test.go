package erros

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrConflito_Error(t *testing.T) {
	err := &ErrConflito{Campo: "email"}
	if got := err.Error(); got != "email já cadastrado" {
		t.Errorf("mensagem inesperada: %q", got)
	}
}

func TestErrNaoEncontrado_Error(t *testing.T) {
	err := &ErrNaoEncontrado{Recurso: "cliente"}
	if got := err.Error(); got != "cliente não encontrado" {
		t.Errorf("mensagem inesperada: %q", got)
	}
}

func TestErrValidacao_Error(t *testing.T) {
	err := &ErrValidacao{Mensagem: "nome é obrigatório"}
	if got := err.Error(); got != "nome é obrigatório" {
		t.Errorf("mensagem inesperada: %q", got)
	}
}

func TestErrNaoProcessavel_Error(t *testing.T) {
	err := &ErrNaoProcessavel{Codigo: "INSUFFICIENT_STOCK", Mensagem: "estoque insuficiente"}
	if got := err.Error(); got != "estoque insuficiente" {
		t.Errorf("mensagem inesperada: %q", got)
	}
}

// TestErros_CompativeisComErrorsAs garante que os erros de domínio embrulhados
// continuam identificáveis via errors.As — é assim que a camada HTTP os mapeia
// para status codes.
func TestErros_CompativeisComErrorsAs(t *testing.T) {
	casos := []struct {
		nome       string
		embrulhado error
		verifica   func(error) bool
	}{
		{"conflito", fmt.Errorf("contexto: %w", &ErrConflito{Campo: "cpf"}), func(err error) bool {
			var e *ErrConflito
			return errors.As(err, &e) && e.Campo == "cpf"
		}},
		{"não encontrado", fmt.Errorf("contexto: %w", &ErrNaoEncontrado{Recurso: "peça"}), func(err error) bool {
			var e *ErrNaoEncontrado
			return errors.As(err, &e) && e.Recurso == "peça"
		}},
		{"validação", fmt.Errorf("contexto: %w", &ErrValidacao{Mensagem: "inválido"}), func(err error) bool {
			var e *ErrValidacao
			return errors.As(err, &e) && e.Mensagem == "inválido"
		}},
		{"não processável", fmt.Errorf("contexto: %w", &ErrNaoProcessavel{Codigo: "X", Mensagem: "y"}), func(err error) bool {
			var e *ErrNaoProcessavel
			return errors.As(err, &e) && e.Codigo == "X"
		}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if !c.verifica(c.embrulhado) {
				t.Errorf("errors.As não identificou o erro embrulhado: %v", c.embrulhado)
			}
		})
	}
}
