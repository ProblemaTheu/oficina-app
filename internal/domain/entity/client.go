package entity

import (
	"time"

	"github.com/google/uuid"
)

// Situações possíveis do cadastro do cliente. Espelham o CHECK constraint
// chk_clientes_status da migration 000005.
const (
	StatusClienteAtivo     = "ativo"
	StatusClienteInativo   = "inativo"
	StatusClienteBloqueado = "bloqueado"
)

type Cliente struct {
	ID      uuid.UUID
	Nome    string
	CpfCnpj string
	// Status do cadastro: ativo, inativo ou bloqueado. Somente clientes
	// ativos obtêm token na Lambda de autenticação.
	Status       string
	Email        *string
	Telefone     *string
	CriadoEm     time.Time
	AtualizadoEm time.Time
}
