# Tech Challenge — API de Oficina Mecânica

> API REST para gestão completa de uma oficina mecânica: autenticação JWT, clientes, veículos, catálogo de serviços, controle de estoque de peças e ordens de serviço com máquina de estados. Construída em Go com arquitetura API-first o contrato OpenAPI é a fonte da verdade e o código de roteamento/serialização é gerado automaticamente.
>
> **Fase 4**: a oficina vira **três microsserviços** — OS Service (este repositório, que também orquestra a saga), [Billing](https://github.com/ProblemaTheu/oficina-billing) e [Execution](https://github.com/ProblemaTheu/oficina-execution) — com banco próprio cada um (PostgreSQL, PostgreSQL e DynamoDB), comunicação assíncrona por **SQS FIFO**, **Saga Pattern orquestrado** com compensações, pagamento real no **Mercado Pago** (sandbox) e testes **BDD** do fluxo ponta a ponta.
>
> **Fase 3**: a aplicação subiu para a nuvem AWS gerenciada — autenticação de clientes por CPF no API Gateway (Lambda authorizer), EKS com autoescalonamento (HPA), RDS PostgreSQL, observabilidade no New Relic (APM, logs correlacionados, dashboards de negócio e alerta) e infraestrutura dividida em repositórios de IaC. Mantém o que veio da Fase 2: webhook de aprovação de orçamento, notificação de status por e-mail, Kubernetes e pipeline de CI/CD.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15.7-336791?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Kustomize%20%7C%20HPA-326CE5?logo=kubernetes&logoColor=white)
![Terraform](https://img.shields.io/badge/Terraform-kind%20%7C%20EKS-7B42BC?logo=terraform&logoColor=white)
![Security](https://img.shields.io/badge/Security-govulncheck%20%7C%20gosec%20%7C%20Trivy-green)

---

## Índice

- [Fase 4 — Microsserviços e Saga Pattern](#fase-4--microsserviços-e-saga-pattern)
  - [Saga Pattern — por que orquestrada](#saga-pattern--por-que-orquestrada)
- [Fase 3 — Nuvem, autenticação por CPF e observabilidade](#fase-3--nuvem-autenticação-por-cpf-e-observabilidade)
- [Fase 2 — Objetivos e arquitetura da solução](#fase-2--objetivos-e-arquitetura-da-solução)
- [Quickstart](#quickstart)
- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
- [Deploy em Kubernetes](#deploy-em-kubernetes)
- [Infraestrutura com Terraform](#infraestrutura-com-terraform)
- [Pré-requisitos](#pré-requisitos)
- [Executando com Docker](#executando-com-docker)
- [Executando sem Docker](#executando-sem-docker)
- [Dados de seed](#dados-de-seed)
- [Autenticação](#autenticação)
- [Variáveis de ambiente](#variáveis-de-ambiente)
- [Endpoints](#endpoints)
- [Máquina de estados — Ordem de Serviço](#máquina-de-estados--ordem-de-serviço)
- [Schema do banco de dados](#schema-do-banco-de-dados)
- [Testes](#testes)
- [CI/CD e Segurança](#cicd-e-segurança)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Gerando código a partir do OpenAPI](#gerando-código-a-partir-do-openapi)
- [Documentação adicional](#documentação-adicional)

---

## Fase 4 — Microsserviços e Saga Pattern

A Fase 4 decompõe a oficina em **três microsserviços**, cada um com repositório, banco, pipeline e deploy próprios, coordenados por uma **saga orquestrada** sobre **Amazon SQS FIFO**:

| Serviço | Repositório | Banco | Responsabilidade |
|---|---|---|---|
| **OS Service** + orquestrador | [`oficina-app`](https://github.com/ProblemaTheu/oficina-app) (este) | PostgreSQL | Clientes, veículos, catálogo, estoque com **reserva**, ciclo de vida da OS e **orquestração da saga** |
| **Billing Service** | [`oficina-billing`](https://github.com/ProblemaTheu/oficina-billing) | PostgreSQL (instância própria) | Orçamento, aprovação/recusa, cobrança no **Mercado Pago** (Checkout Pro, API de Orders), estorno |
| **Execution Service** | [`oficina-execution`](https://github.com/ProblemaTheu/oficina-execution) | **DynamoDB** (NoSQL) | Fila de diagnóstico e reparo, API do mecânico |

```mermaid
flowchart LR
    os["OS Service<br/>+ orquestrador"] -- "comandos" --> qb[["billing-commands.fifo"]] --> bill["Billing"]
    os -- "comandos" --> qe[["execution-commands.fifo"]] --> exec["Execution"]
    bill -- "respostas" --> qr[["os-saga-replies.fifo"]]
    exec -- "respostas" --> qr
    qr --> os
    bill <-- "Orders API · webhook" --> mp["Mercado Pago"]
    os --- pgos[("PostgreSQL OS")]
    bill --- pgb[("PostgreSQL Billing")]
    exec --- ddb[("DynamoDB")]
```

Nenhum serviço lê o banco de outro: toda colaboração passa pelas filas (comandos e respostas da saga) ou por REST autenticado com o mesmo JWT. O desenho completo — componentes na AWS, posse de dados, sequências e catálogo de mensagens — está em **[docs/planejamentos/fase-4/arquitetura.md](docs/planejamentos/fase-4/arquitetura.md)**; as decisões em **[RFC-005](docs/rfcs/RFC-005-decomposicao-em-microsservicos.md)**, **[RFC-006](docs/rfcs/RFC-006-saga-e-consistencia-distribuida.md)** e nas **[ADRs 009–016](docs/architecture-decisions.md#adr-009--banco-por-serviço-postgresql-os-billing-e-dynamodb-execution)**.

### Saga Pattern — por que orquestrada

O fluxo da OS é **linear** (diagnóstico → orçamento → aprovação → pagamento → reparo), tem **três participantes** e inclui **esperas humanas** (aprovação do cliente e pagamento). Nesse cenário a saga **orquestrada** vence a coreografada em tudo que importa:

| Critério | Orquestrada (escolhida) | Coreografada |
|---|---|---|
| Onde está o fluxo | Uma máquina de estados pura, num só arquivo ([`orquestrador.go`](internal/application/saga/orquestrador.go)) | Implícito, espalhado pelos *handlers* dos 3 serviços |
| Quem decide compensar | O orquestrador, com a lista de etapas concluídas na mão | Cada serviço precisa ouvir as falhas dos outros |
| Estado de uma OS no meio do caminho | `SELECT * FROM sagas_os WHERE os_id = …` + histórico em `sagas_os_etapas` | Reconstruir a partir de eventos em 3 bancos |
| Esperas humanas com prazo | Natural: estado + `prazo_em` | Ninguém é dono do timer |
| Acoplamento | O OS conhece os comandos dos outros (catálogo explícito) | Cada serviço conhece os eventos dos outros |
| Muitos participantes | Pior | Melhor |

O custo assumido — o OS Service conhece o contrato de comandos do Billing e do Execution — fica explícito no [catálogo de mensagens](docs/planejamentos/fase-4/arquitetura.md#52-catálogo). A argumentação completa está na [RFC-006](docs/rfcs/RFC-006-saga-e-consistencia-distribuida.md).

**Máquina de estados**

```mermaid
stateDiagram-v2
    [*] --> DIAGNOSTICO_PENDENTE: POST /v1/work-orders (T1: OS + reserva + IniciarDiagnostico na mesma TX)
    DIAGNOSTICO_PENDENTE --> ORCAMENTO_PENDENTE: DiagnosticoConcluido (T2)
    ORCAMENTO_PENDENTE --> AGUARDANDO_APROVACAO: OrcamentoGerado (T3)
    AGUARDANDO_APROVACAO --> AGUARDANDO_PAGAMENTO: OrcamentoAprovado (T4)
    AGUARDANDO_PAGAMENTO --> EXECUCAO_PENDENTE: PagamentoConfirmado (T5)
    EXECUCAO_PENDENTE --> EM_EXECUCAO: ExecucaoIniciada (T6, pivô, baixa do estoque)
    EM_EXECUCAO --> CONCLUIDA: ExecucaoConcluida (T7)
    DIAGNOSTICO_PENDENTE --> COMPENSANDO: DiagnosticoFalhou
    ORCAMENTO_PENDENTE --> COMPENSANDO: OrcamentoFalhou
    AGUARDANDO_APROVACAO --> COMPENSANDO: OrcamentoRecusado
    AGUARDANDO_PAGAMENTO --> COMPENSANDO: PagamentoRecusado
    EXECUCAO_PENDENTE --> COMPENSANDO: ExecucaoFalhou
    COMPENSANDO --> COMPENSADA: compensações confirmadas + reserva liberada
    CONCLUIDA --> [*]
    COMPENSADA --> [*]
```

**Compensações** — sequenciais, em ordem inversa, cada uma esperando a confirmação da anterior:

| Falha em | Compensações (nesta ordem) | Resultado |
|---|---|---|
| T1 — peça sem saldo | — (a transação local não comita) | `422`, nenhuma saga |
| T2 — diagnóstico falhou | `LiberarReserva` (local) | OS `cancelada` |
| T3 — orçamento falhou | `CancelarTarefas` → `LiberarReserva` | OS `cancelada` |
| T4 — **cliente recusou o orçamento** | `CancelarOrcamento` → `CancelarTarefas` → `LiberarReserva` | OS `cancelada` |
| T5 — **pagamento recusado** | `CancelarOrcamento` → `CancelarTarefas` → `LiberarReserva` | OS `cancelada`, order do MP cancelada |
| T6 — **falha ao iniciar o reparo** | `EstornarPagamento` → `CancelarOrcamento` → `CancelarTarefas` → `LiberarReserva` | OS `cancelada`, **refund** no MP |
| Depois de T6 | — **pivô**: peças já baixadas, carro desmontado; trata-se na oficina | — |

**Garantias de consistência**

- **Transactional Outbox:** estado da saga, mudança da OS e comando seguinte gravados na **mesma transação**; um *relay* publica a outbox no SQS ([ADR-012](docs/architecture-decisions.md#adr-012--transactional-outbox--inbox-idempotente-relay-com-líder-único)).
- **Inbox idempotente:** toda mensagem consumida é marcada em `mensagens_processadas` na transação do efeito — reentrega do SQS não duplica nada.
- **Ordem por saga:** `MessageGroupId = saga_id` nas filas FIFO; relay com líder único (*advisory lock* do Postgres).
- **Concorrência:** lock otimista em `sagas_os.versao`; réplicas do OS podem consumir em paralelo.
- **Pagamento tardio:** um `PagamentoConfirmado` que chega com a saga já compensando é **estornado** — o cliente nunca fica pagando por OS cancelada.
- **Falha de compensação:** reentrega até 5 vezes, depois DLQ; a saga fica em `COMPENSANDO` até o reprocessamento. Nunca vira `COMPENSADA` sem todas as confirmações.

### Rodando a saga localmente

Com os três repositórios lado a lado (`…/oficina-app`, `…/oficina-billing`, `…/oficina-execution`), sobe tudo — 3 serviços, 2 Postgres, LocalStack (SQS + DynamoDB), Mailpit e o stub do Mercado Pago:

```bash
cp .env.example .env
docker compose -f docker-compose.saga.yml --profile saga up -d --build
./scripts/demo-saga.sh feliz        # ou: falha-reparo | recusa | cartao-recusado | todos
```

| Porta | Serviço |
|---|---|
| 8080 | OS Service |
| 8081 | Billing |
| 8082 | Execution |
| 8090 | Stub do Mercado Pago (API de Orders + página de checkout) |
| 8025 | Mailpit (e-mails de orçamento) |
| 4566 | LocalStack |

A coleção Postman do fluxo completo está em [`docs/postman/fase-4-saga.json`](docs/postman/fase-4-saga.json), com os ambientes [`local`](docs/postman/local.postman_environment.json) e [`prod`](docs/postman/prod.postman_environment.json). Pelo Collection Runner, rode as pastas 0–6 em ordem (os passos assíncronos repetem sozinhos enquanto a saga avança); a pasta 7 traz os cenários de compensação.

### Testes BDD

Os quatro cenários da saga — caminho feliz, falha no reparo com estorno, orçamento recusado e cartão recusado — estão em [`features/saga_os.feature`](features/saga_os.feature) (Gherkin em português, `godog`). Rodam **só por HTTP** contra os três serviços reais:

```bash
docker compose -f docker-compose.saga.yml --profile saga up -d --build
go test -tags bdd ./features/ -v
```

No CI, o workflow [`bdd.yml`](.github/workflows/bdd.yml) faz checkout dos três repositórios, sobe o compose e roda os cenários a cada PR.

---

## Fase 3 — Nuvem, autenticação por CPF e observabilidade

A Fase 3 leva a aplicação da Fase 2 para uma **nuvem AWS gerenciada**, com autenticação de clientes na borda e observabilidade de ponta a ponta:

- **Autenticação de cliente por CPF na borda** — o **API Gateway** delega a um **Lambda authorizer** a validação do token antes de qualquer requisição chegar ao cluster. Um segundo Lambda (emissor) valida o CPF, consulta o cliente no banco, checa se está ativo e assina o JWT. Requisição sem credencial válida é barrada com **401 no próprio gateway**, sem gastar recurso do EKS;
- **Execução gerenciada no EKS** — o cluster kind local dá lugar ao **Amazon EKS**, com o mesmo autoescalonamento (HPA, 2–5 réplicas) e manifestos Kustomize (`k8s/overlays/prod`);
- **Persistência gerenciada** — o Postgres in-cluster dá lugar ao **RDS PostgreSQL**, provisionado por Terraform no repositório `oficina-infra-db`;
- **Observabilidade no New Relic** — APM automático por requisição, **logs em JSON correlacionados** (`correlation_id` + `trace.id`), **eventos de negócio** (`OrdemServicoEvent`, sem dado pessoal — LGPD por design) que alimentam dashboards de volume diário de OS, tempo médio por status e falhas de processamento, além de alerta de falha;
- **Infra dividida em repositórios de IaC** — a solução é distribuída em **4 repositórios** (aplicação, autenticação serverless, infra de cluster e infra de banco), cada um com seu ciclo de CI/CD.

O desenho completo — diagrama de componentes na AWS, diagramas de sequência da autenticação e da abertura de OS, e o modelo entidade-relacionamento — está em **[docs/planejamentos/fase-3/arquitetura.md](docs/planejamentos/fase-3/arquitetura.md)**. As decisões de nuvem, banco, autenticação e divisão de repositórios estão nas **[RFCs](docs/rfcs/)**; as decisões técnicas (EKS, HPA, padrão de comunicação, observabilidade, notificação por e-mail) nas **[ADRs 004–008](docs/architecture-decisions.md)**.

---

## Fase 2 — Objetivos e arquitetura da solução

Com o aumento da demanda da oficina, a Fase 2 evolui a aplicação da Fase 1 para garantir **qualidade, resiliência e escalabilidade**:

- **Escalabilidade dinâmica** — deploy em Kubernetes com Horizontal Pod Autoscaler (2–5 réplicas conforme CPU), pronto para picos de ordens de serviço;
- **Automação de provisionamento e deploy** — infraestrutura como código (Terraform) e pipeline de CI/CD;
- **Novas integrações** — webhook para aprovação/recusa de orçamento por sistemas externos (assinatura HMAC) e notificação do cliente por e-mail a cada mudança de status relevante;
- **Qualidade contínua** — refatoração Clean Architecture, cobertura ampliada de testes (unitários + integração com Postgres real) e lint zerado no CI.

### Arquitetura da solução

```mermaid
flowchart TB
    subgraph externos["Mundo externo"]
        usuario["👤 Usuário<br/>(portal/Postman)"]
        provedor["📧 Provedor externo<br/>(webhook de orçamento)"]
    end

    subgraph cluster["Cluster Kubernetes — kind (local) / EKS (futuro)"]
        subgraph ns["namespace: oficina"]
            svc["Service api<br/>(ClusterIP 80→8080)"]
            subgraph deploy["Deployment api (2–5 réplicas)"]
                pod1["Pod API"]
                pod2["Pod API"]
            end
            hpa["HPA autoscaling/v2<br/>CPU 50%, min 2 / max 5"]
            cm["ConfigMap<br/>oficina-config"]
            sec["Secret<br/>oficina-secrets"]
            pg[("StatefulSet postgres<br/>+ PVC 1Gi")]
            mp["Mailpit<br/>(SMTP fake local)"]
        end
        ms["metrics-server"]
    end

    usuario -- "JWT Bearer" --> svc
    provedor -- "X-Signature (HMAC-SHA256)" --> svc
    svc --> pod1 & pod2
    pod1 & pod2 --> pg
    pod1 & pod2 -- "notificação por e-mail<br/>(assíncrona)" --> mp
    cm -.envFrom.-> deploy
    sec -.envFrom.-> deploy
    ms -. métricas de CPU .-> hpa
    hpa -- escala --> deploy
```

Em produção (AWS, preparado no épico de IaC): o Postgres in-cluster dá lugar ao **RDS**, o Mailpit a um provedor SMTP real, e o cluster passa a ser o **EKS** — o overlay real é `k8s/overlays/prod`, e a infraestrutura vive nos repositórios `oficina-infra-k8s` e `oficina-infra-db`.

### Fluxo de deploy

```mermaid
flowchart LR
    dev["👨‍💻 Dev"] -- "PR → main" --> gh["GitHub"]

    subgraph actions["GitHub Actions"]
        ci["ci.yml<br/>build + testes -race + lint"]
        it["integration.yml<br/>repositório × Postgres real"]
        sec2["security.yml<br/>govulncheck · gosec · Trivy · Sonar"]
    end

    gh --> ci & it & sec2
    ci -- merge aprovado --> main["branch main"]
    main -- "build + push<br/>(F2-6.2)" --> hub["Docker Hub"]
    hub -- "kubectl apply -k<br/>(F2-6.3)" --> k8s["Cluster Kubernetes"]

    tf["Terraform<br/>infra/environments"] -- provisiona --> k8s
```

> Desde a Fase 3 o `cd.yml` implanta de verdade: push na `main` publica a imagem no Docker Hub e faz o rollout no **EKS** (overlay `k8s/overlays/prod`), autenticando na AWS por OIDC. O ambiente local (kind) continua sendo implantado manualmente com `scripts/k8s-local-deploy.sh` — um runner hospedado do GitHub não alcança um cluster na sua máquina.

### Onde está cada coisa

| Entregável | Local | Instruções |
|---|---|---|
| Manifestos Kubernetes | [`/k8s`](k8s/) | [k8s/README.md](k8s/README.md) |
| Scripts Terraform | [`/infra`](infra/) | [infra/README.md](infra/README.md) |
| Pipelines CI/CD | [`.github/workflows/`](.github/workflows/) | [seção CI/CD](#cicd-e-segurança) |
| Contrato OpenAPI | [`docs/openapi.yaml`](docs/openapi.yaml) | — |
| Coleção Postman | [`docs/postman_collection.json`](docs/postman_collection.json) | inclui webhook com assinatura automática |
| Vídeo demonstrativo | *(link será adicionado na entrega)* | roteiro em [docs/planejamentos/fase-2/roteiro-video.md](docs/planejamentos/fase-2/roteiro-video.md) |

---

## Quickstart

```bash
git clone https://github.com/ProblemaTheu/oficina-app.git
cd oficina-app
cp .env.example .env      # segredos e configurações locais
docker compose up -d --build
```

Aguarde o healthcheck e teste:

```bash
# Verificar se a API está pronta
curl http://localhost:8080/health/ready

# Obter token JWT (usuário admin padrão)
curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@oficina.com","senha":"Admin@123"}' | jq .access_token

# Listar clientes (substituir <token> pelo token retornado)
curl http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <token>"
```

A documentação completa do contrato está em [`docs/openapi.yaml`](docs/openapi.yaml) e a coleção Postman em [`docs/postman_collection.json`](docs/postman_collection.json).

---

## Tecnologias

| Tecnologia | Versão | Uso |
|---|---|---|
| Go | 1.26.2 | Linguagem principal |
| PostgreSQL | 15.7 | Banco de dados relacional |
| chi | v5.2.5 | Router HTTP |
| oapi-codegen | v2.6.0 | Geração de código a partir do OpenAPI |
| golang-migrate | v4.19.1 | Migrations de banco de dados |
| golang-jwt/jwt | v5.3.1 | Autenticação JWT (HS256) |
| golang.org/x/crypto | v0.50.0 | Hash de senhas (bcrypt) |
| lib/pq | v1.12.3 | Driver PostgreSQL |
| Docker / Compose | — | Containerização |

---

## Arquitetura

O projeto segue **Clean Architecture** com geração de código API-first:

```
docs/openapi.yaml               ← fonte da verdade do contrato
        │
        ▼  go generate
internal/infra/http/api/
  api.gen.go                    ← gerado pelo oapi-codegen (NÃO editar)
  server.go                     ← implementação dos handlers
  generate.go                   ← diretiva go:generate

internal/infra/http/middleware/
  jwt.go                        ← validação de JWT Bearer token

internal/application/usecase/   ← regras de negócio
internal/domain/entity/         ← entidades de domínio
internal/domain/valueobject/    ← validações (CPF/CNPJ, placa)
internal/domain/erros/          ← tipos de erro de domínio
internal/infra/repository/      ← acesso ao banco de dados
internal/infra/database/        ← conexão, migrations e arquivos SQL
```

### Fluxo de uma requisição autenticada

```
HTTP Request
  → chi router
  → JWT middleware (valida Bearer token, injeta claims no contexto)
  → StrictHandler (gerado — deserialização e validação de schema)
  → server.go (handler)
  → UseCase (regra de negócio)
  → Repository
  → PostgreSQL
```

### Camadas e responsabilidades

| Camada | Localização | Responsabilidade |
|---|---|---|
| HTTP | `infra/http/api/server.go` | Receber, validar e responder requisições |
| Middleware | `infra/http/middleware/jwt.go` | Autenticação e injeção de claims |
| UseCase | `application/usecase/` | Regras de negócio e orquestração |
| Domain | `domain/entity/` + `domain/valueobject/` | Entidades e value objects |
| Repository | `infra/repository/` | Persistência e consultas SQL |
| Database | `infra/database/` | Conexão, pool e migrations |

---

## Pré-requisitos

### Com Docker
- [Docker](https://docs.docker.com/get-docker/) 24+
- [Docker Compose](https://docs.docker.com/compose/) v2+

### Sem Docker
- [Go](https://go.dev/dl/) 1.26+
- [PostgreSQL](https://www.postgresql.org/download/) 15+
- (Opcional) [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) para rodar migrations manualmente

---

## Executando com Docker

### 1. Clone o repositório

```bash
git clone https://github.com/ProblemaTheu/oficina-app.git
cd oficina-app
```

### 2. Configure o ambiente e suba todos os serviços

```bash
cp .env.example .env      # obrigatório — segredos ficam fora do compose
docker compose up -d --build
```

Isso irá:
- Construir a imagem da aplicação Go (multi-stage, Alpine, ~27MB)
- Subir o PostgreSQL 15.7 com healthcheck (exposto na porta **5433** do host)
- Subir o **Mailpit** (SMTP fake) — os e-mails de notificação de status ficam visíveis em http://localhost:8025
- Aguardar o banco estar pronto antes de iniciar a API
- Executar todas as migrations automaticamente (schema + seed do catálogo)

### 3. Verifique se está rodando

```bash
curl http://localhost:8080/health/ready
```

Resposta esperada:
```json
{
  "status": "UP",
  "components": {
    "db": { "status": "UP" }
  }
}
```

### Portas expostas

| Serviço | Porta (host) | Porta (container) |
|---|---|---|
| API | 8080 | 8080 |
| PostgreSQL | 5433 | 5432 |
| Mailpit (SMTP) | 1025 | 1025 |
| Mailpit (UI web) | 8025 | 8025 |

### Comandos úteis

```bash
# Parar os serviços (mantém os dados)
docker compose down

# Parar e remover todos os dados do banco
docker compose down -v

# Rebuild apenas da aplicação após mudanças no código
docker compose up -d --build app

# Ver logs da aplicação em tempo real
docker compose logs -f app
```

---

## Executando sem Docker

### 1. Configure o PostgreSQL

Crie o banco de dados:

```sql
CREATE DATABASE tech_challenge_db;
```

### 2. Configure as variáveis de ambiente

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=tech_challenge_db
export JWT_SECRET=sua-chave-secreta-aqui
```

### 3. Instale as dependências e execute

As migrations são executadas automaticamente na inicialização:

```bash
go mod download
go run ./cmd/api
```

A API estará disponível em `http://localhost:8080`.

### 4. Build manual

```bash
go build -o api ./cmd/api
./api
```

### 5. (Opcional) Rodar migrations manualmente

```bash
migrate -path internal/infra/database/migrations \
        -database "postgres://postgres:postgres@localhost:5432/tech_challenge_db?sslmode=disable" \
        up
```

---

## Deploy em Kubernetes

Os manifestos ficam em [`/k8s`](k8s/) (Kustomize: `base` + `overlays/local` + `overlays/prod`). O caminho rápido para o ambiente local completo — cluster kind, metrics-server, Secret a partir do `.env`, API, Postgres e Mailpit:

```bash
cp .env.example .env              # se ainda não existir
./scripts/k8s-local-deploy.sh
```

Acessos e demonstração do HPA:

```bash
kubectl -n oficina port-forward svc/api 8080:80        # API
kubectl -n oficina port-forward svc/mailpit 8025:8025  # e-mails
kubectl -n oficina get hpa -w                          # observar a escala
```

Detalhes (passo a passo manual, decisões, segredos): [k8s/README.md](k8s/README.md).

---

## Infraestrutura com Terraform

O provisionamento está em [`/infra`](infra/), separado por ambiente:

```bash
# Local: cria o cluster kind + metrics-server
terraform -chdir=infra/environments/local init
terraform -chdir=infra/environments/local apply

# AWS (EKS + RDS) — OPT-IN, gera custo; ver infra/README.md antes
terraform -chdir=infra/environments/aws plan
```

Recursos criados, variáveis e instruções de destroy: [infra/README.md](infra/README.md).

### Fase 4 — infraestrutura do OS Service (`infra/servico`)

Cada microsserviço provisiona **a sua** parte da saga no próprio repositório (F4-6.2). A do OS Service:

| Recurso | Para quê |
|---|---|
| `os-saga-replies.fifo` + DLQ (redrive após 5) | Fila de entrada do orquestrador: respostas do Billing e do Execution |
| Role `oficina-os-pod` + **EKS Pod Identity** (`oficina-prod` / SA `api`) | Envia só para `billing-commands.fifo` e `execution-commands.fifo` (ARN por convenção, sem ler o state dos outros) e consome só a própria fila |
| Alarme da DLQ (CloudWatch + SNS) | F4-7.4 |
| SSM `/oficina/prod/os/fila-respostas-url` | Lido pelo CD do Billing e do Execution |

O CD gera o ConfigMap `oficina-ambiente` a partir da SSM (`DB_HOST` do `oficina-infra-db` e as três URLs de fila). Assim nenhum endpoint fica fixo no overlay. O Secret continua vindo de `scripts/prod-secret.sh`.

**Ordem da primeira subida da Fase 4** (os três `infra/` aplicam em qualquer ordem entre si):

```bash
# 1. plataforma: oficina-infra-k8s (cluster, namespaces, access entries) → oficina-infra-db (RDS do OS)
# 2. infra de cada serviço — aqui, no oficina-billing e no oficina-execution:
terraform -chdir=infra/servico init && terraform -chdir=infra/servico apply
# 3. Secrets: ./scripts/prod-secret.sh aqui (cria o JWT compartilhado), depois nos outros dois
# 4. deploy: variable DEPLOY_ENABLED=true nos três repositórios e merge/`gh workflow run cd.yml`
```

---

## Dados de seed

### Catálogo (migration 000004 — automático)

Aplicado automaticamente na inicialização. Inclui:
- **3 usuários** de sistema (admin, mecânico, atendente)
- **15 serviços** no catálogo (troca de óleo, alinhamento, freios, etc.)
- **20 peças** no inventário com quantidades mínimas

### Dados mockados (opcional — para testes de endpoints)

Popula o banco com clientes, veículos e ordens de serviço em todos os 6 status possíveis:

```bash
# Com Docker (banco exposto na porta 5433)
psql -h localhost -p 5433 -U postgres -d tech_challenge_db \
     -f scripts/seed_dados_completos.sql

# Sem Docker
psql -U postgres -d tech_challenge_db \
     -f scripts/seed_dados_completos.sql
```

> O script usa `ON CONFLICT DO NOTHING` — seguro re-executar.

### Utilitário de hash de senha

Para gerar hashes bcrypt de novas senhas:

```bash
go run scripts/genhash/main.go
```

### Importando a coleção Postman

1. Abra o Postman
2. Clique em **Import**
3. Selecione `docs/postman_collection.json`
4. Configure a variável de ambiente `base_url` como `http://localhost:8080`
5. Execute o request de login para obter o token e configurar a variável `token`

---

## Autenticação

A API usa **JWT Bearer tokens** com algoritmo HS256, validade de **8 horas**.

### Credenciais padrão (seed do catálogo)

| E-mail | Senha | Papel |
|---|---|---|
| `admin@oficina.com` | `Admin@123` | Administrador |
| `joao@oficina.com` | `Mecanico@123` | Mecânico |
| `ana@oficina.com` | `Atende@123` | Atendente |

### Obtendo um token

```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@oficina.com", "senha": "Admin@123"}'
```

Resposta:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 28800
}
```

### Usando o token

```bash
curl http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <token>"
```

### Rotas públicas (sem autenticação)

| Rota | Motivo |
|---|---|
| `POST /v1/auth/login` | Geração de token |
| `POST /v1/auth/register` | Cadastro de usuário |
| `GET /v1/work-orders/{id}/status` | Consulta de status por clientes externos |

Todas as demais rotas exigem `Authorization: Bearer <token>`.

---

## Variáveis de ambiente

No ambiente local elas vêm do arquivo `.env` (copie de [`.env.example`](.env.example)); no Kubernetes, do ConfigMap `oficina-config` e do Secret `oficina-secrets`.

| Variável | Padrão | Descrição |
|---|---|---|
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_USER` | `postgres` | Usuário do banco |
| `DB_PASSWORD` | `postgres` | Senha do banco |
| `DB_NAME` | `tech_challenge_db` | Nome do banco de dados |
| `JWT_SECRET` | *(inseguro — altere em produção)* | Chave de assinatura dos tokens JWT |
| `WEBHOOK_SECRET` | *(inseguro — altere em produção)* | Segredo do HMAC que assina os webhooks inbound |
| `NEW_RELIC_APP_NAME` | `oficina-api-local` | Nome da aplicação no New Relic |
| `NEW_RELIC_LICENSE_KEY` | *(vazio)* | License key do APM/Logs do New Relic. Sem ela, a app roda sem o agente |
| `NOTIFIER` | `log` | Notificação de status: `log` (console) ou `smtp` (e-mail real/Mailpit) |
| `SMTP_HOST` | `localhost` | Host do servidor SMTP |
| `SMTP_PORT` | `1025` | Porta do servidor SMTP |
| `SMTP_USER` / `SMTP_PASSWORD` | *(vazios)* | Autenticação PLAIN — só para provedor real |
| `EMAIL_FROM` | `oficina@example.com` | Remetente das notificações |

> **Atenção:** em produção, defina `JWT_SECRET` e `WEBHOOK_SECRET` com valores longos e aleatórios (mínimo 32 caracteres). Os padrões são deliberadamente inseguros.

---

## Endpoints

A API está disponível sob o prefixo `/v1`. Rotas marcadas com 🔓 são públicas.

### Health (sem prefixo `/v1`)

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | Status geral (app + dependências) |
| GET | `/health/live` | Liveness — processo está de pé |
| GET | `/health/ready` | Readiness — banco de dados acessível |

### Auth 🔓

| Método | Rota | Descrição |
|---|---|---|
| POST | `/v1/auth/login` | Autenticação — retorna JWT |
| POST | `/v1/auth/register` | Cadastro de novo usuário |

### Clientes

| Método | Rota | Filtros disponíveis | Descrição |
|---|---|---|---|
| GET | `/v1/clients` | `page`, `limit`, `nome`, `documento` | Listar clientes (paginado) |
| POST | `/v1/clients` | — | Cadastrar cliente |
| GET | `/v1/clients/{id}` | — | Buscar cliente por ID |
| PUT | `/v1/clients/{id}` | — | Atualizar cliente |
| DELETE | `/v1/clients/{id}` | — | Remover cliente |
| GET | `/v1/clients/{id}/vehicles` | — | Listar veículos do cliente |

### Veículos

| Método | Rota | Filtros disponíveis | Descrição |
|---|---|---|---|
| GET | `/v1/vehicles` | `page`, `limit`, `placa`, `cliente_id`, `marca` | Listar veículos (paginado) |
| POST | `/v1/vehicles` | — | Cadastrar veículo |
| GET | `/v1/vehicles/{id}` | — | Buscar veículo por ID |
| PUT | `/v1/vehicles/{id}` | — | Atualizar veículo |
| DELETE | `/v1/vehicles/{id}` | — | Remover veículo |

### Serviços

| Método | Rota | Filtros disponíveis | Descrição |
|---|---|---|---|
| GET | `/v1/services` | `page`, `limit`, `nome` | Listar serviços (paginado) |
| POST | `/v1/services` | — | Cadastrar serviço |
| GET | `/v1/services/{id}` | — | Buscar serviço por ID |
| PUT | `/v1/services/{id}` | — | Atualizar serviço |
| DELETE | `/v1/services/{id}` | — | Remover serviço |

### Peças

| Método | Rota | Filtros disponíveis | Descrição |
|---|---|---|---|
| GET | `/v1/parts` | `page`, `limit`, `nome`, `codigo`, `estoque_baixo` | Listar peças (paginado) |
| POST | `/v1/parts` | — | Cadastrar peça |
| GET | `/v1/parts/{id}` | — | Buscar peça por ID |
| PUT | `/v1/parts/{id}` | — | Atualizar peça |
| DELETE | `/v1/parts/{id}` | — | Remover peça |
| PATCH | `/v1/parts/{id}/stock` | — | Ajustar estoque |

**Operações de ajuste de estoque** (`PATCH /v1/parts/{id}/stock`):

| Operação | Efeito |
|---|---|
| `entrada` | Incrementa o estoque atual |
| `saida` | Decrementa o estoque atual |
| `ajuste` | Define o estoque para um valor absoluto |

### Ordens de Serviço

| Método | Rota | Filtros disponíveis | Descrição |
|---|---|---|---|
| GET | `/v1/work-orders` | `page`, `limit`, `status`, `cliente_id`, `veiculo_id`, `incluir_encerradas` | Listar OSs por prioridade de status (mais antigas primeiro); `finalizada`/`entregue` ficam fora da listagem padrão |
| POST | `/v1/work-orders` | — | Abrir nova OS |
| GET | `/v1/work-orders/{id}` | — | Detalhes completos (itens + histórico) |
| GET | `/v1/work-orders/{id}/status` | — | Consultar status (rota pública) |
| PATCH | `/v1/work-orders/{id}/status` | — | Avançar status da OS |
| POST | `/v1/work-orders/{id}/approve` | — | Aprovar orçamento |
| POST | `/v1/work-orders/{id}/reject` | — | Rejeitar orçamento |

### Webhooks

| Método | Rota | Autenticação | Descrição |
|---|---|---|---|
| POST | `/v1/webhooks/budget-response` | Assinatura HMAC-SHA256 (`X-Signature`) | Recebe aprovação/recusa do orçamento vinda de integração externa (idempotente) |

### Relatórios

| Método | Rota | Filtros disponíveis | Descrição |
|---|---|---|---|
| GET | `/v1/reports/avg-execution-time` | `servico_id`, `data_inicio`, `data_fim` | Tempo médio de execução por serviço |

O contrato completo com exemplos de request/response está em [`docs/openapi.yaml`](docs/openapi.yaml). A coleção Postman está em [`docs/postman_collection.json`](docs/postman_collection.json).

---

## Máquina de estados — Ordem de Serviço

Toda OS percorre um fluxo de status com transições controladas. Tentativas de transições inválidas são rejeitadas com erro HTTP 422.

```
        abertura
           │
           ▼
      ┌─────────┐
      │recebida │
      └────┬────┘
           │ PATCH /status
           ▼
    ┌──────────────┐
    │em_diagnostico│
    └──────┬───────┘
           │ PATCH /status
           ▼
  ┌────────────────────┐
  │aguardando_aprovacao│
  └─────────┬──────────┘
       ┌────┴───────┐
    /approve     /reject
       │            │
       ▼            ▼
┌───────────┐  ┌───────────┐
│em_execucao│  │finalizada │
└─────┬─────┘  │(rejeitada)│
      │        └─────┬─────┘
      │ PATCH /status│
      ▼              │
┌───────────┐        │
│finalizada │        │ PATCH /status
│(concluída)│        │
└─────┬─────┘        │
      │ PATCH /status│
      ▼              ▼
   ┌───────────────────┐
   │     entregue      │
   └───────────────────┘
```

---

## Documentação adicional

| Documento | Conteúdo |
|-----------|----------|
| [docs/objectives.md](docs/objectives.md) | Objetivos do sistema, escopo, requisitos funcionais e não-funcionais |
| [docs/planejamentos/fase-4/arquitetura.md](docs/planejamentos/fase-4/arquitetura.md) | Arquitetura da Fase 4: os 3 microsserviços, saga (estados, etapas, compensações), sequências, catálogo de mensagens e modelo de dados por serviço |
| [docs/planejamentos/fase-3/arquitetura.md](docs/planejamentos/fase-3/arquitetura.md) | Arquitetura da Fase 3: diagrama de componentes (AWS), sequência da autenticação e da abertura de OS, e o modelo ER |
| [docs/rfcs/](docs/rfcs/) | RFCs: escolha da nuvem, banco de dados, autenticação, estratégia de ambientes, decomposição em microsserviços e saga |
| [docs/architecture-decisions.md](docs/architecture-decisions.md) | 16 ADRs: PostgreSQL, chi, oapi-codegen, EKS, HPA, padrão de comunicação, observabilidade, e-mail; e da Fase 4 — banco por serviço, SQS FIFO, saga orquestrada, outbox/inbox, mensageria duplicada, Mercado Pago, Lambda via REST e Pod Identity |
| [docs/ubiquitous-language.md](docs/ubiquitous-language.md) | Glossário dos termos do domínio (linguagem ubíqua) |
| [docs/ddd-aggregates.md](docs/ddd-aggregates.md) | Agregados DDD, entidades internas e invariantes |
| [docs/bounded-contexts.md](docs/bounded-contexts.md) | Contextos delimitados e mapa de contextos |
| [docs/openapi.yaml](docs/openapi.yaml) | Contrato OpenAPI completo da API |
| [docs/postman_collection.json](docs/postman_collection.json) | Coleção Postman com todos os endpoints |
| [docs/postman/fase-4-saga.json](docs/postman/fase-4-saga.json) | Coleção Postman da Fase 4: fluxo completo da saga nos 3 serviços |
| [scripts/demo-saga.sh](scripts/demo-saga.sh) | Conduz os 4 cenários da saga no ambiente local |

Os números de OS são gerados automaticamente no formato `OS-YYYY-NNNNN` (ex: `OS-2025-00001`).

---

## Schema do banco de dados

O banco é criado e versionado via migrations em `internal/infra/database/migrations/`. As migrations são executadas automaticamente na inicialização.

### Tabelas principais

| Tabela | Descrição |
|---|---|
| `papeis_usuario` | Papéis de acesso (administrador, mecanico, atendente) |
| `status_ordens` | Status possíveis de uma OS (6 registros) |
| `usuarios` | Usuários do sistema com senhas em bcrypt |
| `clientes` | Clientes (pessoa física ou jurídica, CPF/CNPJ único) |
| `veiculos` | Veículos vinculados a clientes (placa única) |
| `servicos` | Catálogo de serviços da oficina |
| `pecas` | Inventário de peças com controle de estoque mínimo |
| `ordens_servico` | Ordens de serviço com número, status e timestamps |
| `itens_os_servicos` | Serviços incluídos em uma OS (N:N) |
| `itens_os_pecas` | Peças utilizadas em uma OS (N:N) |
| `historicos_status` | Auditoria de todas as transições de status |

### Versões de migration

| Versão | Descrição |
|---|---|
| `000001` | Schema inicial — todas as tabelas |
| `000002` | Seed de papéis e status padrão |
| `000003` | Alterações na tabela `ordens_servico` |
| `000004` | Seed do catálogo (serviços, peças e usuários) |

---

## Testes

```bash
# Gate de cobertura (o mesmo do CI/CD)
./scripts/cobertura.sh

# Rodar todos os testes
go test ./...

# Rodar com verbose
go test -v ./...

# Rodar testes de um pacote específico
go test ./internal/application/usecase/...
```

**Cobertura unitária: 97,9%** no critério do Sonar (mínimo 80%, gate em `./scripts/cobertura.sh`). Entre outros, os testes cobrem a máquina de estados das ordens de serviço (`ordem_servico_usecase_test.go`), validando transições válidas e inválidas com repositórios mockados (sem necessidade de banco de dados).

### Evidências de cobertura e de proteção da `main`

![Cobertura de testes unitários por pacote](docs/evidencias/cobertura.png)

Gerado por `./scripts/cobertura.sh` (o mesmo gate de **80%** que roda no CI e no job de testes do CD, antes do build da imagem). A cobertura publicada no SonarCloud usa o mesmo critério: badge **Coverage** no topo deste README.

A `main` é protegida por ruleset: só entra por **pull request aprovado**, sem push direto, sem force-push e sem deleção. Prova em [`docs/evidencias/push-main-rejeitado.txt`](docs/evidencias/push-main-rejeitado.txt): `push declined due to repository rule violations`.

---

## CI/CD e Segurança

Quatro workflows em `.github/workflows/`, todos com Job Summary detalhado (placar de testes, cobertura, tabelas de vulnerabilidades):

| Workflow | Gatilho | O que faz |
|---|---|---|
| [`ci.yml`](.github/workflows/ci.yml) | PR → `homolog`/`main` | `go vet` + build + testes unitários com `-race` e cobertura + `golangci-lint` + `gitleaks` |
| [`integration.yml`](.github/workflows/integration.yml) | push `main`, PRs | Testes de integração do repositório contra Postgres real (service container) |
| [`security.yml`](.github/workflows/security.yml) | push/PR `main`, manual | Scanners de segurança (tabela abaixo) |
| [`cd.yml`](.github/workflows/cd.yml) | push `main`/tag `v*`, manual | Build + push da imagem no Docker Hub (F2-6.2) e deploy no EKS com rollback automático se o rollout não convergir (F3-1.5) |

### Segredos e variáveis do CI/CD (F2-6.4)

O job de deploy roda sob o [GitHub Environment](https://docs.github.com/actions/deployment/targeting-different-environments/using-environments-for-deployment) `prod` — é o `sub` que a trust policy da role OIDC exige (ver `oficina-infra-k8s/bootstrap`) e onde o *required reviewer* será ligado quando houver permissão de admin no repositório. `homolog` é branch de integração com CI completo, mas não implanta: o ambiente de homologação foi dispensado (corte 10 da Fase 3). Nenhum segredo fica em texto plano no repositório — tudo é referenciado via `secrets`/`vars`:

| Nome | Tipo | Onde é usado |
|---|---|---|
| `DOCKERHUB_USERNAME` | Secret (repo) | Login no Docker Hub (`cd.yml` → build-and-push) |
| `DOCKERHUB_TOKEN` | Secret (repo) | Login no Docker Hub (`cd.yml` → build-and-push) |
| `AWS_ROLE_ARN` | Variable (repo) | Role `gha-oficina-app`, assumida via OIDC — sem chave de longa duração. Só descreve o cluster e lê a SSM; dentro do cluster tem Edit restrito ao namespace `oficina-prod` |
| `AWS_REGION` | Variable (repo) | `aws eks update-kubeconfig`; o nome do cluster vem da SSM (`/oficina/shared/eks/cluster_name`) |
| `NEW_RELIC_APP_ID` | Variable (repo, opcional) | Marcador de deploy no New Relic — o step só roda se estiver definida |
| `NEW_RELIC_API_KEY` | Secret (repo) | idem |
| `SONAR_TOKEN` | Secret (repo, já existente) | `security.yml` → análise SonarCloud |

### Scanners de segurança

| Ferramenta | Tipo | O que verifica |
|---|---|---|
| **govulncheck** | SCA | Vulnerabilidades conhecidas nas dependências Go (vuln.go.dev) |
| **gosec** | SAST | SQL injection, segredos hardcoded, traversal de path, uso inadequado de crypto |
| **Trivy** | Container + SCA | CVEs em pacotes do sistema, segredos expostos, misconfigurações |
| **SonarCloud** | SAST + Quality Gate | Bugs, code smells, cobertura, duplicações, security hotspots |

Os resultados são enviados como SARIF para a aba **Security** do GitHub e disponibilizados como artifacts do workflow.

---

## Estrutura do projeto

```
.
├── cmd/
│   └── api/
│       └── main.go                      # Entry point, wiring de dependências
├── docs/
│   ├── openapi.yaml                     # Contrato OpenAPI 3.1.0 (fonte da verdade)
│   └── postman_collection.json          # Coleção Postman para teste manual
├── internal/
│   ├── application/
│   │   └── usecase/                     # Casos de uso (regras de negócio)
│   │       ├── auth_usecase.go          # Registro e login com JWT
│   │       ├── client_usecase.go
│   │       ├── vehicle_usecase.go
│   │       ├── service_usecase.go
│   │       ├── part_usecase.go
│   │       ├── ordem_servico_usecase.go # OS com máquina de estados
│   │       └── ordem_servico_usecase_test.go
│   ├── domain/
│   │   ├── entity/                      # Entidades de domínio (11 arquivos)
│   │   ├── erros/                       # Tipos de erro de domínio
│   │   └── valueobject/                 # CPF/CNPJ e validação de placa
│   └── infra/
│       ├── database/
│       │   ├── database.go              # Conexão com PostgreSQL
│       │   ├── migrator.go              # Executor de migrations (embed)
│       │   └── migrations/              # Arquivos SQL versionados
│       │       ├── 000001_initial_schema.*
│       │       ├── 000002_seed_default_values.*
│       │       ├── 000003_alter_ordens_servico.*
│       │       └── 000004_seed_catalogo.*
│       ├── http/
│       │   ├── api/
│       │   │   ├── api.gen.go           # Código gerado pelo oapi-codegen (NÃO editar)
│       │   │   ├── generate.go          # Diretiva go:generate
│       │   │   ├── erros.go             # Handler central de erros (mapeamento HTTP)
│       │   │   └── server.go            # Implementação dos handlers
│       │   └── middleware/
│       │       ├── jwt.go               # Middleware JWT Bearer (HS256)
│       │       └── webhook.go           # Validação de assinatura HMAC dos webhooks
│       ├── notification/                # Notificações por e-mail (LogNotifier / SMTPNotifier)
│       └── repository/                  # Acesso ao banco de dados (6 repositórios)
│           └── integration_test.go      # Suíte de integração (build tag `integration`)
├── k8s/                                 # Manifestos Kubernetes (Kustomize)
│   ├── base/                            # namespace, ConfigMap, Deployment, Service, HPA
│   └── overlays/                        # local (kind + Postgres + Mailpit) e aws (EKS + RDS)
├── infra/                               # Terraform
│   ├── modules/kind-cluster/            # Módulo do cluster local
│   └── environments/                    # local (kind + metrics-server) e aws (VPC/EKS/RDS)
├── scripts/
│   ├── seed_dados_completos.sql         # Dados mockados (clientes, veículos, OSs)
│   ├── k8s-local-deploy.sh              # Deploy completo no cluster local
│   ├── test-integration.sh              # Testes de integração com Postgres efêmero
│   └── genhash/
│       └── main.go                      # Utilitário para gerar hashes bcrypt
├── .github/
│   └── workflows/
│       ├── ci.yml                       # Build + testes + lint (PRs para main)
│       ├── integration.yml              # Testes de integração (Postgres real)
│       └── security.yml                 # Segurança (govulncheck, gosec, Trivy, Sonar)
├── docker-compose.yml                   # API + PostgreSQL + Mailpit
├── .env.example                         # Template das variáveis locais
├── Dockerfile                           # Multi-stage build (Go builder + Alpine runtime)
├── go.mod
├── go.sum
└── oapi-codegen.yaml                    # Configuração do gerador de código
```

---

## Gerando código a partir do OpenAPI

Após modificar `docs/openapi.yaml`, regenere o código:

```bash
go generate ./internal/infra/http/api/...
```

O arquivo `oapi-codegen.yaml` controla o que é gerado:

| Opção | Valor | Efeito |
|---|---|---|
| `chi-server` | `true` | Interfaces do router chi |
| `strict-server` | `true` | Request/response tipados (sem parsing manual de JSON) |
| `models` | `true` | Todos os tipos dos schemas OpenAPI |
| `embedded-spec` | `true` | Especificação OpenAPI embutida no binário |

> **Atenção:** nunca edite `api.gen.go` manualmente. Todas as alterações de contrato devem ser feitas no `openapi.yaml` e o código regerado.
