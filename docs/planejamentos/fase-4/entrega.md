# Entrega — Fase 4

Checklist de fechamento, roteiro do vídeo e conteúdo do PDF do portal. O ambiente da demonstração é o **Kubernetes local** ([`local/`](../../../local/README.md), [ADR-017](../../architecture-decisions.md#adr-017--ambiente-kubernetes-local-e-deploy-em-kind-no-cd)); o deploy automatizado é o job **Deploy no Kubernetes (kind)** do CD de cada serviço.

---

## Checklist antes de submeter

### Repositórios (um por microsserviço)
- [x] `oficina-app` (OS Service), `oficina-billing` e `oficina-execution` públicos; `soat-architecture` convidado nos 6 repositórios
- [x] Cada um contém: código-fonte, `Dockerfile`, `k8s/` (base + overlay prod), `infra/` Terraform, `.github/workflows/` (CI + CD)
- [x] `main` protegida com PR obrigatório e checks obrigatórios — push direto rejeitado ([`docs/evidencias/push-main-rejeitado.txt`](../../evidencias/push-main-rejeitado.txt) nos 3)
- [x] CI e CD verdes no último commit de cada `main`
- [x] README com **evidência de cobertura** (badge do Sonar + `docs/evidencias/cobertura.png` + número e critério)
- [x] `docs/arquitetura.md` do Billing e do Execution; [arquitetura da Fase 4](arquitetura.md) para o conjunto
- [x] `docs/openapi.yaml` nos 3; coleção Postman da Fase 4 no `oficina-app` ([`docs/postman/fase-4-saga.json`](../../postman/fase-4-saga.json))
- [x] README do `oficina-app` com **"Saga Pattern — por que orquestrada"**, ambientes, observabilidade e evidências
- [x] ADRs 009–017, RFC-005 e RFC-006
- [x] `gitleaks` limpo nos 3; nenhum token versionado (Secrets do `.env` e do Secrets Manager)

### Funcionalidade (verificada no Kubernetes local com `demo-saga.sh todos`)
- [x] Caminho feliz: OS criada → diagnóstico → orçamento → aprovação → pagamento → reparo → `finalizada`; estoque 10 → 8
- [x] Orçamento recusado → OS `cancelada`, reserva liberada
- [x] Pagamento recusado → OS `cancelada`, order `canceled`, nova tentativa no mesmo link bloqueada
- [x] Falha no reparo (placa `FAL*`) → **estorno** (`refunded`), OS `cancelada`, estoque igual ao inicial
- [x] Mensagem reentregue não duplica efeito (inbox); DLQ após 5 tentativas
- [x] Nenhum serviço com credencial ou rota para o banco de outro (policies em `infra/*/pod.tf`; Lambda sem banco em [oficina-lambda-auth#6](https://github.com/ProblemaTheu/oficina-lambda-auth/pull/6))

### Qualidade
- [x] Cobertura ≥ 80% nos 3 (97,9% · 97,9% · 98,8%), com gate no CI e no CD
- [x] BDD (godog) com os 4 cenários da saga, no CI (`bdd.yml`)
- [x] Quality gate do SonarCloud verde nos 3

### Observabilidade (New Relic, conta `8465573`)
- [x] APM dos 3 serviços
- [x] Trace distribuído com spans dos 3 serviços para a mesma OS
- [x] Logs filtráveis por `saga_id` nos 3
- [x] Dashboard da saga com dados reais (`observability/aplicar-newrelic.py`)
- [x] Alerta de compensação **disparado** e e-mail entregue

### Encerramento
- [ ] Vídeo ≤ 15 min publicado (público ou não listado)
- [ ] PDF submetido no portal (com o link do vídeo)
- [ ] `kind delete cluster --name oficina` depois da gravação

---

## Roteiro do vídeo

**Duração-alvo: 14 minutos.** O enunciado pede quatro demonstrações; o roteiro dedica um bloco a cada uma. Grave por blocos e edite.

### Preparação (antes de ligar a gravação)

- [ ] `./local/subir.sh` com `NEW_RELIC_LICENSE_KEY` no `.env` — só começar quando aparecer **✅** (os serviços já saem aquecidos)
- [ ] ~20 sagas geradas antes (`for i in 1 2 3 4 5; do AMBIENTE=k8s ./scripts/demo-saga.sh todos; done`) para o dashboard ter dados
- [ ] Postman com a coleção da Fase 4 e o ambiente `local`
- [ ] Abas abertas: Mailpit (`localhost:8025`), New Relic (dashboard da saga, *service map*, Distributed tracing, Logs), GitHub Actions do Billing
- [ ] Um PR pequeno preparado no Billing ou no Execution, com os checks verdes, pronto para merge
- [ ] Terminal com `kubectl --context kind-oficina get pods -A` e fonte ≥ 16 pt; notificações desligadas

### Blocos

| Tempo | Bloco | O que mostrar |
|---|---|---|
| **0:00–1:30** | Abertura e arquitetura | Integrantes e o problema (rede nacional de filiais). Diagrama de componentes: *"três serviços, três bancos — dois PostgreSQL e um DynamoDB —, conversando por SQS FIFO; o OS Service orquestra a saga"*. Uma frase sobre **por que orquestrada**. Mostrar os 3 repos, a `main` protegida e os pods nos 3 namespaces (`kubectl get pods -A`). |
| **1:30–6:00** | **Fluxo completo de uma OS** | No Postman (pastas 0–6): atendente abre a OS → peça **reservada** (`estoque_reservado = 2`). Mecânico vê a fila, inicia e conclui o diagnóstico (Execution). A OS vai sozinha para `aguardando_aprovacao`; mostrar o **e-mail do orçamento no Mailpit**. Cliente aprova (Billing) → link de checkout do Mercado Pago → pagamento confirmado (stub; ou o sandbox real com `MP_REAL=1`, confirmado pelo reconciliador). Mecânico inicia (baixa do estoque) e conclui o reparo → OS `finalizada`; `GET /v1/work-orders/{id}` com o histórico de status. |
| **6:00–9:00** | **Saga e tratamento de falhas** | `AMBIENTE=k8s ./scripts/demo-saga.sh falha-reparo`: veículo `FAL*` chega até o pagamento, o Execution falha no reparo → OS `cancelada`, etapas `T5 → T3 → T2 → T1 compensada` **em ordem inversa**, order `refunded` (estorno) e estoque de volta a 10. Em 30 s: orçamento recusado e cartão recusado (nova tentativa bloqueada). Uma frase sobre idempotência (inbox) e DLQ. |
| **9:00–11:30** | **Deploy automatizado com validação de testes** | Mergear o PR preparado (checks obrigatórios verdes). No Actions, o CD daquele serviço: **Testes (gate de 80%)** com a cobertura no Summary → **Build e push** (imagem com o SHA) → **Deploy no Kubernetes (kind)**: dependências, `rollout status` e smoke `/health/ready` no Summary. Mostrar que **só aquele serviço** foi implantado e citar o destino EKS (`DEPLOY_ENABLED`). Citar o BDD rodando nos PRs. |
| **11:30–14:00** | **Monitoramento e rastreamento distribuído** | *Service map* com os 3 serviços. **Abrir um trace** de uma saga: `POST /v1/work-orders` → publicação SQS → consumo no Execution → … Logs filtrados por `saga_id` mostrando os 3 serviços em sequência. Dashboard da saga: desfechos, taxa de compensação, onde as sagas falham, tempo por etapa. Mostrar o **incidente do alerta de compensação** e o e-mail. Encerrar recapitulando os requisitos. |

**Erros que custam pontos:**
- Mostrar só o caminho feliz — a falha **com estorno** é o coração do enunciado.
- Esperar o processamento em tempo real — corte as esperas na edição.
- Dashboard vazio — gere as sagas antes.
- Ler código. Mostre comportamento; o código fica no repositório.
- Estourar 15 minutos.

---

## Conteúdo do PDF

> Documento único a submeter no Portal do Aluno, gerado a partir desta seção. Falta só o link do vídeo.

**Tech Challenge — Fase 4 — SOAT**

| Integrante | RM | GitHub |
|---|---|---|
| Guilherme Mendes da Cunha | RM371632 | Mendeszx |
| Lucas Evangelista Souza Silva | RM374023 | Lucas-Less |
| Matheus Souza Fernandes | RM371906 | ProblemaTheu |

### 1. Repositórios

| Repositório | Papel |
|---|---|
| [`oficina-app`](https://github.com/ProblemaTheu/oficina-app) | **OS Service**: clientes, veículos, catálogo, estoque com reserva, ciclo de vida da OS e **orquestrador da saga** (PostgreSQL) |
| [`oficina-billing`](https://github.com/ProblemaTheu/oficina-billing) | **Billing Service**: orçamento, aprovação, cobrança no Mercado Pago (API de Orders) e estorno (PostgreSQL) |
| [`oficina-execution`](https://github.com/ProblemaTheu/oficina-execution) | **Execution Service**: fila de diagnóstico e reparo, API do mecânico (DynamoDB) |
| [`oficina-lambda-auth`](https://github.com/ProblemaTheu/oficina-lambda-auth) | Autenticação por CPF (Lambda) e rotas do API Gateway |
| [`oficina-infra-k8s`](https://github.com/ProblemaTheu/oficina-infra-k8s) | VPC, EKS, API Gateway, roles OIDC dos pipelines, namespaces e acessos por serviço |
| [`oficina-infra-db`](https://github.com/ProblemaTheu/oficina-infra-db) | RDS PostgreSQL do OS Service |

Todos públicos, com a `main` protegida e o `soat-architecture` como colaborador.

### 2. Vídeo

`[link do YouTube/Vimeo]` — até 15 minutos.

### 3. Arquitetura

Diagrama: [arquitetura.md §1](arquitetura.md#1-diagrama-de-componentes). A tabela abaixo resume quem guarda o quê e quem fala com quem:

| Serviço | Banco (próprio) | Recebe por | Envia para |
|---|---|---|---|
| OS Service | PostgreSQL | REST `/v1/*` · SQS `os-saga-replies.fifo` · REST interno `/internal/*` | SQS `billing-commands.fifo`, `execution-commands.fifo` |
| Billing Service | PostgreSQL (instância separada) | REST `/billing/*` · webhook do Mercado Pago · SQS `billing-commands.fifo` | SQS `os-saga-replies.fifo` · API do Mercado Pago |
| Execution Service | DynamoDB `execution-tarefas` | REST `/execution/*` · SQS `execution-commands.fifo` | SQS `os-saga-replies.fifo` |

Nenhum serviço tem credencial ou rota para o banco de outro. Os mesmos manifests `k8s/base` rodam no Kubernetes local e no CI (kind) e no EKS; muda só o overlay ([arquitetura.md §8](arquitetura.md#8-ambientes-de-execução), ADR-017).

### 4. Saga Pattern: orquestrada

Orquestrador no OS Service, com o estado da saga gravado na mesma transação da OS, do estoque e da mensagem de saída (*Transactional Outbox*). As etapas: abertura com reserva (T1) → diagnóstico (T2) → orçamento (T3) → aprovação (T4) → pagamento (T5) → reparo, o ponto sem volta (T6) → conclusão (T7). As compensações correm sequenciais e em ordem inversa: estorno → cancelamento do orçamento → cancelamento das tarefas → liberação da reserva.

**Por que orquestrada:** fluxo linear com 3 participantes e esperas humanas. A orquestração deixa o fluxo numa máquina de estados só (100% de cobertura), a decisão de compensar num lugar só e o estado consultável numa tabela. A comparação completa está na RFC-006 e no README.

### 5. Justificativas

Divisão em 3 serviços por área de negócio; PostgreSQL para dinheiro e estoque (ACID); DynamoDB para a fila por chave e estado; SQS FIFO ordenado por saga com DLQ; Mercado Pago pela API de Orders (a que estorna no sandbox); EKS + Pod Identity; New Relic; GitHub Actions + SonarCloud + godog.

### 6. Documentação

README do `oficina-app` · [arquitetura.md](arquitetura.md) · `docs/arquitetura.md` do Billing e do Execution · RFC-005 e RFC-006 · ADRs 009–017 · `docs/openapi.yaml` nos 3 · coleção Postman da Fase 4 · `docs/evidencias/` nos 3 · [observability/](../../../observability/README.md)
