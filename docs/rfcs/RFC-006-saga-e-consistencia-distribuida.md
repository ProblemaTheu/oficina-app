# RFC-006 — Saga e consistência distribuída

| Campo | Valor |
|---|---|
| **Status** | Aceita |
| **Autores** | Equipe SOAT — Tech Challenge Fase 4 |
| **Relacionadas** | [RFC-005](RFC-005-decomposicao-em-microsservicos.md), ADRs [010](../architecture-decisions.md#adr-010--mensageria-amazon-sqs-fifo), [011](../architecture-decisions.md#adr-011--saga-orquestrada-no-os-service), [012](../architecture-decisions.md#adr-012--transactional-outbox--inbox-idempotente-relay-com-líder-único), [014](../architecture-decisions.md#adr-014--mercado-pago-checkout-pro-api-de-orders-verificação-ativa-e-estorno) |

## Contexto

Com três bancos (RFC-005), abrir e concluir uma OS deixa de caber numa transação. O fluxo atravessa os três serviços — diagnóstico (Execution) → orçamento e aprovação (Billing) → pagamento (Billing + Mercado Pago) → reparo (Execution) — e inclui **duas esperas humanas** de duração indefinida (aprovação e pagamento). Uma falha em qualquer etapa precisa desfazer as anteriores: liberar a reserva de peças, cancelar tarefas e orçamento e, se já houve pagamento, **estornar**.

O enunciado pede o Saga Pattern, com a escolha entre orquestração e coreografia **justificada**.

## Decisão

### 1. Saga orquestrada, com o orquestrador dentro do OS Service

| Critério | Orquestrada | Coreografada |
|---|---|---|
| Onde está o fluxo | Uma máquina de estados pura (`internal/application/saga/orquestrador.go`, 100% coberta) | Implícito, espalhado pelos *handlers* dos 3 serviços |
| Quem decide compensar | O orquestrador, com a lista de etapas concluídas | Cada serviço precisa ouvir as falhas dos outros |
| Estado de uma OS no meio do caminho | Uma linha em `sagas_os` + histórico em `sagas_os_etapas` | Reconstruído a partir de eventos em 3 bancos |
| Esperas humanas com prazo | Estado + `prazo_em` | Sem dono natural do timer |
| Acoplamento | O OS conhece os comandos dos outros | Cada serviço conhece os eventos dos outros |
| Escala de participantes | Pior com dezenas | Melhor |

Com 3 participantes, fluxo linear e esperas humanas, a orquestração ganha. O orquestrador mora no OS Service porque o estado da saga precisa ser gravado **na mesma transação** que muda a OS e o estoque.

### 2. Etapas, pivô e compensações

| # | Etapa | Serviço | Compensação |
|---|---|---|---|
| T1 | Abrir OS + reservar peças | OS (local) | `LiberarReserva` (local) |
| T2 | Diagnóstico | Execution | `CancelarTarefas` |
| T3 | Orçamento | Billing | `CancelarOrcamento` (cancela a order do MP se ainda aberta) |
| T4 | Aprovação | Billing (humano) | coberta por T3 |
| T5 | Pagamento | Billing + Mercado Pago | `EstornarPagamento` (refund) |
| T6 | **Início do reparo — pivô** | Execution → OS baixa o estoque | — |
| T7 | Conclusão | Execution | — |

- **Pivô em T6.** Depois que o mecânico inicia o reparo, peças saíram do estoque e o carro está desmontado; uma falha posterior é tratada pela oficina, não por compensação automática.
- **Compensações sequenciais, em ordem inversa.** O orquestrador envia uma, espera a confirmação, envia a próxima. Paralelizar economizaria segundos num fluxo que dura dias, ao custo de um estado de "compensações parciais" bem mais difícil de raciocinar.
- **Não existe estado `INICIADA`.** A saga nasce em `DIAGNOSTICO_PENDENTE` na mesma transação que grava OS, reserva e o comando `IniciarDiagnostico` na outbox.

### 3. Entrega confiável

- **Transactional Outbox** em todos os serviços (Postgres no OS e no Billing; `TransactWriteItems` no DynamoDB), com relay publicando no SQS FIFO.
- **Inbox idempotente:** cada mensagem consumida é marcada na mesma transação do efeito; reentrega vira *ack* sem efeito.
- **Ordem por saga:** `MessageGroupId = saga_id`; `MessageDeduplicationId = message_id`.
- **Concorrência no orquestrador:** lock otimista (`sagas_os.versao`); a versão só sobe quando a saga muda.
- **Erros:** erro permanente (mensagem inválida) vai direto para a DLQ; erro transitório é reentregue até 5 vezes e depois vai para a DLQ.

### 4. Casos de borda tratados

| Caso | Tratamento |
|---|---|
| Pagamento chega depois de a saga compensar (cliente pagou no link antigo) | Orquestrador registra `ignorada` e envia `EstornarPagamento`; o Billing também estorna qualquer `approved` de orçamento cancelado |
| Cartão recusado, cliente tenta de novo no mesmo link | O Checkout Pro permite novas tentativas na mesma order. O Billing **cancela a order** antes de emitir `PagamentoRecusado` |
| Webhook não chega (sandbox não notifica orders de teste) | **Reconciliador** consulta `GET /v1/orders/{id}` dos pagamentos pendentes; o webhook (com `x-signature` validada) só antecipa |
| Resposta fora de ordem ou repetida | Registrada como `ignorada` em `sagas_os_etapas`, sem mudar a saga |
| Compensação falha (MP 5xx no refund) | Reentrega até 5×, depois DLQ + alarme; a saga fica em `COMPENSANDO` — **nunca** `COMPENSADA` sem todas as confirmações |

## Fora do escopo desta entrega (evolução)

- **Timeouts das esperas humanas** (orçamento sem resposta, pagamento não concluído). As colunas `prazo_em` existem; falta o *watchdog* que emite `OrcamentoExpirado`/`PagamentoExpirado` como falhas normais da saga.
- **Reprocessamento de DLQ** por ferramenta; hoje é manual (redrive do SQS).
- **Estorno parcial**: o estorno é sempre total.

## Consequências

- **Positivas:** o fluxo inteiro cabe numa tabela de transições testável sem infraestrutura; o estado de qualquer OS é consultável; compensações são comandos comuns, com as mesmas garantias de entrega.
- **Negativas:** o OS Service conhece o catálogo de comandos dos outros dois; consistência eventual visível ao usuário (a OS leva segundos para mudar de status após cada resposta); mais tabelas (`sagas_os`, `sagas_os_etapas`, `outbox`, `mensagens_processadas`) em cada banco.
