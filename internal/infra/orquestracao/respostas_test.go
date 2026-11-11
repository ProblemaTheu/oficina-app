package orquestracao

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/infra/mensageria"
)

type processadorFake struct {
	recebida usecase.RespostaSaga
	err      error
}

func (p *processadorFake) ProcessarRespostaSaga(_ context.Context, r usecase.RespostaSaga) error {
	p.recebida = r
	return p.err
}

func TestHandlersDeResposta_CobremTodoOCatalogoERepassam(t *testing.T) {
	p := &processadorFake{}
	hs := HandlersDeResposta(p)
	if len(hs) != len(TiposDeResposta) || len(hs) != 15 {
		t.Fatalf("handlers = %d; o catálogo tem 15 respostas", len(hs))
	}
	env, _ := mensageria.Novo(saga.TipoDiagnosticoConcluido, uuid.New(), uuid.New(), "", mensageria.OrigemExecution,
		map[string]string{"diagnostico": "x"}, time.Now())

	if err := hs[saga.TipoDiagnosticoConcluido](context.Background(), env); err != nil {
		t.Fatal(err)
	}
	r := p.recebida
	if r.MessageID != env.MessageID || r.Tipo != env.Tipo || r.SagaID != env.SagaID || r.OsID != env.OsID || string(r.Payload) != string(env.Payload) {
		t.Errorf("resposta repassada errada: %+v", r)
	}
}

func TestHandlersDeResposta_ClassificaErros(t *testing.T) {
	env, _ := mensageria.Novo(saga.TipoOrcamentoGerado, uuid.New(), uuid.New(), "", mensageria.OrigemBilling, nil, time.Now())
	semSaga := HandlersDeResposta(&processadorFake{err: usecase.ErrRespostaSemSaga})[saga.TipoOrcamentoGerado]
	if err := semSaga(context.Background(), env); !mensageria.EhPermanente(err) {
		t.Errorf("resposta sem saga deve ser descartada (permanente): %v", err)
	}
	infra := errors.New("lock otimista")
	transitorio := HandlersDeResposta(&processadorFake{err: infra})[saga.TipoOrcamentoGerado]
	if err := transitorio(context.Background(), env); !errors.Is(err, infra) || mensageria.EhPermanente(err) {
		t.Errorf("erro transitório deve voltar para a fila: %v", err)
	}
}
