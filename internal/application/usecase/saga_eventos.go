package usecase

import (
	"context"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
)

// eventosDaDecisao traduz a decisão do orquestrador em eventos de
// observabilidade, um por etapa registrada. Função pura: o caso de uso só
// emite depois do commit.
func eventosDaDecisao(antes saga.Saga, dec saga.Decisao, tipo string, agora time.Time) []EventoSaga {
	depois := antes
	if dec.MudouSaga {
		depois = dec.Saga
	}
	terminou := dec.MudouSaga && !antes.Estado.Terminal() && depois.Estado.Terminal()
	duracaoEtapa := segundosDesde(antes.AtualizadoEm, agora)

	eventos := make([]EventoSaga, 0, len(dec.Etapas))
	for i, e := range dec.Etapas {
		ev := EventoSaga{
			SagaID:         antes.ID.String(),
			OsID:           antes.OsID.String(),
			EstadoAnterior: string(antes.Estado),
			Estado:         string(depois.Estado),
			Etapa:          string(e.Etapa),
			Resultado:      string(e.Resultado),
			TipoMensagem:   tipo,
		}
		switch e.Resultado {
		case saga.ResultadoFalha:
			ev.Motivo = e.Detalhe["motivo"]
		case saga.ResultadoIgnorada:
			ev.Motivo = "resposta_inesperada"
		}
		// O tempo de espera pertence à primeira etapa da transição; as
		// seguintes (ex.: T1 compensada logo após T2) acontecem no mesmo
		// instante.
		if i == 0 && e.Resultado != saga.ResultadoIgnorada {
			ev.DuracaoEtapa = duracaoEtapa
		}
		if terminou && i == len(dec.Etapas)-1 {
			ev.Terminou = true
			ev.DuracaoTotal = segundosDesde(antes.CriadoEm, agora)
		}
		eventos = append(eventos, ev)
	}
	return eventos
}

// eventoAbertura é o T1 da saga, emitido quando a OS é criada.
func eventoAbertura(s saga.Saga) EventoSaga {
	return EventoSaga{
		SagaID:    s.ID.String(),
		OsID:      s.OsID.String(),
		Estado:    string(s.Estado),
		Etapa:     string(saga.EtapaAbertura),
		Resultado: string(saga.ResultadoOK),
	}
}

// registrarEventosSaga emite os eventos quando o emissor sabe fazer isso.
// Observabilidade nunca afeta a saga: sem emissor, é no-op.
func (uc *OrdemServicoUseCase) registrarEventosSaga(ctx context.Context, eventos ...EventoSaga) {
	r, ok := uc.eventos.(sagaEventRecorder)
	if !ok {
		return
	}
	for _, e := range eventos {
		r.RegistrarEventoSaga(ctx, e)
	}
}

func segundosDesde(inicio, agora time.Time) float64 {
	if inicio.IsZero() || agora.Before(inicio) {
		return 0
	}
	return agora.Sub(inicio).Seconds()
}
