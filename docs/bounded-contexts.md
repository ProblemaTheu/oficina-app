# Contextos Delimitados (Bounded Contexts) — Tech Challenge

> Define as fronteiras de responsabilidade de cada contexto dentro do sistema.
> Cada contexto possui sua própria linguagem interna, seus casos de uso e seus repositórios.

---

## Visão Geral

```
┌──────────────────┐     ┌──────────────────┐
│   Atendimento    │────►│     Oficina       │
│                  │     │                  │
│ Clientes         │     │ Ordens de Serviço │
│ Veículos         │     │ Diagnóstico       │
│ Abertura de OS   │     │ Execução          │
└──────────────────┘     │ Transições Status │
         │               └──────────────────┘
         │                        │
         ▼                        ▼
┌──────────────────┐     ┌──────────────────┐
│    Segurança     │     │    Estoque        │
│                  │     │                  │
│ Usuários         │     │ Peças / Insumos   │
│ Autenticação     │     │ Controle Estoque  │
│ Papéis           │     │ Ajustes           │
└──────────────────┘     └──────────────────┘
```

---

## Contexto: Atendimento

**Responsabilidade:** Gerenciar o relacionamento com o cliente e a abertura de ordens de serviço.

### Casos de uso
- Cadastrar, buscar, atualizar e remover clientes
- Buscar cliente por CPF/CNPJ (fluxo de abertura de OS)
- Cadastrar, listar e remover veículos de um cliente
- Abrir uma nova Ordem de Serviço

### Entidades principais
| Entidade | Papel neste contexto |
|----------|---------------------|
| `Cliente` | Proprietário do veículo e solicitante do serviço |
| `Veiculo` | Objeto que será reparado |
| `OrdemServico` | Criada neste contexto; ciclo de vida gerenciado pela Oficina |

### Endpoints
- `GET/POST /v1/clients`
- `GET/PUT/DELETE /v1/clients/{id}`
- `GET/POST /v1/vehicles`
- `GET/DELETE /v1/vehicles/{id}`
- `POST /v1/work-orders`

### Regras de negócio
- Um veículo obrigatoriamente pertence a um cliente existente.
- A identificação do cliente para abertura de OS ocorre pelo CPF/CNPJ (`GET /v1/clients?documento=`).
- A OS nasce no status `recebida`.

---

## Contexto: Oficina

**Responsabilidade:** Gerenciar a execução técnica das ordens de serviço, desde o diagnóstico até a entrega.

### Casos de uso
- Listar e buscar ordens de serviço
- Avançar status da OS (máquina de estados)
- Registrar diagnóstico
- Aprovar ou reprovar orçamento
- Finalizar OS e registrar entrega
- Consultar status público de uma OS (sem autenticação)
- Gerar relatório de tempo médio de atendimento

### Entidades principais
| Entidade | Papel neste contexto |
|----------|---------------------|
| `OrdemServico` | Raiz do agregado; controla o ciclo de vida |
| `ItemOsServico` | Serviços cotados na OS |
| `ItemOsPeca` | Peças cotadas na OS |
| `HistoricoStatus` | Auditoria das transições |

### Endpoints
- `GET /v1/work-orders`
- `GET /v1/work-orders/{id}`
- `PATCH /v1/work-orders/{id}/status`
- `POST /v1/work-orders/{id}/approve`
- `POST /v1/work-orders/{id}/reject`
- `GET /v1/work-orders/{id}/status` *(público)*
- `GET /v1/reports/average-time`

### Regras de negócio
- Somente transições definidas em `validTransitions` são permitidas.
- A dedução de estoque ocorre automaticamente na transição para `em_execucao`.
- O diagnóstico é obrigatório na transição para `aguardando_aprovacao`.

---

## Contexto: Estoque

**Responsabilidade:** Controlar o inventário de peças e insumos utilizados na oficina.

### Casos de uso
- Cadastrar, listar, atualizar e remover peças
- Buscar peça por ID
- Ajustar estoque manualmente (entrada, saída, ajuste)
- Dedução automática de estoque ao iniciar execução de uma OS

### Entidades principais
| Entidade | Papel neste contexto |
|----------|---------------------|
| `Peca` | Item de estoque com código, preço e quantidades |

### Endpoints
- `GET/POST /v1/parts`
- `GET/PUT/DELETE /v1/parts/{id}`
- `POST /v1/parts/{id}/stock`

### Regras de negócio
- `EstoqueAtual` não pode ficar negativo em ajustes manuais de saída.
- A dedução automática (via OS) é coordenada pelo contexto de Oficina usando o repositório de Peças.
- `EstoqueMinimo` é apenas informativo; não bloqueia operações automaticamente.

---

## Contexto: Segurança

**Responsabilidade:** Autenticação de usuários e controle de acesso baseado em papéis (RBAC).

### Casos de uso
- Registrar novo usuário com papel específico
- Autenticar usuário e emitir JWT
- Validar token JWT nas rotas protegidas

### Entidades principais
| Entidade | Papel neste contexto |
|----------|---------------------|
| `Usuario` | Conta de acesso com credenciais e papel |
| `PapelUsuario` | Define o nível de permissão: `mecanico`, `atendente`, `administrador` |

### Endpoints
- `POST /v1/auth/register`
- `POST /v1/auth/login`

### Regras de negócio
- Senhas são armazenadas exclusivamente como hash bcrypt (custo 10).
- O JWT é assinado com `HS256` e expira em 8 horas.
- Todas as rotas exceto `/v1/auth/*` e `GET /v1/work-orders/{id}/status` requerem autenticação.
- O middleware JWT valida a assinatura e a expiração antes de qualquer handler.

---

## Mapa de Contextos (Context Map)

| Contexto | Consome de | Expõe para |
|----------|-----------|------------|
| Atendimento | — | Oficina (`ClienteID`, `VeiculoID`) |
| Oficina | Atendimento, Estoque | — |
| Estoque | — | Oficina (dedução automática) |
| Segurança | — | Todos (middleware JWT) |
