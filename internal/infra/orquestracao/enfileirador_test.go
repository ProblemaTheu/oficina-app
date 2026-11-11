package orquestracao

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/logging"
	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/infra/mensageria"
	"github.com/google/uuid"
)

type outboxFake struct {
	destino string
	env     mensageria.Envelope
	trace   map[string]string
	err     error
}

func (o *outboxFake) Gravar(_ context.Context, destino string, env mensageria.Envelope, trace map[string]string) error {
	o.destino, o.env, o.trace = destino, env, trace
	return o.err
}

type tracerFake struct{}

func (tracerFake) Capturar(context.Context) map[string]string {
	return map[string]string{"traceparent": "00-a-b-01"}
}
func (tracerFake) IniciarConsumo(ctx context.Context, _ string, _ map[string]string) (context.Context, func(error)) {
	return ctx, func(error) {}
}

func TestEnfileirar_MontaEnvelopeDoComando(t *testing.T) {
	o := &outboxFake{}
	e := NovoEnfileirador(o, tracerFake{})
	e.agora = func() time.Time { return time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC) }
	s := saga.Nova(uuid.New(), time.Now())
	ctx := logging.ComCorrelacao(context.Background(), "req-42")

	err := e.Enfileirar(ctx, s.ComandoIniciarDiagnostico(saga.PayloadIniciarDiagnostico{
		NumeroOS: "OS-2026-00001", Veiculo: saga.Veiculo{Placa: "ABC1D23", Modelo: "Gol"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if o.destino != saga.DestinoExecution {
		t.Errorf("destino = %s", o.destino)
	}
	env := o.env
	if env.Tipo != saga.TipoIniciarDiagnostico || env.SagaID != s.ID || env.OsID != s.OsID ||
		env.Origem != mensageria.OrigemOS || env.CorrelationID != "req-42" {
		t.Errorf("envelope errado: %+v", env)
	}
	var p saga.PayloadIniciarDiagnostico
	if err := env.LerPayload(&p); err != nil || p.Veiculo.Placa != "ABC1D23" {
		t.Errorf("payload = %+v (%v)", p, err)
	}
	if o.trace["traceparent"] == "" {
		t.Error("trace não capturado na gravação")
	}
}

func TestEnfileirar_PropagaErros(t *testing.T) {
	falha := errors.New("sem transação")
	e := NovoEnfileirador(&outboxFake{err: falha}, nil)
	s := saga.Nova(uuid.New(), time.Now())
	if err := e.Enfileirar(context.Background(), s.ComandoIniciarDiagnostico(saga.PayloadIniciarDiagnostico{})); !errors.Is(err, falha) {
		t.Errorf("erro da outbox não propagado: %v", err)
	}

	invalido := saga.Comando{Tipo: "", SagaID: s.ID, OsID: s.OsID}
	if err := e.Enfileirar(context.Background(), invalido); err == nil {
		t.Error("comando sem tipo deve falhar antes de gravar")
	}
}
