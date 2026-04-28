package valueobject

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
)

// ErrDocumentoInvalido é retornado quando o CPF ou CNPJ fornecido não é válido.
var ErrDocumentoInvalido = errors.New("documento inválido")

var reNaoDigito = regexp.MustCompile(`\D`)

// Document representa um documento fiscal (CPF ou CNPJ) validado.
type Document struct {
	Value string
}

// NewDocument cria um novo Document a partir de uma string que pode conter
// formatação (pontos, traços, barras). Valida o dígito verificador do CPF (11 dígitos)
// ou CNPJ (14 dígitos). Retorna ErrDocumentoInvalido se a validação falhar.
func NewDocument(value string) (*Document, error) {
	digits := apenasDigitos(value)

	slog.Debug("validando documento", "tamanho", len(digits))

	switch len(digits) {
	case 11:
		if !validarCPF(digits) {
			slog.Warn("CPF inválido fornecido")
			return nil, ErrDocumentoInvalido
		}
		slog.Info("CPF validado com sucesso")
	case 14:
		if !validarCNPJ(digits) {
			slog.Warn("CNPJ inválido fornecido")
			return nil, ErrDocumentoInvalido
		}
		slog.Info("CNPJ validado com sucesso")
	default:
		slog.Warn("documento com tamanho inválido", "tamanho", len(digits))
		return nil, ErrDocumentoInvalido
	}

	slog.Debug("documento válido", "tamanho", len(digits))
	return &Document{Value: digits}, nil
}

// apenasDigitos remove todos os caracteres não numéricos de uma string.
func apenasDigitos(s string) string {
	return reNaoDigito.ReplaceAllString(s, "")
}

// todosIguais verifica se todos os caracteres da string são iguais.
func todosIguais(s string) bool {
	return strings.Count(s, string(s[0])) == len(s)
}

// validarCPF verifica os dois dígitos verificadores de um CPF com 11 dígitos.
// Rejeita sequências repetidas como "11111111111".
func validarCPF(cpf string) bool {
	if todosIguais(cpf) {
		return false
	}

	soma := 0
	for i := 0; i < 9; i++ {
		soma += int(cpf[i]-'0') * (10 - i)
	}
	resto := soma % 11
	digito1 := 0
	if resto >= 2 {
		digito1 = 11 - resto
	}
	if int(cpf[9]-'0') != digito1 {
		return false
	}

	soma = 0
	for i := 0; i < 10; i++ {
		soma += int(cpf[i]-'0') * (11 - i)
	}
	resto = soma % 11
	digito2 := 0
	if resto >= 2 {
		digito2 = 11 - resto
	}
	return int(cpf[10]-'0') == digito2
}

// validarCNPJ verifica os dois dígitos verificadores de um CNPJ com 14 dígitos.
// Rejeita sequências repetidas como "00000000000000".
func validarCNPJ(cnpj string) bool {
	if todosIguais(cnpj) {
		return false
	}

	calcDigito := func(s string, pesos []int) int {
		soma := 0
		for i, p := range pesos {
			soma += int(s[i]-'0') * p
		}
		if resto := soma % 11; resto >= 2 {
			return 11 - resto
		}
		return 0
	}

	pesos1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	pesos2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	return calcDigito(cnpj, pesos1) == int(cnpj[12]-'0') &&
		calcDigito(cnpj, pesos2) == int(cnpj[13]-'0')
}
