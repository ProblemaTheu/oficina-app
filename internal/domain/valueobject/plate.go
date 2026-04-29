package valueobject

import (
	"errors"
	"regexp"
	"strings"
)

var ErrPlacaInvalida = errors.New("placa inválida")

type Plate struct {
	Value string
}

func NewPlate(value string) (*Plate, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "")

	// Formato antigo: ABC1234
	oldPlateRegex := regexp.MustCompile(`^[A-Z]{3}[0-9]{4}$`)

	// Formato Mercosul: ABC1D23
	mercosulPlateRegex := regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)

	if !oldPlateRegex.MatchString(normalized) && !mercosulPlateRegex.MatchString(normalized) {
		return nil, ErrPlacaInvalida
	}

	return &Plate{
		Value: normalized,
	}, nil
}
