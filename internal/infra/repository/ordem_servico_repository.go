package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/google/uuid"
)

// OrdemServicoRepository implementa o acesso ao banco de dados para Ordens de Serviço.
type OrdemServicoRepository struct {
	db *sql.DB

	// cache de nome → UUID para status_ordens
	statusCacheMu sync.RWMutex
	statusCache   map[entity.Status]uuid.UUID
}

// NovoOrdemServicoRepository cria uma nova instância do repositório.
func NovoOrdemServicoRepository(db *sql.DB) *OrdemServicoRepository {
	return &OrdemServicoRepository{
		db:          db,
		statusCache: make(map[entity.Status]uuid.UUID),
	}
}

// BuscarStatusID retorna o UUID do status a partir do nome, usando cache local.
func (r *OrdemServicoRepository) BuscarStatusID(ctx context.Context, nome entity.Status) (uuid.UUID, error) {
	r.statusCacheMu.RLock()
	if id, ok := r.statusCache[nome]; ok {
		r.statusCacheMu.RUnlock()
		return id, nil
	}
	r.statusCacheMu.RUnlock()

	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM status_ordens WHERE nome_status = $1`, string(nome),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("status '%s' não encontrado: %w", nome, err)
	}

	r.statusCacheMu.Lock()
	r.statusCache[nome] = id
	r.statusCacheMu.Unlock()

	return id, nil
}

// PrecarregarStatusCache carrega todos os status do banco na inicialização.
func (r *OrdemServicoRepository) PrecarregarStatusCache(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `SELECT id, nome_status FROM status_ordens`)
	if err != nil {
		return fmt.Errorf("PrecarregarStatusCache: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	r.statusCacheMu.Lock()
	defer r.statusCacheMu.Unlock()

	for rows.Next() {
		var id uuid.UUID
		var nome string
		if err := rows.Scan(&id, &nome); err != nil {
			return err
		}
		r.statusCache[entity.Status(nome)] = id
	}
	return rows.Err()
}

// GerarNumeroOS gera o próximo número sequencial para OS no formato OS-YYYY-NNNNN.
func (r *OrdemServicoRepository) GerarNumeroOS(ctx context.Context) (string, error) {
	var seq int64
	err := r.db.QueryRowContext(ctx, `SELECT nextval('os_numero_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("GerarNumeroOS: %w", err)
	}
	ano := time.Now().Year()
	return fmt.Sprintf("OS-%d-%05d", ano, seq), nil
}

// Criar persiste uma nova OS e seus itens dentro de uma transação.
func (r *OrdemServicoRepository) Criar(
	ctx context.Context,
	os *entity.OrdemServico,
	itensServico []entity.ItemOsServico,
	itensPeca []entity.ItemOsPeca,
) (*entity.OrdemServico, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("OrdemServicoRepository.Criar begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	os.ID = uuid.New()
	os.CriadoEm = time.Now()
	os.AtualizadoEm = time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO ordens_servico
			(id, numero, cliente_id, veiculo_id, status_id, descricao, valor_total, criado_em, atualizado_em)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`,
		os.ID, os.Numero, os.ClienteID, os.VeiculoID, os.StatusID,
		os.Descricao, os.ValorTotal, os.CriadoEm, os.AtualizadoEm,
	)
	if err != nil {
		return nil, fmt.Errorf("OrdemServicoRepository.Criar insert OS: %w", err)
	}

	for i := range itensServico {
		itensServico[i].ID = uuid.New()
		itensServico[i].OsID = os.ID
		itensServico[i].CriadoEm = time.Now()
		itensServico[i].AtualizadoEm = time.Now()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO itens_os_servicos (id, os_id, servico_id, quantidade, preco_unitario, criado_em, atualizado_em)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`,
			itensServico[i].ID, os.ID, itensServico[i].ServicoID,
			itensServico[i].Quantidade, itensServico[i].PrecoUnitario,
			itensServico[i].CriadoEm, itensServico[i].AtualizadoEm,
		)
		if err != nil {
			return nil, fmt.Errorf("OrdemServicoRepository.Criar insert item_servico: %w", err)
		}
	}

	for i := range itensPeca {
		itensPeca[i].ID = uuid.New()
		itensPeca[i].OsID = os.ID
		itensPeca[i].CriadoEm = time.Now()
		itensPeca[i].AtualizadoEm = time.Now()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO itens_os_pecas (id, os_id, peca_id, quantidade, preco_unitario, criado_em, atualizado_em)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`,
			itensPeca[i].ID, os.ID, itensPeca[i].PecaID,
			itensPeca[i].Quantidade, itensPeca[i].PrecoUnitario,
			itensPeca[i].CriadoEm, itensPeca[i].AtualizadoEm,
		)
		if err != nil {
			return nil, fmt.Errorf("OrdemServicoRepository.Criar insert item_peca: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("OrdemServicoRepository.Criar commit: %w", err)
	}

	return os, nil
}

// BuscarPorID retorna a OS completa com itens, cliente e veículo.
func (r *OrdemServicoRepository) BuscarPorID(ctx context.Context, id string) (*entity.OrdemServicoCompleta, error) {
	os := &entity.OrdemServico{}

	err := r.db.QueryRowContext(ctx, `
		SELECT
			o.id, o.numero, o.cliente_id, o.veiculo_id, o.status_id, s.nome_status,
			o.descricao, o.diagnostico, o.valor_total,
			o.aprovado_em, o.reprovado_em, o.iniciado_em, o.finalizado_em, o.entregue_em,
			o.criado_em, o.atualizado_em
		FROM ordens_servico o
		JOIN status_ordens s ON s.id = o.status_id
		WHERE o.id = $1
	`, id).Scan(
		&os.ID, &os.Numero, &os.ClienteID, &os.VeiculoID, &os.StatusID, &os.StatusNome,
		&os.Descricao, &os.Diagnostico, &os.ValorTotal,
		&os.AprovadoEm, &os.ReprovadoEm, &os.IniciadoEm, &os.FinalizadoEm, &os.EntregueEm,
		&os.CriadoEm, &os.AtualizadoEm,
	)
	if err == sql.ErrNoRows {
		return nil, &domainerros.ErrNaoEncontrado{Recurso: "ordem de serviço"}
	}
	if err != nil {
		return nil, fmt.Errorf("OrdemServicoRepository.BuscarPorID: %w", err)
	}

	completa := &entity.OrdemServicoCompleta{OrdemServico: *os}

	// Buscar cliente resumo
	_ = r.db.QueryRowContext(ctx,
		`SELECT id, nome, cpf_cnpj FROM clientes WHERE id = $1`, os.ClienteID,
	).Scan(&completa.Cliente.ID, &completa.Cliente.Nome, &completa.Cliente.CpfCnpj)

	// Buscar veículo resumo
	_ = r.db.QueryRowContext(ctx,
		`SELECT id, placa, marca, modelo, ano FROM veiculos WHERE id = $1`, os.VeiculoID,
	).Scan(&completa.Veiculo.ID, &completa.Veiculo.Placa, &completa.Veiculo.Marca,
		&completa.Veiculo.Modelo, &completa.Veiculo.Ano)

	// Buscar itens de serviço
	rowsS, err := r.db.QueryContext(ctx, `
		SELECT ios.id, ios.servico_id, s.nome, ios.quantidade, ios.preco_unitario
		FROM itens_os_servicos ios
		JOIN servicos s ON s.id = ios.servico_id
		WHERE ios.os_id = $1
	`, os.ID)
	if err != nil {
		return nil, fmt.Errorf("OrdemServicoRepository.BuscarPorID itens_servico: %w", err)
	}
	defer rowsS.Close() //nolint:errcheck

	for rowsS.Next() {
		var item entity.ItemOS
		var serviçoID uuid.UUID
		if err := rowsS.Scan(&item.ID, &serviçoID, &item.Nome, &item.Quantidade, &item.PrecoUnitario); err != nil {
			return nil, err
		}
		item.OsID = os.ID
		item.Tipo = "servico"
		item.ReferenciaID = serviçoID
		item.Subtotal = float64(item.Quantidade) * item.PrecoUnitario
		completa.Itens = append(completa.Itens, item)
	}
	if err := rowsS.Err(); err != nil {
		return nil, err
	}

	// Buscar itens de peça
	rowsP, err := r.db.QueryContext(ctx, `
		SELECT iop.id, iop.peca_id, p.nome, iop.quantidade, iop.preco_unitario
		FROM itens_os_pecas iop
		JOIN pecas p ON p.id = iop.peca_id
		WHERE iop.os_id = $1
	`, os.ID)
	if err != nil {
		return nil, fmt.Errorf("OrdemServicoRepository.BuscarPorID itens_peca: %w", err)
	}
	defer rowsP.Close() //nolint:errcheck

	for rowsP.Next() {
		var item entity.ItemOS
		var pecaID uuid.UUID
		if err := rowsP.Scan(&item.ID, &pecaID, &item.Nome, &item.Quantidade, &item.PrecoUnitario); err != nil {
			return nil, err
		}
		item.OsID = os.ID
		item.Tipo = "peca"
		item.ReferenciaID = pecaID
		item.Subtotal = float64(item.Quantidade) * item.PrecoUnitario
		completa.Itens = append(completa.Itens, item)
	}
	if err := rowsP.Err(); err != nil {
		return nil, err
	}

	if completa.Itens == nil {
		completa.Itens = []entity.ItemOS{}
	}

	return completa, nil
}

// Listar retorna OSs paginadas com filtros opcionais.
func (r *OrdemServicoRepository) Listar(ctx context.Context, params usecase.ListarOSParams) ([]*entity.OrdemServico, int, error) {
	query := `
		SELECT
			o.id, o.numero, o.cliente_id, o.veiculo_id, o.status_id, s.nome_status,
			o.descricao, o.diagnostico, o.valor_total,
			o.aprovado_em, o.reprovado_em, o.iniciado_em, o.finalizado_em, o.entregue_em,
			o.criado_em, o.atualizado_em
		FROM ordens_servico o
		JOIN status_ordens s ON s.id = o.status_id
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM ordens_servico o JOIN status_ordens s ON s.id = o.status_id WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if params.Status != nil {
		cond := fmt.Sprintf(" AND s.nome_status = $%d", argIdx)
		query += cond
		countQuery += cond
		args = append(args, string(*params.Status))
		argIdx++
	} else if !params.IncluirEncerradas {
		// Exclusão lógica: OSs encerradas não aparecem na listagem padrão.
		cond := " AND s.nome_status NOT IN ('finalizada', 'entregue')"
		query += cond
		countQuery += cond
	}
	if params.ClienteID != nil {
		cond := fmt.Sprintf(" AND o.cliente_id = $%d", argIdx)
		query += cond
		countQuery += cond
		args = append(args, *params.ClienteID)
		argIdx++
	}
	if params.VeiculoID != nil {
		cond := fmt.Sprintf(" AND o.veiculo_id = $%d", argIdx)
		query += cond
		countQuery += cond
		args = append(args, *params.VeiculoID)
		argIdx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("OrdemServicoRepository.Listar count: %w", err)
	}

	// Prioridade de atendimento do enunciado; mais antigas primeiro em cada grupo.
	query += fmt.Sprintf(`
		ORDER BY CASE s.nome_status
			WHEN 'em_execucao'          THEN 1
			WHEN 'aguardando_aprovacao' THEN 2
			WHEN 'em_diagnostico'       THEN 3
			WHEN 'recebida'             THEN 4
			ELSE 5
		END, o.criado_em ASC
		LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	offset := (params.Page - 1) * params.Limit
	args = append(args, params.Limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("OrdemServicoRepository.Listar: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var oss []*entity.OrdemServico
	for rows.Next() {
		os := &entity.OrdemServico{}
		if err := rows.Scan(
			&os.ID, &os.Numero, &os.ClienteID, &os.VeiculoID, &os.StatusID, &os.StatusNome,
			&os.Descricao, &os.Diagnostico, &os.ValorTotal,
			&os.AprovadoEm, &os.ReprovadoEm, &os.IniciadoEm, &os.FinalizadoEm, &os.EntregueEm,
			&os.CriadoEm, &os.AtualizadoEm,
		); err != nil {
			return nil, 0, fmt.Errorf("OrdemServicoRepository.Listar scan: %w", err)
		}
		oss = append(oss, os)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return oss, total, nil
}

// AtualizarStatus atualiza o status da OS e os timestamps relacionados.
func (r *OrdemServicoRepository) AtualizarStatus(
	ctx context.Context,
	osID uuid.UUID,
	novoStatusID uuid.UUID,
	diagnostico *string,
	aprovadoEm *time.Time,
	reprovadoEm *time.Time,
	iniciadoEm *time.Time,
	finalizadoEm *time.Time,
	entregueEm *time.Time,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ordens_servico SET
			status_id    = $1,
			diagnostico  = COALESCE($2, diagnostico),
			aprovado_em  = COALESCE($3, aprovado_em),
			reprovado_em = COALESCE($4, reprovado_em),
			iniciado_em  = COALESCE($5, iniciado_em),
			finalizado_em = COALESCE($6, finalizado_em),
			entregue_em  = COALESCE($7, entregue_em),
			atualizado_em = now()
		WHERE id = $8
	`,
		novoStatusID,
		diagnostico,
		aprovadoEm,
		reprovadoEm,
		iniciadoEm,
		finalizadoEm,
		entregueEm,
		osID,
	)
	if err != nil {
		return fmt.Errorf("OrdemServicoRepository.AtualizarStatus: %w", err)
	}
	return nil
}

// RegistrarHistorico insere um registro na tabela historicos_status.
func (r *OrdemServicoRepository) RegistrarHistorico(
	ctx context.Context,
	osID uuid.UUID,
	statusAnteriorID *uuid.UUID,
	statusNovoID uuid.UUID,
	observacao *string,
) error {
	id := uuid.New()
	now := time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO historicos_status
			(id, os_id, status_anterior_id, status_novo_id, observacao, alterado_em, criado_em, atualizado_em)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, id, osID, statusAnteriorID, statusNovoID, observacao, now, now, now)
	if err != nil {
		return fmt.Errorf("OrdemServicoRepository.RegistrarHistorico: %w", err)
	}
	return nil
}

// DeduzirEstoquePecas decrementa o estoque das peças dentro de uma transação.
func (r *OrdemServicoRepository) DeduzirEstoquePecas(ctx context.Context, osID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("DeduzirEstoquePecas begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.QueryContext(ctx, `
		SELECT iop.peca_id, iop.quantidade, p.estoque_atual, p.nome
		FROM itens_os_pecas iop
		JOIN pecas p ON p.id = iop.peca_id
		WHERE iop.os_id = $1
		FOR UPDATE OF p
	`, osID)
	if err != nil {
		return fmt.Errorf("DeduzirEstoquePecas query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	type linhaEstoque struct {
		pecaID       uuid.UUID
		quantidade   int
		estoqueAtual int
		nome         string
	}
	var linhas []linhaEstoque

	for rows.Next() {
		var l linhaEstoque
		if err := rows.Scan(&l.pecaID, &l.quantidade, &l.estoqueAtual, &l.nome); err != nil {
			return err
		}
		linhas = append(linhas, l)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, l := range linhas {
		if l.estoqueAtual < l.quantidade {
			return &domainerros.ErrNaoProcessavel{
				Codigo:   "INSUFFICIENT_STOCK",
				Mensagem: fmt.Sprintf("estoque insuficiente para peça '%s': disponível %d, necessário %d", l.nome, l.estoqueAtual, l.quantidade),
			}
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE pecas SET estoque_atual = estoque_atual - $1, atualizado_em = now()
			WHERE id = $2
		`, l.quantidade, l.pecaID)
		if err != nil {
			return fmt.Errorf("DeduzirEstoquePecas update: %w", err)
		}
	}

	return tx.Commit()
}

// RelatorioTempoMedio calcula o tempo médio de execução agrupado por serviço.
func (r *OrdemServicoRepository) RelatorioTempoMedio(ctx context.Context, params usecase.RelatorioTempoMedioParams) ([]usecase.ItemTempoMedio, error) {
	query := `
		SELECT
			s.id,
			s.nome,
			COUNT(DISTINCT o.id),
			AVG(EXTRACT(EPOCH FROM (o.finalizado_em - o.iniciado_em)) / 60)
		FROM ordens_servico o
		JOIN status_ordens so ON so.id = o.status_id
		JOIN itens_os_servicos ios ON ios.os_id = o.id
		JOIN servicos s ON s.id = ios.servico_id
		WHERE o.iniciado_em IS NOT NULL
		  AND o.finalizado_em IS NOT NULL
		  AND o.reprovado_em IS NULL
		  AND so.nome_status IN ('finalizada', 'entregue')
	`
	args := []interface{}{}
	argIdx := 1

	if params.ServicoID != nil {
		query += fmt.Sprintf(" AND s.id = $%d", argIdx)
		args = append(args, *params.ServicoID)
		argIdx++
	}
	if params.DataInicio != nil {
		query += fmt.Sprintf(" AND o.criado_em >= $%d", argIdx)
		args = append(args, *params.DataInicio)
		argIdx++
	}
	if params.DataFim != nil {
		query += fmt.Sprintf(" AND o.criado_em <= $%d", argIdx)
		args = append(args, *params.DataFim)
	}

	query += " GROUP BY s.id, s.nome ORDER BY s.nome"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("RelatorioTempoMedio: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var resultado []usecase.ItemTempoMedio
	for rows.Next() {
		var item usecase.ItemTempoMedio
		if err := rows.Scan(&item.ServicoID, &item.ServicoNome, &item.TotalExecucoes, &item.TempoMedioMinutos); err != nil {
			return nil, fmt.Errorf("RelatorioTempoMedio scan: %w", err)
		}
		resultado = append(resultado, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if resultado == nil {
		resultado = []usecase.ItemTempoMedio{}
	}

	return resultado, nil
}
