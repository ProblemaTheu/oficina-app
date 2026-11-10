# Plano de execução — 4 dias (ter 10/11 → sex 13/11)

> **Documento de trabalho — replanejado em ter 10/11.** O plano original (7 dias, 05/11 → 11/11) não chegou a começar: em 10/11 não havia repositórios novos, migration nem código de saga (ver [diário](execucao.md#1011--replanejamento)). Este plano começa **hoje** e termina com a **submissão na sex 13/11 até as 18 h**. Sábado e domingo (14–15/11) são **reserva**, não planejamento.

| Item | Quando |
|---|---|
| **D1 — fundação + prazos de terceiros** | ter 10/11, 11 h → 23 h |
| **D2 — saga completa no compose + nuvem provisionada** | qua 11/11 |
| **D3 — nuvem ponta a ponta, qualidade, documentação** | qui 12/11 |
| **Code freeze** | **qui 12/11, 22 h** |
| **D4 — ensaio, gravação, PDF e submissão** | sex 13/11 — **submissão até 18 h** |
| Teardown | sex 13/11, depois da submissão confirmada |
| Reserva (prazo final do portal) | sáb 14/11 – dom 15/11 |

### Premissas

- **3 pessoas, 3 trilhas paralelas**, ~12 h/dia em D1–D3 (~100 h no total). O escopo abaixo, com os cortes, foi dimensionado para ~85 h.
  - **A — OS + saga** (`oficina-app`)
  - **B — Billing + Mercado Pago** (`oficina-billing-service`)
  - **C — Execution + plataforma** (`oficina-execution-service`, `oficina-infra-k8s`, `oficina-infra-db`, `oficina-lambda-auth`)
- **Com 2 pessoas:** aplique a [ordem de sacrifício](#ordem-de-sacrifício-se-atrasar) até o item 5 **antes de começar** e divida a trilha C entre A (Lambda, Gateway) e B (Execution). **Com 1 pessoa, sexta não é viável** — negocie o prazo do portal (15/11) ou o escopo com o grupo hoje.
- **Duas sincronizações por dia, 15 min:** **13 h** e **21 h**. Cada trilha diz se o marco parcial fechou. Corte de escopo **só** nessas sincronizações.
- **Testes junto com o código.** O gate de 80% entra no CI no D3; quem deixar teste para quinta não fecha.
- **Copiar, não inventar.** Billing e Execution nascem como cópia do esqueleto do `oficina-app` (Dockerfile, middleware JWT/correlação, observabilidade, workflows). Nada de framework novo.

---

## Os cortes

Simplificações deliberadas em relação ao [backlog](backlog.md). Cada uma vira uma linha na ADR correspondente, **com o caminho de evolução** — o avaliador aceita simplificação assumida, não simplificação escondida.

### Mantidos do plano anterior

| # | Corte | Desenho completo | Risco na nota |
|---|---|---|---|
| 1 | **Manter o repo `oficina-app`** como OS Service (sem rename) | `oficina-os-service` | Baixo — README deixa claro |
| 2 | **Só SQS FIFO**, sem SNS | SNS com *fan-out* | Nenhum |
| 3 | **Uma compensação por vez** (sequencial) | Compensações em paralelo | Nenhum |
| 4 | **Relay com líder único** (advisory lock no Postgres) | Relay particionado por saga | Nenhum |
| 5 | **Sem watchdog/timeouts** (F4-4.6): "aprovação expirada" vira evolução na RFC-006 | Prazos por etapa + watchdog | Baixo |
| 6 | **Checkout Pro** (link) e **estorno total** | Pix/Transparente, estorno parcial | Nenhum |
| 8 | **BDD só no `oficina-app`** | BDD nos 3 repos | Baixo — o enunciado pede "pelo menos um fluxo" |
| 10 | **Homologação dispensada** (fórum da Fase 3); repos novos só com `main` | `homolog` + `prod` | Médio — **cite o post nos 3 READMEs** |
| 11 | **Código de mensageria copiado** nos 3 repos | Módulo Go compartilhado | Nenhum (ADR-013) |
| 12 | **Injeção de falha por placa `FAL*`** | Falhas configuráveis | Nenhum |
| 13 | **Sem testes de contrato** (F4-5.3) | JSON Schema validado em cada repo | Nenhum visível |
| 14 | **Webhook `budget-response` (HMAC, Fase 2) removido**, não migrado | Migrado para o Billing | Baixo — registrar na ADR |
| 15 | **Terraform dos serviços aplicado localmente**, sem `terraform.yml` | Plan em PR / apply em merge | Baixo — o CI/CD exigido é do serviço |
| 16 | **Dashboard montado pela UI**, exportado em JSON no fim | Dashboard como código | Nenhum |
| 17 | **Consulta da saga** = `saga.estado` + `etapas` no `GET /v1/work-orders/{id}` | F4-1.4 completo | Nenhum |

### Novos (por causa do prazo de 4 dias)

| # | Corte | Desenho completo | Risco na nota |
|---|---|---|---|
| 7′ | **Billing e Execution com 1 réplica fixa**, sem HPA; OS mantém HPA 2–5 | HPA em todos | Baixo — e **resolve a ordem do relay do DynamoDB**, que não tem líder único (ver [arquitetura §6.3](arquitetura.md#63-execution-service-dynamodb-single-table)) |
| 18 | **Contrato de mensagens em Markdown** (catálogo da [arquitetura §5](arquitetura.md#5-contrato-de-mensagens)), sem `docs/mensagens/*.schema.json` | JSON Schema por tipo (F4-0.1 passo 2) | Nenhum |
| 19 | **API do mecânico enxuta:** listar a fila (GSI) e agir por `tarefa_id` (`iniciar`, `concluir-diagnostico`, `concluir`). Sem `POST /fila/proxima` com atribuição concorrente | F4-3.2 passo 3 + teste concorrente | Baixo — a fila continua existindo e ordenada |
| 20 | **BDD com 2 cenários:** caminho feliz e falha no reparo com estorno. Recusa de orçamento e pagamento recusado ficam em teste de tabela do pacote `saga` e no vídeo | 4 cenários | Baixo |
| 21 | **Observabilidade mínima:** APM dos 3, trace pelo SQS (vem pronto no pacote de mensageria), logs com `saga_id`, `SagaEvent`, dashboard de **4 painéis** e **1 alerta** (taxa de compensação). Sem `PagamentoEvent`, `TarefaEvent`, `OutboxSample`, alerta de webhook | F4-7.2 a F4-7.4 completos | Baixo |
| 22 | **Alarme da DLQ = CloudWatch + SNS e-mail num único `aws_cloudwatch_metric_alarm` por fila**, criado no `infra/` de cada serviço; se apertar, só a redrive (DLQ existe e recebe, sem alarme) | Métricas da DLQ no New Relic | Baixo |
| 23 | **Documentação condensada:** README de cada serviço + `docs/arquitetura.md` curto; **RFC-006 (saga)** completa e **RFC-005 (decomposição)** de 1 página; ADRs 009–016 em formato curto (contexto · decisão · consequência, 5–10 linhas cada) | RFCs e ADRs longas | Nenhum |
| 24 | **`security.yml` dos repos novos só com `gitleaks` e `govulncheck`** | + `gosec`, Trivy | Nenhum |
| 25 | **Dados do dashboard:** ~15 sagas na nuvem que terminam em recusa de orçamento (script, sem pagamento) + 3–5 pagas manualmente no sandbox, 2 delas com `FAL*` | 20+ sagas geradas com pagamento automatizado | Baixo |

### O que **não** cortar

Requisito literal do enunciado. Cortar qualquer um destes custa mais nota que qualquer atraso:

- **3 repositórios de serviço**, cada um com **banco próprio** — PostgreSQL (OS), PostgreSQL em **instância RDS própria** (Billing) e DynamoDB (Execution).
- **Nenhum acesso a banco alheio — inclusive a Lambda** (F4-1.5). Grifado em amarelo no PDF.
- **Mensageria assíncrona** de verdade entre os serviços.
- **Compensação demonstrável** com **estorno real** no Mercado Pago (cenário `FAL*`).
- **Estorno do pagamento tardio** (F4-2.5 passo 4). No Checkout Pro um cartão recusado pode não encerrar a cobrança: o cliente pode pagar de novo no mesmo link depois que a saga já compensou (comportamento da order com `OTHE` a confirmar no D2). Sem esse tratamento o dinheiro fica retido com a OS cancelada. *(No plano anterior este item era cortável — não é mais.)*
- **Integração com o Mercado Pago real** (sandbox) ao menos uma vez no vídeo; o stub é para teste.
- **BDD** de um fluxo completo, **cobertura ≥ 80% em cada serviço** com gate no CI, **SonarCloud** no CI.
- **Pipeline independente com deploy no EKS** em cada serviço, com o job de testes antes do deploy.
- **`main` protegida** com PR obrigatório e checks nos 2 repos novos.
- **Justificativa da saga no README** do `oficina-app`.

---

## Cronograma

Legenda: 🔴 bloqueante · 🟡 caminho crítico · 🟢 paralelizável · 🔵 documentação

### 🔴 D1 — ter 10/11 — Fundação e prazos de terceiros (11 h → 23 h)

**11:00–11:45 — todos juntos (sem código):**
- Distribuir as trilhas. Ler juntos o [catálogo de mensagens](arquitetura.md#52-catálogo) e a [máquina de estados](arquitetura.md#32-máquina-de-estados-da-saga) e **congelar** — a partir daqui, mudar nome de mensagem exige acordo na sincronização.
- Combinar os nomes exatos dos jobs que vão no ruleset: `Build e testes`, `Lint`, `Cobertura ≥ 80%`, `SonarCloud`, `gitleaks`.

| Trilha | Tarefas (em ordem) |
|---|---|
| **A** | 🔴 **F4-0.5** pacote `internal/infra/mensageria` (envelope, publisher, consumer, `outbox_pg`, `inbox_pg`, `trace.go` com os headers do New Relic) com testes — **push até 16 h** para B e C copiarem · F4-1.2 migration `000007` ([DDL corrigido](backlog.md#f4-12--status-cancelada-e-reserva-de-estoque)) + `Reservar/Liberar/Baixar` · F4-1.3 `UnitOfWork` e `CriarOS` transacional (OS + reserva + saga + outbox) · relay e consumidor rodando no `cmd/api` |
| **B** | 🔴 **11:45–13:30 D0 do Mercado Pago (F4-0.3):** ✅ **feito em 10/11** — Checkout Pro via **API de Orders**, credenciais de teste da aplicação, vendedor e comprador de teste; cobrança, consulta e estorno validados por `curl` (falta só a assinatura secreta do webhook) · F4-2.1 esqueleto do Billing (cópia do `oficina-app`), migration `000001_billing`, `Centavos int64` · às 16 h copiar a mensageria · consumidor `GerarOrcamento` respondendo `OrcamentoGerado` (já gravando orçamento e itens) |
| **C** | 🔴 **11:45–13:30 D0 de plataforma (F4-0.2/0.3):** `aws eks list-clusters` / quota de vCPU / Budgets · criar os 2 repos com o ruleset exportado, `soat-architecture`, projetos no SonarCloud, repositórios no Docker Hub, roles OIDC no `bootstrap/` + access entries · F4-0.4 `docker-compose.saga.yml` + `localstack-init.sh` (**decidir LocalStack × ElasticMQ até 15 h**) · F4-3.1 esqueleto do Execution com DynamoDB: consumidor `IniciarDiagnostico` → tarefa; `POST .../concluir-diagnostico` → `DiagnosticoConcluido` pela outbox · **20 h:** `terraform apply` do `oficina-infra-k8s` (3 nós + addon Pod Identity, F4-6.4) e do `oficina-infra-db` **em background** — erros de conta (quota, ELBv2) aparecem hoje, não na quinta |

**Sincronização 13 h:** Webhook do MP chegando no ngrok? AWS e repos ok? Se o MP travou, B continua no esqueleto com o stub e alguém reabre o MP às 18 h.

**Marco do dia (21 h):** no compose, criar uma OS faz `IniciarDiagnostico` chegar ao Execution e `DiagnosticoConcluido` voltar ao OS, que registra a etapa T2 em `sagas_os_etapas`. **Se não fechar, D2 começa com A e C juntos nisso.**

---

### 🟡 D2 — qua 11/11 — Saga completa no compose + nuvem provisionada

| Trilha | Tarefas (em ordem) |
|---|---|
| **A** | F4-4.4 orquestrador como tabela pura, com teste de tabela `estado × tipo` (100% no pacote `saga`) · F4-4.5 compensações (pilha inversa, `cancelada`, `LiberarReserva`) · F4-1.1 recorte (remover approve/reject, `PATCH` só `finalizada → entregue`) · F4-1.4 bloco `saga` no `GET` + `estoque_reservado` no `GET /v1/parts` · F4-1.5 lado OS: `GET /internal/clientes/por-cpf/{cpf}` |
| **B** | F4-2.2 aprovar (cria a order) / recusar / `CancelarOrcamento` · F4-2.3 adaptador do MP com testes `httptest` · F4-2.6 stub (`POST /_stub/pagar`) · F4-2.4 webhook ([fluxo corrigido](backlog.md#f4-24--webhook-do-mercado-pago)) — testar a assinatura com *Simular notificação* do painel via ngrok · F4-2.5 estorno **e pagamento tardio** · cobertura ≥ 80% no fim do dia |
| **C** | F4-3.2/3.3/3.4 tarefas `REPARO`, `iniciar`, `concluir`, `CancelarTarefas` · F4-4.7 falha `FAL*` · **a partir das 15 h:** F4-6.2 `infra/` dos 3 serviços (filas + DLQ + alarme, RDS do Billing, tabela DynamoDB, roles e associações do Pod Identity, SSM) aplicados · F4-6.3 manifestos `k8s/` dos 3 |

**Sincronização 13 h:** caminho feliz chega a `AGUARDANDO_APROVACAO` no compose.

**Marco do dia (21 h):** no compose, com o stub: **caminho feliz** até `finalizada`; **recusa de orçamento**, **pagamento recusado** e **falha `FAL*` com estorno** terminando em `COMPENSADA` com o estoque igual ao inicial. Na AWS: cluster com 3 nós, RDS do Billing, 6 filas e a tabela criados.

---

### 🟡 D3 — qui 12/11 — Nuvem ponta a ponta, qualidade e documentação · code freeze 22 h

| Trilha | Manhã (8 h → 13 h) | Tarde (14 h → 19 h) | Noite (19 h → 22 h) 🔵 |
|---|---|---|---|
| **A** | F4-5.2 BDD godog (2 cenários) + `bdd.yml` · `scripts/cobertura.sh` + job `Cobertura ≥ 80%` no `oficina-app` | Ajustar `cd.yml` do OS (`needs: testes`, ConfigMap das filas via SSM, ServiceAccount do Pod Identity) e implantar · F4-7.2 `SagaEvent` + `saga_id` nos logs | README do `oficina-app` com **"Saga Pattern — por que orquestrada"** · RFC-006 · RFC-005 (1 página) |
| **B** | `ci.yml`, `security.yml`, `cd.yml` e Sonar do Billing (cópia do OS) · primeiro deploy por **merge real** | F4-6.5 rotas `/billing/*` e webhook público no Gateway · webhook do painel do MP apontando para a nuvem · **saga real com pagamento sandbox e uma com `FAL*` + estorno na nuvem até 18 h** | F4-7.3 dashboard (4 painéis) · F4-7.4 alerta de compensação **disparado uma vez** · ADRs 009–016 (curtas) |
| **C** | `ci.yml`, `security.yml`, `cd.yml` e Sonar do Execution · deploy por **merge real** · F4-1.5 lado Lambda (REST interno, sem VPC, sem DSN) | Rotas `/execution/*` · conferir `kubectl get pods -A` sem `Pending` · **17 h:** script de ~15 sagas que terminam em recusa (corte 25) | READMEs dos 2 serviços novos e dos repos da Fase 3 · `docs/openapi.yaml` de cada serviço · coleção Postman da Fase 4 · diagrama final exportado |

**Sincronização 13 h:** os 3 serviços `Running` no EKS; BDD e cobertura verdes nos 3 repos. Se o BDD estiver instável no CI, deixe-o rodando no PR como check **não obrigatório** e siga — o relatório do run verde é a evidência.

**Marco do dia (21 h):** [checklist de entrega](entrega.md#checklist-antes-de-submeter) marcado, exceto vídeo e PDF. **Code freeze às 22 h** — depois disso, só correção de bug encontrado no ensaio.

---

### 🏁 D4 — sex 13/11 — Entrega

| Horário | Quem | O que fazer |
|---|---|---|
| 08:00–09:00 | C | Aquecer o ambiente (uma saga completa), `PERMITIR_FALHA_SIMULADA=true`, [preparação do roteiro](entrega.md#preparação-antes-de-ligar-a-gravação) |
| 09:00–12:00 | A + B | Ensaio cronometrado e **gravação por blocos** ([roteiro](entrega.md#roteiro-do-vídeo)). Bloco do pagamento real **primeiro** (sandbox é o ponto mais instável) |
| 09:00–13:00 | C | PDF ([modelo](entrega.md#conteúdo-do-pdf)) com prints: `push` rejeitado na `main`, `soat-architecture` aceito, policies sem acesso cruzado, Sonar dos 3 |
| 12:00–15:00 | A | Edição e upload (não listado) |
| 15:00–16:30 | todos | Revisão cruzada: cada um confere o PDF contra os repositórios e o vídeo contra os 4 itens exigidos |
| **até 17:00** | B | **Submissão no portal** (prazo interno 18 h) |
| após a confirmação | C | Teardown (F4-9.3) e conferência de órfãos |

---

## Ordem de sacrifício (se atrasar)

Corte de cima para baixo, **só nas sincronizações das 13 h e 21 h**. Cada linha diz o que se perde.

| # | Cortar | Perde |
|---|---|---|
| 1 | Alarme da DLQ (manter a redrive) | Alerta automático de mensagem venenosa — mostre a DLQ no console |
| 2 | Campo `saga` no `GET` da OS | Mostre `SELECT` em `sagas_os`/`sagas_os_etapas` no vídeo |
| 3 | Script de sagas para o dashboard | Dashboard só com as sagas feitas à mão |
| 4 | `SagaEvent` e dashboard próprio | Use *service map*, traces, logs por `saga_id` e o dashboard da Fase 3 |
| 5 | BDD rodando no CI | BDD roda local (`go test ./features -tags bdd`) e o relatório entra no README — **o requisito é ter BDD, o CI é bônus** |
| 6 | RFC-005 | Decomposição justificada só no README e na ADR |
| 7 | Trace distribuído atravessando o SQS | Fica a correlação por `saga_id` nos logs — **corte só se o pacote de mensageria atrasar o D1** |

**Nunca na lista:** qualquer item de [O que não cortar](#o-que-não-cortar).

---

## Três regras

1. **O marco do D1 é o plano.** A volta de uma mensagem OS → Execution → OS pelo SQS é o que todo o resto pressupõe. Se não fechar na terça à noite, quarta de manhã é só isso.
2. **Nada sobe para a nuvem sem passar no compose.** Cada `AccessDenied` na AWS custa um ciclo de deploy; cada bug de lógica achado no EKS custa cinco. A infraestrutura sobe cedo (D1 à noite, D2 à tarde) justamente para que só as **permissões** sejam surpresa na quinta.
3. **Registre no [diário](execucao.md) o que custou mais de 30 minutos.** Vira conteúdo para as ADRs e evita repetir o erro na trilha vizinha.
