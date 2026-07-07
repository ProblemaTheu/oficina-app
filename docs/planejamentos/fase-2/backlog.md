# Backlog — Tech Challenge Fase 2

Backlog detalhado por épico. Cada tarefa segue o formato:

- **ID** — identificador estável (`F2-<épico>.<n>`) para referência em commits/PRs.
- **Descrição** — o que precisa ser feito.
- **Critérios de aceite** — como saber que terminou.
- **Sugestão técnica** — abordagem recomendada (não obrigatória — o executor pode propor outra).
- **Estimativa** — esforço relativo em pontos (P=1, M=3, G=5, GG=8).
- **Depende de** — pré-requisitos.

Convenção de estimativa: `P` pequeno (~½ dia), `M` médio (~1 dia), `G` grande (~2–3 dias), `GG` (~1 semana).

---

## E0 — Correções e dívida técnica

### F2-0.1 — Remover `/cancel` e `cancelada` da documentação ✅
- **Status:** Concluída. README sem `/cancel`/`cancelada`; diagrama de estados redesenhado fiel a `validTransitions`; OpenAPI já não tinha a rota.
- **Descrição:** O cancelamento de OS está documentado (README + diagrama de estados) mas **não existe no código nem nos PDFs** da Fase 1/2. A fonte da verdade são os PDFs, que definem apenas 6 status sem cancelamento. Alinhar a documentação à especificação e ao código.
- **Critérios de aceite:**
  - README, diagrama de estados e OpenAPI **não citam mais** `/cancel` nem `cancelada`.
  - O diagrama de estados no README reflete exatamente as transições de `validTransitions`.
  - Não há divergência entre documentação e código ao final.
- **Sugestão técnica:** Editar o README (seção "Máquina de estados" e tabela de endpoints de OS) removendo as referências a cancelamento. Confirmar que o OpenAPI não possui a rota.
- **Nota:** Implementar cancelamento é um **extra opcional** (não exigido pelos PDFs). Se decidir fazê-lo depois: adicionar `StatusCancelada`, migration de seed em `status_ordens`, handler `POST /v1/work-orders/{id}/cancel` com justificativa obrigatória e restauração de estoque (`DeduzirEstoquePecas` invertido) quando a OS estava `em_execucao`.
- **Estimativa:** P
- **Depende de:** —

### F2-0.2 — Corrigir violação de dependência de camadas (Clean Architecture) ✅
- **Status:** Concluída (commit `eef16bf`). DTOs movidos para `internal/application/usecase/ports.go`; infra passou a importar a aplicação.
- **Descrição:** A camada `application/usecase` importa `infra/repository`. Inverter a dependência.
- **Critérios de aceite:**
  - `internal/application/**` não importa mais `internal/infra/**` (validável por `go list`/grep).
  - Structs de parâmetros e resultados (`ListarOSParams`, `RelatorioTempoMedioParams`, `ItemTempoMedio`) vivem na camada de aplicação (ou domínio), e a infra os consome.
  - Testes continuam passando.
- **Sugestão técnica:** Mover os DTOs para o pacote `usecase` (ou um novo `application/dto`). As interfaces de repositório (`ports.go`) já pertencem à aplicação — apenas os tipos precisam migrar. A implementação em `infra/repository` passa a importar a aplicação, respeitando a regra de dependência apontar para dentro.
- **Estimativa:** M
- **Depende de:** —

---

## E1 — Refatoração e testes

### F2-1.1 — Auditoria de Clean Code ✅
- **Status:** Concluída. Removidos `var _ = sql.ErrNoRows`, `BuscarItensPecaParaVerificacao`, `errNaoImplementado`, mocks não usados em testes e 20 comentários `TODO` obsoletos. `.golangci.yml` adicionado (padrão + revive + gofmt); `golangci-lint run` com 0 apontamentos.
- **Descrição:** Revisar nomes, coesão de funções, remoção de código morto e comentários desatualizados.
- **Critérios de aceite:**
  - Remover artefatos como `var _ = sql.ErrNoRows` (comentário "apenas para testar…") em `ordem_servico_usecase.go`.
  - Remover `BuscarItensPecaParaVerificacao` se não utilizado (verificar referências).
  - Comentários `TODO: implementar…` em `server.go` que já estão implementados devem ser removidos.
  - `golangci-lint run` sem apontamentos novos.
- **Sugestão técnica:** Adotar `golangci-lint` com um `.golangci.yml` mínimo (govet, staticcheck, revive, errcheck, ineffassign). Integrar no CI (E6).
- **Estimativa:** M
- **Depende de:** —

### F2-1.2 — Padronizar tratamento de erros nos handlers ✅
- **Status:** Concluída. Handler central de erros (`internal/infra/http/api/erros.go`) registrado via `StrictHTTPServerOptions`; novo tipo `ErrValidacao` (400) no domínio substitui `errors.New` soltos nos use cases; repositórios agora convertem `sql.ErrNoRows` em `ErrNaoEncontrado`; switches repetidos removidos dos handlers. Verificado ponta a ponta via docker-compose (404/409/422/400).
- **Descrição:** Vários handlers retornam `nil, err` cru (ex.: `GetClientsId`, `PutClientsId`), gerando 500 genérico em casos que deveriam ser 404/400.
- **Critérios de aceite:** Handlers mapeiam `ErrNaoEncontrado`→404, `ErrConflito`→409, `ErrNaoProcessavel`→422, validação→400 de forma consistente. Erros inesperados logam contexto e retornam 500.
- **Sugestão técnica:** Extrair um helper de mapeamento erro→resposta reutilizável, evitando o switch repetido.
- **Estimativa:** M
- **Depende de:** —

### F2-1.3 — Cobrir fluxos críticos com testes automatizados ✅
- **Status:** Concluída. Use cases em 85,9% (Fase 1: ~82%); middleware em 89,8% (JWT: token válido/ausente/expirado/malformado/segredo errado, rotas públicas); handler central de erros 100% (mapeamento completo, erro embrulhado, 500 sem vazar detalhes); CriarOS 62%→86% (caminhos de erro + valor total); notificação 69%. Listagem e webhook cobertos nas F2-2.1/2.2. Suíte roda com `-race`.
- **Descrição:** Garantir testes dos fluxos que o enunciado destaca: abertura de OS, transições de status, aprovação/recusa via webhook, exclusão lógica na listagem.
- **Critérios de aceite:**
  - Testes unitários para a nova ordenação/filtro de listagem (E2).
  - Testes do webhook de aprovação (aprovar/recusar, idempotência, payload inválido).
  - Cobertura mantida ou superior à Fase 1 (~82% use cases).
  - Fluxos de erro (estoque insuficiente, transição inválida) testados.
- **Sugestão técnica:** Manter mocks de repositório (padrão já existente). Considerar testes de integração com Postgres via `testcontainers-go` para o repositório (opcional, mas valorizado).
- **Estimativa:** G
- **Depende de:** E2

### F2-1.4 — (Opcional) Testes de integração de repositório ✅
- **Status:** Concluída. Suíte com build tag `integration` em `internal/infra/repository/integration_test.go` valida SQL real: ordenação/exclusão lógica do `Listar`, transação de `DeduzirEstoquePecas` (sucesso e rollback), `AtualizarStatus` com diagnóstico/timestamps e `GerarNumeroOS`. Execução local via `scripts/test-integration.sh` (Postgres efêmero em Docker); CI via `.github/workflows/integration.yml` (service container).
- **Descrição:** Validar SQL real (ordenação, filtros, transações de estoque) contra um Postgres efêmero.
- **Critérios de aceite:** Suite de integração roda em CI com container Postgres e cobre `Listar`, `DeduzirEstoquePecas` e transições.
- **Sugestão técnica:** `testcontainers-go` ou service container do GitHub Actions. Marcar com build tag `//go:build integration` para separar do unit test rápido.
- **Estimativa:** G
- **Depende de:** F2-1.3

---

## E2 — Evolução das APIs de Ordem de Serviço

### F2-2.1 — Nova regra de listagem de OS ✅
- **Status:** Concluída. `ORDER BY` por prioridade de status + `criado_em ASC`; `finalizada`/`entregue` excluídas por padrão; parâmetro `incluir_encerradas` no contrato (OpenAPI regenerado). Testes unitários + verificação ponta a ponta via docker-compose.
- **Descrição:** Alterar `GET /v1/work-orders` para: (a) ordenar por prioridade de status **Em Execução > Aguardando Aprovação > Em Diagnóstico > Recebida**; (b) dentro de cada status, **mais antigas primeiro**; (c) **excluir logicamente** OS `finalizada` e `entregue` da listagem padrão.
- **Critérios de aceite:**
  - Listagem padrão não retorna `finalizada`/`entregue`.
  - Ordenação segue a prioridade exata do enunciado, e depois `criado_em ASC`.
  - Um parâmetro opcional (ex.: `incluir_encerradas=true` ou filtro `status`) permite consultar as encerradas quando necessário, sem quebrar relatórios.
  - Coberto por teste.
- **Sugestão técnica:** No `Listar` do repositório, trocar `ORDER BY o.criado_em DESC` por `ORDER BY CASE s.nome_status WHEN 'em_execucao' THEN 1 WHEN 'aguardando_aprovacao' THEN 2 WHEN 'em_diagnostico' THEN 3 WHEN 'recebida' THEN 4 ELSE 5 END, o.criado_em ASC`. Adicionar cláusula `AND s.nome_status NOT IN ('finalizada','entregue')` quando o filtro de status não for informado e `incluir_encerradas` for falso. "Exclusão lógica" aqui = filtrar da listagem (não deletar); confirmar que não se pretende adicionar coluna `deleted_at`.
- **Estimativa:** M
- **Depende de:** F2-0.2

### F2-2.2 — Webhook de aprovação/recusa de orçamento ✅
- **Status:** Concluída. `POST /v1/webhooks/budget-response` com assinatura HMAC-SHA256 (`X-Signature`, segredo em `WEBHOOK_SECRET`), idempotência via estado da OS (`AprovadoEm`/`ReprovadoEm`), reutilizando `AprovarOrcamento`/`RejeitarOrcamento`. Testes de use case + middleware; verificado ponta a ponta (200/200 idempotente/401/404/422/400).
- **Descrição:** Endpoint inbound para receber notificações externas de aprovação ou recusa do orçamento (substitui/complementa os atuais `approve`/`reject` acionados manualmente).
- **Critérios de aceite:**
  - `POST /v1/webhooks/budget-response` (ou nome equivalente) recebe `{ os_id | numero, decisao: "aprovado"|"recusado", motivo? }`.
  - Aplica a mesma regra de domínio: só processa OS em `aguardando_aprovacao`; aprovar → `em_execucao` (deduz estoque); recusar → `finalizada`/rejeitada.
  - Autenticação apropriada para webhook (assinatura HMAC do provedor OU token dedicado), **não** o JWT de usuário.
  - Idempotência: reprocessar a mesma notificação não gera efeito duplicado.
  - Payload inválido → 400; OS inexistente → 404; estado inválido → 422.
  - Coberto por teste.
- **Sugestão técnica:** Reutilizar `AprovarOrcamento`/`RejeitarOrcamento`. Validar assinatura via header (`X-Signature`) comparando HMAC-SHA256 do corpo com um segredo em `Secret`/env. Idempotência via chave de evento (armazenar `event_id` processado, ou usar o próprio estado da OS como guarda). Registrar rota como pública no middleware JWT (como já é feito para `/status`), porém protegida pela verificação de assinatura.
- **Estimativa:** G
- **Depende de:** F2-0.2

### F2-2.3 — Notificação de mudança de status por e-mail ✅
- **Status:** Concluída. Porta `notifier` na aplicação; implementações `LogNotifier` (padrão local) e `SMTPNotifier` (produção/MailHog) em `internal/infra/notification`, selecionadas via env `NOTIFIER`. Disparo assíncrono (goroutine) nas transições relevantes (aguardando_aprovacao, em_execucao, finalizada, entregue) sem bloquear/falhar a transição. Testes com `-race`; verificado ponta a ponta pelos logs do compose.
- **Descrição:** Ao mudar o status de uma OS, notificar o cliente por e-mail (requisito "atualização de status via ferramenta como e-mail").
- **Critérios de aceite:**
  - Em transições relevantes (ex.: `aguardando_aprovacao`, `em_execucao`, `finalizada`, `entregue`), um e-mail é enviado ao cliente.
  - O envio é **assíncrono/desacoplado** e não bloqueia nem falha a transição se o provedor estiver fora (log de erro, retry opcional).
  - Configurável por variáveis de ambiente/Secret (provedor, API key, remetente).
  - Em ambiente local, um "mailer" fake/console permite demonstrar sem provedor real.
- **Sugestão técnica:** Definir uma porta `Notifier` na aplicação (interface) com implementações: `SMTPNotifier`/`SendGridNotifier` (produção) e `LogNotifier` (local/testes). Disparar via goroutine com contexto ou, melhor, uma fila simples (channel) para desacoplar. Para K8s, credenciais em `Secret`. Alternativa robusta: publicar evento `StatusDaOsAlterado` e um worker consumir — porém pode ser over-engineering para o escopo; documentar a evolução.
- **Estimativa:** G
- **Depende de:** —

### F2-2.4 — Atualizar contrato OpenAPI e regenerar código ✅
- **Status:** Concluída. `openapi.yaml` já cobria listagem (F2-2.1) e webhook (F2-2.2); regeneração confirmada sem drift (`oapi-codegen v1.16.3`). Coleção Postman sincronizada: parâmetro `incluir_encerradas`, pasta Webhooks com pre-request script que assina o corpo (HMAC) e variáveis `webhookSecret`/`webhookSignature`. Tabela de endpoints do README atualizada.
- **Descrição:** Refletir no `docs/openapi.yaml` as mudanças (novo parâmetro de listagem, webhook, eventual `/cancel`) e regenerar.
- **Critérios de aceite:** `go generate ./internal/infra/http/api/...` executado; `api.gen.go` atualizado; contrato válido; coleção Postman sincronizada (E7).
- **Sugestão técnica:** Editar sempre o YAML primeiro (fonte da verdade). Validar com `oapi-codegen`/`kin-openapi` antes de commitar.
- **Estimativa:** M
- **Depende de:** F2-2.1, F2-2.2

---

## E3 — Containerização (revisão)

### F2-3.1 — Revisar Dockerfile
- **Descrição:** Endurecer e otimizar a imagem.
- **Critérios de aceite:**
  - Build reprodutível (versão de Go fixada — já usa `golang:1.26-alpine`).
  - Usuário não-root (já presente) e imagem final mínima.
  - `HEALTHCHECK` presente (já existe) e coerente com os endpoints de saúde.
  - Considerar imagem `scratch`/`distroless` para reduzir superfície (opcional).
  - Trivy no CI sem CRITICAL/HIGH sem fix.
- **Sugestão técnica:** Adicionar `-ldflags="-s -w"` no build para reduzir binário; considerar `gcr.io/distroless/static`. Garantir `.dockerignore` para não copiar `pdfs/`, `docs/`, `.git`.
- **Estimativa:** P
- **Depende de:** —

### F2-3.2 — Revisar docker-compose para desenvolvimento local
- **Descrição:** Facilitar o ciclo local e alinhar com K8s.
- **Critérios de aceite:**
  - Segredos via `.env` (não hardcoded no compose); `.env.example` versionado.
  - Volume de dados do Postgres e healthcheck (já existem).
  - Opcional: serviço de mailer fake (ex.: MailHog) para demonstrar notificações por e-mail localmente.
- **Sugestão técnica:** Adicionar `env_file: .env`. Incluir MailHog (`axllent/mailpit` ou `mailhog/mailhog`) exposto para visualizar e-mails no fluxo de demonstração.
- **Estimativa:** P
- **Depende de:** F2-2.3

---

## E4 — Kubernetes (`/k8s`)

### F2-4.1 — Estrutura base e namespace
- **Descrição:** Organizar `/k8s` e criar namespace dedicado.
- **Critérios de aceite:** `/k8s` com layout claro (base + overlays, ou pastas por recurso). `kubectl apply` cria o namespace `oficina` (ou similar).
- **Sugestão técnica:** Usar **Kustomize** (`base/` + `overlays/local` + `overlays/aws`) para reaproveitar manifestos entre ambientes.
- **Estimativa:** P
- **Depende de:** —

### F2-4.2 — ConfigMap e Secret
- **Descrição:** Externalizar configuração e segredos.
- **Critérios de aceite:**
  - `ConfigMap` com variáveis não sensíveis (DB_HOST, DB_PORT, DB_NAME, portas, remetente de e-mail).
  - `Secret` com sensíveis (JWT_SECRET, DB_PASSWORD, API key de e-mail, segredo do webhook).
  - App consome ambos via `envFrom`.
  - Secret de exemplo versionado como template (sem valores reais); valores reais fora do git.
- **Sugestão técnica:** `secret.example.yaml` versionado; instrução para `kubectl create secret` ou selar com Sealed Secrets/SOPS (mencionar como evolução).
- **Estimativa:** M
- **Depende de:** F2-4.1

### F2-4.3 — Deployment da aplicação
- **Descrição:** Deployment da API com boas práticas.
- **Critérios de aceite:**
  - `readinessProbe` → `/health/ready`; `livenessProbe` → `/health/live`.
  - `resources.requests`/`limits` de CPU e memória definidos (**pré-requisito do HPA**).
  - `replicas` inicial ≥ 2; imagem parametrizada por tag (do Docker Hub).
  - Rollout sem downtime (`RollingUpdate`).
- **Sugestão técnica:** Alinhar probes com os handlers já existentes em `main.go`. Definir requests modestos (ex.: 100m CPU / 128Mi) para o HPA ter margem de escala na demonstração.
- **Estimativa:** M
- **Depende de:** F2-4.2

### F2-4.4 — Service
- **Descrição:** Expor a aplicação no cluster.
- **Critérios de aceite:** `Service` ClusterIP para a API; acesso externo via `port-forward` (local) ou Ingress/LoadBalancer (AWS).
- **Sugestão técnica:** ClusterIP + Ingress (nginx) no overlay. Local: `kubectl port-forward` documentado.
- **Estimativa:** P
- **Depende de:** F2-4.3

### F2-4.5 — Horizontal Pod Autoscaler (HPA)
- **Descrição:** Escalar pods conforme CPU/memória.
- **Critérios de aceite:**
  - `HPA` escala a API entre min e max réplicas por utilização de CPU (e/ou memória).
  - Comprovadamente escala sob carga (para o vídeo).
  - `metrics-server` disponível no cluster.
- **Sugestão técnica:** `autoscaling/v2`, target ~50% CPU, min 2 / max 5. Instalar `metrics-server` (no kind requer flags TLS). Gerar carga com `hey`/`k6`/`ab` para o vídeo.
- **Estimativa:** M
- **Depende de:** F2-4.3

### F2-4.6 — Banco de dados no cluster (ambiente local)
- **Descrição:** Provisionar Postgres para o ambiente local em K8s.
- **Critérios de aceite:** `StatefulSet` (ou Deployment) do Postgres com `PVC`, `Service` e credenciais via Secret. App conecta com sucesso; migrations rodam no start.
- **Sugestão técnica:** `StatefulSet` + `PersistentVolumeClaim`. Em **AWS**, substituir por **RDS** (provisionado no Terraform, E5) — daí o overlay `aws` não inclui o Postgres in-cluster.
- **Estimativa:** M
- **Depende de:** F2-4.2

### F2-4.7 — (Opcional) Job de migrations
- **Descrição:** Rodar migrations como Job/initContainer em vez de no boot da app.
- **Critérios de aceite:** Migrations aplicadas de forma controlada antes do rollout; app não corre migrations concorrentemente em múltiplas réplicas.
- **Sugestão técnica:** Como a app roda migrations no start e haverá ≥2 réplicas, avaliar mover para um `Job`/`initContainer` para evitar corrida. `golang-migrate` já é usado.
- **Estimativa:** M
- **Depende de:** F2-4.3

---

## E5 — Terraform / IaC (`/infra`)

### F2-5.1 — Estrutura de módulos e backend
- **Descrição:** Organizar o Terraform em módulos reutilizáveis com separação local/AWS.
- **Critérios de aceite:** `/infra` com `modules/` e `environments/local` + `environments/aws`; `versions.tf` com providers fixados; backend de state definido (local agora, S3 no futuro).
- **Sugestão técnica:** `environments/local` provisiona cluster kind e aplica manifestos; `environments/aws` provisiona VPC, EKS, RDS e ECR. Manter `aws` funcional mas não aplicado por padrão.
- **Estimativa:** M
- **Depende de:** —

### F2-5.2 — Provisionamento do cluster local (kind/minikube)
- **Descrição:** Subir o cluster local via Terraform.
- **Critérios de aceite:** `terraform apply` cria o cluster local e deixa o `kubeconfig` acessível; opcionalmente instala metrics-server e ingress.
- **Sugestão técnica:** Provider `tehcyx/kind` ou `null_resource` chamando `kind create cluster`. Documentar pré-requisitos (Docker, kind).
- **Estimativa:** M
- **Depende de:** F2-5.1

### F2-5.3 — Módulo AWS (EKS + RDS + ECR) — preparado para o futuro
- **Descrição:** Escrever (sem necessariamente aplicar) o provisionamento de cluster e banco na AWS.
- **Critérios de aceite:** Módulos para VPC, EKS, RDS (Postgres) e ECR; variáveis e outputs documentados; `terraform plan` válido.
- **Sugestão técnica:** Usar módulos oficiais `terraform-aws-modules/{vpc,eks,rds}`. Deixar claro no README que é opt-in e gera custo. Não commitar credenciais.
- **Estimativa:** GG
- **Depende de:** F2-5.1

### F2-5.4 — Documentação do Terraform
- **Descrição:** Documentar quais recursos são criados e como aplicar.
- **Critérios de aceite:** `infra/README.md` com pré-requisitos, comandos (`init/plan/apply/destroy`), variáveis e diagrama dos recursos, para local e AWS.
- **Sugestão técnica:** Gerar tabela de variáveis (ou usar `terraform-docs`).
- **Estimativa:** P
- **Depende de:** F2-5.2

---

## E6 — CI/CD

### F2-6.1 — Workflow de CI (build + test + lint)
- **Descrição:** Validar cada push/PR.
- **Critérios de aceite:** Workflow roda `go build`, `go test` (com cobertura), `golangci-lint` e `go vet`; falha bloqueia o merge.
- **Sugestão técnica:** Reaproveitar cache do `actions/setup-go`. Manter o `security.yml` existente (govulncheck/gosec/Trivy/Sonar) como parte da suíte de qualidade.
- **Estimativa:** M
- **Depende de:** F2-1.1

### F2-6.2 — Build e push da imagem Docker (Docker Hub)
- **Descrição:** Construir e publicar a imagem no Docker Hub.
- **Critérios de aceite:** Em push na `main` (ou tag), imagem é construída e publicada com tag por SHA e `latest`; login via secrets do GitHub (`DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`).
- **Sugestão técnica:** `docker/login-action` + `docker/build-push-action` com `cache-from/to`. Multi-plataforma (amd64/arm64) opcional.
- **Estimativa:** M
- **Depende de:** F2-3.1, F2-6.1

### F2-6.3 — Deploy no cluster Kubernetes
- **Descrição:** Aplicar os manifestos e o banco no cluster a partir da pipeline.
- **Critérios de aceite:** Após publicar a imagem, o job faz `kubectl apply`/`kustomize build | apply` no cluster alvo, atualiza a tag da imagem e aguarda o rollout; migrations aplicadas.
- **Sugestão técnica:** Local não é acessível pelo runner hospedado — para a demonstração use **self-hosted runner** ou execute o deploy manualmente/por script no vídeo, deixando o job pronto para EKS (kubeconfig via secret/OIDC). Documentar essa limitação com honestidade.
- **Estimativa:** G
- **Depende de:** F2-6.2, E4

### F2-6.4 — Gestão de secrets da pipeline
- **Descrição:** Centralizar segredos do CI/CD.
- **Critérios de aceite:** Segredos (`DOCKERHUB_TOKEN`, `KUBECONFIG`/OIDC, `SONAR_TOKEN` já existente) configurados no repositório e referenciados; nenhum segredo em texto plano no código.
- **Sugestão técnica:** GitHub Environments com required reviewers para o ambiente de deploy.
- **Estimativa:** P
- **Depende de:** F2-6.2

---

## E7 — Documentação e entrega

### F2-7.1 — Atualizar README com arquitetura da Fase 2
- **Descrição:** Documentar solução, arquitetura e instruções.
- **Critérios de aceite:** README com: descrição/objetivos da Fase 2; desenho da arquitetura (componentes da aplicação, infraestrutura provisionada, fluxo de deploy); instruções de execução local, deploy K8s e Terraform; links da coleção e do vídeo.
- **Sugestão técnica:** Diagramas em Mermaid (versionáveis) e/ou exportados do Miro. Reaproveitar o board existente.
- **Estimativa:** M
- **Depende de:** E4, E5, E6

### F2-7.2 — Desenho da arquitetura
- **Descrição:** Diagrama dos componentes e do fluxo de deploy.
- **Critérios de aceite:** Diagrama mostra API, banco, K8s (pods/HPA/service/ingress), Docker Hub, pipeline CI/CD e integrações (webhook/e-mail). Exportado em imagem e embutido no README/PDF.
- **Sugestão técnica:** Atualizar o board do Miro e exportar; ou Mermaid `flowchart`.
- **Estimativa:** M
- **Depende de:** F2-7.1

### F2-7.3 — Coleção de APIs atualizada
- **Descrição:** Sincronizar Postman/OpenAPI com as novas rotas.
- **Critérios de aceite:** `docs/postman_collection.json` e `docs/openapi.yaml` cobrem webhook, nova listagem e eventual `/cancel`; link no README.
- **Sugestão técnica:** Exportar do Postman após validar manualmente; ou servir Swagger UI a partir do OpenAPI embutido.
- **Estimativa:** P
- **Depende de:** F2-2.4

### F2-7.4 — Vídeo demonstrativo (≤15 min)
- **Descrição:** Gravar demonstração do ambiente em execução.
- **Critérios de aceite:** Vídeo publicado (YouTube/Vimeo, público ou não listado) demonstrando: deploy da aplicação, execução do CI/CD, consumo das APIs e escalabilidade automática (simulação de carga). Link no README e no PDF.
- **Sugestão técnica:** Roteiro: (1) `terraform apply` sobe cluster; (2) pipeline builda/publica; (3) deploy no K8s; (4) chamadas de API (abertura de OS, webhook de aprovação, e-mail no MailHog); (5) `hey`/`k6` gera carga e HPA escala (`kubectl get hpa -w`).
- **Estimativa:** M
- **Depende de:** Todos os épicos

### F2-7.5 — Entrega no portal
- **Descrição:** Submeter a entrega formal.
- **Critérios de aceite:** Repositório compartilhado com o usuário `soat-architecture`; PDF com link do repo, desenho da arquitetura e link do vídeo submetido no portal do aluno.
- **Sugestão técnica:** Conferir permissão de acesso ao repo antes da submissão.
- **Estimativa:** P
- **Depende de:** F2-7.4

---

## Resumo de esforço por épico

| Épico | Tarefas | Estimativa somada |
|-------|---------|-------------------|
| E0 | 2 | M + M |
| E1 | 4 | M + M + G + G |
| E2 | 4 | M + G + G + M |
| E3 | 2 | P + P |
| E4 | 7 | P + M + M + P + M + M + M |
| E5 | 4 | M + M + GG + P |
| E6 | 4 | M + M + G + P |
| E7 | 5 | M + M + P + M + P |

> Total aproximado: ~13–18 dias de trabalho focado para uma pessoa; paralelizável entre 2–3 integrantes (código × infra × CI-CD/doc).
