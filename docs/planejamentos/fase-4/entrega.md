# Entrega — Fase 4

Checklist de fechamento, roteiro do vídeo e conteúdo do PDF do portal.

---

## Checklist antes de submeter

### Repositórios (um por microsserviço)
- [ ] `oficina-app` (OS Service), `oficina-billing-service` e `oficina-execution-service` públicos, com `soat-architecture` **aceito** (print guardado)
- [ ] Cada um contém: código-fonte, `Dockerfile`, `k8s/` (base + overlay prod), `infra/` Terraform, `.github/workflows/`
- [ ] `main` protegida com PR obrigatório e checks obrigatórios — comprovado por um `git push origin main` rejeitado
- [ ] CI verde no último commit de cada `main`
- [ ] README com **evidência de cobertura** (badge do Sonar + print + número ≥ 80% e o critério usado)
- [ ] `docs/arquitetura.md` do serviço (componentes, mensagens consumidas/produzidas, modelo de dados)
- [ ] `docs/openapi.yaml` atualizado; coleção Postman da Fase 4 no `oficina-app`
- [ ] README do `oficina-app` com a seção **"Saga Pattern — por que orquestrada"**
- [ ] Print do post do fórum dispensando homologação citado nos 3 READMEs
- [ ] `gitleaks` limpo nos 3; nenhum token do Mercado Pago versionado

### Funcionalidade
- [ ] Caminho feliz: OS criada → diagnóstico → orçamento → aprovação → **pagamento sandbox** → reparo → `finalizada` → `entregue`
- [ ] Orçamento recusado → OS `cancelada`, reserva liberada
- [ ] Pagamento recusado (titular `OTHE`) → OS `cancelada`
- [ ] Falha no reparo (placa `FAL*`) → **estorno no painel do MP**, OS `cancelada`, estoque igual ao inicial
- [ ] Mensagem reentregue não duplica efeito; mensagem venenosa termina na DLQ
- [ ] Lambda de autenticação sem credencial de banco; `/auth/token` com os 4 resultados da Fase 3
- [ ] Billing e Execution sem acesso ao banco do OS (print do `simulate-principal-policy` ou das policies)

### Qualidade
- [ ] Cobertura ≥ 80% nos 3 serviços, com gate no CI
- [ ] BDD (godog) com os 2 cenários obrigatórios verdes (CI ou local), relatório anexado
- [ ] Quality gate do SonarCloud verde nos 3

### Observabilidade
- [ ] APM dos 3 serviços no New Relic e *service map* ligando os três
- [ ] Um trace distribuído com spans dos 3 serviços para a mesma OS
- [ ] Logs filtráveis por `saga_id`
- [ ] Dashboard da saga com dados reais (~15 sagas por script + 3–5 pagas no sandbox, 2 com `FAL*` — corte 25)
- [ ] Alerta de compensação **disparado e recebido**; alarme da DLQ testado (se não sacrificado — item 1)

### Encerramento
- [ ] Vídeo ≤ 15 min publicado (público ou não listado)
- [ ] PDF submetido no portal
- [ ] `terraform destroy` nos 6 repos e verificação de órfãos (CLBs, ENIs, RDS billing, filas, KMS)

---

## Roteiro do vídeo

**Duração-alvo: 14 minutos.** O enunciado pede quatro demonstrações; o roteiro dedica um bloco a cada uma. Grave por blocos e edite.

### Preparação (antes de ligar a gravação)

- [ ] Ambiente provisionado, 3 serviços `Running`, serviços aquecidos (uma saga completa minutos antes)
- [ ] `PERMITIR_FALHA_SIMULADA=true` no ConfigMap
- [ ] Dois veículos cadastrados: `ABC1D23` (normal) e `FAL1A23` (falha no reparo)
- [ ] Postman com a coleção da Fase 4 e as variáveis do ambiente `prod`
- [ ] Janela anônima logada como **comprador de teste** do Mercado Pago, com os cartões anotados
- [ ] New Relic aberto: dashboard da saga, *service map*, um trace pronto, Logs filtrado
- [ ] Terminal com `psql` no RDS do OS pronto para `SELECT estado, etapa_atual FROM sagas_os ...` (plano B do bloco 3)
- [ ] Uma PR preparada no Billing ou Execution, pronta para merge
- [ ] Fonte do terminal ≥ 16 pt; notificações desligadas

### Blocos

| Tempo | Bloco | O que mostrar |
|---|---|---|
| **0:00–1:30** | Abertura e arquitetura | Integrantes e o problema (rede nacional de filiais). Diagrama de componentes: *"três serviços, três bancos — dois PostgreSQL e um DynamoDB —, conversando por SQS; o OS Service orquestra a saga"*. Uma frase sobre **por que orquestrada**. Mostrar rápido os 3 repos e a `main` protegida. |
| **1:30–6:00** | **Fluxo completo de uma OS** | No Postman: atendente cria OS (`ABC1D23`) → mostrar a OS `recebida` e a peça **reservada**. Mecânico vê a fila, inicia e conclui o diagnóstico (Execution). OS vai sozinha para `aguardando_aprovacao`; mostrar o e-mail do orçamento. Cliente autentica por CPF e aprova (Billing) → link do Mercado Pago → **pagar no sandbox** com `APRO`. OS segue para execução; mecânico inicia e conclui. `GET /v1/work-orders/{id}` com o bloco `saga` mostrando todas as etapas `ok`. |
| **6:00–9:00** | **Saga e tratamento de falhas** | Repetir com o veículo `FAL1A23` até o pagamento. Ao enviar para execução, o Execution falha → mostrar a OS virando `cancelada`, as etapas `compensada` **em ordem inversa** e o **estorno no painel do Mercado Pago**. Mostrar o estoque de volta ao valor inicial. Em 30 s: uma recusa de pagamento (`OTHE`) e uma recusa de orçamento. Mencionar em uma frase a idempotência (inbox) e a DLQ. |
| **9:00–11:30** | **Deploy automatizado com validação de testes** | Mergear a PR preparada. No Actions: job de testes (cobertura ≥ 80% no Summary) → Sonar → build → deploy no EKS → `rollout status` → smoke test. `kubectl get pods -n billing-prod -w` no rolling update. Mostrar que **só aquele serviço** foi implantado. Marcador de deploy no New Relic. Citar o BDD rodando nos PRs (abrir o relatório por 10 s). |
| **11:30–14:00** | **Monitoramento e rastreamento distribuído** | *Service map* com os 3 serviços. **Abrir o trace** da OS do bloco 2: `POST /v1/work-orders` → span de produção SQS → consumo no Execution → … Logs filtrados por `saga_id` mostrando os 3 serviços em sequência. Dashboard da saga: desfechos, taxa de compensação, onde as sagas falham, tempo por etapa. Mostrar o **incidente do alerta de compensação** aberto pelas falhas do bloco 3. Encerrar recapitulando os requisitos. |

**Erros que custam pontos:**
- Mostrar só o caminho feliz — a falha **com estorno** é o coração do enunciado.
- Pagamento pelo stub no vídeo sem mostrar o Mercado Pago real ao menos uma vez.
- Esperar o diagnóstico/aprovação em tempo real — deixe os prazos curtos e corte a espera na edição.
- Dashboard vazio — gere sagas nos dias anteriores.
- Ler código. Mostre comportamento; o código fica no repositório.
- Estourar 15 minutos.

---

## Conteúdo do PDF

> Documento único a submeter no Portal do Aluno. **Confira cada item contra o que foi realmente implantado** — o avaliador abre os repositórios.

---

**Tech Challenge — Fase 4 — SOAT**
**Grupo:** *[nomes completos, RMs e usuários do GitHub dos integrantes]*

### 1. Repositórios

| # | Repositório | Papel | Link |
|---|---|---|---|
| 1 | `oficina-app` | **OS Service** — atendimento, catálogo, estoque, ciclo de vida da OS e orquestrador da saga (PostgreSQL) | `https://github.com/ProblemaTheu/oficina-app` |
| 2 | `oficina-billing-service` | **Billing Service** — orçamentos e pagamentos via Mercado Pago (PostgreSQL) | `https://github.com/ProblemaTheu/oficina-billing-service` |
| 3 | `oficina-execution-service` | **Execution Service** — fila de diagnóstico e reparo (DynamoDB) | `https://github.com/ProblemaTheu/oficina-execution-service` |
| 4 | `oficina-lambda-auth` | Autenticação por CPF e rotas do API Gateway (Fase 3, atualizado) | `https://github.com/ProblemaTheu/oficina-lambda-auth` |
| 5 | `oficina-infra-k8s` | VPC, EKS, API Gateway (Fase 3, atualizado) | `https://github.com/ProblemaTheu/oficina-infra-k8s` |
| 6 | `oficina-infra-db` | RDS do OS Service (Fase 3) | `https://github.com/ProblemaTheu/oficina-infra-db` |

✅ **Usuário `soat-architecture` adicionado como colaborador em todos os repositórios.**

### 2. Vídeo de demonstração

`[COLAR LINK DO YOUTUBE/VIMEO]` — duração: `[XX:XX]`

### 3. Arquitetura final

`[INSERIR IMAGEM do diagrama de componentes — arquitetura.md §1, exportado de mermaid.live]`

| Serviço | Banco | Recebe por | Envia para |
|---|---|---|---|
| OS Service | PostgreSQL (RDS `oficina`) | REST (`/v1/*`) · SQS `os-saga-replies.fifo` | SQS `billing-commands.fifo`, `execution-commands.fifo` |
| Billing Service | PostgreSQL (RDS `billing`) | REST (`/billing/*`) · webhook Mercado Pago · SQS `billing-commands.fifo` | SQS `os-saga-replies.fifo` · API Mercado Pago |
| Execution Service | DynamoDB (`execution-tarefas`) | REST (`/execution/*`) · SQS `execution-commands.fifo` | SQS `os-saga-replies.fifo` |
| λ auth-token | — | API Gateway `/auth/token` | REST interno do OS Service |

Nenhum serviço possui credencial ou rota de rede para o banco de outro.

### 4. Estratégia do Saga Pattern

**Orquestrada**, com o orquestrador no OS Service.

- **Fluxo:** abertura da OS com reserva de peças → diagnóstico (Execution) → orçamento (Billing) → aprovação do cliente → pagamento no Mercado Pago → reparo (Execution) → finalização.
- **Compensações**, em ordem inversa: estorno do pagamento → cancelamento do orçamento → cancelamento das tarefas → liberação da reserva de peças. A OS termina `cancelada`.
- **Garantias:** estado da saga persistido na mesma transação da OS e da mensagem (*Transactional Outbox*); consumidores idempotentes (*inbox* por `message_id`); SQS FIFO ordenado por saga; DLQ com alarme; pagamento aprovado após a compensação é estornado automaticamente.
- **Por que não coreografia:** `[resumo da tabela da RFC-006 — fluxo linear com espera humana, compensação decidida em um lugar, estado consultável em uma tabela]`.

### 5. Justificativa da divisão e das tecnologias

| Escolha | Justificativa |
|---|---|
| 3 serviços (OS, Billing, Execution) | Três áreas do negócio com donos e ritmos de mudança distintos (atendimento, cobrança, oficina); estoque permanece no OS por ser reservado na abertura da OS |
| PostgreSQL no OS e no Billing | Dados relacionais e transações ACID; dinheiro e idempotência de pagamento exigem unicidade e consistência forte |
| DynamoDB no Execution | Fila acessada por chave e por estado; sem servidor, custo ≈ zero, permissões por IAM |
| SQS FIFO | Gerenciado, ordenação por saga, DLQ nativa, custo ≈ zero no volume da operação |
| Mercado Pago Checkout Pro | Integração exigida; sandbox com usuários e cartões de teste; estorno via API como compensação |
| EKS + Pod Identity | Mantém a plataforma da Fase 3; permissões AWS mínimas por serviço |
| New Relic | Mantido da Fase 3; traces distribuídos atravessando as filas |
| godog, SonarCloud, GitHub Actions | BDD no ecossistema Go; qualidade e cobertura no CI; pipelines independentes por repositório |

**Simplificações assumidas e registradas em ADR** *(remova as que não se aplicarem)*: repositório do OS mantém o nome `oficina-app`; apenas SQS, sem SNS; compensações sequenciais; sem timeouts automáticos de aprovação/pagamento (evolução na RFC-006); Billing e Execution com réplica única; ambiente único de produção (orientação do fórum, `[LINK]`); estorno apenas total.

### 6. Documentação

| Documento | Link |
|---|---|
| Arquitetura de cada serviço | `.../<repo>/blob/main/docs/arquitetura.md` |
| RFC-005 — Decomposição em microsserviços | `.../oficina-app/blob/main/docs/rfcs/RFC-005-...md` |
| RFC-006 — Saga e consistência distribuída | `.../oficina-app/blob/main/docs/rfcs/RFC-006-...md` |
| ADRs 009–016 | `.../oficina-app/blob/main/docs/architecture-decisions.md` |
| OpenAPI de cada serviço | `.../<repo>/blob/main/docs/openapi.yaml` |
| Coleção Postman da Fase 4 | `.../oficina-app/blob/main/docs/postman/fase-4-saga.json` |
| Dashboard New Relic | `[permalink read-only]` |
