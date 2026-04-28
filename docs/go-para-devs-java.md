# Go para Desenvolvedores Java/Spring

## Sumário
1. [Estrutura de Projeto](#1-estrutura-de-projeto)
2. [Módulos e Dependências](#2-módulos-e-dependências)
3. [Tipos, Structs e Interfaces](#3-tipos-structs-e-interfaces)
4. [Tratamento de Erros](#4-tratamento-de-erros)
5. [Concorrência](#5-concorrência)
6. [Injeção de Dependência](#6-injeção-de-dependência)
7. [HTTP e Rotas](#7-http-e-rotas)
8. [Banco de Dados](#8-banco-de-dados)
9. [Testes](#9-testes)
10. [Boas Práticas Gerais](#10-boas-práticas-gerais)

---

## 1. Estrutura de Projeto

### Java/Spring
```
src/
  main/
    java/com/empresa/projeto/
      controller/
      service/
      repository/
      model/
  test/
    java/...
```

### Go (Standard Layout)
```
cmd/
  api/
    main.go          ← entrypoint da aplicação
internal/            ← código privado (não pode ser importado por outros módulos)
  domain/
    entity/
    valueobject/
  infra/
    database/
    http/
      handler/
pkg/                 ← código público (pode ser importado por outros módulos)
```

**Boas práticas:**
- `cmd/` → um subdiretório por executável
- `internal/` → código privado ao projeto (o Go impede importação externa)
- `pkg/` → bibliotecas reutilizáveis por outros módulos
- Evite criar pastas `utils/` ou `helpers/` genéricas — prefira nomear pelo domínio

---

## 2. Módulos e Dependências

### Java/Spring
```xml
<!-- pom.xml -->
<dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-web</artifactId>
    <version>3.2.0</version>
</dependency>
```

### Go
```go
// go.mod
module github.com/empresa/projeto

go 1.22

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/golang-migrate/migrate/v4 v4.17.0
)
```

**Comandos equivalentes:**

| Java (Maven)          | Go                          |
|-----------------------|-----------------------------|
| `mvn install`         | `go mod tidy`               |
| Adicionar dependência | `go get pacote@versão`      |
| `mvn clean package`   | `go build ./...`            |
| `mvn test`            | `go test ./...`             |
| `mvn spring-boot:run` | `go run ./cmd/api/`         |

---

## 3. Tipos, Structs e Interfaces

### Java/Spring
```java
// Classe com anotações
@Entity
@Table(name = "clientes")
public class Cliente {
    @Id
    private UUID id;
    private String nome;
    
    // getters e setters obrigatórios
}

// Interface
public interface ClienteService {
    Cliente buscarPorId(UUID id);
}

// Implementação
@Service
public class ClienteServiceImpl implements ClienteService {
    @Override
    public Cliente buscarPorId(UUID id) { ... }
}
```

### Go
```go
// Struct — sem anotações, sem getters/setters obrigatórios
type Cliente struct {
    ID    uuid.UUID
    Nome  string
    Email string
}

// Interface — simples, sem implements explícito
type ClienteService interface {
    BuscarPorID(id uuid.UUID) (*Cliente, error)
}

// Implementação — satisfaz a interface automaticamente (duck typing)
type clienteService struct {
    repo ClienteRepository
}

func (s *clienteService) BuscarPorID(id uuid.UUID) (*Cliente, error) {
    return s.repo.FindByID(id)
}
```

**Diferenças importantes:**
- Go não tem herança, apenas composição
- Interfaces são satisfeitas implicitamente (sem `implements`)
- Não existem getters/setters — acesse os campos diretamente
- Campos exportados começam com letra **maiúscula** (`Nome`), privados com **minúscula** (`nome`)

---

## 4. Tratamento de Erros

Esta é a maior diferença cultural entre Java e Go.

### Java/Spring
```java
// Exceções são lançadas e capturadas
try {
    Cliente cliente = clienteService.buscarPorId(id);
} catch (ClienteNaoEncontradoException e) {
    throw new ResponseStatusException(HttpStatus.NOT_FOUND, e.getMessage());
} catch (Exception e) {
    throw new ResponseStatusException(HttpStatus.INTERNAL_SERVER_ERROR);
}
```

### Go
```go
// Erros são valores retornados, não lançados
cliente, err := clienteService.BuscarPorID(id)
if err != nil {
    // trate o erro aqui
    return nil, fmt.Errorf("buscar cliente: %w", err)
}
```

**Boas práticas em Go:**
```go
// ✅ Correto: erros descritivos com contexto usando %w (wrapping)
return fmt.Errorf("clienteService.BuscarPorID: %w", err)

// ✅ Correto: erros sentinela para comparação
var ErrClienteNaoEncontrado = errors.New("cliente não encontrado")

// ✅ Correto: checar tipo de erro
if errors.Is(err, ErrClienteNaoEncontrado) {
    // retornar 404
}

// ❌ Evite: ignorar erros
resultado, _ := funcaoQuePodeErrar()
```

---

## 5. Concorrência

### Java/Spring
```java
// Threads e CompletableFuture
CompletableFuture<Cliente> future = CompletableFuture
    .supplyAsync(() -> clienteService.buscarPorId(id));
```

### Go
```go
// Goroutines — leves e baratas (não são threads OS)
go func() {
    resultado := buscarDados()
    canal <- resultado
}()

// Channels — comunicação segura entre goroutines
canal := make(chan Cliente, 1)
go func() {
    canal <- clienteService.BuscarPorID(id)
}()
cliente := <-canal
```

**Boas práticas:**
- Prefira channels para comunicar dados entre goroutines
- Use `sync.Mutex` apenas para proteger estado compartilhado
- Use `context.Context` para cancelamento e timeout
- Nunca compartilhe memória — comunique-se via channels

```go
// ✅ Context para timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

cliente, err := clienteService.BuscarPorID(ctx, id)
```

---

## 6. Injeção de Dependência

### Java/Spring
```java
// Spring IoC Container gerencia tudo automaticamente
@RestController
@RequiredArgsConstructor
public class ClienteController {
    
    @Autowired
    private final ClienteService clienteService;
}
```

### Go
```go
// Go não tem container de DI nativo — use construtores explícitos
// Simples, direto, sem "mágica"

type ClienteHandler struct {
    service ClienteService
}

func NewClienteHandler(service ClienteService) *ClienteHandler {
    return &ClienteHandler{service: service}
}

// No main.go — montagem manual (wiring)
func main() {
    repo := database.NewClienteRepository(db)
    service := domain.NewClienteService(repo)
    handler := http.NewClienteHandler(service)
    
    router.GET("/clientes/:id", handler.BuscarPorID)
}
```

**Boas práticas:**
- Monte as dependências no `main.go` (composition root)
- Use interfaces como parâmetros dos construtores (facilita testes)
- Para projetos grandes, considere o pacote `google/wire` (geração de código) ou `uber/fx`

---

## 7. HTTP e Rotas

### Java/Spring
```java
@RestController
@RequestMapping("/clientes")
public class ClienteController {

    @GetMapping("/{id}")
    public ResponseEntity<ClienteDTO> buscar(@PathVariable UUID id) {
        return ResponseEntity.ok(clienteService.buscarPorId(id));
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public ClienteDTO criar(@RequestBody @Valid CriarClienteRequest request) {
        return clienteService.criar(request);
    }
}
```

### Go (com Gin)
```go
type ClienteHandler struct {
    service ClienteService
}

func (h *ClienteHandler) BuscarPorID(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
        return
    }

    cliente, err := h.service.BuscarPorID(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, domain.ErrClienteNaoEncontrado) {
            c.JSON(http.StatusNotFound, gin.H{"erro": "cliente não encontrado"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro interno"})
        return
    }

    c.JSON(http.StatusOK, cliente)
}

// Registro das rotas
func (h *ClienteHandler) RegisterRoutes(r *gin.Engine) {
    clientes := r.Group("/clientes")
    clientes.GET("/:id", h.BuscarPorID)
    clientes.POST("", h.Criar)
}
```

**Frameworks HTTP populares em Go:**

| Biblioteca      | Equivalente Spring        | Uso                          |
|-----------------|---------------------------|------------------------------|
| `net/http`      | Servlet puro              | Stdlib, sem dependências     |
| `gin-gonic/gin` | Spring MVC                | Mais popular, rápido         |
| `go-chi/chi`    | Spring MVC (leve)         | Minimalista, bom middleware  |
| `labstack/echo` | Spring MVC                | Alternativa ao Gin           |

---

## 8. Banco de Dados

### Java/Spring
```java
// JPA/Hibernate — ORM completo
@Repository
public interface ClienteRepository extends JpaRepository<Cliente, UUID> {
    Optional<Cliente> findByEmail(String email);
}
```

### Go
```go
// Opções populares:
// 1. database/sql (stdlib) — SQL puro
// 2. sqlx — extensão da stdlib
// 3. GORM — ORM (mais próximo do Hibernate)
// 4. sqlc — geração de código a partir de SQL

// Exemplo com database/sql
type ClienteRepository struct {
    db *sql.DB
}

func (r *ClienteRepository) FindByID(ctx context.Context, id uuid.UUID) (*Cliente, error) {
    query := `SELECT id, nome, email FROM clientes WHERE id = $1`
    
    var c Cliente
    err := r.db.QueryRowContext(ctx, query, id).Scan(&c.ID, &c.Nome, &c.Email)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrClienteNaoEncontrado
        }
        return nil, fmt.Errorf("FindByID: %w", err)
    }
    return &c, nil
}
```

**Boas práticas:**
- Prefira SQL explícito ao invés de ORM para ter controle total das queries
- Sempre use `context.Context` nas queries para suportar cancelamento/timeout
- Use `$1, $2` (PostgreSQL) ou `?` (MySQL) como placeholders — nunca interpolação de strings (SQL injection)
- Use `database/sql` ou `sqlc` para projetos novos

---

## 9. Testes

### Java/Spring
```java
@SpringBootTest
class ClienteServiceTest {
    
    @MockBean
    private ClienteRepository clienteRepository;
    
    @Autowired
    private ClienteService clienteService;
    
    @Test
    void deveBuscarClientePorId() {
        var cliente = new Cliente(UUID.randomUUID(), "João");
        when(clienteRepository.findById(any())).thenReturn(Optional.of(cliente));
        
        var resultado = clienteService.buscarPorId(cliente.getId());
        
        assertNotNull(resultado);
        assertEquals("João", resultado.getNome());
    }
}
```

### Go
```go
// Go tem testing nativo — sem framework obrigatório
func TestBuscarClientePorID(t *testing.T) {
    // Arrange
    repo := &mockClienteRepository{
        cliente: &Cliente{ID: uuid.New(), Nome: "João"},
    }
    service := NewClienteService(repo)
    
    // Act
    resultado, err := service.BuscarPorID(context.Background(), repo.cliente.ID)
    
    // Assert
    if err != nil {
        t.Fatalf("erro inesperado: %v", err)
    }
    if resultado.Nome != "João" {
        t.Errorf("esperado 'João', obtido '%s'", resultado.Nome)
    }
}

// Mock manual — Go não tem Mockito nativo
type mockClienteRepository struct {
    cliente *Cliente
}

func (m *mockClienteRepository) FindByID(_ context.Context, _ uuid.UUID) (*Cliente, error) {
    return m.cliente, nil
}
```

**Bibliotecas de teste populares:**

| Java               | Go                   | Propósito              |
|--------------------|----------------------|------------------------|
| JUnit              | `testing` (stdlib)   | Testes unitários       |
| Mockito            | `testify/mock`       | Mocks                  |
| AssertJ            | `testify/assert`     | Assertions fluentes    |
| Hamcrest           | `testify/require`    | Assertions que param   |

**Comando para executar:**
```bash
go test ./...                  # todos os testes
go test ./... -v               # verbose
go test ./... -cover           # com cobertura
go test -run TestBuscar ./...  # filtrar por nome
```

---

## 10. Boas Práticas Gerais

### Nomenclatura
| Java                     | Go                        |
|--------------------------|---------------------------|
| `getClienteById()`       | `ClienteByID()` ou `FindByID()` |
| `ClienteServiceImpl`     | `clienteService` (struct privada) |
| `IClienteRepository`     | `ClienteRepository` (interface sem prefixo I) |
| `CONSTANTE_EM_UPPER`     | `ConstanteEmPascalCase`   |
| `clienteId`              | `clienteID` (siglas em maiúsculo) |

### Pacotes
```go
// ✅ Nomes de pacotes: curtos, minúsculos, sem underscore
package handler
package repository
package domain

// ❌ Evite
package clienteHandler
package cliente_handler
package handlers // plural
```

### Evite "Java em Go"
```go
// ❌ Não faça isso — muito Java
type IClienteService interface { ... }      // prefixo I
type ClienteServiceImpl struct { ... }     // sufixo Impl
type ClienteDTO struct { ... }             // DTO desnecessário

// ✅ Idiomático em Go
type ClienteService interface { ... }
type clienteService struct { ... }         // privado, sem Impl
type Cliente struct { ... }               // struct direto
```

### Inicialização e zero values
```go
// Em Go, todo tipo tem um "zero value" — não precisa inicializar
var nome string    // ""
var count int      // 0
var ativo bool     // false
var cliente *Cliente // nil

// Structs são inicializadas com zero values automaticamente
var c Cliente // c.Nome == "", c.ID == uuid.Nil
```

### Defer para limpeza de recursos
```go
// Equivalente ao try-with-resources do Java
func buscarDados(db *sql.DB) error {
    rows, err := db.Query("SELECT id FROM clientes")
    if err != nil {
        return err
    }
    defer rows.Close() // executado ao final da função, sempre
    
    for rows.Next() { ... }
    return rows.Err()
}
```

---

## Resumo Rápido

| Conceito              | Java/Spring                  | Go                               |
|-----------------------|------------------------------|----------------------------------|
| Gerenciador de deps   | Maven/Gradle + pom.xml       | `go mod` + go.mod                |
| Entrypoint            | `@SpringBootApplication`     | `func main()` em `cmd/`          |
| DI/IoC                | Spring Container (`@Autowired`) | Manual via construtores        |
| ORM                   | JPA/Hibernate                | `database/sql`, `sqlc`, GORM     |
| Tratamento de erro    | `try/catch/throw`            | `if err != nil { return err }`   |
| Concorrência          | Threads, CompletableFuture   | Goroutines + Channels            |
| Testes                | JUnit + Mockito              | `testing` stdlib + testify       |
| HTTP                  | Spring MVC                   | Gin, Chi, Echo, `net/http`       |
| Annotations           | `@Component`, `@Service`...  | Não existem — código explícito   |
| Build                 | `mvn package` → .jar         | `go build` → binário estático    |
