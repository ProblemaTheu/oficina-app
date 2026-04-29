# Tech Challenge — API de Oficina Mecânica

API REST para gestão de uma oficina mecânica, cobrindo clientes, veículos, serviços, peças e ordens de serviço. Construída em Go com arquitetura API-first: o contrato OpenAPI é a fonte da verdade, e o código de roteamento/serialização é gerado automaticamente.

## Índice

- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
- [Pré-requisitos](#pré-requisitos)
- [Executando com Docker](#executando-com-docker)
- [Executando sem Docker](#executando-sem-docker)
- [Variáveis de ambiente](#variáveis-de-ambiente)
- [Endpoints](#endpoints)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Gerando código a partir do OpenAPI](#gerando-código-a-partir-do-openapi)

---

## Tecnologias

| Tecnologia | Versão | Uso |
|---|---|---|
| Go | 1.26 | Linguagem principal |
| PostgreSQL | 15.7 | Banco de dados relacional |
| chi | v5.2.5 | Router HTTP |
| oapi-codegen | v2.6.0 | Geração de código a partir do OpenAPI |
| golang-migrate | v4.19.1 | Migrations de banco de dados |
| lib/pq | v1.12.3 | Driver PostgreSQL |
| Docker / Compose | — | Containerização |

---

## Arquitetura

O projeto segue **Clean Architecture** com geração de código API-first:

```
docs/openapi.yaml          ← fonte da verdade do contrato
        │
        ▼
internal/infra/http/api/
  api.gen.go               ← gerado pelo oapi-codegen (NÃO editar)
  server.go                ← implementação dos handlers (editar aqui)
  generate.go              ← diretiva go:generate

internal/application/usecase/   ← regras de negócio
internal/domain/entity/         ← entidades de domínio
internal/domain/valueobject/    ← validações (CPF/CNPJ)
internal/domain/erros/          ← tipos de erro de domínio
internal/infra/repository/      ← acesso ao banco de dados
internal/infra/database/        ← conexão e migrations
```

Fluxo de uma requisição:
```
HTTP Request → chi router → StrictHandler (gerado) → server.go → UseCase → Repository → PostgreSQL
```

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
- Construir a imagem da aplicação Go
- Subir o PostgreSQL com healthcheck
- Aguardar o banco estar pronto antes de iniciar a API
- Executar as migrations automaticamente na inicialização

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

### Parar os serviços

```bash
docker compose down
```

Para remover também o volume de dados:

```bash
docker compose down -v
```

### Rebuild após mudanças no código

```bash
docker compose up -d --build app
```

---

## Executando sem Docker

### 1. Configure o PostgreSQL

Crie o banco de dados:

```sql
CREATE DATABASE tech_challenge_db;
```

### 2. Configure as variáveis de ambiente

Crie um arquivo `.env` na raiz do projeto ou exporte as variáveis no shell:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=tech_challenge_db
```

### 3. Instale as dependências

```bash
go mod download
```

### 4. Execute a aplicação

As migrations são executadas automaticamente na inicialização:

```bash
go run ./cmd/api
```

A API estará disponível em `http://localhost:8080`.

### 5. (Opcional) Rodar migrations manualmente

Caso prefira rodar as migrations com a CLI do golang-migrate:

```bash
migrate -path internal/infra/database/migrations \
        -database "postgres://postgres:postgres@localhost:5432/tech_challenge_db?sslmode=disable" \
        up
```

---

## Variáveis de ambiente

| Variável | Padrão | Descrição |
|---|---|---|
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_USER` | `postgres` | Usuário do banco |
| `DB_PASSWORD` | `postgres` | Senha do banco |
| `DB_NAME` | `tech_challenge_db` | Nome do banco de dados |

---

## Endpoints

A API está disponível sob o prefixo `/v1`.

### Health

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | Status geral (app + dependências) |
| GET | `/health/live` | Liveness — processo está de pé |
| GET | `/health/ready` | Readiness — banco de dados acessível |

### Auth

| Método | Rota | Descrição |
|---|---|---|
| POST | `/v1/auth/login` | Autenticação e geração de JWT |

### Clientes

| Método | Rota | Descrição |
|---|---|---|
| GET | `/v1/clients` | Listar clientes (paginado) |
| POST | `/v1/clients` | Cadastrar cliente |
| GET | `/v1/clients/{id}` | Buscar cliente por ID |
| PUT | `/v1/clients/{id}` | Atualizar cliente |
| DELETE | `/v1/clients/{id}` | Remover cliente |
| GET | `/v1/clients/{id}/vehicles` | Listar veículos do cliente |

### Veículos

| Método | Rota | Descrição |
|---|---|---|
| GET | `/v1/vehicles` | Listar veículos (paginado) |
| POST | `/v1/vehicles` | Cadastrar veículo |
| GET | `/v1/vehicles/{id}` | Buscar veículo por ID |
| PUT | `/v1/vehicles/{id}` | Atualizar veículo |
| DELETE | `/v1/vehicles/{id}` | Remover veículo |

### Serviços

| Método | Rota | Descrição |
|---|---|---|
| GET | `/v1/services` | Listar serviços (paginado) |
| POST | `/v1/services` | Cadastrar serviço |
| GET | `/v1/services/{id}` | Buscar serviço por ID |
| PUT | `/v1/services/{id}` | Atualizar serviço |
| DELETE | `/v1/services/{id}` | Remover serviço |

### Peças

| Método | Rota | Descrição |
|---|---|---|
| GET | `/v1/parts` | Listar peças (paginado) |
| POST | `/v1/parts` | Cadastrar peça |
| GET | `/v1/parts/{id}` | Buscar peça por ID |
| PUT | `/v1/parts/{id}` | Atualizar peça |
| DELETE | `/v1/parts/{id}` | Remover peça |
| PATCH | `/v1/parts/{id}/stock` | Ajustar estoque (entrada/saída/ajuste) |

O contrato completo com exemplos de request/response está em [`docs/openapi.yaml`](docs/openapi.yaml).

---

## Estrutura do projeto

```
.
├── cmd/
│   └── api/
│       └── main.go                  # Entry point, wiring de dependências
├── docs/
│   └── openapi.yaml                 # Contrato OpenAPI 3.1.0 (fonte da verdade)
├── internal/
│   ├── application/
│   │   └── usecase/                 # Casos de uso (regras de negócio)
│   ├── domain/
│   │   ├── entity/                  # Entidades de domínio
│   │   ├── erros/                   # Tipos de erro de domínio
│   │   └── valueobject/             # Value objects (CPF/CNPJ)
│   └── infra/
│       ├── database/
│       │   ├── database.go          # Conexão com PostgreSQL
│       │   ├── migrator.go          # Executor de migrations
│       │   └── migrations/          # Arquivos SQL de migration
│       ├── http/
│       │   └── api/
│       │       ├── api.gen.go       # Código gerado pelo oapi-codegen
│       │       ├── generate.go      # Diretiva go:generate
│       │       └── server.go        # Implementação dos handlers
│       └── repository/              # Acesso ao banco de dados
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── go.sum
```

---

## Gerando código a partir do OpenAPI

Após modificar `docs/openapi.yaml`, regenere o código:

```bash
go generate ./internal/infra/http/api/...
```

> **Atenção:** nunca edite `api.gen.go` manualmente. Todas as alterações de contrato devem ser feitas no `openapi.yaml`.

O arquivo `oapi-codegen.yaml` na raiz do projeto contém a configuração do gerador (pacote de saída, tipos habilitados etc.).
