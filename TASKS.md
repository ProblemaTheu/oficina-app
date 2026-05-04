# Tarefas Pendentes — Tech Challenge

> Itens identificados como incompletos após revisão do código em 03/05/2025.
> Todos os fluxos de negócio principais estão funcionando. O que falta é documentação complementar, cobertura de testes e um ajuste na máquina de estados.

---

## 1. ✅ Estado `cancelada` na máquina de estados da OS

- [x] Criar migration `000005` adicionando `cancelada` na tabela `status_ordens` + coluna `cancelado_em`
- [x] Adicionar constante `StatusCancelada` em `internal/domain/entity/ordem_servico.go`
- [x] Definir transições válidas: `recebida`, `em_diagnostico`, `aguardando_aprovacao`, `em_execucao` → `cancelada`
- [x] Adicionar endpoint `POST /v1/work-orders/{id}/cancel` no `docs/openapi.yaml`
- [x] Adicionar schema `CancelarOSRequest` e campo `cancelado_em` em `OrdemServicoResponse`
- [x] Regenerar código com `oapi-codegen`
- [x] Implementar `CancelarOS` em `internal/application/usecase/ordem_servico_usecase.go`
- [x] Implementar handler `PostWorkOrdersIdCancel` em `internal/infra/http/api/server.go`
- [x] Restaurar estoque de peças automaticamente quando cancelado a partir de `em_execucao`
- [x] Atualizar diagrama da máquina de estados no `README.md`

---

## 2. Testes unitários — cobertura ≥ 80%

Existe 1 arquivo de teste (`ordem_servico_usecase_test.go`). Cobertura estimada: ~10–15%.

### Value Objects

- [ ] `internal/domain/valueobject/document_test.go`
  - [ ] CPF válido aceito
  - [ ] CPF inválido rejeitado (dígitos verificadores errados)
  - [ ] CNPJ válido aceito
  - [ ] CNPJ inválido rejeitado
  - [ ] Entradas com máscara (`123.456.789-09`) e sem máscara (`12345678909`)
- [ ] `internal/domain/valueobject/plate_test.go`
  - [ ] Placa no formato antigo (`ABC1234`) aceita
  - [ ] Placa no formato Mercosul (`ABC1D23`) aceita
  - [ ] Placa inválida rejeitada

### Use Cases

- [ ] `internal/application/usecase/auth_usecase_test.go`
  - [ ] Registro com e-mail duplicado retorna `ErrConflito`
  - [ ] Login com senha errada retorna erro
  - [ ] Login bem-sucedido retorna token JWT válido
- [ ] `internal/application/usecase/client_usecase_test.go`
  - [ ] Criação com CPF duplicado retorna `ErrConflito`
  - [ ] Busca por ID inexistente retorna `ErrNaoEncontrado`
  - [ ] Atualização de cliente existente persiste corretamente
- [ ] `internal/application/usecase/vehicle_usecase_test.go`
  - [ ] Placa duplicada retorna `ErrConflito`
  - [ ] Veículo vinculado a cliente inexistente retorna `ErrNaoEncontrado`
- [ ] `internal/application/usecase/part_usecase_test.go`
  - [ ] Ajuste de estoque `entrada` incrementa corretamente
  - [ ] Ajuste de estoque `saida` com quantidade maior que estoque retorna erro
  - [ ] Ajuste de estoque `ajuste` define valor absoluto
- [ ] `internal/application/usecase/ordem_servico_usecase_test.go` (expandir existente)
  - [ ] Criação de OS com veículo que não pertence ao cliente retorna erro *(já coberto)*
  - [ ] Criação de OS com estoque insuficiente retorna erro *(já coberto)*
  - [ ] `valor_total` calculado corretamente a partir dos itens
  - [ ] Cancelamento a partir de cada estado válido
  - [ ] Cancelamento a partir de estado inválido (`finalizada`, `entregue`) retorna erro

### Executar e verificar cobertura

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
# Alvo: total >= 80%
```

---

## 3. Testes de integração

Nenhum teste de integração existe. Os testes devem rodar contra um banco PostgreSQL real (pode ser via Docker em CI).

- [ ] Criar `docker-compose.test.yml` com PostgreSQL isolado para testes
- [ ] Criar helper `internal/testutil/db.go` para setup/teardown do banco nos testes
  - [ ] Subir migrations antes de cada suite
  - [ ] Truncar tabelas entre testes (ou usar transactions com rollback)
- [ ] `internal/infra/repository/client_repository_test.go`
  - [ ] Criar cliente e buscar por ID
  - [ ] Buscar por documento (CPF/CNPJ)
  - [ ] Criar duplicado retorna erro de constraint
  - [ ] Deletar e confirmar `ErrNaoEncontrado`
- [ ] `internal/infra/repository/ordem_servico_repository_test.go`
  - [ ] Criar OS com itens e buscar detalhes completos
  - [ ] Avançar status e verificar `historico_status` gerado
  - [ ] Relatório de tempo médio retorna apenas OSs com `iniciado_em` preenchido
- [ ] `internal/infra/repository/peca_repository_test.go`
  - [ ] Ajuste de estoque reflete no banco
  - [ ] Filtro `estoque_baixo` retorna apenas peças abaixo do mínimo
- [ ] Fluxo E2E da OS (do tipo integração, sem HTTP):
  - [ ] Criar cliente → veículo → OS → avançar todos os estados → entregue
  - [ ] Criar OS → avançar até `aguardando_aprovacao` → rejeitar → confirmar `finalizada`

```bash
# Rodar apenas testes de integração (usar build tag)
go test ./... -tags=integration
```

---

## 4. Diagramas DDD e Linguagem Ubíqua

O código segue DDD mas os artefatos de documentação visual não existem no repositório.

### Linguagem Ubíqua

- [ ] Criar `docs/ubiquitous-language.md` com glossário dos termos do domínio:
  - Ordem de Serviço (OS), Diagnóstico, Orçamento, Aprovação, Execução, Entrega
  - Cliente, Veículo, Placa, Serviço, Peça/Insumo
  - Estoque Mínimo, Estoque Atual, Ajuste, Entrada, Saída
  - Status, Transição, Histórico de Status
  - Mecânico, Atendente, Administrador

### Diagramas DDD

- [ ] Criar `docs/ddd-aggregates.md` (ou imagem exportada do Miro) documentando:
  - [ ] Agregado **OrdemServico** (raiz) → `ItensOsServico`, `ItensOsPeca`, `HistoricoStatus`
  - [ ] Agregado **Cliente** → `Veiculo`
  - [ ] Agregados independentes: **Servico**, **Peca**, **Usuario**
- [ ] Criar `docs/bounded-contexts.md` documentando os contextos delimitados:
  - [ ] Contexto **Atendimento** (clientes, veículos, abertura de OS)
  - [ ] Contexto **Oficina** (execução de serviços, uso de peças, transições de status)
  - [ ] Contexto **Estoque** (peças, controle de inventário)
  - [ ] Contexto **Segurança** (usuários, autenticação, papéis)
- [ ] Referenciar os diagramas do Miro no `README.md`

---

## 5. Justificativa de banco de dados e documentação de objetivos

- [ ] Criar `docs/architecture-decisions.md` com:
  - [ ] **Escolha do PostgreSQL**: justificar frente às alternativas (MySQL, SQLite, MongoDB)
    - Suporte a transações ACID (crítico para controle de estoque e transições de status)
    - Constraints de integridade referencial (FKs entre OS, clientes, veículos)
    - Suporte nativo a tipos decimais precisos (preços e valores financeiros)
  - [ ] **Escolha do chi** como router vs Echo, Gin, Fiber
  - [ ] **Escolha do oapi-codegen** (API-first) vs geração manual de handlers
- [ ] Criar `docs/objectives.md` com:
  - [ ] Objetivo do sistema (problema que resolve)
  - [ ] Escopo do desafio técnico
  - [ ] Requisitos funcionais atendidos
  - [ ] Requisitos não-funcionais (segurança, containerização, observabilidade)
- [ ] Referenciar ambos os documentos no `README.md`

---

## Resumo

| # | Tarefa | Complexidade | Prioridade |
|---|---|---|---|
| 1 | Estado `cancelada` na OS | Baixa | Alta |
| 2 | Testes unitários ≥ 80% | Alta | Alta |
| 3 | Testes de integração | Alta | Média |
| 4 | Diagramas DDD + Linguagem Ubíqua | Média | Média |
| 5 | Justificativa de BD + objetivos | Baixa | Baixa |
