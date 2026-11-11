package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
)

// SagaRepository persiste o estado da saga (sagas_os) e o histórico de etapas
// (sagas_os_etapas). As escritas exigem a transação do contexto: estado da
// saga, efeito e mensagem de saída mudam juntos ou não mudam.
type SagaRepository struct {
	db *sql.DB
}

// NovoSagaRepository cria o repositório.
func NovoSagaRepository(db *sql.DB) *SagaRepository { return &SagaRepository{db: db} }

// Criar insere a saga de uma OS.
func (r *SagaRepository) Criar(ctx context.Context, s saga.Saga) error {
	tx, err := txObrigatoria(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sagas_os (id, os_id, estado, etapa_atual, versao, criado_em, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.OsID, string(s.Estado), string(s.EtapaAtual), s.Versao, s.CriadoEm, s.AtualizadoEm); err != nil {
		return fmt.Errorf("SagaRepository.Criar: %w", err)
	}
	return nil
}

// RegistrarEtapa grava uma linha no histórico de etapas da saga.
func (r *SagaRepository) RegistrarEtapa(ctx context.Context, sagaID uuid.UUID, etapa saga.Etapa,
	resultado saga.Resultado, messageID *uuid.UUID, detalhe any) error {
	tx, err := txObrigatoria(ctx)
	if err != nil {
		return err
	}
	var det []byte
	if detalhe != nil {
		if det, err = json.Marshal(detalhe); err != nil {
			return fmt.Errorf("SagaRepository.RegistrarEtapa detalhe: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sagas_os_etapas (id, saga_id, etapa, resultado, message_id, detalhe)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), sagaID, string(etapa), string(resultado), messageID, det); err != nil {
		return fmt.Errorf("SagaRepository.RegistrarEtapa: %w", err)
	}
	return nil
}

// BuscarPorOS devolve a saga de uma OS.
func (r *SagaRepository) BuscarPorOS(ctx context.Context, osID uuid.UUID) (*saga.Saga, error) {
	var (
		s                    saga.Saga
		estado, etapa        string
		etapaFalha           sql.NullString
		orcamento, pagamento uuid.NullUUID
		prazo                sql.NullTime
	)
	err := execDe(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, os_id, estado, etapa_atual, etapa_falha, orcamento_id, pagamento_id,
		       prazo_em, versao, criado_em, atualizado_em
		FROM sagas_os WHERE os_id = $1`, osID).
		Scan(&s.ID, &s.OsID, &estado, &etapa, &etapaFalha, &orcamento, &pagamento,
			&prazo, &s.Versao, &s.CriadoEm, &s.AtualizadoEm)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "saga"}
	}
	if err != nil {
		return nil, fmt.Errorf("SagaRepository.BuscarPorOS: %w", err)
	}
	s.Estado, s.EtapaAtual = saga.Estado(estado), saga.Etapa(etapa)
	if etapaFalha.Valid {
		e := saga.Etapa(etapaFalha.String)
		s.EtapaFalha = &e
	}
	if orcamento.Valid {
		s.OrcamentoID = &orcamento.UUID
	}
	if pagamento.Valid {
		s.PagamentoID = &pagamento.UUID
	}
	if prazo.Valid {
		s.PrazoEm = &prazo.Time
	}
	return &s, nil
}

// EtapaRegistrada é uma linha de sagas_os_etapas (consulta da saga, F4-1.4).
type EtapaRegistrada struct {
	Etapa     saga.Etapa
	Resultado saga.Resultado
}

// Etapas devolve o histórico de etapas da saga, em ordem.
func (r *SagaRepository) Etapas(ctx context.Context, sagaID uuid.UUID) ([]EtapaRegistrada, error) {
	rows, err := execDe(ctx, r.db).QueryContext(ctx,
		`SELECT etapa, resultado FROM sagas_os_etapas WHERE saga_id = $1 ORDER BY ocorrido_em, id`, sagaID)
	if err != nil {
		return nil, fmt.Errorf("SagaRepository.Etapas: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	var out []EtapaRegistrada
	for rows.Next() {
		var etapa, resultado string
		if err := rows.Scan(&etapa, &resultado); err != nil {
			return nil, err
		}
		out = append(out, EtapaRegistrada{Etapa: saga.Etapa(etapa), Resultado: saga.Resultado(resultado)})
	}
	return out, rows.Err()
}
