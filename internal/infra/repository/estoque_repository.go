package repository

import (
	"context"
	"database/sql"
	"fmt"

	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
)

// EstoqueRepository reserva, libera e baixa as peças de uma OS (F4-1.2).
//
// As três operações recebem a transação de quem chama para compor com a saga e
// a outbox numa transação só — é o que elimina o dual-write do fluxo antigo
// (DeduzirEstoquePecas numa tx, AtualizarStatus em outra).
//
// Cada operação só age sobre itens no estado de origem (reserva_estado), o que
// as torna idempotentes por OS: chamar duas vezes não altera o saldo duas vezes.
type EstoqueRepository struct{}

// NovoEstoqueRepository cria o repositório.
func NovoEstoqueRepository() *EstoqueRepository { return &EstoqueRepository{} }

type itemEstoque struct {
	itemID     uuid.UUID
	pecaID     uuid.UUID
	quantidade int
	nome       string
}

// estadoReservada é o reserva_estado dos itens que Liberar e Baixar consomem.
var estadoReservada = "reservada"

// itens lista os itens de peça da OS num estado de reserva, em ordem de peça —
// a ordem fixa evita deadlock entre duas OS que reservam as mesmas peças.
//
// estado nil seleciona os itens ainda não reservados (reserva_estado NULL).
// IS NOT DISTINCT FROM compara NULL como valor, o que mantém a query estática
// e parametrizada (sem concatenar SQL — gosec G202 / Sonar).
func itens(ctx context.Context, tx *sql.Tx, osID uuid.UUID, estado *string) ([]itemEstoque, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT iop.id, iop.peca_id, iop.quantidade, p.nome
		FROM itens_os_pecas iop
		JOIN pecas p ON p.id = iop.peca_id
		WHERE iop.os_id = $1 AND iop.reserva_estado IS NOT DISTINCT FROM $2::varchar
		ORDER BY iop.peca_id`, osID, estado)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var out []itemEstoque
	for rows.Next() {
		var i itemEstoque
		if err := rows.Scan(&i.itemID, &i.pecaID, &i.quantidade, &i.nome); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func marcarItem(ctx context.Context, tx *sql.Tx, itemID uuid.UUID, estado string) error {
	_, err := tx.ExecContext(ctx, `UPDATE itens_os_pecas SET reserva_estado = $1, atualizado_em = now() WHERE id = $2`, estado, itemID)
	return err
}

// Reservar segura as peças da OS sem retirá-las do estoque. Falha com
// ErrNaoProcessavel (INSUFFICIENT_STOCK) se o disponível (atual - reservado)
// não bastar para algum item — e, como a transação é de quem chama, nada da OS
// é persistido (T1 da saga).
//
// Sem SELECT antes do UPDATE: a condição de saldo vai no próprio UPDATE, então
// duas OS disputando a última unidade não passam ambas (a segunda espera o lock
// da linha, reavalia a condição e não encontra saldo).
func (EstoqueRepository) Reservar(ctx context.Context, tx *sql.Tx, osID uuid.UUID) error {
	lista, err := itens(ctx, tx, osID, nil)
	if err != nil {
		return fmt.Errorf("EstoqueRepository.Reservar itens: %w", err)
	}
	for _, i := range lista {
		res, err := tx.ExecContext(ctx, `
			UPDATE pecas SET estoque_reservado = estoque_reservado + $1, atualizado_em = now()
			WHERE id = $2 AND estoque_atual - estoque_reservado >= $1`, i.quantidade, i.pecaID)
		if err != nil {
			return fmt.Errorf("EstoqueRepository.Reservar update: %w", err)
		}
		if n, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("EstoqueRepository.Reservar rows: %w", err)
		} else if n == 0 {
			return &domainerros.ErrNaoProcessavel{
				Codigo:   "INSUFFICIENT_STOCK",
				Mensagem: fmt.Sprintf("estoque insuficiente para peça '%s': necessário %d", i.nome, i.quantidade),
			}
		}
		if err := marcarItem(ctx, tx, i.itemID, "reservada"); err != nil {
			return fmt.Errorf("EstoqueRepository.Reservar marcar: %w", err)
		}
	}
	return nil
}

// Liberar desfaz a reserva (compensação de T1).
func (EstoqueRepository) Liberar(ctx context.Context, tx *sql.Tx, osID uuid.UUID) error {
	lista, err := itens(ctx, tx, osID, &estadoReservada)
	if err != nil {
		return fmt.Errorf("EstoqueRepository.Liberar itens: %w", err)
	}
	for _, i := range lista {
		if _, err := tx.ExecContext(ctx, `
			UPDATE pecas SET estoque_reservado = estoque_reservado - $1, atualizado_em = now()
			WHERE id = $2`, i.quantidade, i.pecaID); err != nil {
			return fmt.Errorf("EstoqueRepository.Liberar update: %w", err)
		}
		if err := marcarItem(ctx, tx, i.itemID, "liberada"); err != nil {
			return fmt.Errorf("EstoqueRepository.Liberar marcar: %w", err)
		}
	}
	return nil
}

// Baixar converte a reserva em saída (início do reparo, pivô T6):
// estoque_atual -= n e estoque_reservado -= n.
func (EstoqueRepository) Baixar(ctx context.Context, tx *sql.Tx, osID uuid.UUID) error {
	lista, err := itens(ctx, tx, osID, &estadoReservada)
	if err != nil {
		return fmt.Errorf("EstoqueRepository.Baixar itens: %w", err)
	}
	for _, i := range lista {
		if _, err := tx.ExecContext(ctx, `
			UPDATE pecas SET estoque_atual = estoque_atual - $1, estoque_reservado = estoque_reservado - $1,
			       atualizado_em = now()
			WHERE id = $2`, i.quantidade, i.pecaID); err != nil {
			return fmt.Errorf("EstoqueRepository.Baixar update: %w", err)
		}
		if err := marcarItem(ctx, tx, i.itemID, "baixada"); err != nil {
			return fmt.Errorf("EstoqueRepository.Baixar marcar: %w", err)
		}
	}
	return nil
}
