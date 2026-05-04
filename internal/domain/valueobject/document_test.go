package valueobject_test

import (
	"errors"
	"testing"

	"github.com/problematheu/tech-challenge-1/internal/domain/valueobject"
)

func TestNewDocument_CPFValido(t *testing.T) {
	casos := []string{
		"529.982.247-25",
		"52998224725",
	}
	for _, c := range casos {
		if _, err := valueobject.NewDocument(c); err != nil {
			t.Errorf("CPF %q deveria ser válido, obteve: %v", c, err)
		}
	}
}

func TestNewDocument_CPFInvalido(t *testing.T) {
	casos := []string{
		"111.111.111-11", // todos iguais
		"000.000.000-00",
		"529.982.247-26", // último dígito errado
		"12345678900",    // dígitos verificadores errados
		"1234",           // muito curto
	}
	for _, c := range casos {
		_, err := valueobject.NewDocument(c)
		if !errors.Is(err, valueobject.ErrDocumentoInvalido) {
			t.Errorf("CPF %q deveria ser inválido, obteve: %v", c, err)
		}
	}
}

func TestNewDocument_CNPJValido(t *testing.T) {
	casos := []string{
		"11.222.333/0001-81",
		"11222333000181",
	}
	for _, c := range casos {
		if _, err := valueobject.NewDocument(c); err != nil {
			t.Errorf("CNPJ %q deveria ser válido, obteve: %v", c, err)
		}
	}
}

func TestNewDocument_CNPJInvalido(t *testing.T) {
	casos := []string{
		"00.000.000/0000-00", // todos iguais
		"11.111.111/1111-11",
		"11.222.333/0001-82", // último dígito errado
		"1234567890123",      // 13 dígitos
	}
	for _, c := range casos {
		_, err := valueobject.NewDocument(c)
		if !errors.Is(err, valueobject.ErrDocumentoInvalido) {
			t.Errorf("CNPJ %q deveria ser inválido, obteve: %v", c, err)
		}
	}
}

func TestNewDocument_ValorArmazenaSemMascara(t *testing.T) {
	doc, err := valueobject.NewDocument("529.982.247-25")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if doc.Value != "52998224725" {
		t.Errorf("esperava '52998224725', obteve '%s'", doc.Value)
	}
}
