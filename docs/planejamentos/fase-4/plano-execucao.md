# Plano de execução — 7 dias (05/11 → 11/11)

> **Documento de trabalho.** Entrega no portal em **dom 15/11**; o código fecha na **qua 11/11** (metade da semana da entrega). Sobram qui–dom para gravação, PDF e imprevistos. O [roadmap](roadmap.md) descreve o "jeito certo com tempo"; este documento diz **o que cortar** para caber em 7 dias e **o que nunca cortar**.

| Item | Data |
|---|---|
| D0 — prazos de terceiros (curtos, já) | sáb 31/10 → qua 04/11 |
| **D1 → D7 — implementação** | **qui 05/11 → qua 11/11** |
| **Code freeze** | qua 11/11, 23 h |
| Ensaio e gravação | qui 12/11 |
| Edição do vídeo e PDF | sex 13/11 |
| **Submissão** (com um dia de folga) | **sáb 14/11** |
| Prazo final | dom 15/11 |

### Premissas

- **Três trilhas em paralelo**, uma por serviço, alinhadas com a [divisão do roadmap](roadmap.md#divisão-sugerida-por-integrante):
  - **A — OS + saga** (`oficina-app`)
  - **B — Billing + Mercado Pago** (`oficina-billing-service`)
  - **C — Execution + plataforma** (`oficina-execution-service`, `oficina-infra-k8s`, `oficina-lambda-auth`)
- **~8 h por trilha por dia** (~170 h no total, incluindo o fim de semana de 07–08/11). O escopo com cortes cabe em ~150 h.
- **Com menos de 3 pessoas** o escopo não cabe em 7 dias: aplique a [ordem de sacrifício](#ordem-de-sacrifício-se-atrasar) até o item 6 **antes** do D1 e junte as trilhas B e C.
- **Branch protection já está ativa** nos repositórios existentes; nos 2 repos novos basta replicar o mesmo ruleset no D0.
- **Sincronização diária de 15 min às 9 h**: cada trilha diz se o marco do dia anterior fechou. Marco que não fechou é tratado **antes** de qualquer tarefa nova.

---

## Os cortes

Simplificações deliberadas em relação ao [backlog](backlog.md). Cada uma vira uma linha na ADR correspondente, **com o caminho de evolução** — o avaliador aceita simplificação assumida; não aceita simplificação escondida.

| # | Corte | Desenho completo | Risco na nota |
|---|---|---|---|
| 1 | **Manter o repo `oficina-app`** como OS Service (sem rename) | Renomear para `oficina-os-service` | Baixo — README deixa claro |
| 2 | **Só SQS FIFO**, sem SNS | SNS para eventos de domínio com *fan-out* | Nenhum |
| 3 | **Uma compensação por vez** (sequencial) | Compensações independentes em paralelo | Nenhum |
| 4 | **Relay com líder único** (advisory lock) | Relay particionado por saga | Nenhum |
| 5 | **Sem watchdog/timeouts** (F4-4.6): o cenário "aprovação expirada" vira evolução na RFC-006 | Prazos por etapa + watchdog | Baixo — os outros 3 cenários de falha cobrem o requisito |
| 6 | **Checkout Pro** (link) e **estorno total** | Checkout Transparente/Pix, estorno parcial | Nenhum |
| 7 | **HPA 1–3** em Billing e Execution; HPA 2–5 só no OS | HPA 2–5 em todos | Baixo |
| 8 | **BDD só no `oficina-app`**, 2 cenários obrigatórios (feliz + falha após pagamento) e 2 se sobrar tempo | BDD nos 3 repos, 4 cenários | Baixo — o enunciado pede "pelo menos um fluxo" |
| 9 | **DLQ alertada pelo CloudWatch + SNS e-mail** | Métricas da DLQ no New Relic | Baixo |
| 10 | **Homologação dispensada** (orientação do fórum da Fase 3) | Ambientes `homolog` e `prod` | Médio — **cite o post nos 3 READMEs** |
| 11 | **Código de mensageria copiado** nos 3 repos | Módulo Go compartilhado | Nenhum (ADR-013) |
| 12 | **Injeção de falha por placa `FAL*`** | Falhas configuráveis | Nenhum |
| 13 | **Sem testes de contrato** (F4-5.3); o contrato fica em `docs/mensagens/` e o BDD pega quebras | JSON Schema validado em cada repo | Nenhum visível |
| 14 | **Webhook `budget-response` (HMAC, Fase 2) removido**, não migrado | Migrado para o Billing | Baixo — registrar na ADR |
| 15 | **Terraform dos serviços aplicado localmente**, sem `terraform.yml` nos repos novos | Plan em PR / apply em merge | Baixo — o requisito de CI/CD é do **serviço**, e o `cd.yml` fica |
| 16 | **Dashboard montado pela UI** e exportado em JSON no fim | Dashboard como código | Nenhum |
| 17 | **Consulta da saga** = campo `saga.estado` + `etapas` no `GET /v1/work-orders/{id}`, sem schema elaborado | F4-1.4 completo | Nenhum |

### O que não cortar

Estes itens são **requisito literal** do enunciado. Cortar qualquer um compromete a nota mais do que qualquer atraso:

- **3 repositórios de serviço**, cada um com **banco próprio** — e pelo menos um SQL **e** um NoSQL.
- **Nenhum acesso a banco alheio** — inclusive a Lambda (F4-1.5). Está grifado em amarelo no PDF.
- **Mensageria assíncrona** de verdade entre os serviços (não HTTP disfarçado).
- **Compensação demonstrável** — pelo menos o cenário de falha **após o pagamento**, com estorno.
- **Integração com o Mercado Pago** — o stub é para teste; o vídeo precisa mostrar o sandbox real ao menos uma vez.
- **BDD** de um fluxo completo, **cobertura ≥ 80% em cada serviço** e **Sonar no CI**.
- **Pipeline independente com deploy no Kubernetes** em cada serviço.
- **`main` protegida** com PR obrigatório e checks — já ativa; garantir nos 2 repos novos.
- **Justificativa da saga no README** do `oficina-app`.

---

## D0 — sáb 31/10 → qua 04/11 · prazos de terceiros (~1 h por dia, uma pessoa)

Nada disso depende de código, e tudo bloqueia o D1 se ficar para depois.

- [ ] **Mercado Pago:** aplicação Checkout Pro, vendedor e comprador de teste, `MP_ACCESS_TOKEN` e assinatura secreta do webhook no Secrets Manager (`oficina/prod/billing`). Uma preference criada por `curl` (F4-0.3).
- [ ] **AWS:** confirmar quota de vCPU ≥ 8, estado do ambiente (`aws eks list-clusters`) e AWS Budgets. Se o ambiente foi destruído, agendar o `apply` para o D3.
- [ ] **Repos novos:** criar `oficina-billing-service` e `oficina-execution-service`, replicar o **mesmo ruleset** da `main` dos repos existentes, convidar `soat-architecture`, criar os repositórios no Docker Hub e os projetos no SonarCloud (F4-0.2).
- [ ] **OIDC:** roles `gha-oficina-billing-service` e `gha-oficina-execution-service` no `bootstrap/` + access entries por namespace.
- [ ] **New Relic:** rotacionar as chaves (pendência da Fase 3).
- [ ] **Contrato (F4-0.1):** catálogo de mensagens e máquina de estados revisados **pelas três trilhas** — é a única coisa de código que precisa estar congelada antes do D1.

---

## Cronograma

Legenda: 🔴 bloqueante · 🟡 caminho crítico · 🟢 paralelizável · 🔵 documentação

### 🔴 D1 — qui 05/11 — Fundação

| Trilha | Tarefas |
|---|---|
| **A** | F4-0.5 pacote de mensageria (envelope, publisher, consumer, outbox, inbox Postgres) com testes · F4-1.2 migration 000007 |
| **B** | F4-2.1 esqueleto do Billing (copiando o pacote de A no fim do dia) · F4-2.3 adaptador do Mercado Pago (preference) com testes contra `httptest` |
| **C** | F4-0.4 `docker-compose.saga.yml` + `localstack-init.sh` (**decidir LocalStack × ElasticMQ hoje**) · F4-3.1 esqueleto do Execution com DynamoDB |

**Marco do dia:** `docker compose -f docker-compose.saga.yml up` sobe os 3 serviços saudáveis; o pacote de mensageria publica e consome no LocalStack.

### 🟡 D2 — sex 06/11 — A volta completa

| Trilha | Tarefas |
|---|---|
| **A** | F4-1.3 `CriarOS` numa transação (OS + reserva + saga + outbox) · F4-4.2 relay · F4-4.3 consumidor no OS · F4-4.4 tabela `fluxo` (sem compensação) |
| **B** | F4-2.2 orçamento: consumidor `GerarOrcamento`, e-mail no Mailpit, aprovar e recusar |
| **C** | F4-3.2 fila e tarefas · F4-3.3 API do mecânico · F4-3.4 comandos (`IniciarDiagnostico`, `ExecutarReparo`) |

**Marco do dia (o mais importante do plano):** criar uma OS no compose leva a saga sozinha até `AGUARDANDO_APROVACAO`, atravessando Execution e Billing pelo SQS. **Se não fechar, o D3 começa por aqui — com as três trilhas juntas.**

### 🟡 D3 — sáb 07/11 — Pagamento e caminho feliz

| Trilha | Tarefas |
|---|---|
| **A** | F4-4.4 orquestrador completo (caminho feliz) com teste de tabela · F4-1.1 recortar o OS (remover approve/reject, restringir o PATCH) · F4-1.4 campo `saga` no `GET` |
| **B** | F4-2.4 webhook do MP (`x-signature`, idempotência, verificação ativa) — testar com *Simular notificação* do painel via ngrok · F4-2.6 stub do MP |
| **C** | F4-6.4 3º nó + addon Pod Identity · F4-6.2 `infra/` dos 3 serviços (filas, RDS billing, DynamoDB, roles) — **a nuvem sobe hoje** |

**Marco do dia:** caminho feliz completo no compose (pagando pelo stub) **e** filas, RDS billing e tabela criadas na AWS.

### 🟡 D4 — dom 08/11 — Falhas

| Trilha | Tarefas |
|---|---|
| **A** | F4-4.5 compensações (pilha em ordem inversa, `cancelada`, `LiberarReserva`) · F4-4.7 falha simulada por placa `FAL*` |
| **B** | F4-2.5 estorno e cancelamento, incluindo pagamento tardio · cobertura do Billing até ≥ 80% |
| **C** | F4-6.3 manifestos dos 3 serviços + primeiro deploy manual no EKS · F4-1.5 endpoint interno no OS e Lambda sem banco |

**Marco do dia:** os cenários **recusa de orçamento**, **pagamento recusado** e **falha no reparo com estorno** compensam corretamente no compose; os 3 serviços `Running` no EKS.

### 🟡 D5 — seg 09/11 — Qualidade

| Trilha | Tarefas |
|---|---|
| **A** | F4-5.2 BDD godog (feliz + falha após pagamento; os outros 2 se sobrar tempo) + `bdd.yml` · cobertura do OS no critério do Sonar |
| **B** | F4-5.1 gate de cobertura (`scripts/cobertura.sh`) nos 3 repos · F4-5.4 SonarCloud com quality gate nos 3 |
| **C** | F4-6.1 `ci.yml` e `cd.yml` (com `needs: testes`) nos 2 repos novos; ajustar o `cd.yml` do `oficina-app` · cobertura do Execution até ≥ 80% |

**Marco do dia:** BDD verde no CI; os 3 repos com cobertura ≥ 80% e Sonar verde; checks obrigatórios do ruleset batendo com os nomes dos jobs.

### 🔴 D6 — ter 10/11 — Nuvem ponta a ponta

| Trilha | Tarefas |
|---|---|
| **A** | F4-7.1 trace atravessando o SQS · F4-7.2 logs com `saga_id` e `SagaEvent` |
| **B** | F4-6.5 rotas `/billing/*`, `/execution/*` e webhook público no Gateway · webhook do painel do MP apontando para a nuvem · **uma saga completa com pagamento sandbox na nuvem** |
| **C** | Validar o CD de cada serviço por **merge real** na `main` · iniciar o script de geração de sagas (~25% com placa `FAL*`) para popular o dashboard |

**Marco do dia:** saga real com pagamento sandbox e estorno funcionando no EKS; cada merge implanta só o próprio serviço.

### 🔵 D7 — qua 11/11 — Observabilidade, documentação e code freeze

| Trilha | Tarefas |
|---|---|
| **A** | README do `oficina-app` com a seção **"Saga Pattern — por que orquestrada"** · RFC-005 e RFC-006 |
| **B** | F4-7.3 dashboard da saga (pela UI) · F4-7.4 alertas de compensação e DLQ — **disparar cada um uma vez** |
| **C** | READMEs dos 3 serviços (cobertura, arquitetura, links) e dos repos da Fase 3 · ADRs 009–016 · OpenAPI de cada serviço · coleção Postman da Fase 4 · diagrama final exportado |

**Marco do dia:** checklist de [entrega.md](entrega.md#checklist-antes-de-submeter) inteiro marcado, exceto vídeo e PDF. **Code freeze às 23 h**: depois disso, só correção de bug que apareça no ensaio.

### 🏁 Qui 12/11 → dom 15/11 — Entrega

| Dia | O que fazer |
|---|---|
| **qui 12/11** | Ensaio cronometrado completo; gravação por blocos ([roteiro](entrega.md#roteiro-do-vídeo)). Ambiente aquecido, prazos curtos no ConfigMap. |
| **sex 13/11** | Edição e upload (não listado); PDF ([modelo](entrega.md#conteúdo-do-pdf)) conferido contra o que foi implantado. |
| **sáb 14/11** | **Submissão no portal.** Depois: teardown (F4-9.3) e conferência de órfãos. |
| dom 15/11 | Reserva — não planeje nada para este dia. |

---

## Ordem de sacrifício (se atrasar)

Corte de cima para baixo, **sempre na sincronização da manhã**, nunca no meio do dia. Cada linha diz o que se perde.

| # | Cortar | Perde |
|---|---|---|
| 1 | 3º e 4º cenários do BDD | Cobertura comportamental extra |
| 2 | Pagamento tardio (F4-2.5 passo 4) | Um caso de borda; registrar como risco conhecido na RFC-006 |
| 3 | Campo `saga` no `GET` da OS | Mostre `SELECT` em `sagas_os` no vídeo |
| 4 | Alerta da DLQ | Mantenha o de compensação — é o que o vídeo mostra |
| 5 | Endpoint interno + Lambda sem banco (F4-1.5) | ⚠️ **Fere a regra grifada do PDF.** Só corte se for isso ou a saga; nesse caso, registre na ADR-015 com a justificativa e o plano |
| 6 | HPA nos serviços novos | Réplica fixa de 1 em Billing e Execution |
| 7 | `SagaEvent` e dashboard próprio da saga | Use o *service map*, os traces e o dashboard da Fase 3 |

**Nunca na lista:** qualquer item de [O que não cortar](#o-que-não-cortar).

---

## Três regras

1. **O marco do D2 é o plano.** A volta completa de uma mensagem pelo SQS é o que todo o resto pressupõe. Se ela não fechar na sexta, o sábado começa com as três trilhas nela.
2. **Nada sobe para a nuvem sem passar no compose.** Cada `AccessDenied` na AWS custa um ciclo de deploy; cada bug de lógica achado no EKS custa cinco.
3. **Registre no [diário](execucao.md) o que custou mais de 30 minutos.** Na Fase 3, esses registros (TLS do RDS, ELBv2 bloqueado, `-target` sem IGW) evitaram repetir erros — e viraram conteúdo para as ADRs.
