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
