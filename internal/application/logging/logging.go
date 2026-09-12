// Package logging define a porta de logging correlacionado usada pelos casos
// de uso. A infraestrutura HTTP decora o contexto com um *slog.Logger por
// requisição (ver middleware.Correlacao); a aplicação apenas lê — a camada de
// dentro define a porta, a de fora implementa, sem violar a regra de
// dependência da Clean Architecture.
package logging

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

// chave é a chave de contexto usada para guardar o logger decorado.
var chave = ctxKey{}

// Log devolve o logger correlacionado da requisição atual, ou o logger
// default do slog quando chamado fora de um contexto HTTP (testes, jobs
// internos, inicialização).
func Log(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(chave).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// ComLogger devolve um novo contexto carregando o logger informado.
func ComLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, chave, logger)
}
