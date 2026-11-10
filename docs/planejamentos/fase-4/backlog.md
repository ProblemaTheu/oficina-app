# Backlog — Tech Challenge Fase 4

> Cada tarefa tem **objetivo**, **passo a passo** e **critérios de aceite**. Os IDs (`F4-x.y`) são referenciados no [roadmap](roadmap.md) e no [plano de execução](plano-execucao.md). Decisões de fundo estão no [README](README.md#decisões-técnicas-recomendadas); desenho em [arquitetura.md](arquitetura.md).

> **Revisão de 10/11 — plano de 4 dias.** O backlog continua descrevendo o desenho completo; o que **não** será feito nesta entrega está marcado com ✂️ e o número do corte no [plano de execução](plano-execucao.md#os-cortes). Correções de desenho feitas nesta revisão: DDL da migration `000007` (coluna `nome_status`, `reserva_estado`, `ignorada`, `falhou_em`), fluxo do webhook do Mercado Pago (F4-2.4), relay do DynamoDB com réplica única (F4-3.1), numeração das etapas no orquestrador (F4-4.4) e exposição real de `/internal/*` (F4-1.5).

Convenções usadas em todo o backlog:

| Item | Convenção |
|---|---|
| Repositórios | `ProblemaTheu/oficina-app` (OS), `ProblemaTheu/oficina-billing-service`, `ProblemaTheu/oficina-execution-service` |
| Namespaces K8s | `oficina-prod` (OS — mantido da Fase 3), `billing-prod`, `execution-prod` |
| Imagens | `problematheu/oficina-api` (já existe), `problematheu/oficina-billing`, `problematheu/oficina-execution` |
| Filas | `os-saga-replies.fifo`, `billing-commands.fifo`, `execution-commands.fifo` + `*-dlq.fifo` |
| SSM | `/oficina/prod/<servico>/<chave>` (mesmo padrão da Fase 3) |
| Segredos | `oficina/prod/app` (JWT, já existe), `oficina/prod/billing` (DB, MP, HMAC), `oficina/prod/internal` (token interno) |
| Tags | `Projeto=oficina`, `Ambiente=prod`, `Servico=<os|billing|execution>`, `Repositorio=<repo>` |

---

## E0 — Fundação e contratos

### F4-0.1 — Fechar decisões e o contrato de mensagens

**Objetivo:** congelar o que os três repositórios precisam concordar **antes** de escrever código: lista de mensagens, envelope, filas e estados da saga.

**Passo a passo**

1. Revisar a tabela de [decisões](README.md#decisões-técnicas-recomendadas). Qualquer troca (ex.: RabbitMQ no lugar de SQS) muda F4-4.1, F4-6.2 e o compose local — decida agora.
2. ✂️ *(corte 18 — o contrato fica no catálogo em Markdown da arquitetura §5)* Criar `docs/mensagens/` no `oficina-app` com um JSON Schema por tipo de mensagem do [catálogo](arquitetura.md#52-catálogo) e um `envelope.schema.json`:

   ```json
   {
     "$schema": "https://json-schema.org/draft/2020-12/schema",
     "$id": "envelope.schema.json",
     "type": "object",
     "required": ["message_id", "tipo", "versao", "saga_id", "os_id", "ocorrido_em", "origem", "payload"],
     "properties": {
       "message_id": {"type": "string", "format": "uuid"},
       "tipo": {"type": "string"},
       "versao": {"type": "integer", "minimum": 1},
       "saga_id": {"type": "string", "format": "uuid"},
       "os_id": {"type": "string", "format": "uuid"},
       "correlation_id": {"type": "string"},
       "ocorrido_em": {"type": "string", "format": "date-time"},
       "origem": {"enum": ["os-service", "billing-service", "execution-service"]},
       "payload": {"type": "object"}
     },
     "additionalProperties": false
   }
   ```

3. Abrir rascunho da **RFC-005 (decomposição)** e da **RFC-006 (saga)** em `docs/rfcs/` — o contexto está fresco agora; finalizar em F4-8.2.

**Critérios de aceite**
- [ ] ~~`docs/mensagens/*.schema.json` cobre as 21 mensagens do catálogo.~~ ✂️ corte 18 — catálogo da arquitetura §5 revisado pelas 3 trilhas no D1.
- [ ] Máquina de estados da saga ([arquitetura §3.2](arquitetura.md#32-máquina-de-estados-da-saga)) revisada e sem transição ambígua.
- [ ] RFC-005 e RFC-006 em rascunho com contexto e alternativas.

---

### F4-0.2 — Repositórios, proteção e acessos

**Objetivo:** os 3 repos de serviço existem, estão protegidos e com tudo de que o CI precisa — incluindo as pendências que ficaram da Fase 3.

**Passo a passo**

1. **Branch protection já está ativa** nos repositórios existentes. Exporte o ruleset atual para replicá-lo idêntico nos repos novos:

   ```bash
   gh api repos/ProblemaTheu/oficina-app/rulesets --jq '.[].id'
   gh api repos/ProblemaTheu/oficina-app/rulesets/<id> > /tmp/ruleset-atual.json
   ```
2. Criar os repos novos a partir de um esqueleto comum:

   ```bash
   for r in oficina-billing-service oficina-execution-service; do
     gh repo create "ProblemaTheu/$r" --public --description "Oficina — ${r#oficina-} (Tech Challenge Fase 4)"
     gh api -X PUT "repos/ProblemaTheu/$r/collaborators/soat-architecture" -f permission=pull
   done
   ```

3. Branches `main` e `homolog` nos dois; README inicial já no formato do enunciado (propósito, tecnologias, execução, arquitetura, cobertura, links).
4. **Ruleset na `main` dos 2 repos novos** — use o exportado no passo 1 (sem os campos `id`, `source`, `_links`, `created_at`, `updated_at`) acrescentando os checks novos; o modelo abaixo serve de referência. Acrescente `Cobertura ≥ 80%` e `SonarCloud` também no ruleset do `oficina-app`:

   ```bash
   cat > /tmp/ruleset.json <<'EOF'
   {
     "name": "main protegida",
     "target": "branch",
     "enforcement": "active",
     "conditions": {"ref_name": {"include": ["refs/heads/main", "refs/heads/homolog"], "exclude": []}},
     "rules": [
       {"type": "deletion"},
       {"type": "non_fast_forward"},
       {"type": "pull_request", "parameters": {
         "required_approving_review_count": 0,
         "dismiss_stale_reviews_on_push": true,
         "require_last_push_approval": false,
         "required_review_thread_resolution": false
       }},
       {"type": "required_status_checks", "parameters": {
         "strict_required_status_checks_policy": true,
         "required_status_checks": [
           {"context": "Build e testes"},
           {"context": "Lint"},
           {"context": "Cobertura ≥ 80%"},
           {"context": "SonarCloud"},
           {"context": "gitleaks"}
         ]
       }}
     ]
   }
   EOF
   gh api -X POST repos/ProblemaTheu/oficina-billing-service/rulesets --input /tmp/ruleset.json
   ```

   > `required_approving_review_count: 0` mantém o PR obrigatório sem travar um time pequeno (mesma decisão da Fase 3). Os nomes em `required_status_checks` precisam bater **exatamente** com o `name:` dos jobs — confira depois do primeiro PR.

5. **OIDC para os repos novos.** No `bootstrap/` (Fase 3), acrescentar as roles `gha-oficina-billing-service` e `gha-oficina-execution-service` (+ as `-plan` read-only), com `sub` restrito a `repo:ProblemaTheu/<repo>:environment:prod` e `pull_request`. Cada role precisa de **access entry** no EKS com `AmazonEKSEditPolicy` limitada ao namespace do serviço:

   ```hcl
   access_entries = {
     gha_billing = {
       principal_arn = "arn:aws:iam::706215605178:role/gha-oficina-billing-service"
       policy_associations = {
         edit = {
           policy_arn   = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSEditPolicy"
           access_scope = { type = "namespace", namespaces = ["billing-prod"] }
         }
       }
     }
   }
   ```

6. Variables e secrets por repo: `AWS_ROLE_ARN`, `AWS_REGION`, `TF_BUCKET`, `DOCKERHUB_USERNAME/TOKEN`, `NEW_RELIC_LICENSE_KEY`, `NEW_RELIC_API_KEY`, `SONAR_TOKEN`, `SONAR_PROJECT_KEY`.
7. **SonarCloud:** criar os 2 projetos; resolver a pendência da chave `tech-challenge-1` → `oficina-app` (*Update key* no SonarCloud **antes** de trocar a variable); desligar a análise automática onde o workflow faz a análise.
8. Docker Hub: criar os repositórios `oficina-billing` e `oficina-execution` (o rename da Fase 3 esqueceu esse passo e custou um dia).

**Critérios de aceite**
- [ ] 3 repos de serviço existem, públicos, com `soat-architecture` **aceito** (print guardado).
- [ ] `git push origin main` é rejeitado nos 6 repos.
- [ ] Um PR de teste em cada repo novo assume a role `-plan` e a de deploy só funciona com `environment: prod`.
- [ ] Projetos SonarCloud criados e chave do `oficina-app` corrigida.

---

### F4-0.3 — Contas e prazos de terceiros

**Objetivo:** tudo que depende de outra pessoa ou empresa pedido no dia 1.

**Passo a passo**

1. **Mercado Pago**
   > **Validado em 10/11** (ver [diário](execucao.md#1011--mercado-pago-validado)). O que funcionou — e o que **não** funciona:
   - Criar a aplicação na **conta pessoal** de um integrante, no [painel](https://www.mercadopago.com.br/developers/panel/app): solução **Checkout Pro**, tipo de API **API de Orders**.
   - Em *Contas de teste*, criar **vendedor** e **comprador** (ambos Brasil). Código de verificação no login = o que aparece em *Credenciais → Conta de teste*.
   - Usar as **Credenciais de teste** da aplicação (o Access Token começa com `APP_USR-` e termina com o id do **vendedor de teste** vinculado — é normal). ❌ **Não** usar credenciais de produção de uma aplicação criada *dentro* do vendedor de teste: cobram, mas o estorno responde `401 Unauthorized use of live credentials`.
   - ❌ A **API de Preferences** (`/checkout/preferences` + `/v1/payments/{id}/refunds`) cobra, mas **não estorna** com conta de teste (mesmo 401). Com a **API de Orders** cobrança, consulta e estorno funcionam (F4-2.3).
   - Pagar como **comprador de teste em outro navegador ou no perfil Convidado** — janelas anônimas do Chrome compartilham a sessão do vendedor e o checkout responde "A transação não aceita este meio de pagamento". Cartão gerado no painel (*Cartões de teste*), titular `APRO`/`OTHE`, CPF `12345678909`.
   - Em *Webhooks*, cadastrar a URL `https://<api-gateway>/billing/webhooks/mercadopago` (ngrok no D2), evento **Order (Mercado Pago)** — *confirmar o nome do evento no painel ao configurar* — e copiar a **assinatura secreta** (valida o `x-signature`).
   - Guardar `MP_ACCESS_TOKEN` e `MP_WEBHOOK_SECRET` em `oficina/prod/billing` no Secrets Manager — nunca no repo.
   - Anotar os [cartões de teste](https://www.mercadopago.com.br/developers/pt/docs/checkout-pro/additional-content/your-integrations/test/cards) e os nomes de titular que forçam resultado (`APRO` aprovado, `OTHE` recusado). **Confira na documentação vigente** — o painel muda com frequência.
2. **AWS:** conferir o pedido de quota de vCPU (32) que estava `CASE_OPENED` e o estado do ambiente (`aws eks list-clusters`, `aws rds describe-db-instances`). Se destruído, o `apply` completo leva ~25 min — agende.
3. **New Relic:** rotacionar as chaves que passaram pelo chat na Fase 3 (pendência registrada) e atualizar os secrets dos repos.
4. Confirmar no fórum se a dispensa de homologação continua valendo na Fase 4 e guardar o print.

**Critérios de aceite**
- [x] Uma order criada via `curl` com o token de teste devolve `checkout_url`, um pagamento `APRO` deixa a order `processed` e o estorno via API deixa a order `refunded` (10/11).
- [ ] Quota de vCPU ≥ 8 confirmada (3 × `t3.small` = 6).
- [ ] Chaves do New Relic rotacionadas.

---

### F4-0.4 — Ambiente local multi-serviço

**Objetivo:** subir a saga inteira na máquina (e no CI) sem nuvem.

**Passo a passo**

1. Criar `docker-compose.saga.yml` no `oficina-app` (o compose atual continua servindo para o OS isolado):

   ```yaml
   services:
     os-db:
       image: postgres:15.7
       environment: { POSTGRES_DB: oficina, POSTGRES_USER: oficina, POSTGRES_PASSWORD: oficina }
     billing-db:
       image: postgres:15.7
       environment: { POSTGRES_DB: billing, POSTGRES_USER: billing, POSTGRES_PASSWORD: billing }
     aws:
       image: localstack/localstack:4
       environment: { SERVICES: "sqs,dynamodb" }
       volumes: ["./scripts/localstack-init.sh:/etc/localstack/init/ready.d/init.sh:ro"]
     mailpit:
       image: axllent/mailpit
       ports: ["8025:8025"]
     mp-stub:
       image: problematheu/oficina-billing:latest
       command: ["/app/mp-stub"]        # F4-2.6
     os:
       build: .
       environment:
         DB_HOST: os-db
         AWS_ENDPOINT_URL: http://aws:4566
         FILA_RESPOSTAS_URL: http://aws:4566/000000000000/os-saga-replies.fifo
         FILA_BILLING_URL: http://aws:4566/000000000000/billing-commands.fifo
         FILA_EXECUTION_URL: http://aws:4566/000000000000/execution-commands.fifo
       ports: ["8080:8080"]
     billing:
       image: problematheu/oficina-billing:${BILLING_TAG:-latest}
       environment:
         DB_HOST: billing-db
         AWS_ENDPOINT_URL: http://aws:4566
         MP_BASE_URL: http://mp-stub:8090
       ports: ["8081:8080"]
     execution:
       image: problematheu/oficina-execution:${EXECUTION_TAG:-latest}
       environment:
         AWS_ENDPOINT_URL: http://aws:4566
         DYNAMODB_TABELA: execution-tarefas
       ports: ["8082:8080"]
   ```

2. `scripts/localstack-init.sh` cria as 6 filas (3 + DLQs) e a tabela, **com os mesmos atributos do Terraform** (FIFO, `maxReceiveCount`, GSI):

   ```bash
   #!/bin/sh
   set -e
   for f in os-saga-replies billing-commands execution-commands; do
     awslocal sqs create-queue --queue-name "$f-dlq.fifo" --attributes FifoQueue=true
     dlq_arn=$(awslocal sqs get-queue-attributes --queue-url "http://localhost:4566/000000000000/$f-dlq.fifo" \
       --attribute-names QueueArn --query Attributes.QueueArn --output text)
     awslocal sqs create-queue --queue-name "$f.fifo" --attributes \
       "FifoQueue=true,VisibilityTimeout=60,RedrivePolicy={\"deadLetterTargetArn\":\"$dlq_arn\",\"maxReceiveCount\":\"5\"}"
   done
   awslocal dynamodb create-table --table-name execution-tarefas \
     --billing-mode PAY_PER_REQUEST \
     --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=sk,AttributeType=S \
       AttributeName=gsi1pk,AttributeType=S AttributeName=gsi1sk,AttributeType=S \
     --key-schema AttributeName=pk,KeyType=HASH AttributeName=sk,KeyType=RANGE \
     --global-secondary-indexes '[{"IndexName":"gsi1","KeySchema":[{"AttributeName":"gsi1pk","KeyType":"HASH"},{"AttributeName":"gsi1sk","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}}]'
   ```

3. O SDK da AWS para Go v2 respeita `AWS_ENDPOINT_URL` — **não** escreva `if local { endpoint = ... }` no código.

> ⚠️ **Plano B do LocalStack.** Se a imagem do LocalStack passar a exigir token de conta no momento da execução, troque `aws` por `softwaremill/elasticmq-native` (SQS compatível, com FIFO) + `amazon/dynamodb-local`, com `AWS_ENDPOINT_URL_SQS` e `AWS_ENDPOINT_URL_DYNAMODB` separados. Decida no dia em que montar o compose, não no dia do BDD.

**Critérios de aceite**
- [ ] `docker compose -f docker-compose.saga.yml up` sobe os 3 serviços saudáveis em < 2 min.
- [ ] `awslocal sqs list-queues` mostra 6 filas; `awslocal dynamodb describe-table` mostra o GSI.

---

### F4-0.5 — Pacote de mensageria

**Objetivo:** escrever **uma vez** o código de envelope, publicação, consumo, outbox e inbox, e copiá-lo para os outros dois repos.

**Decisão:** copiar `internal/infra/mensageria` para os 3 repos em vez de criar um módulo Go compartilhado. Um módulo comum seria um 4º repositório de código, com versionamento próprio — e acoplaria os deploys, que é o que a fase quer evitar. São ~400 linhas estáveis; a duplicação é o preço consciente da independência (registrar na ADR-013).

**Estrutura**

```
internal/infra/mensageria/
├── envelope.go      # tipo Envelope, NovoEnvelope, validação
├── publisher.go     # SQS SendMessage com MessageGroupId/DeduplicationId e headers de trace
├── consumer.go      # long polling, dispatch por tipo, ack/nack, visibilidade
├── outbox_pg.go     # Gravar(tx, destino, env) + Relay (Postgres)
├── inbox_pg.go      # JaProcessada/Marcar dentro da tx (Postgres)
└── trace.go         # inserir/aceitar headers de DT do New Relic
```

**Envelope**

```go
// Envelope é o formato comum de todas as mensagens da saga. O contrato está em
// docs/mensagens/envelope.schema.json no oficina-app.
type Envelope struct {
	MessageID     uuid.UUID       `json:"message_id"`
	Tipo          string          `json:"tipo"`
	Versao        int             `json:"versao"`
	SagaID        uuid.UUID       `json:"saga_id"`
	OsID          uuid.UUID       `json:"os_id"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	OcorridoEm    time.Time       `json:"ocorrido_em"`
	Origem        string          `json:"origem"`
	Payload       json.RawMessage `json:"payload"`
}
```

**Publicação (FIFO por saga)**

```go
func (p *Publisher) Publicar(ctx context.Context, filaURL string, env Envelope, trace map[string]string) error {
	corpo, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("serializar envelope: %w", err)
	}
	attrs := map[string]types.MessageAttributeValue{
		"tipo": {DataType: aws.String("String"), StringValue: aws.String(env.Tipo)},
	}
	for k, v := range trace {
		attrs[k] = types.MessageAttributeValue{DataType: aws.String("String"), StringValue: aws.String(v)}
	}
	_, err = p.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:               aws.String(filaURL),
		MessageBody:            aws.String(string(corpo)),
		MessageGroupId:         aws.String(env.SagaID.String()),
		MessageDeduplicationId: aws.String(env.MessageID.String()),
		MessageAttributes:      attrs,
	})
	return err
}
```

**Critérios de aceite**
- [ ] Pacote com testes unitários ≥ 90% (cliente SQS atrás de interface, mockado).
- [ ] Mesmo pacote compilando nos 3 repos (diferença só no adaptador DynamoDB do Execution — F4-3.1).

---

## E1 — OS Service (refatoração do `oficina-app`)

### F4-1.1 — Recortar responsabilidades

**Objetivo:** o OS Service deixa de fazer o que passou a ser do Billing e do Execution, e para de permitir que alguém "pule" a saga.

**Passo a passo**

1. **Remover do `docs/openapi.yaml`** (e regenerar com `go generate ./...`):
   - `POST /v1/work-orders/{id}/approve` e `/reject` → migram para `POST /billing/v1/orcamentos/{id}/aprovar|recusar`;
   - `POST /v1/webhooks/budget-response` → migra para o Billing com a **mesma** assinatura HMAC (`X-Signature`), para não quebrar integradores.
2. **Restringir `PATCH /v1/work-orders/{id}/status`** a `finalizada → entregue`. Qualquer outro destino → `422 TRANSICAO_CONTROLADA_PELA_SAGA`:

   ```go
   // transicoesManuais são as únicas que um usuário pode disparar. As demais
   // pertencem ao orquestrador da saga (F4-4.4).
   var transicoesManuais = map[entity.Status]entity.Status{
   	entity.StatusFinalizada: entity.StatusEntregue,
   }
   ```

3. Remover `AprovarOrcamento`, `RejeitarOrcamento` e `ProcessarRespostaOrcamento` do use case — e os testes correspondentes migram para o Billing (não descarte: são a especificação do comportamento).
4. Manter `AvancarStatus` como **porta interna** usada pelo orquestrador, sem rota HTTP direta.

**Critérios de aceite**
- [ ] `PATCH .../status` com `em_execucao` → 422; com `entregue` a partir de `finalizada` → 200.
- [ ] Coleção Postman sem as rotas removidas.
- [ ] Cobertura do pacote `usecase` continua ≥ 90%.

---

### F4-1.2 — Status `cancelada` e reserva de estoque

**Objetivo:** dar à saga um estado terminal de falha e eliminar o dual-write de estoque (achados 1, 3 e 4).

**Passo a passo**

1. Migration `000007_fase4_saga.up.sql`:

   ```sql
   -- a coluna é nome_status (000001_initial_schema); id fixo segue o padrão do seed 000002
   INSERT INTO status_ordens (id, nome_status, descricao) VALUES
     ('660e8400-e29b-41d4-a716-446655440007', 'cancelada',
      'OS cancelada pela compensação da saga (recusa, pagamento recusado ou falha na execução).')
   ON CONFLICT (nome_status) DO NOTHING;

   -- 'finalizada' deixa de significar "rejeitada"
   UPDATE status_ordens SET descricao = 'Serviço concluído.' WHERE nome_status = 'finalizada';

   ALTER TABLE ordens_servico
     ADD COLUMN cancelado_em         timestamptz,
     ADD COLUMN motivo_cancelamento  text;

   ALTER TABLE pecas
     ADD COLUMN estoque_reservado integer NOT NULL DEFAULT 0,
     ADD CONSTRAINT pecas_reserva_valida
       CHECK (estoque_reservado >= 0 AND estoque_reservado <= estoque_atual);

   -- idempotência de Liberar/Baixar por item (critério de aceite abaixo)
   ALTER TABLE itens_os_pecas
     ADD COLUMN reserva_estado text NOT NULL DEFAULT 'baixada'
       CHECK (reserva_estado IN ('reservada', 'liberada', 'baixada'));
   -- DEFAULT 'baixada' para as OS antigas, cujo estoque já foi deduzido;
   -- o CriarOS grava 'reservada' explicitamente.

   CREATE TABLE sagas_os (
     id             uuid PRIMARY KEY,
     os_id          uuid NOT NULL UNIQUE REFERENCES ordens_servico(id),
     estado         text NOT NULL,
     etapa_atual    text NOT NULL,
     etapa_falha    text,
     orcamento_id   uuid,
     pagamento_id   uuid,
     prazo_em       timestamptz,
     versao         integer NOT NULL DEFAULT 1,
     criado_em      timestamptz NOT NULL DEFAULT now(),
     atualizado_em  timestamptz NOT NULL DEFAULT now()
   );
   CREATE INDEX sagas_os_prazo_idx ON sagas_os (prazo_em)
     WHERE estado NOT IN ('CONCLUIDA', 'COMPENSADA');

   CREATE TABLE sagas_os_etapas (
     id          uuid PRIMARY KEY,
     saga_id     uuid NOT NULL REFERENCES sagas_os(id),
     etapa       text NOT NULL,
     resultado   text NOT NULL CHECK (resultado IN ('ok', 'falha', 'compensada', 'ignorada')),
     message_id  uuid,
     detalhe     jsonb,
     ocorrido_em timestamptz NOT NULL DEFAULT now()
   );
   CREATE INDEX sagas_os_etapas_saga_idx ON sagas_os_etapas (saga_id, ocorrido_em);

   CREATE TABLE outbox (
     id            uuid PRIMARY KEY,
     destino       text NOT NULL,
     tipo          text NOT NULL,
     envelope      jsonb NOT NULL,
     trace_headers jsonb NOT NULL DEFAULT '{}',
     criado_em     timestamptz NOT NULL DEFAULT now(),
     enviado_em    timestamptz,
     tentativas    integer NOT NULL DEFAULT 0,
     falhou_em     timestamptz          -- preenchido após 10 tentativas (F4-4.2)
   );
   CREATE INDEX outbox_pendentes_idx ON outbox (criado_em)
     WHERE enviado_em IS NULL AND falhou_em IS NULL;

   CREATE TABLE mensagens_processadas (
     message_id    uuid PRIMARY KEY,
     tipo          text NOT NULL,
     processado_em timestamptz NOT NULL DEFAULT now()
   );
   ```

2. Na entidade, `StatusCancelada` alcançável a partir de **qualquer** status não terminal — mas só pelo orquestrador.
3. Repositório de estoque com três operações, **todas recebendo `*sql.Tx`** (para compor com a saga e a outbox numa transação só):

   ```go
   // Reservar segura n unidades sem retirá-las do estoque. Falha com
   // ErrEstoqueInsuficiente se o disponível (atual - reservado) não bastar.
   Reservar(ctx context.Context, tx *sql.Tx, osID uuid.UUID) error
   // Liberar desfaz a reserva (compensação de T1).
   Liberar(ctx context.Context, tx *sql.Tx, osID uuid.UUID) error
   // Baixar converte a reserva em saída: atual -= n, reservado -= n.
   Baixar(ctx context.Context, tx *sql.Tx, osID uuid.UUID) error
   ```

   A reserva é um `UPDATE ... WHERE estoque_atual - estoque_reservado >= $n` por peça; `RowsAffected() == 0` → `ErrEstoqueInsuficiente`. Sem `SELECT` antes (evita a corrida).

4. Expor `estoque_reservado` (e `estoque_disponivel = atual - reservado`) no schema `Peca` do `GET /v1/parts/{id}` — o BDD (F4-5.2) verifica a reserva **só por HTTP**.
5. `DeduzirEstoquePecas` sai de cena; testes de integração (`integration.yml`) cobrem as três operações contra Postgres real, incluindo a corrida de duas reservas para a última peça.

**Critérios de aceite**
- [ ] Duas OS concorrentes para a última unidade: uma cria, a outra recebe `422 ESTOQUE_INSUFICIENTE` (teste de integração).
- [ ] `Liberar` e `Baixar` são idempotentes por OS (chamar duas vezes não altera o saldo duas vezes) — usar uma coluna `reserva_estado` em `itens_os_pecas` (`reservada | liberada | baixada`).
- [ ] O `CHECK` do banco barra reserva maior que o estoque mesmo se o código errar.

---

### F4-1.3 — Abertura de OS inicia a saga

**Objetivo:** `POST /v1/work-orders` grava OS, reserva, saga e primeiro comando **numa transação**.

**Passo a passo**

1. Introduzir uma porta `UnitOfWork` no use case (`Executar(ctx, func(tx Tx) error) error`) — o repositório atual abre transação internamente (`ordem_servico_repository.go:97`); passe a recebê-la.
2. Fluxo do `CriarOS`:

   ```go
   err := uc.uow.Executar(ctx, func(tx Tx) error {
   	os, err := uc.osRepo.Criar(ctx, tx, novaOS, itensServico, itensPeca)
   	if err != nil {
   		return err
   	}
   	if err := uc.estoque.Reservar(ctx, tx, os.ID); err != nil {
   		return err // nenhuma saga nasce se a reserva falhar (T1)
   	}
   	saga := saga.Nova(os.ID, uc.relogio.Agora()) // nasce em DIAGNOSTICO_PENDENTE, etapa T1 ok
   	if err := uc.sagas.Criar(ctx, tx, saga); err != nil {
   		return err
   	}
   	return uc.outbox.Gravar(ctx, tx, destinoExecution,
   		saga.Comando(TipoIniciarDiagnostico, payloadDiagnostico(os)), traceDe(ctx))
   })
   ```

3. A resposta HTTP continua `201` com a OS `recebida` — a saga é assíncrona; o cliente acompanha por `GET /v1/work-orders/{id}`.

**Critérios de aceite**
- [ ] Falha forçada na gravação da outbox (teste) → nenhuma OS, reserva ou saga persistida.
- [ ] Após `201`, existe exatamente 1 linha pendente na `outbox` para a OS.

---

### F4-1.4 — Consulta de status e histórico com a saga

**Objetivo:** o requisito "consulta de status e histórico" passa a mostrar onde a OS está **na saga**.

**Passo a passo**

1. `GET /v1/work-orders/{id}` ganha o bloco `saga`:

   ```json
   "saga": {
     "estado": "AGUARDANDO_PAGAMENTO",
     "etapa_atual": "T5",
     "prazo_em": "2026-11-10T15:00:00Z",
     "etapas": [
       {"etapa": "T1", "resultado": "ok", "ocorrido_em": "..."},
       {"etapa": "T2", "resultado": "ok", "ocorrido_em": "..."}
     ]
   }
   ```

2. `GET /v1/work-orders/{id}/status` (público) continua retornando só o status visível — **não** exponha estado interno da saga em rota pública.
3. `historicos_status` recebe uma linha por mudança de status da OS feita pelo orquestrador, com `observacao` = tipo da mensagem que a causou.

**Critérios de aceite**
- [ ] OpenAPI atualizado com o schema `SagaResumo`.
- [ ] Uma OS compensada mostra as etapas com `resultado: compensada` em ordem inversa.

---

### F4-1.5 — Endpoint interno para a Lambda (fim do acesso direto ao banco)

**Objetivo:** cumprir "nenhum serviço acessa o banco de outro" também para a Lambda de autenticação (achado 2).

**Passo a passo**

1. No OS Service, rota **fora do OpenAPI público** e fora do API Gateway:

   ```
   GET /internal/clientes/por-cpf/{cpf}
   Header: X-Internal-Token: <oficina/prod/internal:token>
   200 {"id": "...", "nome": "...", "status": "ativo"} · 404
   ```

   Middleware compara o token com `subtle.ConstantTimeCompare`. Rota responde 404 (não 401) sem o header, para não revelar que existe.

   > **Atenção:** "fora do API Gateway" não é "privada". A Lambda sai da VPC e chama o **CLB público** do OS, então `/internal/*` é alcançável pela internet e o token é a única proteção. Assuma isso na ADR-015 (evolução: CLB interno + Lambda na VPC sem rota para o RDS).
2. No `oficina-lambda-auth`:
   - trocar a query em `clientes` por um cliente HTTP com timeout de 2 s para `OS_INTERNAL_URL` (DNS do CLB do OS Service);
   - **remover a configuração de VPC da Lambda** e o SG de acesso ao RDS — a Lambda não precisa mais da rede privada. Bônus: some o cold start de ENI;
   - remover o DSN do segredo lido pela Lambda.
3. Manter os mesmos códigos de resposta (400/403/404/200); 5xx do OS → 503 `servico_indisponivel`.

**Critérios de aceite**
- [ ] A role da Lambda não tem nenhuma permissão nem rota de rede para o RDS (`terraform plan` mostra a remoção).
- [ ] Os 4 cenários de `/auth/token` da Fase 3 continuam passando.
- [ ] `curl` em `/internal/...` pelo DNS do CLB, sem o header → 404; com token errado → 404.

---

## E2 — Billing Service

### F4-2.1 — Esqueleto do serviço

**Objetivo:** repo `oficina-billing-service` no mesmo padrão do `oficina-app` (Go 1.26, Clean Architecture, API-first com `oapi-codegen`, chi, `golang-migrate`, `slog` JSON + New Relic).

**Passo a passo**

1. Copiar do `oficina-app`: `Dockerfile`, `.golangci.yml`, `oapi-codegen.yaml`, middleware JWT + correlação + APM, `internal/infra/observability`, `internal/infra/mensageria` (F4-0.5), workflows (ajustados em F4-6.1).
2. `docs/openapi.yaml` com:

   | Método | Rota | Auth | Descrição |
   |---|---|---|---|
   | GET | `/billing/v1/orcamentos/{id}` | JWT | Detalhe do orçamento |
   | GET | `/billing/v1/orcamentos?os_id=` | JWT | Orçamento de uma OS |
   | POST | `/billing/v1/orcamentos/{id}/aprovar` | JWT (cliente dono ou funcionário) | Aprova e devolve `link_pagamento` |
   | POST | `/billing/v1/orcamentos/{id}/recusar` | JWT | Recusa com motivo |
   | POST | `/billing/v1/webhooks/budget-response` | HMAC `X-Signature` | Migrado do OS (Fase 2) |
   | POST | `/billing/webhooks/mercadopago` | `x-signature` do MP | Notificação de pagamento |
   | GET | `/billing/v1/pagamentos/{id}` | JWT | Status do pagamento |
   | GET | `/health/live`, `/health/ready` | — | Probes |

3. **Dinheiro em `numeric(12,2)` no banco e inteiro de centavos no Go** (`type Centavos int64`). O OS envia `preco` como string decimal no payload (`"89.90"`) — converter com `shopspring/decimal` ou parse manual para centavos; **nunca** `float64` no Billing.
4. Migration `000001_billing.up.sql` com as tabelas do [modelo](arquitetura.md#62-billing-service-postgresql) + `outbox` e `mensagens_processadas` (mesmo DDL do OS).
5. Autorização: cliente (`tipo=cliente` no JWT) só aprova/recusa orçamento com `cliente_id == sub`; funcionário pode qualquer um.

**Critérios de aceite**
- [ ] `docker compose up` do repo sobe o serviço isolado com o próprio Postgres.
- [ ] `go test ./...` verde, lint zerado.

---

### F4-2.2 — Orçamento: gerar, enviar, aprovar, recusar, expirar

**Objetivo:** responder aos comandos `GerarOrcamento` e `CancelarOrcamento` e às ações do cliente.

**Passo a passo**

1. **`GerarOrcamento`** (consumidor): numa transação — inbox, `orcamentos` + `itens_orcamento` com `expira_em = now() + PRAZO_ORCAMENTO` (default 48 h; 2 min na demo), outbox `OrcamentoGerado`. Depois do commit, e-mail ao cliente (porta `Notifier`, mesmo SMTP/Mailpit da Fase 2) com o link `GET /billing/v1/orcamentos/{id}`.
   - Orçamento já existente para a OS → reenvia `OrcamentoGerado` com o mesmo `orcamento_id` (idempotência de negócio, além da inbox).
2. **Aprovar:** `gerado → aprovado`; cria a **order** no Mercado Pago (F4-2.3) **fora** da transação; grava `pagamentos(status=pendente, mp_order_id)` e outbox `OrcamentoAprovado{link_pagamento}`. Se o MP falhar, a aprovação **não** é gravada e o cliente recebe `502` — pode tentar de novo.
3. **Recusar:** `gerado → recusado` + outbox `OrcamentoRecusado`.
4. **Webhook `budget-response`** migrado: mesma validação HMAC da Fase 2 (`internal/infra/http/middleware/webhook.go`), traduzido em aprovar/recusar.
5. **Expiração:** o timeout de aprovação é do **orquestrador** (F4-4.6), não do Billing. O Billing só recusa aprovar um orçamento vencido (`422 ORCAMENTO_EXPIRADO`).
6. **`CancelarOrcamento`** (compensação): qualquer status não terminal → `cancelado`; se houver pagamento `aprovado` desse orçamento, **estorna antes** (F4-2.5); se houver order ainda não paga, **cancela-a** (`POST /v1/orders/{id}/cancel` — confira na referência quais status aceitam cancelamento; falha aqui só gera log, porque o passo 4 de F4-2.5 cobre o pagamento tardio). Responde `OrcamentoCancelado` mesmo se já estava cancelado.

**Critérios de aceite**
- [ ] Mesmo `GerarOrcamento` entregue duas vezes → um orçamento, duas respostas iguais.
- [ ] Aprovar orçamento de outro cliente → 403.
- [ ] E-mail do orçamento aparece no Mailpit.

---

### F4-2.3 — Integração com o Mercado Pago: cobrança

**Objetivo:** gerar o link de pagamento do orçamento aprovado.

**Passo a passo**

1. Porta no use case:

   ```go
   type gatewayPagamento interface {
   	CriarCobranca(ctx context.Context, c Cobranca) (ResultadoCobranca, error)       // POST /v1/orders
   	ConsultarCobranca(ctx context.Context, mpOrderID string) (Cobranca, error)      // GET  /v1/orders/{id}
   	Estornar(ctx context.Context, mpOrderID, chaveIdempotencia string) (Estorno, error) // POST /v1/orders/{id}/refund
   	Cancelar(ctx context.Context, mpOrderID string) error                           // POST /v1/orders/{id}/cancel
   }
   ```

2. Adaptador HTTP `internal/infra/mercadopago` sobre a **API de Orders** (decisão de 10/11 — a API de Preferences não estorna com conta de teste; ver F4-0.3). Sem SDK — são 4 chamadas:

   ```go
   // POST {MP_BASE_URL}/v1/orders — valores como STRING decimal ("89.90"), não float
   corpo := map[string]any{
   	"type":               "online",
   	"processing_mode":    "manual", // como no exemplo oficial do Checkout Pro (Orders)
   	"total_amount":       c.Valor.String(),
   	"external_reference": c.OrcamentoID.String(),
   	"payer":              map[string]any{"email": c.ClienteEmail},
   	"items": []map[string]any{{
   		"title":      "Orçamento " + c.NumeroOS,
   		"unit_price": c.Valor.String(),
   		"quantity":   1,
   	}},
   	// config: URLs de retorno/notificação e restrições — conferir os campos na referência
   }
   req.Header.Set("Authorization", "Bearer "+a.accessToken)
   req.Header.Set("X-Idempotency-Key", c.OrcamentoID.String())
   ```

   Resposta (`201`, validado): `id` (`ORDTST...` com credencial de teste) → `pagamentos.mp_order_id`; **`checkout_url`** → `link_pagamento`; `status: created`.

   Respostas observadas em 10/11:

   | Chamada | Resultado |
   |---|---|
   | `GET /v1/orders/{id}` após pagar | `status: processed`, `status_detail: accredited`, `total_paid_amount`, `transactions.payments[0].id` (`PAY...`) e `reference_id` (id numérico em `/v1/payments`) |
   | `POST /v1/orders/{id}/refund` sem corpo | `201`, `status: refunded`, `transactions.refunds[0].id` (`REF...`) |
   | Mesmo refund com a **mesma** `X-Idempotency-Key` | `201` com o **mesmo** `REF...` — idempotente |
   | Refund com chave **nova** em order já estornada | `409 order_already_refunded` → tratar como **sucesso** (já estornado) |
3. `MP_BASE_URL` configurável (`https://api.mercadopago.com` em prod, `http://mp-stub:8090` no compose e no BDD).
4. Timeout de 5 s, retry só em erro de rede/5xx (máx. 2), nunca em 4xx.

**Critérios de aceite**
- [ ] Teste do adaptador contra `httptest.Server` cobre 201, 400, 401, 5xx e timeout.
- [ ] Aprovação real em sandbox devolve um `checkout_url` que abre o Checkout Pro.
- [x] **Validado em 10/11 — cartão recusado:** a tentativa fica `failed / rejected_by_issuer`, mas a **order continua aberta** (`action_required / waiting_retry`) e aceita nova tentativa no mesmo link — até 20. `config.online.retries.allowed: false` na criação é **ignorado** pelo Checkout Pro (volta `true`, `max_attempts: 20`).
- [x] **Validado em 10/11 — cancelamento:** `POST /v1/orders/{id}/cancel` numa order `action_required` → `200`, order `canceled`; o checkout passa a mostrar "O que você queria pagar não está disponível" e não aceita pagamento.

---

### F4-2.4 — Webhook do Mercado Pago

**Objetivo:** registrar e **verificar** o pagamento com segurança e idempotência.

**Passo a passo**

1. Rota pública `POST /billing/webhooks/mercadopago` (sem authorizer no Gateway — F4-6.5).
2. **Validar `x-signature`** (formato `ts=<epoch>,v1=<hmac>`; **validado em 10/11** com notificações do painel: `ts` em **segundos** (10 dígitos), query `?data.id=<id>&type=order`, mesmo segredo para `order` e `payment`). O manifesto assinado é `id:<data.id>;request-id:<x-request-id>;ts:<ts>;`, com `data.id` vindo da **query string** (em minúsculas se alfanumérico — **atenção:** com a API de Orders o `data.id` é o id da order, alfanumérico e maiúsculo, `ORDTST...`; a regra das minúsculas passa a importar de verdade):

   ```go
   func assinaturaValida(segredo, xSignature, xRequestID, dataID string, agora time.Time) bool {
   	var ts, v1 string
   	for _, parte := range strings.Split(xSignature, ",") {
   		k, v, _ := strings.Cut(strings.TrimSpace(parte), "=")
   		switch k {
   		case "ts":
   			ts = v
   		case "v1":
   			v1 = v
   		}
   	}
   	if ts == "" || v1 == "" {
   		return false
   	}
   	// janela anti-replay de 5 min
   	if seg, err := strconv.ParseInt(ts, 10, 64); err != nil || agora.Sub(time.Unix(seg, 0)).Abs() > 5*time.Minute {
   		return false
   	}
   	manifesto := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", strings.ToLower(dataID), xRequestID, ts)
   	mac := hmac.New(sha256.New, []byte(segredo))
   	mac.Write([]byte(manifesto))
   	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(v1))
   }
   ```

   > Responder `200` em até **22 s** (senão o MP reenvia a cada 15 min). **Limitação do sandbox (10/11):** a aplicação só tem webhook de **produção** (`webhooks?env=production`) e o painel mostra *"Notificações enviadas: nenhuma"* para os 5 pagamentos de orders `ORDTST...` — **orders de teste não disparam webhook**. O handler só é exercitável por *Simular notificação* (validado) e pelo stub. Por isso existe o **reconciliador** (passo 7).

3. Assinatura inválida → `401`.
4. **Verificação ativa:** `ConsultarCobranca(data.id)` (`GET /v1/orders/{id}`); usar `external_reference` para achar o orçamento e `status`:

   | `status` da order | Ação |
   |---|---|
   | `processed` (`accredited`) | `pagamentos → aprovado` (`mp_order_id` UNIQUE) + outbox `PagamentoConfirmado` |
   | **`action_required` / `waiting_retry`** (tentativa recusada) | **`POST /v1/orders/{id}/cancel`** → se `200`/`canceled`: `pagamentos → recusado` + outbox `PagamentoRecusado{status_detail da tentativa}`. Se o cancel falhar porque a order já foi paga (corrida com uma 2ª tentativa), reconsultar e tratar como `processed` |
   | `failed`, `canceled`, `expired` | `pagamentos → recusado` + outbox `PagamentoRecusado` (idempotente se já recusado) |
   | `created`, `processing` | nada; aguarda próxima notificação |
   | `refunded` | conferência do estorno (F4-2.5) |

   > Observados em 10/11: `processed`, `refunded`, `action_required/waiting_retry` e `canceled`. **Por que cancelar antes de recusar:** a order recusada continua pagável e o Checkout Pro ignora `retries.allowed: false`; sem o cancel, o cliente paga depois que a saga já compensou.

5. **Ordem do handler (corrigida em 10/11)** — síncrono, nesta sequência:
   1. validar assinatura;
   2. `ConsultarCobranca(data.id)` — se falhar, responder **`500`** e **não gravar nada**, para o reenvio do MP ser processado de verdade;
   3. numa transação: atualizar `pagamentos` com `UPDATE ... WHERE mp_order_id = $1 AND status = 'pendente'` **+** outbox **+** `notificacoes_mp` (`ON CONFLICT (x_request_id) DO NOTHING`, só auditoria);
   4. responder `200`.

   A idempotência vem do **`mp_order_id` + transição de status**: a segunda notificação do mesmo pagamento não encontra mais `pendente` e não gera outra mensagem. *(O desenho anterior gravava `notificacoes_mp` antes do `GET` e respondia "200 sem efeito" no conflito — um `GET` falho fazia o pagamento nunca ser processado.)*
6. Pagamento `approved` para orçamento `cancelado` → estorno imediato (F4-2.5 passo 4), sem `PagamentoConfirmado`.
7. **Reconciliador (mecanismo principal de confirmação — decisão de 10/11).** Goroutine no Billing que, a cada `MP_RECONCILIAR_INTERVALO` (default 10 s), lista `pagamentos` com `status = 'pendente'` e chama a **mesma** função do handler a partir do passo 2 (`ConsultarCobranca` → tabela de status → transação). Sem líder único necessário: o Billing roda com 1 réplica (corte 7′), e a idempotência por `mp_order_id` + transição de status torna inofensiva a corrida com o webhook.
   - O webhook vira **acelerador**: quando chega (produção, ou *Simular notificação*), o processamento é imediato; quando não chega (sandbox), o reconciliador resolve em ≤ 10 s.
   - Parar de reconciliar um pagamento quando o orçamento sair de `aprovado` (pago, recusado, cancelado) — a consulta é só para pendentes.
   - Justificativa para a ADR-014: o webhook diz "algo mudou"; a verdade vem da API — o reconciliador é a mesma verificação ativa, sem depender da entrega.

**Critérios de aceite**
- [ ] Testes de assinatura: válida, `v1` errado, `ts` fora da janela, header ausente, `data.id` em maiúsculas.
- [ ] Mesma notificação reenviada 3× → um `PagamentoConfirmado`.
- [ ] `GET /v1/orders/{id}` falha (stub com 500) na primeira notificação e funciona no reenvio → um `PagamentoConfirmado`.
- [ ] **Sem nenhum webhook**, pagar uma order em sandbox → `PagamentoConfirmado` em ≤ 10 s (reconciliador).
- [ ] Webhook e reconciliador processando o mesmo pagamento ao mesmo tempo → um `PagamentoConfirmado`.
- [ ] Notificação com corpo forjado mas `data.id` de pagamento recusado → `PagamentoRecusado` (prova que a verdade vem do `GET`).

---

### F4-2.5 — Estorno e cancelamento (compensações do Billing)

**Objetivo:** desfazer o pagamento quando a saga falha depois dele.

**Passo a passo**

1. **`EstornarPagamento`**: buscar o pagamento `aprovado` do orçamento; `POST /v1/orders/{mp_order_id}/refund` sem corpo (estorno total) e `X-Idempotency-Key = message_id` do comando; `409 order_already_refunded` = sucesso; gravar `mp_refund_id` (`REF...`), `status = estornado` e outbox `PagamentoEstornado`.
2. Sem pagamento aprovado (falha antes do pagamento, ou já estornado) → responde `PagamentoEstornado{refund_id: null}` — compensação de algo que não aconteceu é sucesso.
3. Erro 5xx/rede do MP → **não** dá ack na mensagem: o SQS reentrega após o visibility timeout; após 5 tentativas vai para a DLQ e dispara alerta (F4-7.4). A chave de idempotência evita estorno duplo nas retentativas.
4. **Pagamento aprovado chegando depois da compensação** — **não cortável** no plano de 4 dias. A 1ª defesa é cancelar a order antes de emitir `PagamentoRecusado` (F4-2.4) e no `CancelarOrcamento` (F4-2.2); esta é a 2ª, para a corrida entre o cancel e uma nova tentativa: o webhook encontra o orçamento `cancelado` → estorna na hora e **não** envia `PagamentoConfirmado`.

**Critérios de aceite**
- [ ] Estorno visível no painel do MP para a OS de teste.
- [ ] Teste: MP responde 500 duas vezes e 201 na terceira → um estorno, uma resposta.
- [ ] Teste do pagamento tardio.

---

### F4-2.6 — Stub do Mercado Pago

**Objetivo:** permitir BDD e demo local sem browser e sem internet.

**Passo a passo**

1. Binário `cmd/mp-stub` no repo do Billing (entra na mesma imagem): implementa `POST /v1/orders`, `GET /v1/orders/{id}`, `POST /v1/orders/{id}/refund` (com o `409 order_already_refunded`) e `POST /v1/orders/{id}/cancel` em memória, com os mesmos formatos de resposta registrados em F4-2.3.
2. Rota de controle `POST /_stub/pagar {order_id, resultado: "processed"|"failed"}`: cria o pagamento (a order passa a `processed` ou `action_required`); o reconciliador do Billing encontra sozinho. Opcional: `?notificar=true` também **chama o webhook do Billing** com `x-signature` válido (mesmo segredo do compose), para exercitar o handler no BDD.
3. **Nunca** habilitado em produção: o manifesto `k8s/overlays/prod` não referencia o stub, e o `MP_BASE_URL` de prod vem do ConfigMap.

**Critérios de aceite**
- [ ] BDD (F4-5.2) roda o cenário de pagamento recusado só com o stub.

---

## E3 — Execution Service

### F4-3.1 — Esqueleto com DynamoDB

**Objetivo:** repo `oficina-execution-service` no mesmo padrão, com persistência em DynamoDB via `aws-sdk-go-v2`.

**Passo a passo**

1. Mesma cópia de F4-2.1 (sem `golang-migrate`; a tabela é do Terraform).
2. Repositório `internal/infra/repository/dynamo` com o [design single-table](arquitetura.md#63-execution-service-dynamodb-single-table).
3. **Outbox e inbox no DynamoDB** — o efeito, a marca de processada e a mensagem de saída num `TransactWriteItems`:

   ```go
   _, err := r.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
   	TransactItems: []types.TransactWriteItem{
   		{Put: &types.Put{ // inbox: falha se já processada
   			TableName:           r.tabela,
   			Item:                itemInbox(env.MessageID),
   			ConditionExpression: aws.String("attribute_not_exists(pk)"),
   		}},
   		{Put: &types.Put{TableName: r.tabela, Item: itemTarefa(tarefa)}},
   		{Put: &types.Put{TableName: r.tabela, Item: itemOutbox(resposta)}},
   	},
   })
   var cancel *types.TransactionCanceledException
   if errors.As(err, &cancel) && cancel.CancellationReasons[0].Code != nil &&
   	*cancel.CancellationReasons[0].Code == "ConditionalCheckFailed" {
   	return ErrJaProcessada // ack sem efeito
   }
   ```

4. Relay da outbox: `Query gsi1 gsi1pk = OUTBOX#PENDENTE` em ordem de `gsi1sk`, publica, e `UpdateItem REMOVE gsi1pk, gsi1sk` (item sai do índice esparso). Item de inbox com `ttl` de 7 dias.
   **O relay do DynamoDB não tem líder único** — por isso o Execution roda com **1 réplica fixa** (corte 7′). Com 2+ réplicas, mensagens da mesma saga podem sair fora de ordem; a evolução é um *lease* em `RELAY#LIDER` (`UpdateItem` condicional em `expira_em < :agora`).

**Critérios de aceite**
- [ ] Testes do repositório contra DynamoDB Local/LocalStack no CI (job de integração, como o `integration.yml` do OS).
- [ ] Reentrega do mesmo comando → `ErrJaProcessada`, sem tarefa duplicada.

---

### F4-3.2 — Fila de execução e tarefas

**Objetivo:** "gerenciar a fila de execução da OS".

**Passo a passo**

1. Entidade `Tarefa{ID, OsID, SagaID, Tipo(DIAGNOSTICO|REPARO), Estado, Mecanico, Prioridade, ...}` com máquina de estados: `AGUARDANDO → EM_ANDAMENTO → CONCLUIDA`, e `AGUARDANDO|EM_ANDAMENTO → CANCELADA`.
2. Prioridade: `REPARO` antes de `DIAGNOSTICO` (carro já pago esperando é o pior cenário para a oficina); dentro do mesmo tipo, FIFO por `criado_em`. `gsi1sk = <prioridade>#<criado_em>`.
3. ✂️ *(corte 19)* **Pegar a próxima** sem dois mecânicos pegarem a mesma: `Query` no GSI + `UpdateItem` condicional `estado = AGUARDANDO`; se falhar a condição, tenta a próxima (máx. 3).

**Critérios de aceite**
- [ ] `GET /execution/v1/fila?tipo=REPARO` lista as tarefas na ordem do GSI (prioridade, `criado_em`).
- [ ] ~~Teste concorrente de `fila/proxima`~~ ✂️ corte 19.

---

### F4-3.3 — API do mecânico

| Método | Rota | Efeito | Resposta à saga |
|---|---|---|---|
| GET | `/execution/v1/fila?tipo=` | Lista tarefas aguardando, em ordem | — |
| POST | ✂️ `/execution/v1/fila/proxima` *(corte 19)* | Atribui a próxima ao mecânico do JWT | — |
| POST | `/execution/v1/tarefas/{id}/concluir-diagnostico` | `{diagnostico}` → `CONCLUIDA` | `DiagnosticoConcluido` |
| POST | `/execution/v1/tarefas/{id}/iniciar` | Tarefa → `EM_ANDAMENTO`, mecânico do JWT | `DiagnosticoIniciado` (diagnóstico) · `ExecucaoIniciada` (reparo) |
| POST | `/execution/v1/tarefas/{id}/concluir` | Reparo → `CONCLUIDA` | `ExecucaoConcluida` |
| GET | `/execution/v1/tarefas?os_id=` | Tarefas de uma OS | — |

Autorização: `tipo=usuario` com papel `mecanico` ou `administrador`. `diagnostico` obrigatório (regra que já existia no OS — migra junto com o teste).

**Critérios de aceite**
- [ ] OpenAPI do serviço gerando os handlers com `oapi-codegen`.
- [ ] Concluir tarefa já cancelada → `409 TAREFA_CANCELADA`.

---

### F4-3.4 — Comandos, respostas e cancelamento

**Passo a passo**

1. Consumidor da `execution-commands.fifo`:
   - `IniciarDiagnostico` → cria tarefa `DIAGNOSTICO/AGUARDANDO`;
   - `ExecutarReparo` → cria tarefa `REPARO/AGUARDANDO`. **Ponto de falha da demo** (F4-4.7);
   - `CancelarTarefas` → todas as tarefas não concluídas da OS viram `CANCELADA`; responde `TarefasCanceladas{quantidade}` (0 é resposta válida).
2. Erro de validação do comando (payload inválido) → resposta de falha (`DiagnosticoFalhou`/`ExecucaoFalhou`) e **ack** — não adianta reentregar mensagem malformada. Erro de infraestrutura (DynamoDB fora) → **sem ack**, reentrega.

**Critérios de aceite**
- [ ] Tabela de testes cobrindo os 3 comandos × (sucesso, duplicado, inválido, erro de infra).

---

## E4 — Saga e mensageria

### F4-4.1 — Filas SQS FIFO, DLQ e contrato SSM

**Objetivo:** cada serviço provisiona a **sua fila de entrada** no próprio `infra/`.

```hcl
# oficina-billing-service/infra/filas.tf
resource "aws_sqs_queue" "comandos_dlq" {
  name                      = "billing-commands-dlq.fifo"
  fifo_queue                = true
  message_retention_seconds = 1209600 # 14 dias para investigar
}

resource "aws_sqs_queue" "comandos" {
  name                        = "billing-commands.fifo"
  fifo_queue                  = true
  deduplication_scope         = "messageGroup"
  fifo_throughput_limit       = "perMessageGroupId"
  visibility_timeout_seconds  = 60   # > tempo máximo de processamento (inclui chamada ao MP)
  receive_wait_time_seconds   = 20   # long polling: menos requisições, custo ~0
  message_retention_seconds   = 345600
  sqs_managed_sse_enabled     = true
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.comandos_dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_ssm_parameter" "fila_comandos_url" {
  name  = "/oficina/prod/billing/fila-comandos-url"
  type  = "String"
  value = aws_sqs_queue.comandos.url
}
```

Permissões (identity policy na role do **remetente**, ARN por convenção — evita a [dependência circular](arquitetura.md#7-topologia-dos-repositórios)):

```hcl
# oficina-app/infra/iam.tf — o OS envia comandos para as filas dos outros
data "aws_caller_identity" "atual" {}
locals {
  fila_arn = { for f in ["billing-commands", "execution-commands"] :
    f => "arn:aws:sqs:us-east-1:${data.aws_caller_identity.atual.account_id}:${f}.fifo" }
}
data "aws_iam_policy_document" "os_pod" {
  statement {
    actions   = ["sqs:SendMessage"]
    resources = values(local.fila_arn)
  }
  statement {
    actions   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:ChangeMessageVisibility", "sqs:GetQueueAttributes"]
    resources = [aws_sqs_queue.respostas.arn]
  }
}
```

**Critérios de aceite**
- [ ] Os três `infra/` aplicam em qualquer ordem a partir do zero.
- [ ] Uma mensagem que falha 5× aparece na DLQ correspondente.

---

### F4-4.2 — Outbox e relay

**Objetivo:** publicar tudo que foi gravada na `outbox`, em ordem, com pelo menos uma entrega.

**Passo a passo**

1. `Gravar(ctx, tx, destino, env, traceHeaders)` — `INSERT` na mesma `tx` do efeito. Os headers de trace são capturados **na gravação** (a transação do New Relic da requisição já terá terminado quando o relay rodar).
2. Relay em goroutine, a cada 500 ms, **com um único relay ativo por serviço** (advisory lock), preservando a ordem por saga mesmo com várias réplicas:

   ```go
   func (r *Relay) ciclo(ctx context.Context) error {
   	conn, err := r.db.Conn(ctx)
   	if err != nil {
   		return err
   	}
   	defer conn.Close()

   	var lider bool
   	if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, chaveRelay).Scan(&lider); err != nil || !lider {
   		return err // outra réplica é a líder
   	}
   	defer conn.ExecContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, chaveRelay)

   	linhas, err := conn.QueryContext(ctx, `
   		SELECT id, destino, envelope, trace_headers FROM outbox
   		WHERE enviado_em IS NULL AND falhou_em IS NULL ORDER BY criado_em LIMIT 50`)
   	// ... para cada linha: Publicar; em sucesso UPDATE enviado_em = now();
   	//     em erro UPDATE tentativas = tentativas + 1 e PARA o ciclo
   	//     (não pula mensagem: pular quebraria a ordem da saga)
   	//     com tentativas >= 10: UPDATE falhou_em = now() + log error com saga_id —
   	//     a linha sai do índice e deixa de travar as OUTRAS sagas (head-of-line)
   }
   ```

3. Limpeza: `DELETE FROM outbox WHERE enviado_em < now() - interval '7 days'` diário.
4. Métrica: `SELECT count(*) FROM outbox WHERE enviado_em IS NULL` exposta no `/health` e como evento — outbox crescendo = SQS inacessível.

**Critérios de aceite**
- [ ] Teste de integração: 3 réplicas do relay em goroutines, 100 mensagens de 10 sagas → cada saga recebe suas mensagens em ordem.
- [ ] Derrubar o SQS (parar o LocalStack) por 1 min e religar → nenhuma mensagem perdida.

---

### F4-4.3 — Consumidor e inbox idempotente

**Passo a passo**

1. Loop de `ReceiveMessage` (long polling 20 s, até 10 mensagens, `MessageAttributeNames: ["All"]`).
2. Por mensagem: iniciar transação do New Relic aceitando os headers de trace (F4-7.1) → abrir tx → `INSERT INTO mensagens_processadas ... ON CONFLICT DO NOTHING` → se `RowsAffected == 0`, **commit e ack** (duplicata) → senão, handler do tipo na mesma tx → commit → `DeleteMessage`.
3. Classificação de erro:

   | Erro | Ação |
   |---|---|
   | Duplicata | ack |
   | Tipo desconhecido / payload inválido | log `error` + ack (vai para análise pelo log, não pela DLQ) |
   | Erro de negócio esperado | resposta de falha à saga + ack |
   | Erro de infraestrutura | **sem ack** → reentrega → DLQ após 5 |

4. Shutdown gracioso: no `SIGTERM`, parar de receber, terminar as mensagens em voo (≤ visibility timeout) e só então sair — o `terminationGracePeriodSeconds` do pod precisa ser > 60 s.

**Critérios de aceite**
- [ ] Teste: mesma mensagem processada 2× → um efeito.
- [ ] `kubectl rollout restart` durante uma saga não perde nem duplica etapa.

---

### F4-4.4 — Orquestrador

**Objetivo:** a máquina de estados da [arquitetura §3.2](arquitetura.md#32-máquina-de-estados-da-saga) em código puro e testável.

**Passo a passo**

1. Pacote `internal/application/saga`, **sem dependência de infraestrutura**: recebe `(saga, resposta)` e devolve `Decisao{NovoEstado, NovoStatusOS, Comandos[], Etapa}`.

   ```go
   type chave struct {
   	estado Estado
   	tipo   string
   }

   // fluxo é a saga inteira em uma tabela. Ler esta tabela é ler o processo.
   var fluxo = map[chave]passo{
   	{DiagnosticoPendente, "DiagnosticoConcluido"}: {para: OrcamentoPendente, comando: "GerarOrcamento", etapaOK: "T2"},
   	{OrcamentoPendente, "OrcamentoGerado"}:         {para: AguardandoAprovacao, statusOS: entity.StatusAguardandoAprovacao, etapaOK: "T3"},
   	{AguardandoAprovacao, "OrcamentoAprovado"}:     {para: AguardandoPagamento, etapaOK: "T4"},
   	{AguardandoPagamento, "PagamentoConfirmado"}:   {para: ExecucaoPendente, comando: "ExecutarReparo", etapaOK: "T5"},
   	{ExecucaoPendente, "ExecucaoIniciada"}:         {para: EmExecucao, statusOS: entity.StatusEmExecucao, baixarReserva: true, etapaOK: "T6"}, // pivô
   	{EmExecucao, "ExecucaoConcluida"}:              {para: Concluida, statusOS: entity.StatusFinalizada, etapaOK: "T7"},
   }
   // A saga nasce em DiagnosticoPendente com T1 ok (F4-1.3) — não há estado INICIADA.
   // prazo: só documentação nesta entrega (corte 5); o campo existe para o watchdog futuro.

   // falhas leva cada resposta de falha ao início da compensação.
   var falhas = map[chave]string{ // valor = etapa que falhou
   	{DiagnosticoPendente, "DiagnosticoFalhou"}:  "T2",
   	{OrcamentoPendente, "OrcamentoFalhou"}:      "T3",
   	{AguardandoAprovacao, "OrcamentoRecusado"}:  "T4",
   	{AguardandoPagamento, "PagamentoRecusado"}:  "T5",
   	{ExecucaoPendente, "ExecucaoFalhou"}:        "T6",
   }
   ```

2. Aplicação da decisão (no use case, numa tx): `UPDATE sagas_os SET ..., versao = versao + 1 WHERE id = $1 AND versao = $2` (lock otimista — 0 linhas = outra réplica avançou; retorna erro de infra e a mensagem é reentregue), `AvancarStatus` da OS, `sagas_os_etapas`, `outbox`.
3. **Resposta inesperada para o estado atual** (`fluxo` e `falhas` sem a chave): registrar etapa `ignorada` e ack — **exceto** `PagamentoConfirmado` em `COMPENSANDO/COMPENSADA`, que gera `EstornarPagamento` (pagamento tardio, defesa em profundidade com F4-2.5).
4. Eventos informativos (`DiagnosticoIniciado`) só mudam o status da OS para `em_diagnostico`, sem mudar o estado da saga.

**Critérios de aceite**
- [ ] Teste de tabela cobrindo **todas** as combinações `estado × tipo` (as válidas, as de falha e as inesperadas).
- [ ] Pacote `saga` com 100% de cobertura (é lógica pura — não há desculpa).

---

### F4-4.5 — Compensações

**Passo a passo**

1. Ao entrar em `COMPENSANDO`, o orquestrador monta a **pilha** de compensações a partir das etapas `ok` registradas em `sagas_os_etapas`, em ordem inversa:

   | Etapa concluída | Compensação | Resposta esperada |
   |---|---|---|
   | T5 (pagamento) | `EstornarPagamento` → Billing | `PagamentoEstornado` |
   | T3 (orçamento) | `CancelarOrcamento` → Billing | `OrcamentoCancelado` |
   | T2 (diagnóstico) / reparo criado | `CancelarTarefas` → Execution | `TarefasCanceladas` |
   | T1 (reserva) | `LiberarReserva` — **local**, na mesma tx da última resposta | — |

2. **Uma compensação por vez**: envia a próxima só quando chega a resposta da anterior. Mais lento, mas mantém a regra "no máximo uma mensagem pendente por saga" de que o relay depende para garantir ordem.
3. Status da OS vai para `cancelada` **no início** da compensação (o cliente não deve ver "aguardando pagamento" enquanto o estorno corre); `cancelado_em` e `motivo_cancelamento` preenchidos.
4. `COMPENSADA` só após a última resposta + `LiberarReserva`. Notificação ao cliente por e-mail ("OS cancelada — pagamento estornado", se for o caso).
5. Pilha por estado (para conferência rápida): T2 falhou → [`LiberarReserva`]; T3 → [`CancelarTarefas`, `LiberarReserva`]; T4/T5 → [`CancelarOrcamento`, `CancelarTarefas`, `LiberarReserva`]; T6 → [`EstornarPagamento`, `CancelarOrcamento`, `CancelarTarefas`, `LiberarReserva`]. Não existe `DevolverEstoque`: a baixa só acontece no pivô (T6 ok), depois do qual não se compensa.

**Critérios de aceite**
- [ ] Os 5 cenários da [tabela de falhas](arquitetura.md#33-etapas-e-compensações) testados no pacote `saga`; no BDD, a falha T6 (`FAL*`) com estorno (corte 20).
- [ ] Estoque igual ao inicial e pagamento `estornado` ao fim do cenário T6.

---

### F4-4.6 — Timeouts e watchdog ✂️ *(corte 5 — evolução registrada na RFC-006)*

**Passo a passo**

1. `prazo_em` definido ao entrar em estados com espera humana (`AGUARDANDO_APROVACAO`: `SAGA_PRAZO_APROVACAO`, default 48 h; `AGUARDANDO_PAGAMENTO`: `SAGA_PRAZO_PAGAMENTO`, default 24 h) e em estados que esperam resposta de máquina (`*_PENDENTE`: 15 min).
2. Watchdog a cada 60 s (mesma técnica de líder único do relay):

   ```sql
   SELECT id FROM sagas_os
   WHERE estado NOT IN ('CONCLUIDA', 'COMPENSADA') AND prazo_em < now()
   ORDER BY prazo_em LIMIT 20 FOR UPDATE SKIP LOCKED;
   ```

   - estados de espera humana → trata como falha da etapa (inicia compensação, `motivo = prazo_expirado`);
   - `*_PENDENTE` e `COMPENSANDO` vencidos → **não compensa sozinho** (pode ser só lentidão): emite `SagaPresaEvent` para o alerta (F4-7.4) e estende o prazo.
3. Na demo, `SAGA_PRAZO_APROVACAO=2m` no ConfigMap para mostrar a expiração ao vivo.

**Critérios de aceite**
- [ ] Teste com relógio injetado (`Relogio` como porta — o use case já recebe `time.Now()` em vários pontos; centralize).

---

### F4-4.7 — Injeção de falha para demonstração

**Objetivo:** provocar a falha de T6 (pagamento já feito) de forma **determinística** no vídeo e no BDD.

**Passo a passo**

1. No Execution: se `PERMITIR_FALHA_SIMULADA=true` **e** a placa do veículo no payload começa com `FAL`, `ExecutarReparo` responde `ExecucaoFalhou{motivo: "falha_simulada"}`.
2. A flag é `false` por padrão e só é ligada no ConfigMap durante a gravação; documentar no README do serviço que é um gatilho de demonstração.
3. Alternativas para outros pontos: `GerarOrcamento` falha para OS com descrição `[falha:orcamento]` — só se sobrar tempo.

> Por que placa: atravessa os 3 serviços no payload, funciona com várias réplicas sem estado compartilhado e é visível na tela ("este é o carro que vai falhar").

**Critérios de aceite**
- [ ] Veículo `FAL1A23` → saga `COMPENSADA` com estorno; qualquer outra placa → fluxo normal.

---

## E5 — Testes e qualidade

### F4-5.1 — Cobertura ≥ 80% por serviço, com gate no CI

**Passo a passo**

1. Mesmas exclusões no gate e no Sonar — **uma lista só**, em `scripts/cobertura.sh`:

   ```bash
   #!/usr/bin/env bash
   # Mantenha EXCLUSOES igual a sonar.coverage.exclusions em sonar-project.properties.
   set -euo pipefail
   MINIMO="${MINIMO:-80}"
   EXCLUSOES='(api\.gen\.go|/cmd/|/internal/infra/repository/|/internal/infra/database/|/scripts/)'

   go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
   grep -vE "$EXCLUSOES" coverage.out > coverage.unit.out
   total=$(go tool cover -func=coverage.unit.out | awk '/^total:/ { sub("%", "", $3); print $3 }')
   echo "Cobertura (critério do Sonar): ${total}%"
   echo "cobertura=${total}" >> "${GITHUB_OUTPUT:-/dev/null}"
   awk -v t="$total" -v m="$MINIMO" 'BEGIN { exit !(t >= m) }' \
     || { echo "::error::Cobertura ${total}% abaixo do mínimo de ${MINIMO}%"; exit 1; }
   ```

2. Job `Cobertura ≥ 80%` no `ci.yml` de cada repo chamando o script; resultado no Job Summary.
3. README de cada serviço mostra **o número e o critério** ("unitária, excluindo código gerado e adaptadores de banco cobertos por integração") + badge do Sonar. Seja transparente: o avaliador pode rodar `go test -cover` e ver um número menor (achado 7).
4. Para o Billing e o Execution, escreva os testes **junto** com o código: o gate quebra o PR, e recuperar cobertura no fim custa o dobro.

**Critérios de aceite**
- [ ] Os 3 repos com o job obrigatório no ruleset e ≥ 80%.
- [ ] Print do relatório do Sonar em cada README.

---

### F4-5.2 — BDD do fluxo completo (godog)

**Objetivo:** "pelo menos um fluxo completo testado com BDD" — a saga de ponta a ponta, contra os 3 serviços reais.

> **Plano de 4 dias (corte 20):** obrigatórios só o 1º (caminho feliz) e o 4º (falha no reparo com estorno). Os outros dois entram se sobrar tempo no D3.

**Passo a passo**

1. `go get github.com/cucumber/godog` no `oficina-app`; `features/saga_os.feature`:

   ```gherkin
   # language: pt
   Funcionalidade: Ciclo de vida da ordem de serviço coordenado por saga
     Para atender clientes de várias filiais com segurança
     Como oficina
     Quero que abertura, orçamento, pagamento e execução se completem ou se desfaçam juntos

     Contexto:
       Dado um cliente ativo com o veículo de placa "ABC1D23"
       E a peça "Filtro de óleo" com 5 unidades em estoque

     Cenário: OS orçada, paga e executada
       Quando o atendente abre uma OS com 1 "Filtro de óleo"
       Então a OS fica "recebida" e 1 unidade de "Filtro de óleo" fica reservada
       Quando o mecânico conclui o diagnóstico "troca do filtro"
       Então eventualmente a OS fica "aguardando_aprovacao" com um orçamento gerado
       Quando o cliente aprova o orçamento
       E o Mercado Pago aprova o pagamento
       E o mecânico inicia e conclui o reparo
       Então eventualmente a OS fica "finalizada"
       E "Filtro de óleo" tem 4 unidades e nenhuma reservada
       E a saga termina "CONCLUIDA"

     Cenário: cliente recusa o orçamento
       Quando o atendente abre uma OS com 1 "Filtro de óleo"
       E o mecânico conclui o diagnóstico "troca do filtro"
       E eventualmente a OS fica "aguardando_aprovacao" com um orçamento gerado
       Quando o cliente recusa o orçamento
       Então eventualmente a OS fica "cancelada"
       E "Filtro de óleo" tem 5 unidades e nenhuma reservada
       E a saga termina "COMPENSADA"

     Cenário: pagamento recusado
       # ... até a aprovação
       Quando o Mercado Pago recusa o pagamento
       Então eventualmente a OS fica "cancelada"
       E a saga termina "COMPENSADA"

     Cenário: falha no reparo estorna o pagamento
       Dado um cliente ativo com o veículo de placa "FAL1A23"
       # ... até o pagamento aprovado
       Então eventualmente a OS fica "cancelada"
       E o pagamento do orçamento está "estornado"
       E "Filtro de óleo" tem 5 unidades e nenhuma reservada
       E a saga termina "COMPENSADA"
   ```

2. Steps em `features/steps_test.go` falando **só HTTP** com os 3 serviços (nada de acessar banco no teste — a mesma regra vale para o BDD). "Eventualmente" = polling a cada 200 ms por até 15 s.
3. Pagamento via `POST /_stub/pagar` do stub (F4-2.6).
4. Workflow `bdd.yml` no `oficina-app`: sobe `docker-compose.saga.yml` com o OS buildado do PR e Billing/Execution na tag `latest` do Docker Hub, roda `go test ./features -tags bdd`, publica o relatório (`--format cucumber:bdd.json`) como artefato.
5. Nos repos de Billing e Execution, o mesmo workflow roda com **eles** buildados do PR e os outros em `latest` — o BDD vira o teste de integração entre serviços nos três.

**Critérios de aceite**
- [ ] 2 cenários obrigatórios verdes (no CI ou, no pior caso, localmente com o relatório anexado — sacrifício 5), em < 5 min.
- [ ] Relatório do BDD linkado no README.

---

### F4-5.3 — Testes de contrato das mensagens ✂️ *(corte 13)*

**Passo a passo**

1. Em cada repo, um teste que **produz** cada mensagem que o serviço envia e valida contra o JSON Schema de `docs/mensagens/` (copiado com um script `make sync-contratos` que baixa do `oficina-app` pela API do GitHub).
2. E o inverso: cada consumidor desserializa os exemplos (`docs/mensagens/exemplos/*.json`) sem erro.
3. Lib: `github.com/santhosh-tekuri/jsonschema/v6` (já é dependência indireta do `oficina-app`).

**Critérios de aceite**
- [ ] Renomear um campo do payload em um serviço quebra o teste de contrato **no PR**, não em produção.

---

### F4-5.4 — SonarCloud nos três repos

**Passo a passo**

1. `sonar-project.properties` em cada repo com as exclusões do script de cobertura.
2. Job `SonarCloud` com `SonarSource/sonarqube-scan-action@v8` (a `sonarcloud-github-action` está arquivada — achado da Fase 3) e **quality gate aguardado** (`sonar.qualitygate.wait=true`), para o check falhar de verdade.
3. Quality gate: "Sonar way" + cobertura em código novo ≥ 80%.

**Critérios de aceite**
- [ ] Check `SonarCloud` obrigatório e verde nos 3.

---

## E6 — CI/CD e infraestrutura

### F4-6.1 — Pipelines independentes por serviço

**Objetivo:** cada repo tem build → testes → qualidade → deploy no EKS, sem depender dos outros.

| Workflow | Gatilho | Jobs |
|---|---|---|
| `ci.yml` | PR → `homolog`, `main` | `Build e testes`, `Lint`, `Cobertura ≥ 80%`, `Contratos`, `Integração` (Postgres/DynamoDB Local) |
| `bdd.yml` | PR → `main` | `BDD — saga` |
| `security.yml` | PR + semanal | `gitleaks`, `govulncheck`, `gosec`, `Trivy`, `SonarCloud` |
| `terraform.yml` | PR (plan comentado) / push `main` (apply) | `plan`, `apply` (`environment: prod`) |
| `cd.yml` | push `main` | `Testes` → `Build e push` → `Deploy no EKS` → `Smoke test` → `Change tracking NR` |

Pontos de atenção no `cd.yml` (base: o do `oficina-app`, que já funciona):

1. **`needs: testes`** antes do build — o vídeo pede "deploy automatizado com validação de testes"; mostrar o job de testes no mesmo run é a evidência.
2. ConfigMap com as URLs das filas lidas da SSM no deploy:

   ```yaml
   - name: Config do ambiente (SSM → ConfigMap)
     run: |
       FILA=$(aws ssm get-parameter --name /oficina/prod/os/fila-respostas-url --query Parameter.Value --output text)
       kubectl -n "$NAMESPACE" create configmap billing-filas \
         --from-literal=FILA_RESPOSTAS_URL="$FILA" \
         --from-literal=FILA_COMANDOS_URL="$(aws ssm get-parameter --name /oficina/prod/billing/fila-comandos-url --query Parameter.Value --output text)" \
         --dry-run=client -o yaml | kubectl apply -f -
   ```

3. `kustomize edit set image` com o SHA, `rollout status`, rollback automático só se `apply`/`rollout` falharam (mesma lógica do `oficina-app`).
4. Smoke test: `curl -fsS https://<gateway>/<prefixo>/health/ready`.
5. `DEPLOY_ENABLED` e `workflow_dispatch` como no `oficina-app`.

**Critérios de aceite**
- [ ] Merge na `main` de cada repo implanta **só** aquele serviço.
- [ ] Um teste quebrado num PR impede o merge; um teste quebrado direto na `main` (simulado) impede o deploy.

---

### F4-6.2 — Terraform por serviço

**Objetivo:** "cada um com seu próprio repositório, infraestrutura e banco de dados".

| Repo | `infra/` provisiona |
|---|---|
| `oficina-app` | `os-saga-replies.fifo` + DLQ, role do pod (Pod Identity) com SQS + Secrets, associação ao namespace `oficina-prod`, parâmetros SSM |
| `oficina-billing-service` | `billing-commands.fifo` + DLQ, **RDS PostgreSQL `billing`** (`db.t4g.micro`, criptografado, subnet privada, SG aceitando só o SG dos nós), segredo `oficina/prod/billing`, role do pod, SSM |
| `oficina-execution-service` | `execution-commands.fifo` + DLQ, **tabela DynamoDB** (on-demand, PITR, TTL), role do pod com acesso só à tabela, SSM |

Backend: o mesmo bucket `oficina-tfstate-706215605178`, chave `<repo>/terraform.tfstate`, `use_lockfile = true`. Rede e cluster lidos da SSM publicada pelo `oficina-infra-k8s`.

Pod Identity (exemplo do Execution):

```hcl
resource "aws_iam_role" "pod" {
  name = "oficina-execution-pod"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "pods.eks.amazonaws.com" }
      Action    = ["sts:AssumeRole", "sts:TagSession"]
    }]
  })
}

resource "aws_iam_role_policy" "pod" {
  role = aws_iam_role.pod.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem", "dynamodb:Query", "dynamodb:TransactWriteItems", "dynamodb:ConditionCheckItem"]
        Resource = [aws_dynamodb_table.tarefas.arn, "${aws_dynamodb_table.tarefas.arn}/index/*"]
      },
      {
        Effect   = "Allow"
        Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:ChangeMessageVisibility", "sqs:GetQueueAttributes"]
        Resource = aws_sqs_queue.comandos.arn
      },
      {
        Effect   = "Allow"
        Action   = "sqs:SendMessage"
        Resource = "arn:aws:sqs:us-east-1:${data.aws_caller_identity.atual.account_id}:os-saga-replies.fifo"
      }
    ]
  })
}

resource "aws_eks_pod_identity_association" "execution" {
  cluster_name    = "oficina"
  namespace       = "execution-prod"
  service_account = "execution"
  role_arn        = aws_iam_role.pod.arn
}
```

**Critérios de aceite**
- [ ] `terraform destroy` de um serviço não afeta os outros.
- [ ] `aws iam simulate-principal-policy` prova que a role do Execution **não** acessa o RDS nem as filas do Billing (print para o PDF).

---

### F4-6.3 — Manifestos Kubernetes por serviço

**Passo a passo**

1. Em cada repo, `k8s/base` (Namespace, ServiceAccount, Deployment, Service `LoadBalancer`, HPA, ConfigMap) + `k8s/overlays/prod` — mesmo formato do `oficina-app`.
2. Deployment: probes `/health/live` e `/health/ready` (ready inclui banco **e** SQS), `terminationGracePeriodSeconds: 90` (F4-4.3), `resources.requests` modestos (`cpu: 50m`, `memory: 64Mi`) — o cluster é pequeno.
3. HPA: OS 2–5 (como na Fase 3); Billing e Execution com **1 réplica fixa, sem HPA** (corte 7′ — garante a ordem do relay do DynamoDB, ver F4-3.1).
4. **Manter o OS no namespace `oficina-prod`** da Fase 3 em vez de criar `os-prod`: trocar o namespace troca o Service e, com ele, o DNS do CLB para o qual o API Gateway aponta. Os serviços novos usam `billing-prod` e `execution-prod`.

**Critérios de aceite**
- [ ] `kubectl get pods -A` mostra os 3 serviços `Running` com o número mínimo de réplicas.

---

### F4-6.4 — Capacidade do cluster e addons

**Objetivo:** caber 3 serviços + `nri-bundle` no cluster sem pods `Pending` (achado 8).

**Passo a passo**

1. Medir antes de decidir:

   ```bash
   kubectl get nodes -o custom-columns=NOME:.metadata.name,PODS:.status.allocatable.pods
   kubectl get pods -A -o wide --field-selector spec.nodeName=<no> | wc -l
   kubectl describe node <no> | sed -n '/Allocated resources/,/Events/p'
   ```

2. Opções, na ordem de preferência:
   - **3 × `t3.small`** (`desired_size = 3`, `max_size = 4`) — +US$ 0,50/dia, cabe na quota de 8 vCPUs;
   - **prefix delegation** no VPC CNI (`ENABLE_PREFIX_DELEGATION=true` no addon `vpc-cni` + `max-pods` ajustado no node group) — sobe para ~110 pods/nó sem custo, mas exige recriar os nós e é mais um ponto de falha; só se a opção anterior não bastar;
   - `t3.medium` — 17 pods e 4 GB, ~US$ 2/dia a mais.
3. Addon **`eks-pod-identity-agent`** no `oficina-infra-k8s` (DaemonSet — conta 1 pod por nó no orçamento de pods).
4. Publicar na SSM o que os serviços leem: `cluster-name`, `sg-nos`, `subnets-privadas`.

**Critérios de aceite**
- [ ] HPA do OS escala a 5 sob carga com Billing e Execution no mínimo, sem pod `Pending`.

---

### F4-6.5 — Rotas no API Gateway

**Passo a passo**

No `oficina-lambda-auth` (dono das rotas desde o dia 5 da Fase 3):

| Rota | Integração | Authorizer |
|---|---|---|
| `ANY /v1/{proxy+}` | CLB do OS | sim (já existe) |
| `ANY /billing/v1/{proxy+}` | CLB do Billing | sim |
| `POST /billing/webhooks/mercadopago` | CLB do Billing | **não** — validação por `x-signature` |
| `POST /billing/v1/webhooks/budget-response` | CLB do Billing | **não** — HMAC |
| `ANY /execution/v1/{proxy+}` | CLB do Execution | sim |
| `/internal/*` | — | **sem rota** (não exposto) |

O DNS de cada CLB vem da SSM (`/oficina/prod/<servico>/lb-dns`), publicado pelo CD de cada serviço após o rollout. Manter o header compartilhado da Fase 3 (corte 1) em todas as integrações.

**Critérios de aceite**
- [ ] `curl https://<gateway>/billing/v1/orcamentos/x` sem token → 401 no Gateway.
- [ ] Webhook simulado pelo painel do MP chega ao Billing (log com `x-request-id`) e passa na validação do `x-signature` (orders de teste não disparam webhook real — F4-2.4).

---

## E7 — Observabilidade distribuída

### F4-7.1 — Trace distribuído atravessando as filas

**Objetivo:** um trace do New Relic que mostra `POST /v1/work-orders` → `IniciarDiagnostico` no Execution → ... (requisito "rastreamento dos fluxos distribuídos").

**Passo a passo**

1. Na gravação da outbox (dentro da transação HTTP ou de consumo), capturar os headers:

   ```go
   func traceDe(ctx context.Context) map[string]string {
   	txn := newrelic.FromContext(ctx)
   	if txn == nil {
   		return nil
   	}
   	h := http.Header{}
   	txn.InsertDistributedTraceHeaders(h)
   	out := make(map[string]string, len(h))
   	for k := range h {
   		out[strings.ToLower(k)] = h.Get(k)
   	}
   	return out
   }
   ```

2. No consumidor:

   ```go
   txn := app.StartTransaction("sqs/" + tipo)
   defer txn.End()
   h := http.Header{}
   for k, v := range msg.MessageAttributes {
   	if v.StringValue != nil {
   		h.Set(k, *v.StringValue)
   	}
   }
   txn.AcceptDistributedTraceHeaders(newrelic.TransportQueue, h)
   ctx = newrelic.NewContext(ctx, txn)
   ```

3. Segmentos de mensageria: `newrelic.MessageProducerSegment{Library: "SQS", DestinationType: newrelic.MessageQueue, DestinationName: fila}` em volta do `SendMessage`; `nrawssdk-v2` para instrumentar as chamadas ao SQS/DynamoDB.
4. Limite do SQS: 10 *message attributes* por mensagem — `tipo` + 3 de trace (`newrelic`, `traceparent`, `tracestate`) cabem.

> A espera humana (aprovação/pagamento) quebra o trace em dois ou três — é esperado. O fio que une tudo é o `saga_id` (F4-7.2).

**Critérios de aceite**
- [ ] Print de um trace com spans dos 3 serviços para a mesma OS.
- [ ] *Service map* do New Relic mostrando OS ↔ Billing ↔ Execution.

---

### F4-7.2 — Logs com `saga_id` e `SagaEvent`

**Passo a passo**

1. Todo log dentro de consumo ou do orquestrador leva `saga_id`, `os_id`, `message_id`, `tipo_mensagem` além do `correlation_id` e `trace.id` da Fase 3. O `correlation_id` da requisição original viaja no envelope e é restaurado no consumidor.
2. Custom event `SagaEvent` emitido pelo OS a cada transição (estende o `eventRecorder` existente):

   | Atributo | Exemplo |
   |---|---|
   | `sagaId`, `osId` | UUIDs |
   | `estadoAnterior`, `estado` | `AGUARDANDO_PAGAMENTO` → `COMPENSANDO` |
   | `etapa` | `T5` |
   | `resultado` | `ok` \| `falha` \| `compensada` \| `ignorada` |
   | `motivo` | `pagamento_recusado` |
   | `duracaoEtapaSegundos` | `42.1` |
   | `duracaoTotalSegundos` | só em estados terminais |

   Sem CPF, e-mail ou valor pago (mesma regra de LGPD do `OrdemServicoEvent`).
3. Billing emite `PagamentoEvent{status, mpStatusDetail, valorFaixa}` e Execution `TarefaEvent{tipo, estado, duracao}`.
4. Antes de tudo: conferir se o `OrdemServicoEvent` da Fase 3 está chegando (achado 10) — `SELECT count(*) FROM OrdemServicoEvent SINCE 1 week ago`.

**Critérios de aceite**
- [ ] Busca no Logs do New Relic por `saga_id:<id>` mostra a saga inteira, dos 3 serviços, em ordem.

---

### F4-7.3 — Dashboard da saga

> **Plano de 4 dias (corte 21):** pela UI, só os 4 primeiros painéis (desfechos, taxa de compensação, onde falham, tempo por etapa) + os 2 de latência/erro por serviço. Dados: corte 25.

Dashboard como código (JSON versionado em `oficina-app/observability/dashboard-saga.json`, aplicado via NerdGraph ou Terraform `newrelic_one_dashboard_json`):

| Painel | NRQL |
|---|---|
| Sagas por desfecho | `SELECT count(*) FROM SagaEvent WHERE estado IN ('CONCLUIDA','COMPENSADA') FACET estado TIMESERIES` |
| Taxa de compensação | `SELECT percentage(count(*), WHERE estado = 'COMPENSADA') FROM SagaEvent WHERE estado IN ('CONCLUIDA','COMPENSADA')` |
| Onde as sagas falham | `SELECT count(*) FROM SagaEvent WHERE resultado = 'falha' FACET etapa, motivo` |
| Tempo médio por etapa | `SELECT average(duracaoEtapaSegundos) FROM SagaEvent WHERE resultado = 'ok' FACET etapa` |
| Duração ponta a ponta (p50/p95) | `SELECT percentile(duracaoTotalSegundos, 50, 95) FROM SagaEvent WHERE estado = 'CONCLUIDA'` |
| Sagas em andamento por estado | `SELECT uniqueCount(sagaId) FROM SagaEvent FACET estado SINCE 1 day ago` (aproximação; o exato é a query no banco) |
| Pagamentos | `SELECT count(*) FROM PagamentoEvent FACET status TIMESERIES` |
| Fila do Execution | `SELECT latest(tamanho) FROM TarefaFilaSample FACET tipo` (amostra periódica emitida pelo serviço) |
| Latência por serviço | `SELECT percentile(duration, 95) FROM Transaction FACET appName TIMESERIES` |
| Erros por serviço | `SELECT percentage(count(*), WHERE error IS true) FROM Transaction FACET appName` |

**Critérios de aceite**
- [ ] Dashboard com dados de ao menos 20 sagas (geradas por um script de carga leve com o stub — F4-9.1).

---

### F4-7.4 — Alertas

> **Plano de 4 dias (corte 21/22):** só a **taxa de compensação** (New Relic) e o **alarme da DLQ** (CloudWatch + SNS). As demais linhas são evolução.

| Condição | Fonte | Limite |
|---|---|---|
| Taxa de compensação alta | `SagaEvent` (NRQL) | > 30% em 15 min |
| Saga presa | `SagaPresaEvent` (NRQL) | `count > 0` em 5 min |
| Mensagem na DLQ | CloudWatch `ApproximateNumberOfMessagesVisible` das 3 DLQs → alarme → tópico SNS com e-mail (Terraform em cada `infra/`) | `> 0` |
| Outbox acumulando | evento `OutboxSample{pendentes}` | `> 50` por 5 min |
| Erro de webhook do MP | `Transaction` com `request.uri LIKE '%mercadopago%'` e `httpResponseCode >= 400` | > 3 em 5 min |

> A DLQ fica no CloudWatch porque é métrica da AWS, não da aplicação. Integrar o CloudWatch ao New Relic (*AWS integration / metric streams*) é possível, mas custa mais configuração que um alarme com SNS — fica como evolução.

**Critérios de aceite**
- [ ] Alerta de compensação disparado com 4 sagas `FAL` seguidas, e e-mail recebido.
- [ ] Mensagem venenosa injetada manualmente → alarme da DLQ disparado.

---

## E8 — Documentação

### F4-8.1 — READMEs

Cada repo de serviço, no formato pedido pelo enunciado:

1. Propósito e responsabilidades (o que faz **e o que não faz**).
2. Tecnologias e banco (com o porquê, linkando a ADR).
3. **Arquitetura do serviço** (`docs/arquitetura.md`): componentes internos, mensagens que consome/produz, modelo de dados.
4. Como rodar local, testar, BDD.
5. **Evidência de cobertura**: badge do Sonar + print + número do critério (F4-5.1).
6. CI/CD: diagrama dos workflows e o que bloqueia o merge.
7. Links: OpenAPI, Postman, dashboard, outros repos.

No `oficina-app`, além disso, a seção **"Saga Pattern — por que orquestrada"** (exigência explícita: *documentar a escolha e justificar no README*), com o diagrama de estados e a tabela de compensações.

Atualizar também os READMEs de `oficina-lambda-auth` (sem acesso ao banco), `oficina-infra-k8s` (Pod Identity, nós, rotas) e `oficina-infra-db` (agora só o banco do OS).

---

### F4-8.2 — RFCs e ADRs da Fase 4

| Documento | Conteúdo |
|---|---|
| **RFC-005** — Decomposição em microsserviços | Contextos, por que 3, onde ficou o estoque, alternativas (4–5 serviços), evolução |
| **RFC-006** — Saga e consistência distribuída | Orquestração × coreografia (tabela da arquitetura §3.1), etapas, compensações, timeouts, falha de compensação |
| ADR-009 | Banco por serviço: PostgreSQL (OS, Billing) e DynamoDB (Execution) |
| ADR-010 | Mensageria: SQS FIFO (alternativas: RabbitMQ, Kafka/MSK, SNS+SQS) |
| ADR-011 | Saga orquestrada no OS Service |
| ADR-012 | Transactional Outbox + inbox idempotente; relay com líder único |
| ADR-013 | Código de mensageria duplicado nos repos em vez de módulo compartilhado |
| ADR-014 | Mercado Pago Checkout Pro + verificação ativa + estorno como compensação |
| ADR-015 | Lambda de autenticação passa a usar REST interno (fim do acesso ao banco) |
| ADR-016 | EKS Pod Identity para acesso dos pods à AWS |

Numeração continua a de `docs/architecture-decisions.md` (ADR-001…008) e `docs/rfcs/` (RFC-001…004).

---

### F4-8.3 — OpenAPI e Postman

1. `docs/openapi.yaml` em cada serviço, servido como Swagger UI em `/docs` (opcional, ajuda na demo).
2. Coleção Postman **única do fluxo** (`docs/postman/fase-4-saga.json` no `oficina-app`), com pastas por serviço e na ordem da saga: login funcionário → criar OS → mecânico conclui diagnóstico → token do cliente por CPF → ver orçamento → aprovar → (pagar no link) → mecânico inicia/conclui → consultar OS e histórico da saga. Variáveis encadeadas por script (`pm.collectionVariables.set("os_id", ...)`).
3. Ambientes Postman: `local` (portas 8080/8081/8082) e `prod` (gateway).

---

### F4-8.4 — Diagramas finais

Revisar os diagramas de [arquitetura.md](arquitetura.md) contra o que foi implantado (`terraform state list` dos 6 repos, `kubectl get all -A`), exportar o de componentes como PNG (mermaid.live) para o PDF e copiá-los para o `docs/` do `oficina-app` (fora de `planejamentos/`).

---

## E9 — Entrega

### F4-9.1 — Vídeo de demonstração (≤ 15 min)

Roteiro cronometrado em [entrega.md](entrega.md#roteiro-do-vídeo). Antes de gravar: gerar ~20 sagas com um script (`scripts/gerar-sagas.sh`, usando o stub em ambiente local apontando para a nuvem, ou o sandbox) para o dashboard ter dados.

### F4-9.2 — PDF do portal

Modelo em [entrega.md](entrega.md#conteúdo-do-pdf): participantes, links dos repositórios, vídeo, diagrama geral, estratégia da saga, justificativa da divisão e das tecnologias.

### F4-9.3 — Teardown e custo

Ordem de destroy (inversa): `lambda-auth` (rotas) → `oficina-execution-service/infra` ∥ `oficina-billing-service/infra` ∥ `oficina-app/infra` → apagar os Services `LoadBalancer` dos 3 namespaces (os CLBs são do Kubernetes, não do Terraform — lição da Fase 3) → `infra-db` → `infra-k8s`. Conferir órfãos: CLBs, ENIs, snapshots finais do RDS billing (`skip_final_snapshot = true` na demo), filas, chaves KMS em `PendingDeletion`.
