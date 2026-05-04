# Agregados DDD — Tech Challenge

> Documento os agregados do domínio, suas raízes, entidades internas e invariantes.
> Cada agregado é uma fronteira de consistência: operações que cruzam agregados são coordenadas pela camada de use case.

---

## Agregado: OrdemServico

**Raiz:** `OrdemServico`

```
OrdemServico (raiz)
├── ItemOsServico[]     — serviços incluídos no orçamento
├── ItemOsPeca[]        — peças incluídas no orçamento
└── HistoricoStatus[]   — linha do tempo de transições de status
```

### Entidades internas

| Entidade | Responsabilidade |
|----------|-----------------|
| `OrdemServico` | Raiz do agregado. Controla o ciclo de vida, valida transições de status e armazena diagnóstico, valor total e datas-chave. |
| `ItemOsServico` | Registro de um serviço cotado na OS: referência ao `Servico`, quantidade e preço unitário no momento da abertura. |
| `ItemOsPeca` | Registro de uma peça cotada na OS: referência à `Peca`, quantidade e preço unitário no momento da abertura. |
| `HistoricoStatus` | Registro imutável de cada transição. Guarda status anterior, novo status, timestamp e observação. |

### Invariantes

- O status só pode avançar para transições definidas em `validTransitions` (`ordem_servico.go`).
- `AprovadoEm`, `ReprovadoEm`, `IniciadoEm`, `FinalizadoEm` e `EntregueEm` são preenchidos exatamente uma vez, na transição correspondente.
- A dedução de estoque das peças ocorre atomicamente na transição para `em_execucao`.
- O `ValorTotal` é calculado como `Σ (ItemOsServico.Subtotal) + Σ (ItemOsPeca.Subtotal)`.

---

## Agregado: Cliente

**Raiz:** `Cliente`

```
Cliente (raiz)
└── Veiculo[]   — veículos pertencentes ao cliente
```

### Entidades internas

| Entidade | Responsabilidade |
|----------|-----------------|
| `Cliente` | Raiz do agregado. Identifica o proprietário do veículo por CPF ou CNPJ (value object `Document`). |
| `Veiculo` | Veículo vinculado ao cliente. A placa é validada pelo value object `Plate` (formato antigo ou Mercosul). |

### Invariantes

- Um `Veiculo` não pode existir sem um `Cliente` dono (`ClienteID` é NOT NULL no banco).
- CPF/CNPJ e e-mail do cliente são únicos no sistema.
- A placa do veículo é única no sistema.

---

## Agregados Independentes

Estes agregados não possuem entidades internas — são raízes simples sem filhos dentro do mesmo agregado.

### Servico

```
Servico (raiz)
```

| Campo | Descrição |
|-------|-----------|
| `Nome` | Nome comercial do serviço. |
| `PrecoBase` | Preço de referência; pode ser ajustado na cotação da OS. |
| `TempoMinutos` | Tempo estimado de execução. |

### Peca

```
Peca (raiz)
```

| Campo | Descrição |
|-------|-----------|
| `Codigo` | Código interno único da peça. |
| `Preco` | Preço unitário atual. |
| `EstoqueAtual` | Quantidade disponível. |
| `EstoqueMinimo` | Gatilho de alerta de reposição. |

### Usuario

```
Usuario (raiz)
```

| Campo | Descrição |
|-------|-----------|
| `Email` | Identificador de login (único). |
| `SenhaHash` | Hash bcrypt da senha. Nunca exposto pela API. |
| `PapelID` | FK para `PapelUsuario` (mecanico / atendente / administrador). |

---

## Diagrama de Relacionamentos

```
┌─────────────────────────────────────────────────────────────┐
│                    Agregado OrdemServico                     │
│                                                             │
│  OrdemServico ──── ItemOsServico[] ──── ref ──► Servico     │
│       │       └─── ItemOsPeca[]    ──── ref ──► Peca        │
│       │       └─── HistoricoStatus[]                        │
│       │                                                     │
│       ├── ClienteID ──────────────────────────► Cliente     │
│       └── VeiculoID ──────────────────────────► Veiculo     │
└─────────────────────────────────────────────────────────────┘

┌───────────────────────┐
│  Agregado Cliente     │
│  Cliente ─── Veiculo[]│
└───────────────────────┘

  Servico      Peca      Usuario
  (simples)  (simples)  (simples)
```

> As setas `ref` indicam referência por ID entre agregados — não composição.
> `Servico` e `Peca` são referenciados pelos itens da OS mas pertencem a agregados próprios.
