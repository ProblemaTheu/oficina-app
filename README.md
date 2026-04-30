# Tech Challenge — API de Oficina Mecânica

API REST para gestão de uma oficina mecânica, cobrindo autenticação JWT, clientes, veículos, serviços, peças e ordens de serviço com máquina de estados. Construída em Go com arquitetura API-first: o contrato OpenAPI é a fonte da verdade, e o código de roteamento/serialização é gerado automaticamente.

## Índice

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
- [Testes](#testes)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Gerando código a partir do OpenAPI](#gerando-código-a-partir-do-openapi)

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
        ▼
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

Fluxo de uma requisição autenticada:

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

### Comandos úteis

```bash
# Parar os serviços (mantém os dados)
docker compose down

# Parar e remover todos os dados do banco
docker compose down -v

# Rebuild apenas da aplicação após mudanças no código
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

Exporte as variáveis no shell ou crie um arquivo `.env`:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=tech_challenge_db
export JWT_SECRET=sua-chave-secreta-aqui
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

```bash
migrate -path internal/infra/database/migrations \
        -database "postgres://postgres:postgres@localhost:5432/tech_challenge_db?sslmode=disable" \
        up
```

---

## Dados de seed

### Catálogo (migration 000004 — automático)

Aplicado automaticamente na inicialização. Inclui 3 usuários de sistema, 15 serviços e 20 peças.

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

---

## Autenticação

A API usa **JWT Bearer tokens** com algoritmo HS256.

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

### Credenciais padrão (seed do catálogo)

| E-mail | Senha | Papel |
|---|---|---|
| `admin@oficina.com` | `Admin@123` | Administrador |
| `joao@oficina.com` | `Mecanico@123` | Mecânico |
| `ana@oficina.com` | `Atende@123` | Atendente |

### Rotas públicas (sem autenticação)

| Rota | Motivo |
|---|---|
| `POST /v1/auth/login` | Geração de token |
| `POST /v1/auth/register` | Cadastro de usuário |
| `GET /v1/work-orders/{id}/status` | Consulta de status por clientes externos |

Todas as demais rotas exigem `Authorization: Bearer <token>`.

> **Expiração:** tokens têm validade de **8 horas**.

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

> **Atenção:** em produção, defina `JWT_SECRET` com um valor longo e aleatório. O valor padrão é deliberadamente inseguro.

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
| PATCH | `/v1/parts/{id}/stock` | — | Ajustar estoque (`entrada` / `saida` / `ajuste`) |

### Ordens de Serviço

| Método | Rota | Filtros disponíveis | Descrição |
|---|---|---|---|
| GET | `/v1/work-orders` | `page`, `limit`, `status`, `cliente_id`, `veiculo_id` | Listar OSs (paginado) |
| POST | `/v1/work-orders` | — | Abrir nova OS |
| GET | `/v1/work-orders/{id}` | — | Detalhes completos da OS (itens + histórico) |
| GET 🔓 | `/v1/work-orders/{id}/status` | — | Consultar status (rota pública para clientes) |
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

Toda OS percorre um fluxo de status com transições controladas:

```
                    ┌─────────────┐
        abertura    │   recebida  │
        ──────────► │             │
                    └──────┬──────┘
                           │ avançar status
                    ┌──────▼──────┐
                    │em_diagnos-  │
                    │   tico      │
                    └──────┬──────┘
                           │ avançar status
                    ┌──────▼──────┐
                    │ aguardando  │
                    │ aprovação   │
                    └──────┬──────┘
                    ┌──────┴──────┐
              aprovar│             │rejeitar
                    ▼             ▼
             ┌──────────┐  ┌──────────┐
             │em_execu- │  │finalizada│
             │   cão    │  │(rejeitada)│
             └────┬─────┘  └──────────┘
                  │ avançar status
             ┌────▼─────┐
             │finalizada│
             │(concluída)│
             └────┬─────┘
                  │ avançar status
             ┌────▼─────┐
             │ entregue │
             └──────────┘
```

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
│   │   ├── entity/                      # Entidades de domínio
│   │   ├── erros/                       # Tipos de erro de domínio
│   │   └── valueobject/                 # Value objects (CPF/CNPJ, placa)
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
│       └── repository/                  # Acesso ao banco de dados
│           ├── client_repository.go
│           ├── vehicle_repository.go
│           ├── servico_repository.go
│           ├── peca_repository.go
│           ├── usuario_repository.go
│           └── ordem_servico_repository.go
├── scripts/
│   ├── seed_dados_completos.sql         # Dados mockados (clientes, veículos, OSs)
│   └── genhash/
│       └── main.go                     # Utilitário para gerar hashes bcrypt
├── docker-compose.yml
├── Dockerfile
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

> **Atenção:** nunca edite `api.gen.go` manualmente. Todas as alterações de contrato devem ser feitas no `openapi.yaml`.

O arquivo `oapi-codegen.yaml` na raiz do projeto contém a configuração do gerador (pacote de saída, tipos habilitados etc.).
