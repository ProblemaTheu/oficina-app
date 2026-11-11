package orquestracao

import (
	"context"
	"errors"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/infra/mensageria"
)

// processador é o que os handlers usam do caso de uso de OS.
type processador interface {
	ProcessarRespostaSaga(ctx context.Context, r usecase.RespostaSaga) error
}

// TiposDeResposta são as mensagens que o orquestrador consome em
// os-saga-replies.fifo (arquitetura §5.2).
var TiposDeResposta = []string{
	saga.TipoDiagnosticoIniciado, saga.TipoDiagnosticoConcluido, saga.TipoDiagnosticoFalhou,
	saga.TipoOrcamentoGerado, saga.TipoOrcamentoFalhou, saga.TipoOrcamentoAprovado, saga.TipoOrcamentoRecusado,
	saga.TipoPagamentoConfirmado, saga.TipoPagamentoRecusado,
	saga.TipoExecucaoIniciada, saga.TipoExecucaoFalhou, saga.TipoExecucaoConcluida,
	saga.TipoPagamentoEstornado, saga.TipoOrcamentoCancelado, saga.TipoTarefasCanceladas,
}

// HandlersDeResposta liga cada tipo de resposta ao orquestrador.
//
// Resposta sem saga correspondente é erro permanente (descartada com log —
// reentregar não cria a saga); qualquer outro erro, inclusive lock otimista,
// volta para a fila e é reprocessado.
func HandlersDeResposta(p processador) map[string]mensageria.Handler {
	h := func(ctx context.Context, env mensageria.Envelope) error {
		err := p.ProcessarRespostaSaga(ctx, usecase.RespostaSaga{
			MessageID: env.MessageID, Tipo: env.Tipo, SagaID: env.SagaID, OsID: env.OsID, Payload: env.Payload,
		})
		if errors.Is(err, usecase.ErrRespostaSemSaga) {
			return mensageria.Permanente(err)
		}
		return err
	}
	out := make(map[string]mensageria.Handler, len(TiposDeResposta))
	for _, t := range TiposDeResposta {
		out[t] = h
	}
	return out
}
