package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/ProblemaTheu/oficina-app/internal/application/saga"
	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/infra/mensageria"
	"github.com/ProblemaTheu/oficina-app/internal/infra/orquestracao"
	"github.com/ProblemaTheu/oficina-app/internal/infra/repository"
)

// filasSaga são as URLs das filas SQS. No EKS vêm da SSM via ConfigMap
// (F4-6.1); no compose, do docker-compose.saga.yml.
type filasSaga struct {
	Execution string // execution-commands.fifo — comandos para o Execution
	Billing   string // billing-commands.fifo   — comandos para o Billing
	Respostas string // os-saga-replies.fifo    — respostas para o orquestrador
}

func lerFilas() filasSaga {
	return filasSaga{
		Execution: os.Getenv("FILA_EXECUTION_URL"),
		Billing:   os.Getenv("FILA_BILLING_URL"),
		Respostas: os.Getenv("FILA_RESPOSTAS_URL"),
	}
}

// componentesSaga separa o que entra no caso de uso (opcoes) do que roda em
// segundo plano (iniciar): o consumidor de respostas precisa do caso de uso já
// montado pelo servidor HTTP.
type componentesSaga struct {
	opcoes  []usecase.OSUseCaseOption
	iniciar func(ctx context.Context, uc *usecase.OrdemServicoUseCase)
}

// montarSaga liga a saga da Fase 4 quando FILA_EXECUTION_URL existe; sem ela o
// OS roda como na Fase 3. O consumidor de respostas (orquestrador) sobe quando
// FILA_RESPOSTAS_URL também existe.
//
// O cliente SQS usa a cadeia padrão de credenciais (Pod Identity no EKS) e
// respeita AWS_ENDPOINT_URL (LocalStack no compose) — sem `if local` no código.
func montarSaga(ctx context.Context, db *sql.DB, nrApp *newrelic.Application, filas filasSaga) (componentesSaga, error) {
	if filas.Execution == "" {
		slog.Info("saga desligada: FILA_EXECUTION_URL não definida")
		return componentesSaga{iniciar: func(context.Context, *usecase.OrdemServicoUseCase) {}}, nil
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return componentesSaga{}, fmt.Errorf("configuração AWS: %w", err)
	}
	cliente := sqs.NewFromConfig(cfg)
	outbox := repository.NovoOutboxRepository(db)
	osRepo := repository.NovoOrdemServicoRepository(db)
	tracer := mensageria.TracerNewRelic{App: nrApp}

	opt := usecase.ComSaga(usecase.DependenciasSaga{
		UoW:          repository.NovaUnitOfWork(db),
		Estoque:      repository.NovoEstoqueRepository(),
		Sagas:        repository.NovoSagaRepository(db),
		Comandos:     orquestracao.NovoEnfileirador(outbox, tracer),
		Inbox:        repository.NovoInboxRepository(),
		Cancelamento: osRepo,
	})

	destinos := map[string]string{saga.DestinoExecution: filas.Execution}
	if filas.Billing != "" {
		destinos[saga.DestinoBilling] = filas.Billing
	}
	relay := mensageria.NovoRelay(outbox, mensageria.NovoPublisher(cliente), mensageria.ConfigRelay{Destinos: destinos})

	iniciar := func(ctx context.Context, uc *usecase.OrdemServicoUseCase) {
		go executar(ctx, "relay da outbox", relay.Executar)
		if filas.Respostas == "" {
			slog.Warn("orquestrador desligado: FILA_RESPOSTAS_URL não definida (a saga só envia, não avança)")
			return
		}
		consumidor := mensageria.NovoConsumer(cliente, mensageria.ConfigConsumer{
			FilaURL: filas.Respostas, Handlers: orquestracao.HandlersDeResposta(uc), Tracer: tracer,
		})
		go executar(ctx, "orquestrador (consumidor de respostas)", consumidor.Executar)
		slog.Info("saga ligada", "fila_execution", filas.Execution, "fila_billing", filas.Billing, "fila_respostas", filas.Respostas)
	}
	return componentesSaga{opcoes: []usecase.OSUseCaseOption{opt}, iniciar: iniciar}, nil
}

func executar(ctx context.Context, nome string, laco func(context.Context) error) {
	if err := laco(ctx); err != nil {
		slog.Error(nome+" parou", "error", err)
	}
}
