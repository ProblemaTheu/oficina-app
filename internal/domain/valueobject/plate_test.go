package valueobject_test

import (
	"errors"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/domain/valueobject"
)

func TestNewPlate_FormatoAntigo(t *testing.T) {
	casos := []string{
		"ABC1234",
		"abc1234",   // minúsculas são normalizadas
		"XYZ-9999",  // com hífen
		" DEF5678 ", // com espaços
	}
	for _, c := range casos {
		if _, err := valueobject.NewPlate(c); err != nil {
			t.Errorf("placa %q deveria ser válida (formato antigo), obteve: %v", c, err)
		}
	}
}

func TestNewPlate_FormatoMercosul(t *testing.T) {
	casos := []string{
		"ABC1D23",
		"abc1d23",
	}
	for _, c := range casos {
		if _, err := valueobject.NewPlate(c); err != nil {
			t.Errorf("placa %q deveria ser válida (Mercosul), obteve: %v", c, err)
		}
	}
}

func TestNewPlate_Invalida(t *testing.T) {
	casos := []string{
		"",
		"ABC123",   // 6 caracteres
		"ABCD1234", // 4 letras no início
		"1234ABC",  // dígitos antes das letras
		"ABC12D3",  // Mercosul com padrão errado
		"AB1234",   // só 2 letras no início
	}
	for _, c := range casos {
		_, err := valueobject.NewPlate(c)
		if !errors.Is(err, valueobject.ErrPlacaInvalida) {
			t.Errorf("placa %q deveria ser inválida, obteve: %v", c, err)
		}
	}
}

func TestNewPlate_NormalizaMaiusculas(t *testing.T) {
	p, err := valueobject.NewPlate("abc1d23")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if p.Value != "ABC1D23" {
		t.Errorf("esperava 'ABC1D23', obteve '%s'", p.Value)
	}
}
