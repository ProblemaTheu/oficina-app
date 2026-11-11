package main

import (
	"context"
	"database/sql"
	"fmt"
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

// montarSaga cria as dependências da saga e o relay da outbox.
//
// O cliente SQS usa a cadeia padrão de credenciais (Pod Identity no EKS) e
// respeita AWS_ENDPOINT_URL (LocalStack no compose) — sem `if local` no código.
func montarSaga(ctx context.Context, db *sql.DB, nrApp *newrelic.Application, filas filasSaga) (usecase.OSUseCaseOption, *mensageria.Relay, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("configuração AWS: %w", err)
	}
	cliente := sqs.NewFromConfig(cfg)
	outbox := repository.NovoOutboxRepository(db)

	opt := usecase.ComSaga(usecase.DependenciasSaga{
		UoW:      repository.NovaUnitOfWork(db),
		Estoque:  repository.NovoEstoqueRepository(),
		Sagas:    repository.NovoSagaRepository(db),
		Comandos: orquestracao.NovoEnfileirador(outbox, mensageria.TracerNewRelic{App: nrApp}),
	})

	destinos := map[string]string{saga.DestinoExecution: filas.Execution}
	if filas.Billing != "" {
		destinos[saga.DestinoBilling] = filas.Billing
	}
	relay := mensageria.NovoRelay(outbox, mensageria.NovoPublisher(cliente), mensageria.ConfigRelay{Destinos: destinos})
	return opt, relay, nil
}
