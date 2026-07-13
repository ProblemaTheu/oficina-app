# Tech Challenge — API de Oficina Mecânica

> API REST para gestão completa de uma oficina mecânica: autenticação JWT, clientes, veículos, catálogo de serviços, controle de estoque de peças e ordens de serviço com máquina de estados. Construída em Go com arquitetura API-first — o contrato OpenAPI é a fonte da verdade e o código de roteamento/serialização é gerado automaticamente.
>
> **Fase 2**: a aplicação evoluiu para escalar — webhook de aprovação de orçamento, notificação por e-mail, deploy em Kubernetes com autoescalonamento (HPA), infraestrutura provisionada com Terraform e pipeline de CI/CD.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15.7-336791?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Kustomize%20%7C%20HPA-326CE5?logo=kubernetes&logoColor=white)
![Terraform](https://img.shields.io/badge/Terraform-kind%20%7C%20EKS-7B42BC?logo=terraform&logoColor=white)
![Security](https://img.shields.io/badge/Security-govulncheck%20%7C%20gosec%20%7C%20Trivy-green)

---

## Índice

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

Em produção (AWS, preparado no épico de IaC): o Postgres in-cluster dá lugar ao **RDS**, o Mailpit a um provedor SMTP real, e o cluster passa a ser o **EKS** — tudo já esboçado em `k8s/overlays/aws` e `infra/environments/aws`.

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

> O job de push da imagem (F2-6.2) roda a cada push na `main`. O job de deploy (F2-6.3) fica pronto para um EKS real (autenticação via OIDC) mas só é executado quando a variável de repositório `DEPLOY_ENABLED=true` está setada — um runner hospedado do GitHub não alcança um cluster local (kind/minikube), então na demonstração em vídeo o deploy local é feito manualmente com `scripts/k8s-local-deploy.sh` sobre o cluster provisionado pelo Terraform.

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
git clone https://github.com/problematheu/tech-challenge-1.git
cd tech-challenge-1
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
git clone https://github.com/problematheu/tech-challenge-1.git
cd tech-challenge-1
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

Os manifestos ficam em [`/k8s`](k8s/) (Kustomize: `base` + `overlays/local` + `overlays/aws`). O caminho rápido para o ambiente local completo — cluster kind, metrics-server, Secret a partir do `.env`, API, Postgres e Mailpit:

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
| [docs/architecture-decisions.md](docs/architecture-decisions.md) | ADRs: escolha do PostgreSQL, chi e oapi-codegen |
| [docs/ubiquitous-language.md](docs/ubiquitous-language.md) | Glossário dos termos do domínio (linguagem ubíqua) |
| [docs/ddd-aggregates.md](docs/ddd-aggregates.md) | Agregados DDD, entidades internas e invariantes |
| [docs/bounded-contexts.md](docs/bounded-contexts.md) | Contextos delimitados e mapa de contextos |
| [docs/openapi.yaml](docs/openapi.yaml) | Contrato OpenAPI completo da API |
| [docs/postman_collection.json](docs/postman_collection.json) | Coleção Postman com todos os endpoints |

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
# Rodar todos os testes
go test ./...

# Rodar com verbose
go test -v ./...

# Rodar testes de um pacote específico
go test ./internal/application/usecase/...
```

Os testes cobrem a máquina de estados das ordens de serviço (`ordem_servico_usecase_test.go`), validando transições válidas e inválidas com repositórios mockados (sem necessidade de banco de dados).

---

## CI/CD e Segurança

Quatro workflows em `.github/workflows/`, todos com Job Summary detalhado (placar de testes, cobertura, tabelas de vulnerabilidades):

| Workflow | Gatilho | O que faz |
|---|---|---|
| [`ci.yml`](.github/workflows/ci.yml) | PR → `main` | `go vet` + build + testes unitários com `-race` e cobertura + `golangci-lint` |
| [`integration.yml`](.github/workflows/integration.yml) | push `main`, PRs | Testes de integração do repositório contra Postgres real (service container) |
| [`security.yml`](.github/workflows/security.yml) | push/PR `main`, manual | Scanners de segurança (tabela abaixo) |
| [`cd.yml`](.github/workflows/cd.yml) | push `main`/tag `v*`, manual | Build + push da imagem no Docker Hub (F2-6.2) e deploy no cluster (F2-6.3) |

### Segredos e variáveis do CI/CD (F2-6.4)

O job de deploy roda sob o [GitHub Environment](https://docs.github.com/actions/deployment/targeting-different-environments/using-environments-for-deployment) `production`, que pode exigir *required reviewers* antes de liberar a execução. Nenhum segredo fica em texto plano no repositório — tudo é referenciado via `secrets`/`vars`:

| Nome | Tipo | Onde é usado |
|---|---|---|
| `DOCKERHUB_USERNAME` | Secret (repo) | Login no Docker Hub (`cd.yml` → build-and-push) |
| `DOCKERHUB_TOKEN` | Secret (repo) | Login no Docker Hub (`cd.yml` → build-and-push) |
| `AWS_DEPLOY_ROLE_ARN` | Secret (environment `production`) | `aws-actions/configure-aws-credentials` via OIDC — sem chave de longa duração |
| `EKS_CLUSTER_NAME`, `AWS_REGION` | Variable (environment `production`) | `aws eks update-kubeconfig` |
| `DEPLOY_ENABLED` | Variable (repo) | Liga/desliga o job de deploy — evita falha em ambientes sem cluster acessível |
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
