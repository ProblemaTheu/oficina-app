# Tech Challenge — API de Oficina Mecânica

> API REST para gestão completa de uma oficina mecânica: autenticação JWT, clientes, veículos, catálogo de serviços, controle de estoque de peças e ordens de serviço com máquina de estados. Construída em Go com arquitetura API-first — o contrato OpenAPI é a fonte da verdade e o código de roteamento/serialização é gerado automaticamente.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15.7-336791?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)
![Security](https://img.shields.io/badge/Security-govulncheck%20%7C%20gosec%20%7C%20Trivy-green)

---

## Índice

- [Quickstart](#quickstart)
- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
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

---

## Quickstart

```bash
git clone https://github.com/problematheu/tech-challenge-1.git
cd tech-challenge-1
docker compose up -d --build
```

Aguarde o healthcheck e teste:

```bash
# Verificar se a API está pronta
curl http://localhost:8080/health/ready

# Obter token JWT (usuário admin padrão)
curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@oficina.com","password":"Admin@123"}' | jq .token

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

### 2. Suba todos os serviços

```bash
docker compose up -d --build
```

Isso irá:
- Construir a imagem da aplicação Go (multi-stage, Alpine)
- Subir o PostgreSQL 15.7 com healthcheck (exposto na porta **5433** do host)
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
  -d '{"email": "admin@oficina.com", "password": "Admin@123"}'
```

Resposta:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
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

| Variável | Padrão | Descrição |
|---|---|---|
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_USER` | `postgres` | Usuário do banco |
| `DB_PASSWORD` | `admin` | Senha do banco |
| `DB_NAME` | `tech_challenge_db` | Nome do banco de dados |
| `JWT_SECRET` | *(inseguro — altere em produção)* | Chave de assinatura dos tokens JWT |

> **Atenção:** em produção, defina `JWT_SECRET` com um valor longo e aleatório (mínimo 32 caracteres). O valor padrão é deliberadamente inseguro.

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
| GET | `/v1/work-orders` | `page`, `limit`, `status`, `cliente_id`, `veiculo_id` | Listar OSs (paginado) |
| POST | `/v1/work-orders` | — | Abrir nova OS |
| GET | `/v1/work-orders/{id}` | — | Detalhes completos (itens + histórico) |
| GET | `/v1/work-orders/{id}/status` | — | Consultar status (rota pública) |
| PATCH | `/v1/work-orders/{id}/status` | — | Avançar status da OS |
| POST | `/v1/work-orders/{id}/approve` | — | Aprovar orçamento |
| POST | `/v1/work-orders/{id}/reject` | — | Rejeitar orçamento |

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
   ┌───────────────────┐
   │aguardando_aprovacao│
   └────────┬──────────┘
      ┌─────┴──────┐
      │            │
   /approve     /reject
      │            │
      ▼            ▼
 ┌──────────┐  ┌──────────┐
 │em_execucao│  │finalizada│
 └─────┬────┘  │(rejeitada)│
       │        └──────────┘
       │ PATCH /status
       ▼
  ┌──────────┐
  │finalizada│
  │(concluída)│
  └─────┬────┘
        │ PATCH /status
        ▼
   ┌─────────┐
   │entregue │
   └─────────┘
```

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

O repositório inclui um pipeline de segurança em `.github/workflows/security.yml` com três scanners complementares.

### Quando é executado

- Push nas branches `main`, `feature/*` e `fix/*`
- Pull requests para `main`
- Semanalmente (segundas-feiras às 08:00 UTC)
- Manualmente via `workflow_dispatch`

### Scanners

| Ferramenta | Tipo | O que verifica |
|---|---|---|
| **govulncheck** | SCA | Vulnerabilidades conhecidas nas dependências Go (vuln.go.dev) |
| **gosec** | SAST | SQL injection, segredos hardcoded, traversal de path, uso inadequado de crypto |
| **Trivy** | Container + SCA | CVEs em pacotes do sistema, segredos expostos, misconfigurações |

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
│       │   │   └── server.go            # Implementação dos handlers
│       │   └── middleware/
│       │       └── jwt.go               # Middleware JWT Bearer (HS256)
│       └── repository/                  # Acesso ao banco de dados (6 repositórios)
├── scripts/
│   ├── seed_dados_completos.sql         # Dados mockados (clientes, veículos, OSs)
│   └── genhash/
│       └── main.go                      # Utilitário para gerar hashes bcrypt
├── .github/
│   └── workflows/
│       └── security.yml                 # Pipeline de segurança (govulncheck, gosec, Trivy)
├── docker-compose.yml
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
