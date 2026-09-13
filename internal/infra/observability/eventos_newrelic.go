// Package observability implementa as portas de observabilidade da camada de
// aplicação sobre o New Relic — em especial os eventos de negócio de Ordem de
// Serviço (usecase.EventoOS), que alimentam os dashboards (volume diário de OS,
// tempo médio por status) e o alerta de falha no processamento de OS.
package observability

import (
	"context"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// TipoEventoOS é o eventType do custom event no New Relic. As consultas NRQL dos
// dashboards e do alerta referenciam exatamente este nome.
const TipoEventoOS = "OrdemServicoEvent"

// NewRelicEventRecorder emite eventos de negócio como custom events do New Relic.
// Satisfaz a porta usecase.eventRecorder por composição estrutural.
type NewRelicEventRecorder struct{}

// NovoEventRecorder cria o emissor de eventos do New Relic.
func NovoEventRecorder() *NewRelicEventRecorder {
	return &NewRelicEventRecorder{}
}

// RegistrarEventoOS publica um OrdemServicoEvent no New Relic. É um no-op quando
// não há transação New Relic no contexto (ex.: New Relic desabilitado no
// ambiente local), de modo que o caso de uso nunca é afetado pela observabilidade.
//
// O evento não carrega nenhum dado sensível (CPF, e-mail): apenas identificadores
// da OS e metadados de status, mais o trace.id/span.id para correlacionar o
// gráfico com os traces e logs da mesma requisição.
func (NewRelicEventRecorder) RegistrarEventoOS(ctx context.Context, e usecase.EventoOS) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return
	}
	app := txn.Application()
	if app == nil {
		return
	}

	md := txn.GetTraceMetadata()
	app.RecordCustomEvent(TipoEventoOS, montarAtributos(e, md.TraceID, md.SpanID))
}

// montarAtributos converte um EventoOS no mapa de atributos do custom event.
// Campos opcionais (status_anterior, motivo) só entram quando preenchidos;
// trace.id/span.id só entram quando há trace ativo. É uma função pura para
// manter a lógica de composição testável independente do agente New Relic.
func montarAtributos(e usecase.EventoOS, traceID, spanID string) map[string]interface{} {
	atributos := map[string]interface{}{
		"os_id":                   e.OsID,
		"numero":                  e.Numero,
		"status":                  e.Status,
		"resultado":               e.Resultado,
		"duracao_status_segundos": e.DuracaoStatusSegundos,
	}
	if e.StatusAnterior != "" {
		atributos["status_anterior"] = e.StatusAnterior
	}
	if e.Motivo != "" {
		atributos["motivo"] = e.Motivo
	}
	if traceID != "" {
		atributos["trace.id"] = traceID
		atributos["span.id"] = spanID
	}
	return atributos
}
