// Package orquestracao liga a saga da OS (internal/application/saga) à
// mensageria: transforma comandos da saga em envelopes e os grava na outbox.
// É específico do OS Service — o pacote mensageria continua genérico para ser
// copiado nos outros serviços.
package orquestracao

import (
	"context"
	"fmt"
	"time"

	"github.com/ProblemaTheu/oficina-app/internal/application/logging"
	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/infra/mensageria"
)

// gravadorOutbox é a parte da outbox usada aqui (repository.OutboxRepository).
type gravadorOutbox interface {
	Gravar(ctx context.Context, destino string, env mensageria.Envelope, trace map[string]string) error
}

// Enfileirador implementa a porta filaComandos do caso de uso.
type Enfileirador struct {
	outbox gravadorOutbox
	tracer mensageria.Tracer
	agora  func() time.Time
}

// NovoEnfileirador cria o adaptador. tracer nil = sem propagação de trace.
func NovoEnfileirador(outbox gravadorOutbox, tracer mensageria.Tracer) *Enfileirador {
	if tracer == nil {
		tracer = mensageria.TracerNulo{}
	}
	return &Enfileirador{outbox: outbox, tracer: tracer, agora: time.Now}
}

// Enfileirar grava o comando na outbox, na transação do contexto. O
// correlation_id da requisição viaja no envelope e os headers de trace são
// capturados agora — quando o relay publicar, a transação HTTP já terminou.
func (e *Enfileirador) Enfileirar(ctx context.Context, c saga.Comando) error {
	env, err := mensageria.Novo(c.Tipo, c.SagaID, c.OsID, logging.Correlacao(ctx), mensageria.OrigemOS, c.Payload, e.agora())
	if err != nil {
		return fmt.Errorf("enfileirar %s: %w", c.Tipo, err)
	}
	return e.outbox.Gravar(ctx, c.Destino, env, e.tracer.Capturar(ctx))
}
