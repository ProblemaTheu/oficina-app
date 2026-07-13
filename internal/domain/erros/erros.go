// Package erros define os tipos de erro de domínio utilizados em todas as
// camadas da aplicação. Usando tipos concretos, a camada HTTP consegue mapear
// cada erro ao status code correto sem conhecer detalhes de infraestrutura.
package erros

import "fmt"

// ErrConflito é retornado quando há violação de unicidade (HTTP 409).
// O campo Campo indica qual atributo causou o conflito.
type ErrConflito struct {
	Campo string
}

func (e *ErrConflito) Error() string {
	return fmt.Sprintf("%s já cadastrado", e.Campo)
}

// ErrNaoEncontrado é retornado quando o recurso solicitado não existe (HTTP 404).
// O campo Recurso indica qual entidade não foi localizada.
type ErrNaoEncontrado struct {
	Recurso string
}

func (e *ErrNaoEncontrado) Error() string {
	return fmt.Sprintf("%s não encontrado", e.Recurso)
}

// ErrValidacao é retornado quando a entrada não satisfaz validações de
// obrigatoriedade ou formato (HTTP 400).
type ErrValidacao struct {
	Mensagem string
}

func (e *ErrValidacao) Error() string {
	return e.Mensagem
}

// ErrNaoProcessavel é retornado quando uma regra de negócio impede o
// processamento da requisição (HTTP 422). O campo Codigo identifica a regra
// violada de forma legível por máquina.
type ErrNaoProcessavel struct {
	Codigo   string
	Mensagem string
}

func (e *ErrNaoProcessavel) Error() string {
	return e.Mensagem
}
