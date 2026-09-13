package observability

import (
	"context"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func TestMontarAtributos_CamposObrigatorios(t *testing.T) {
	attrs := montarAtributos(usecase.EventoOS{
		OsID:                  "os-1",
		Numero:                "OS-2026-00001",
		Status:                "recebida",
		Resultado:             usecase.ResultadoSucesso,
		DuracaoStatusSegundos: 12.5,
	}, "", "")

	if attrs["os_id"] != "os-1" || attrs["numero"] != "OS-2026-00001" {
		t.Errorf("identificadores não mapeados: %#v", attrs)
	}
	if attrs["status"] != "recebida" || attrs["resultado"] != usecase.ResultadoSucesso {
		t.Errorf("status/resultado não mapeados: %#v", attrs)
	}
	if attrs["duracao_status_segundos"] != 12.5 {
		t.Errorf("duracao = %v, esperava 12.5", attrs["duracao_status_segundos"])
	}
	// Campos opcionais ausentes não devem aparecer no evento.
	if _, ok := attrs["status_anterior"]; ok {
		t.Error("status_anterior não deveria estar presente quando vazio")
	}
	if _, ok := attrs["motivo"]; ok {
		t.Error("motivo não deveria estar presente quando vazio")
	}
	if _, ok := attrs["trace.id"]; ok {
		t.Error("trace.id não deveria estar presente sem trace ativo")
	}
}

func TestMontarAtributos_CamposOpcionaisETrace(t *testing.T) {
	attrs := montarAtributos(usecase.EventoOS{
		OsID:           "os-2",
		Numero:         "OS-2026-00002",
		Status:         "em_execucao",
		StatusAnterior: "aguardando_aprovacao",
		Resultado:      usecase.ResultadoFalha,
		Motivo:         "deducao_estoque",
	}, "trace-abc", "span-xyz")

	if attrs["status_anterior"] != "aguardando_aprovacao" {
		t.Errorf("status_anterior = %v", attrs["status_anterior"])
	}
	if attrs["motivo"] != "deducao_estoque" {
		t.Errorf("motivo = %v", attrs["motivo"])
	}
	if attrs["trace.id"] != "trace-abc" || attrs["span.id"] != "span-xyz" {
		t.Errorf("trace/span não mapeados: %#v", attrs)
	}
}

// Sem transação New Relic no contexto (ambiente local sem a licença), o
// recorder deve ser um no-op silencioso — nunca deve panicar.
func TestRegistrarEventoOS_SemTransacaoNoContexto(t *testing.T) {
	NovoEventRecorder().RegistrarEventoOS(context.Background(), usecase.EventoOS{
		OsID:      "os-1",
		Status:    "recebida",
		Resultado: usecase.ResultadoSucesso,
	})
}

// Com uma transação no contexto, o recorder percorre todo o caminho de emissão
// (Application + RecordCustomEvent). Usa um app desabilitado para não abrir
// conexão de rede: o RecordCustomEvent vira no-op interno, mas o código é exercido.
func TestRegistrarEventoOS_ComTransacao(t *testing.T) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("test"),
		newrelic.ConfigEnabled(false),
	)
	if err != nil {
		t.Fatalf("falha ao criar app de teste: %v", err)
	}
	txn := app.StartTransaction("teste")
	defer txn.End()

	ctx := newrelic.NewContext(context.Background(), txn)
	NovoEventRecorder().RegistrarEventoOS(ctx, usecase.EventoOS{
		OsID:           "os-3",
		Numero:         "OS-2026-00003",
		Status:         "finalizada",
		StatusAnterior: "em_execucao",
		Resultado:      usecase.ResultadoFalha,
		Motivo:         "atualizacao_status",
	})
}
