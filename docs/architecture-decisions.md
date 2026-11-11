# Architecture Decision Records — Tech Challenge

> Registro das decisões arquiteturais relevantes tomadas durante o desenvolvimento.
> Cada ADR descreve o contexto, as alternativas consideradas e a justificativa da escolha.

---

## ADR-001 — Banco de dados: PostgreSQL

### Contexto

O sistema precisa de um banco de dados para persistir clientes, veículos, peças, serviços e ordens de serviço. O controle de estoque e as transições de status da OS exigem garantias fortes de consistência.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **PostgreSQL** | ACID completo, FKs, tipos decimais nativos, maturidade | Infraestrutura ligeiramente mais pesada que SQLite |
| MySQL / MariaDB | Popular, boa performance de leitura | Tipos `DECIMAL` menos precisos, FK enforcement opcional por engine |
| SQLite | Zero infraestrutura, simples para dev | Sem suporte a concorrência real, inadequado para produção com múltiplas conexões |
| MongoDB | Flexibilidade de schema, escalabilidade horizontal | Sem transações multi-documento em versões antigas, sem FKs, inadequado para dados relacionais fortemente ligados |

### Decisão: PostgreSQL 15.7

**1. Transações ACID são críticas para este domínio**

Dois fluxos exigem atomicidade entre múltiplas tabelas:

- **Dedução de estoque**: ao avançar uma OS para `em_execucao`, o sistema debita as peças do estoque. Se qualquer `UPDATE` falhar, toda a operação deve ser revertida — sem PostgreSQL, o estoque poderia ficar inconsistente.
- **Transições de status**: registrar a transição na `ordens_servico` e inserir o `historico_status` deve ocorrer atomicamente.

**2. Integridade referencial via foreign keys**

O schema usa FKs em todas as relações relevantes:

```sql
ALTER TABLE veiculos        ADD CONSTRAINT fk_veiculos_cliente   FOREIGN KEY (cliente_id)  REFERENCES clientes(id);
ALTER TABLE ordens_servico  ADD CONSTRAINT fk_os_cliente         FOREIGN KEY (cliente_id)  REFERENCES clientes(id);
ALTER TABLE ordens_servico  ADD CONSTRAINT fk_os_veiculo         FOREIGN KEY (veiculo_id)  REFERENCES veiculos(id);
ALTER TABLE itens_os_pecas  ADD CONSTRAINT fk_itens_os_peca      FOREIGN KEY (os_id)       REFERENCES ordens_servico(id);
```

O banco garante que não existirá uma OS apontando para um cliente deletado — sem FK no nível do banco, isso dependeria exclusivamente da aplicação.

**3. Tipos decimais precisos para valores financeiros**

Preços de peças e serviços são armazenados como `NUMERIC(10,2)` no PostgreSQL. MySQL usa `FLOAT` por padrão em alguns contextos, o que pode introduzir erros de arredondamento em somatórios de notas fiscais.

**4. Suporte maduro a migrações incrementais**

`golang-migrate` tem suporte de primeira classe para PostgreSQL, com transações por migration — se a migration 3 falhar no meio, o banco volta ao estado da migration 2 automaticamente.

---

## ADR-002 — Router HTTP: chi

### Contexto

A API precisa de um router HTTP que suporte parâmetros de rota (`/v1/clients/{id}`), middlewares por grupo de rotas e seja compatível com a interface padrão `net/http` — requisito do `oapi-codegen` no modo `chi-server`.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **chi** | Interface `net/http` pura, leve (~1k LOC), compatível com oapi-codegen | Menos "batteries included" que Echo/Gin |
| Echo | Popular, bom suporte a middleware, validação integrada | Interface própria (não `net/http`), incompatível com oapi-codegen chi-server mode |
| Gin | Alta performance, muito popular | Interface própria, contexto custom, incompatível com oapi-codegen chi-server mode |
| Fiber | Alta performance (fasthttp), familiar para devs Express | Não é `net/http`, incompatível com oapi-codegen |
| `net/http` puro | Zero dependências, máxima compatibilidade | Sem roteamento avançado, verboso para parâmetros de rota |

### Decisão: chi v5

**1. Requisito de compatibilidade com oapi-codegen**

O `oapi-codegen` com `chi-server: true` gera código que registra rotas diretamente no `chi.Router`. Usar Echo ou Gin exigiria mode diferente ou adaptadores, adicionando complexidade sem benefício.

**2. Interface `net/http` pura**

Chi não introduce contexto customizado — usa `*http.Request` e `http.ResponseWriter` nativos. Qualquer middleware da stdlib ou de terceiros que siga o padrão `net/http` funciona sem adaptação.

**3. Leveza e sem "magic"**

Chi tem ~1.000 linhas de código. Não há reflexão em runtime, sem injeção de dependência implícita. O comportamento é previsível e fácil de depurar.

**4. Middlewares por grupo**

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.JWTAuth(cfg.JWTSecret))
    // rotas protegidas
})
```

Permite aplicar JWT apenas nas rotas que precisam, sem contaminar as rotas públicas (`/health`, `/v1/auth/*`).

---

## ADR-003 — Abordagem API-first com oapi-codegen

### Contexto

A API precisa ter um contrato OpenAPI bem definido. A questão é: o contrato é gerado a partir do código (code-first) ou o código é gerado a partir do contrato (API-first)?

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **oapi-codegen (API-first)** | Contrato é a fonte da verdade, tipos Go garantidos, sem drift | Requer re-geração ao mudar o contrato |
| Swaggo / swag (code-first) | Escreve anotações no Go, gera OpenAPI | Anotações verbose, drift fácil, schema menos preciso |
| Handlers manuais | Controle total, sem geração | Manutenção trabalhosa, schema e código podem divergir, serialização manual |
| gRPC + Gateway | Performance, tipagem forte | Overhead de Protobuf, complexidade maior para API REST simples |

### Decisão: oapi-codegen v2 (modo strict-server + chi-server)

**1. O contrato é a fonte da verdade**

`docs/openapi.yaml` define todos os endpoints, schemas, parâmetros e respostas. O código Go em `api.gen.go` é derivado — nunca editado manualmente. Isso elimina a possibilidade de o código e a documentação divergirem.

**2. Strict Server — tipos compilados por endpoint**

O modo `strict-server: true` gera um `RequestObject` e um `ResponseObject` tipado para cada operação:

```go
// Gerado automaticamente — o compilador garante que o handler
// retorna exatamente o tipo esperado
func (s *Server) GetClients(ctx context.Context, req GetClientsRequestObject) (GetClientsResponseObject, error)
```

Erros de contrato (retornar 200 onde deveria ser 201, campo faltando) são **erros de compilação**, não bugs em runtime.

**3. Validação de schema gratuita**

O `StrictHandler` gerado valida automaticamente o body da requisição contra o schema OpenAPI antes de chamar o handler. Não é necessário escrever validação manual de campos obrigatórios no handler.

**4. Regeneração simples**

```bash
go generate ./internal/infra/http/api/...
```

Um único comando atualiza todo o código gerado ao modificar o contrato.

---

## ADR-004 — Orquestração de containers: Kubernetes gerenciado (EKS)

### Contexto

A Fase 3 exige que a aplicação rode em nuvem, escale sob carga e seja provisionada por IaC. É preciso escolher a plataforma de execução dos containers na AWS.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **EKS (Kubernetes gerenciado)** | Padrão de mercado, HPA nativo, manifestos portáveis, control plane gerenciado | Curva de aprendizado, custo do control plane |
| ECS + Fargate | Sem gerência de nós, integração AWS fluida | Autoescalonamento e conceitos proprietários, menos portável |
| EC2 + Docker Compose | Simples de entender | Sem autoescalonamento real, sem self-healing, operação manual |
| Lambda (app inteira) | Escala a zero, sem servidor | Inadequado para API stateful de longa duração com pool de conexões ao RDS |

### Decisão: Amazon EKS

**1. HPA nativo atende o requisito de escala**

O requisito de "escalar sob carga (2 → ≥4 réplicas)" é atendido de forma declarativa pelo `HorizontalPodAutoscaler` do Kubernetes (ver ADR-005), sem código nem serviço proprietário.

**2. Manifestos portáveis e versionados**

Todo o estado desejado do cluster vive em `k8s/` (Kustomize com `base` + `overlays/prod` e `overlays/local`). O mesmo manifesto roda localmente (kind/minikube) e em produção, mudando apenas o overlay.

**3. Separação clara com a infraestrutura**

O cluster e seus add-ons são provisionados por Terraform no repositório `oficina-infra-k8s`; a aplicação apenas entrega manifestos e imagem. Essa fronteira sustenta a divisão em 4 repositórios.

---

## ADR-005 — Autoescalonamento horizontal (HPA)

### Contexto

A aplicação precisa absorver picos de tráfego sem intervenção manual e reduzir custo quando ocioso. É preciso definir a estratégia e os parâmetros de autoescalonamento.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **HPA por CPU (autoscaling/v2)** | Nativo, métrica simples e previsível, fácil de demonstrar | Reage a CPU, não diretamente a latência/RPS |
| HPA por métrica custom (RPS/latência) | Escala pelo sinal de negócio real | Exige metrics-adapter/Prometheus adapter, mais peças móveis |
| Escalonamento manual (`kubectl scale`) | Controle total | Não atende o requisito de escala automática |
| Cluster Autoscaler apenas | Ajusta nós | Não ajusta réplicas do pod — resolve outro problema |

### Decisão: HPA v2 por utilização de CPU, `min=2` / `max=5` / alvo 50%

```yaml
minReplicas: 2
maxReplicas: 5
metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
behavior:
  scaleDown:
    stabilizationWindowSeconds: 60
```

**1. `min=2` garante disponibilidade**

Duas réplicas em subnets/AZs distintas evitam ponto único de falha e permitem rolling update sem downtime.

**2. CPU a 50% dá margem para o pico**

Com alvo de 50%, o HPA começa a criar réplicas antes da saturação — quando um teste de carga (`hey`) satura os 2 pods, o autoscaler sobe rapidamente até 5.

**3. `stabilizationWindowSeconds: 60` no scale-down evita flapping**

Após o pico, a redução espera 60s antes de remover réplicas, o que também torna a demonstração no vídeo estável e legível (sobe sob carga, volta a 2 depois).

---

## ADR-006 — Padrão de comunicação: REST síncrono + webhook assíncrono assinado

### Contexto

O sistema tem dois padrões de interação distintos: chamadas cliente→API (abrir OS, consultar status) e a resposta de aprovação/recusa de orçamento, que chega de um sistema externo em momento indeterminado. É preciso decidir como cada uma se comunica.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **REST síncrono + webhook HTTP assinado (HMAC)** | Simples, sem broker, resposta imediata onde faz sentido, callback desacoplado no tempo | Webhook exige endpoint público e verificação de autenticidade |
| Fila/mensageria (SQS/SNS) para tudo | Desacoplamento total, resiliência | Overhead operacional desproporcional ao volume atual, complexidade de entrega/idempotência |
| Polling do cliente pelo status do orçamento | Nenhuma porta de entrada extra | Latência, desperdício de requisições, pior UX |

### Decisão: REST síncrono para o fluxo do usuário; webhook HTTP com HMAC para a resposta de orçamento

**1. Síncrono onde a resposta é imediata**

Abrir OS, avançar status e consultar são operações request/response — o cliente precisa do resultado na hora. REST sobre o contrato OpenAPI (ADR-003) cobre isso.

**2. Webhook assíncrono para o que é assíncrono por natureza**

A decisão do cliente sobre o orçamento chega quando ele responde — pode ser minutos ou horas depois. Um `POST /v1/webhooks/budget-response` recebe esse callback sem manter conexão aberta.

**3. Autenticidade por HMAC, não por JWT**

O webhook vem de um sistema externo, não de um usuário. Ele é validado pela assinatura HMAC do corpo (middleware `AssinaturaWebhook`), com o segredo compartilhado `WEBHOOK_SECRET` — não por token de usuário. O processamento é idempotente: reenvio do provedor não deduz estoque duas vezes.

**4. Sem broker por decisão de proporcionalidade**

O volume e os requisitos atuais não justificam SQS/SNS. A comunicação assíncrona necessária (um único callback) é atendida por webhook, sem introduzir um broker para operar e monitorar.

---

## ADR-007 — Observabilidade: New Relic (APM + eventos de negócio)

### Contexto

A Fase 3 exige APM, logs correlacionados, dashboards de negócio e alerta de falha no processamento de OS. É preciso escolher a plataforma e como instrumentar sem vazar dado sensível (LGPD).

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **New Relic (APM + Custom Events + Logs in Context)** | APM, infra, logs, eventos e alertas em uma plataforma; agente Go maduro | SaaS pago, dado sai do ambiente |
| Prometheus + Grafana + Loki (self-hosted) | Open-source, sem custo de licença | Operação de vários componentes, sem APM/tracing pronto |
| CloudWatch | Nativo AWS | Tracing/APM e dashboards de negócio mais trabalhosos |

### Decisão: New Relic, com APM automático + `OrdemServicoEvent` como custom event

**1. Duas camadas complementares**

- **APM automático**: o middleware envolve cada requisição numa transação New Relic, dando latência, throughput e erros sem código de negócio.
- **Eventos de negócio**: cada transição relevante de OS emite um `OrdemServicoEvent` (`app.RecordCustomEvent`), que alimenta os dashboards de volume diário, tempo médio por status e falhas, além do alerta.

**2. Correlação trace ↔ log ↔ evento**

O evento carrega `trace.id`/`span.id` da transação corrente (`GetTraceMetadata`), permitindo pular de um ponto no gráfico para o trace distribuído e para os logs JSON daquela mesma requisição (`correlation_id` + `trace.id`).

**3. LGPD: o evento não carrega dado pessoal**

`OrdemServicoEvent` só leva identificadores da OS e metadados de status (`os_id`, `numero`, `status`, `resultado`, `duracao_status_segundos`, `motivo`). Nunca CPF, nome ou e-mail.

**4. Observabilidade nunca quebra o negócio**

A emissão é no-op quando não há transação no contexto (ex.: ambiente local sem licença) e o caso de uso jamais falha por causa dela — é um efeito colateral, não um passo do fluxo.

---

## ADR-008 — Notificação de status ao cliente: e-mail via SMTP

### Contexto

O cliente deve ser notificado nas mudanças de status relevantes da OS (aguardando aprovação, em execução, finalizada, entregue). É preciso escolher o canal e como isolá-lo do fluxo transacional. O requisito da Fase 3 pede a escolha registrada em ADR.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|-------------|------|---------|
| **SMTP (SES na nuvem, Mailpit local)** | Um só código para os dois ambientes, protocolo padrão, `net/smtp` na stdlib | Entrega best-effort, sem histórico de entregas rico |
| SDK específico da AWS SES | Recursos avançados (templates, métricas de entrega) | Acopla o código ao provedor, sem equivalente local trivial |
| SNS / fila de notificação | Desacoplamento, retry gerenciado | Complexidade desproporcional para um e-mail simples |

### Decisão: notificador SMTP único, apontando para SES (nuvem) ou Mailpit (local), envio assíncrono best-effort

**1. Um código, dois ambientes**

`SMTPNotifier` usa `net/smtp`: com `SMTP_USER` definido, autentica via PLAIN (interface SMTP do Amazon SES em produção); sem usuário, conecta sem autenticação ao Mailpit/MailHog local (`localhost:1025`). A escolha do ambiente é só configuração (`SMTP_HOST`/`SMTP_PORT`), sem `if` no código.

**2. Assíncrono e best-effort — nunca bloqueia a transição**

A notificação é disparada numa goroutine com timeout próprio. Falha de envio (cliente sem e-mail, provedor fora) é apenas logada; a transição de status da OS é concluída de qualquer forma. Comunicação com o cliente não pode derrubar a operação.

**3. Escolha para a demonstração**

No vídeo, a notificação é demonstrada com **Mailpit** local capturando o e-mail — evita depender da saída de porta 25/verificação de domínio do SES no momento da gravação, mantendo o mesmo caminho de código de produção.

---

# Fase 4 — Microsserviços e Saga

> ADRs 009–016. As decisões de maior alcance (decomposição e estratégia de saga) estão nas [RFC-005](rfcs/RFC-005-decomposicao-em-microsservicos.md) e [RFC-006](rfcs/RFC-006-saga-e-consistencia-distribuida.md).

---

## ADR-009 — Banco por serviço: PostgreSQL (OS, Billing) e DynamoDB (Execution)

### Contexto

Cada microsserviço precisa de banco próprio, sem acesso ao banco dos outros, com pelo menos um relacional e um NoSQL.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| **PostgreSQL no OS e no Billing, DynamoDB no Execution** | Relacional onde há dinheiro e integridade referencial; NoSQL onde o acesso é por chave | Dois modelos de outbox/inbox para manter |
| PostgreSQL nos três (schemas separados) | Um só modelo | Não atende ao requisito de NoSQL; schema separado na mesma instância isola mal |
| DynamoDB no Billing | Sem servidor | Orçamento com itens, valores em centavos e conciliação pedem transações e consultas relacionais |
| MongoDB/DocumentDB no Execution | Flexível | Mais um servidor para operar; DocumentDB não tem free tier |

### Decisão

- **OS Service:** o PostgreSQL da Fase 3, acrescido de `sagas_os`, `sagas_os_etapas`, `outbox`, `mensagens_processadas` e da reserva de estoque (migration `000007`).
- **Billing:** instância PostgreSQL **própria**, credencial própria. Dinheiro sempre em centavos (`bigint`), nunca `float`.
- **Execution:** DynamoDB *single-table* (`execution-tarefas`): tarefa, ponteiro OS→tarefa, inbox e outbox na mesma tabela, para caberem no mesmo `TransactWriteItems`. GSI para a fila por tipo e estado.

### Consequências

O isolamento é por **credencial e role**: cada pod recebe só o segredo ou a permissão do próprio banco. Os três serviços dividem os nós do EKS, então a rede ainda alcança os dois RDS. *Security Groups for Pods* fica como evolução.

---

## ADR-010 — Mensageria: Amazon SQS FIFO

### Contexto

A saga troca comandos e respostas entre os três serviços. As mensagens de uma mesma OS precisam chegar **em ordem**; reentregas precisam ser toleradas; falhas repetidas precisam ser isoladas.

### Alternativas consideradas

| Alternativa | Prós | Contras |
|---|---|---|
| **SQS FIFO** | Gerenciado, ordem por `MessageGroupId`, deduplicação, DLQ nativa, custo ≈ 0, IAM | Throughput limitado por grupo (irrelevante aqui); não é *pub/sub* |
| SNS + SQS | *Fan-out* | Sem consumidor múltiplo no desenho orquestrado; mais um recurso |
| RabbitMQ (Amazon MQ ou in-cluster) | Roteamento rico | Broker para operar e pagar 24/7 |
| Kafka / MSK | Log replay, alto throughput | Custo e operação desproporcionais para 3 serviços |

### Decisão

Três filas FIFO — `execution-commands.fifo`, `billing-commands.fifo`, `os-saga-replies.fifo` — cada uma com DLQ após **5 recebimentos**. `MessageGroupId = saga_id` (ordem por saga, paralelismo entre sagas) e `MessageDeduplicationId = message_id`. Envelope comum (`message_id`, `tipo`, `saga_id`, `os_id`, `correlation_id`, `origem`, `ocorrido_em`, `payload`) e contexto de trace do New Relic nos atributos da mensagem. Localmente, LocalStack.

### Consequências

Ordem e deduplicação ficam com a fila; idempotência de negócio continua necessária (a deduplicação do SQS vale só por 5 minutos — ADR-012).

---

## ADR-011 — Saga orquestrada no OS Service

### Contexto

O enunciado pede o Saga Pattern com a escolha entre orquestração e coreografia justificada. A comparação completa está na [RFC-006](rfcs/RFC-006-saga-e-consistencia-distribuida.md).

### Decisão

Saga **orquestrada**, com o orquestrador **dentro do OS Service** (`internal/application/saga`):

- `Decidir(saga, tipo, dados, agora) → Decisão` é uma **função pura**: recebe a saga e a resposta e devolve o novo estado, as etapas a registrar, os comandos a enviar e a mudança de status da OS. Sem I/O, 100% coberta por testes de tabela.
- O adaptador aplica a decisão numa **única transação**: inbox, saga (com lock otimista em `versao`), etapas, status da OS, estoque e outbox.
- Compensações **sequenciais em ordem inversa**; pivô em T6 (início do reparo).

### Alternativas consideradas

Coreografia (rejeitada: fluxo implícito em 3 serviços, sem lugar natural para as esperas humanas) e orquestrador como serviço próprio (rejeitado: o estado da saga precisa estar na mesma transação que a OS).

### Consequências

O OS Service conhece o catálogo de comandos dos outros dois. Qualquer mudança de fluxo é uma edição em duas tabelas (`fluxo` e `falhas`) do orquestrador.

---

## ADR-012 — Transactional Outbox + inbox idempotente; relay com líder único

### Contexto

Mudar o banco **e** publicar na fila não é atômico. Se o processo cai entre um e outro, a saga para (mudou, mas não avisou) ou diverge (avisou, mas não mudou). O SQS entrega *at-least-once*.

### Decisão

- **Outbox:** a mensagem é gravada na tabela `outbox` na mesma transação do efeito. Um **relay** lê os pendentes em ordem (`seq`), publica e marca como enviados; falha de envio registra `falhou_em` e tenta de novo.
- **Inbox:** `mensagens_processadas(message_id)` é gravada na transação do efeito; uma reentrega encontra a marca e vira *ack* sem efeito.
- **Líder único do relay:** no Postgres, `pg_try_advisory_lock` garante que só uma réplica publique, preservando a ordem. O Execution (DynamoDB, sem lock equivalente) roda com **uma réplica**.
- **No DynamoDB:** efeito, inbox e outbox no mesmo `TransactWriteItems` com `attribute_not_exists(pk)`.
- **Erros:** `Permanente` (payload inválido) vai direto para a DLQ; o resto é reentregue.

### Consequências

Entrega *at-least-once* com efeito *exactly-once*. Latência extra do intervalo de *polling* do relay (500 ms por padrão). Billing e Execution limitados a uma réplica de relay nesta entrega.

---

## ADR-013 — Código de mensageria duplicado nos repositórios

### Contexto

Os três serviços usam o mesmo envelope, publisher, consumer e relay. A alternativa natural seria um módulo Go compartilhado.

### Decisão

O pacote `mensageria` (e, no Billing, outbox/inbox e UnitOfWork) é **copiado** em cada repositório.

| Alternativa | Por que não |
|---|---|
| Módulo Go compartilhado (repositório próprio, versionado) | Mais um repositório, CI e versionamento; toda mudança vira PR em cadeia (lib → 3 serviços) — no prazo de 4 dias, o acoplamento de release custaria mais do que a duplicação |
| *Monorepo* | Contraria a exigência de um repositório por serviço |

### Consequências

~400 linhas duplicadas, cobertas por testes em cada repositório. O **contrato** (envelope e catálogo) é a fonte da verdade e está documentado; o código pode divergir internamente sem quebrar ninguém. Evolução: extrair o módulo quando o contrato estabilizar.

---

## ADR-014 — Mercado Pago Checkout Pro: API de Orders, verificação ativa e estorno

### Contexto

O pagamento real precisa de link de checkout, confirmação confiável e estorno (compensação T5). No sandbox, validamos em 10/11/2026 que: (1) a API de *Preferences* não estorna pagamentos de teste (`401`); (2) o sandbox **não envia webhook** para orders de teste; (3) após uma recusa, a order aceita novas tentativas (`action_required`), mesmo com `retries.allowed: false`.

### Decisão

- **API de Orders** do Checkout Pro (`POST /v1/orders`, `GET /v1/orders/{id}`, `/refund`, `/cancel`), com idempotência por `X-Idempotency-Key`.
- **Webhook com `x-signature` validada** (HMAC-SHA256 do manifesto `id:<data.id>;request-id:<x-request-id>;ts:<ts>;`). O webhook nunca é confiado pelo corpo: ele só dispara a **consulta ativa** da order.
- **Reconciliador** consulta periodicamente os pagamentos pendentes — é o mecanismo primário de confirmação no sandbox e a rede de segurança em produção.
- **Recusa:** o Billing cancela a order antes de emitir `PagamentoRecusado`, fechando a porta a uma nova tentativa no mesmo link.
- **Estorno total** como compensação; pagamento que chega para orçamento cancelado é estornado automaticamente.
- Localmente, um **stub** (`oficina-billing/cmd/mp-stub`) imita a API de Orders e a página de checkout, permitindo BDD sem rede.

### Consequências

Confirmação em até um ciclo do reconciliador (≈ 10 s) quando o webhook não chega. Credenciais do MP ficam no segredo do Billing.

---

## ADR-015 — Lambda de autenticação passa a usar REST interno

> **Status:** aceita — **implementação pendente** (depende da recriação da conta AWS).

### Contexto

Na Fase 3 a Lambda `auth-token` lia o RDS do OS para validar o CPF. Na Fase 4 nenhum componente pode ler o banco de outro.

### Decisão

A Lambda consulta o OS Service por **REST interno** (`GET /internal/clientes/por-cpf`), autenticada por um segredo de serviço, e perde a configuração de VPC, o SG de acesso ao RDS e o DSN.

### Consequências

A emissão do token passa a depender da disponibilidade do OS Service (que já é pré-requisito de qualquer operação do cliente). A Lambda fica mais simples e sem credencial de banco.

---

## ADR-016 — EKS Pod Identity para acesso dos pods à AWS

> **Status:** aceita — **implementação pendente** (depende da recriação da conta AWS).

### Contexto

Os pods precisam de SQS, DynamoDB e Secrets Manager. Credencial estática em Secret do Kubernetes é rotacionável só manualmente e vaza fácil.

### Alternativas consideradas

| Alternativa | Contras |
|---|---|
| Access key em Secret | Credencial de longa duração |
| IRSA (OIDC do cluster) | Funciona, mas exige anotação por ServiceAccount e *trust policy* por cluster |
| **EKS Pod Identity** | — (addon gerenciado, associação role ↔ ServiceAccount via API) |

### Decisão

Uma role por serviço, associada à ServiceAccount do seu namespace, com **mínimo privilégio**: consumir só a própria fila de comandos, publicar só nas filas de destino, DynamoDB só em `table/execution-tarefas*`, segredos só `oficina/prod/<serviço>`.

### Consequências

Sem credencial estática nos pods; o SDK da AWS resolve credenciais sozinho. A política IAM de cada role é a prova do isolamento entre serviços.
