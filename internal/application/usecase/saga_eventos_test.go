package usecase_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
)

// fakeEventRecorderSaga captura também os SagaEvent (porta opcional).
type fakeEventRecorderSaga struct {
	fakeEventRecorder
	sagas []usecase.EventoSaga
}

func (f *fakeEventRecorderSaga) RegistrarEventoSaga(_ context.Context, e usecase.EventoSaga) {
	f.sagas = append(f.sagas, e)
}

func perto(a, b float64) bool { return math.Abs(a-b) < 5 }

// orquestradorComEventos monta o cenário com a última transição há 30 s e a
// saga aberta há 10 min, para conferir as durações.
func orquestradorComEventos(estado saga.Estado, statusOS entity.Status) (cenarioOrquestrador, *fakeEventRecorderSaga) {
	rec := &fakeEventRecorderSaga{}
	c := novoOrquestrador(estado, statusOS, nil, usecase.ComEventRecorder(rec))
	c.sagas.criada.CriadoEm = time.Now().Add(-10 * time.Minute)
	c.sagas.criada.AtualizadoEm = time.Now().Add(-30 * time.Second)
	return c, rec
}

func TestSagaEvent_PassoDoCaminhoFeliz(t *testing.T) {
	c, rec := orquestradorComEventos(saga.DiagnosticoPendente, entity.StatusEmDiagnostico)

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, map[string]string{"diagnostico": "x"})); err != nil {
		t.Fatal(err)
	}
	if len(rec.sagas) != 1 {
		t.Fatalf("eventos = %+v", rec.sagas)
	}
	e := rec.sagas[0]
	if e.SagaID != c.sagas.criada.ID.String() || e.OsID != c.os.os.ID.String() ||
		e.EstadoAnterior != "DIAGNOSTICO_PENDENTE" || e.Estado != "ORCAMENTO_PENDENTE" ||
		e.Etapa != "T2" || e.Resultado != "ok" || e.TipoMensagem != saga.TipoDiagnosticoConcluido {
		t.Errorf("evento = %+v", e)
	}
	if !perto(e.DuracaoEtapa, 30) || e.Terminou || e.DuracaoTotal != 0 {
		t.Errorf("duração etapa %.1f, terminou %v, total %.1f", e.DuracaoEtapa, e.Terminou, e.DuracaoTotal)
	}
}

func TestSagaEvent_FalhaIniciaCompensacaoComMotivo(t *testing.T) {
	c, rec := orquestradorComEventos(saga.ExecucaoPendente, entity.StatusEmExecucao)

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoExecucaoFalhou, map[string]string{"motivo": "falha_simulada"})); err != nil {
		t.Fatal(err)
	}
	e := rec.sagas[0]
	if e.Resultado != "falha" || e.Estado != "COMPENSANDO" || e.Motivo != "execucao_falhou: falha_simulada" || e.Terminou {
		t.Errorf("evento = %+v", e)
	}
}

// Última compensação: duas etapas na mesma transição (T2 e T1 compensadas).
// Só a última marca terminou; a espera fica só na primeira.
func TestSagaEvent_FimDaCompensacaoMarcaTerminouUmaVez(t *testing.T) {
	c, rec := orquestradorComEventos(saga.Compensando, entity.StatusCancelada)
	f := saga.EtapaOrcamento
	c.sagas.criada.EtapaFalha, c.sagas.criada.EtapaAtual = &f, saga.EtapaDiagnostico

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoTarefasCanceladas, nil)); err != nil {
		t.Fatal(err)
	}
	if len(rec.sagas) != 2 {
		t.Fatalf("eventos = %+v", rec.sagas)
	}
	primeira, ultima := rec.sagas[0], rec.sagas[1]
	if primeira.Etapa != "T2" || primeira.Resultado != "compensada" || primeira.Terminou || !perto(primeira.DuracaoEtapa, 30) {
		t.Errorf("1º evento = %+v", primeira)
	}
	if ultima.Etapa != "T1" || ultima.Estado != "COMPENSADA" || !ultima.Terminou || ultima.DuracaoEtapa != 0 || !perto(ultima.DuracaoTotal, 600) {
		t.Errorf("último evento = %+v", ultima)
	}
}

func TestSagaEvent_ConclusaoTerminaComDuracaoTotal(t *testing.T) {
	c, rec := orquestradorComEventos(saga.EmExecucao, entity.StatusEmExecucao)

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoExecucaoConcluida, nil)); err != nil {
		t.Fatal(err)
	}
	e := rec.sagas[len(rec.sagas)-1]
	if e.Estado != "CONCLUIDA" || !e.Terminou || !perto(e.DuracaoTotal, 600) {
		t.Errorf("evento = %+v", e)
	}
}

// Resposta atrasada numa saga já terminal: evento "ignorada", sem terminou
// (não conta como desfecho de novo) e sem duração.
func TestSagaEvent_IgnoradaNaoContaComoDesfecho(t *testing.T) {
	c, rec := orquestradorComEventos(saga.Concluida, entity.StatusFinalizada)

	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoOrcamentoGerado, nil)); err != nil {
		t.Fatal(err)
	}
	e := rec.sagas[0]
	if e.Resultado != "ignorada" || e.Motivo != "resposta_inesperada" || e.Terminou || e.DuracaoEtapa != 0 || e.Estado != "CONCLUIDA" {
		t.Errorf("evento = %+v", e)
	}
}

func TestSagaEvent_DuplicataNaoEmite(t *testing.T) {
	c, rec := orquestradorComEventos(saga.DiagnosticoPendente, entity.StatusEmDiagnostico)
	r := c.resposta(saga.TipoDiagnosticoConcluido, map[string]string{"diagnostico": "x"})
	_ = c.uc.ProcessarRespostaSaga(context.Background(), r)
	_ = c.uc.ProcessarRespostaSaga(context.Background(), r)
	if len(rec.sagas) != 1 {
		t.Errorf("duplicata emitiu evento: %d eventos", len(rec.sagas))
	}
}

// Emissor sem a capacidade de saga (só EventoOS): a saga segue normalmente.
func TestSagaEvent_EmissorSemSuporteENaoQuebra(t *testing.T) {
	c := novoOrquestrador(saga.DiagnosticoPendente, entity.StatusEmDiagnostico, nil, usecase.ComEventRecorder(&fakeEventRecorder{}))
	if err := c.uc.ProcessarRespostaSaga(context.Background(), c.resposta(saga.TipoDiagnosticoConcluido, nil)); err != nil {
		t.Fatal(err)
	}
	if c.sagas.criada.Estado != saga.OrcamentoPendente {
		t.Errorf("saga = %s", c.sagas.criada.Estado)
	}
}

func TestSagaEvent_AberturaDaOS(t *testing.T) {
	c := novoCenarioSaga(&estoqueSagaFake{}, &filaComandosFake{})
	if _, err := c.uc.CriarOS(context.Background(), c.input); err != nil {
		t.Fatal(err)
	}
	if len(c.eventos.sagas) != 1 {
		t.Fatalf("eventos = %+v", c.eventos.sagas)
	}
	e := c.eventos.sagas[0]
	if e.SagaID != c.sagas.criada.ID.String() || e.Etapa != "T1" || e.Resultado != "ok" || e.Estado != "DIAGNOSTICO_PENDENTE" || e.EstadoAnterior != "" {
		t.Errorf("evento = %+v", e)
	}
}

// Falha na abertura: nenhuma saga persistida, nenhum SagaEvent.
func TestSagaEvent_AberturaQueFalhaNaoEmite(t *testing.T) {
	c := novoCenarioSaga(&estoqueSagaFake{}, &filaComandosFake{err: errors.New("outbox fora")})
	_, _ = c.uc.CriarOS(context.Background(), c.input)
	if len(c.eventos.sagas) != 0 {
		t.Errorf("eventos = %+v", c.eventos.sagas)
	}
}
