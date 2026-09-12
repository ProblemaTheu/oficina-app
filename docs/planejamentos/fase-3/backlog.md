# Backlog — Tech Challenge Fase 3

Backlog detalhado por épico. Cada tarefa segue o formato:

- **ID** — identificador estável (`F3-<épico>.<n>`) para referência em commits/PRs.
- **Descrição** — o que precisa ser feito e **por quê**.
- **Como fazer** — passo a passo com comandos e código. É um ponto de partida testável, não um dogma.
- **Critérios de aceite** — como saber que terminou.
- **Estimativa** — `P` (~½ dia), `M` (~1 dia), `G` (~2–3 dias), `GG` (~1 semana).
- **Depende de** — pré-requisitos.

**Convenção de nomes usada em todo o documento:**

| Item | Valor |
|---|---|
| Prefixo de recursos | `oficina` |
| Região | `us-east-1` |
| Ambientes | `homolog`, `prod` |
| Repos | `oficina-app` (atual), `oficina-lambda-auth`, `oficina-infra-k8s`, `oficina-infra-db` |
| Namespace SSM | `/oficina/<env>/<chave>` |

---

## E0 — Fundação e contratos

> Este épico é **bloqueante**: sem ele, os quatro repositórios não conseguem se coordenar. É pouca coisa, mas mal feita aqui vira retrabalho em toda a fase.

### F3-0.1 — Backend remoto do Terraform (S3 com lock nativo) ✅

- **Descrição:** Hoje o state do Terraform está **versionado no repositório** (`infra/environments/local/terraform.tfstate`). Isso não funciona para uma equipe: dois `apply` simultâneos corrompem o state, e o state contém segredos em texto puro (a senha do RDS gerada por `random_password` está lá). Com dois repos de infra separados, um backend remoto compartilhado deixa de ser boa prática e vira **requisito funcional**.

- **Como fazer:**

  Este é o único recurso criado "na mão" — é o ovo antes da galinha. Crie um diretório `bootstrap/` no repo `oficina-infra-k8s` com backend local, aplique uma vez e nunca mais mexa:

  ```hcl
  # bootstrap/main.tf
  terraform {
    required_version = ">= 1.9"
    required_providers {
      aws = { source = "hashicorp/aws", version = "~> 5.60" }
    }
  }

  provider "aws" {
    region = "us-east-1"
    default_tags {
      tags = {
        Projeto   = "oficina"
        Fase      = "3"
        ManagedBy = "terraform"
      }
    }
  }

  resource "aws_s3_bucket" "tfstate" {
    bucket        = "oficina-tfstate-${data.aws_caller_identity.atual.account_id}"
    force_destroy = false
  }

  data "aws_caller_identity" "atual" {}

  resource "aws_s3_bucket_versioning" "tfstate" {
    bucket = aws_s3_bucket.tfstate.id
    versioning_configuration { status = "Enabled" }
  }

  # O state guarda segredos: criptografia em repouso é obrigatória
  resource "aws_s3_bucket_server_side_encryption_configuration" "tfstate" {
    bucket = aws_s3_bucket.tfstate.id
    rule {
      apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
    }
  }

  resource "aws_s3_bucket_public_access_block" "tfstate" {
    bucket                  = aws_s3_bucket.tfstate.id
    block_public_acls       = true
    block_public_policy     = true
    ignore_public_acls      = true
    restrict_public_buckets = true
  }

  # SEM tabela DynamoDB: a partir do Terraform 1.10 o backend S3 tem lock
  # nativo (`use_lockfile = true`), e `dynamodb_table` está deprecado.
  # Um recurso a menos para criar, pagar e destruir.

  output "bucket" { value = aws_s3_bucket.tfstate.bucket }
  ```

  ```bash
  cd bootstrap && terraform init && terraform apply
  ```

  Depois disso, **cada** repo de infra declara o backend apontando para uma *key* diferente:

  ```hcl
  # oficina-infra-k8s/terraform/backend.tf
  terraform {
    backend "s3" {
      bucket       = "oficina-tfstate-706215605178"
      key          = "infra-k8s/terraform.tfstate"
      region       = "us-east-1"
      use_lockfile = true   # lock nativo do S3 (TF >= 1.10)
      encrypt      = true
    }
  }
  ```

  ```hcl
  # oficina-infra-db/terraform/backend.tf → key = "infra-db/terraform.tfstate"
  # oficina-lambda-auth/terraform/backend.tf → key = "lambda-auth/terraform.tfstate"
  ```

  **Workspaces: só onde fazem sentido.** É tentador criar `homolog`/`prod` como workspaces em todos os repos, mas isso significa **duplicar tudo** — inclusive o cluster EKS (+US$ 73/mês) e a VPC. Nossa divisão:

  > ⏱️ **[Cortes 10 e 14](plano-10-dias.md#os-cortes)** — com um ambiente só, nenhum repo usa workspaces: use o `default` nos três. A tabela abaixo descreve o desenho de dois ambientes.

  | Repo | Workspaces? | Por quê |
  |---|---|---|
  | `oficina-infra-k8s` | **não** — state único | VPC, EKS e ALB são compartilhados; o que varia por ambiente sai de um `for_each` |
  | `oficina-infra-db` | **não** — state único | duas instâncias RDS pequenas criadas por `for_each`, no mesmo state |
  | `oficina-lambda-auth` | **sim** | o código da função muda por branch; cada ambiente tem seu deploy independente |

  ```bash
  # apenas em oficina-lambda-auth
  terraform workspace new homolog && terraform workspace new prod
  ```

  O S3 grava o workspace em `env:/<workspace>/<key>` — states isolados, um backend só. Leia a [Estratégia de ambientes](#estratégia-de-ambientes-leia-antes-de-escrever-hcl) antes de escrever qualquer HCL.

  > ⚠️ **Antes de commitar qualquer coisa:** remova os `.tfstate` do repositório atual e do histórico se possível, e garanta que o `.gitignore` cobre `*.tfstate*`, `.terraform/` e `*.tfvars` (exceto `*.example.tfvars`). Rode `git log --all --full-history -- "*.tfstate"` para conferir o estrago; se houver senha exposta, ela precisa ser rotacionada.

- **Status:** ✅ Concluída — ver [execucao.md](execucao.md#valores-do-ambiente). Código em `oficina-infra-k8s/bootstrap/`.

- **Critérios de aceite:**
  - Bucket S3 versionado e criptografado existe; lock via `use_lockfile`.
  - Os 3 repos com Terraform usam backend S3 com keys distintas.
  - Workspaces `homolog` e `prod` criados.
  - Nenhum `.tfstate` versionado; `.gitignore` atualizado nos 3 repos.
- **Estimativa:** P
- **Depende de:** —

---

### F3-0.2 — Definir o contrato do JWT entre Lambda e aplicação

- **Descrição:** Dois emissores diferentes (a aplicação, para funcionários; a Lambda, para clientes) vão assinar tokens com o **mesmo segredo**. Sem `iss`, `aud` e `tipo`, a aplicação não consegue distinguir um token de cliente de um de funcionário — e um cliente poderia acessar rotas administrativas. Este contrato precisa estar fechado **antes** de escrever a Lambda.

- **Como fazer:**

  Documente em `docs/rfcs/RFC-003-autenticacao.md` (E6) e implemente exatamente isto:

  ```jsonc
  // Token emitido pela λ auth-token (cliente final)
  {
    "sub":   "9f1c...",                  // UUID do cliente
    "cpf":   "52998224725",              // apenas dígitos
    "nome":  "Maria Silva",
    "tipo":  "cliente",                  // ← discriminador
    "iss":   "oficina-auth-lambda",
    "aud":   "oficina-api",
    "iat":   1772000000,
    "exp":   1772003600                  // 1 hora
  }

  // Token emitido pela aplicação (funcionário) — POST /v1/auth/login
  {
    "sub":   "3ab8...",                  // UUID do usuário
    "email": "atendente@oficina.com",
    "nome":  "João Souza",
    "papel": "atendente",                // ← NOVO: hoje o claim não existe
    "tipo":  "usuario",                  // ← NOVO
    "iss":   "oficina-api",              // ← NOVO
    "aud":   "oficina-api",              // ← NOVO
    "iat":   1772000000,
    "exp":   1772028800                  // 8 horas (mantido)
  }
  ```

  Regras de autorização derivadas do contrato:

  | Rota | `tipo` aceito | Observação |
  |---|---|---|
  | `POST /auth/token` | — | pública (é onde o token nasce) |
  | `POST /v1/auth/login` | — | pública |
  | `GET /v1/work-orders/{id}/status` | — | pública (consulta por número da OS) |
  | `POST /v1/webhooks/budget-response` | — | HMAC, sem JWT |
  | `GET /v1/work-orders` (minhas OS) | `cliente`, `usuario` | cliente vê **só as suas** — filtro por `sub` |
  | `POST /v1/work-orders`, `PATCH .../status` | `usuario` | operação interna |
  | `POST /v1/clients`, `/v1/parts`, `/v1/services` | `usuario` | operação interna |
  | `GET /v1/reports/*` | `usuario` | relatório gerencial |

  Escreva o contrato como **teste** antes de implementar (`internal/infra/http/middleware/jwt_test.go`), garantindo que um token com `tipo: "cliente"` receba 403 em `POST /v1/work-orders`.

- **Critérios de aceite:**
  - Contrato de claims documentado e revisado pelo time.
  - Tabela de autorização por rota acordada.
  - Testes escritos (ainda falhando) que expressam as regras.
- **Estimativa:** P
- **Depende de:** —

---

### F3-0.3 — Convenções de nomes, tags e contrato SSM entre repositórios

- **Descrição:** Com quatro repositórios aplicando Terraform na mesma conta, sem convenção vira caos: recursos órfãos, colisão de nomes e impossibilidade de rastrear custo por componente.

- **Como fazer:**

  **1. Tags padrão** — em todo provider AWS dos 3 repos de infra:

  ```hcl
  provider "aws" {
    region = var.regiao
    default_tags {
      tags = {
        Projeto     = "oficina"
        Fase        = "3"
        Repositorio = "oficina-infra-k8s"          # muda por repo
        ManagedBy   = "terraform"
      }
    }
  }
  ```

  Note que `Ambiente` **não** entra em `default_tags`: em `infra-k8s`/`infra-db` o state é único e o ambiente varia por recurso. Marque-o onde ele existe de fato:

  ```hcl
  resource "aws_lb_target_group" "api" {
    for_each = local.ambientes
    # ...
    tags = { Ambiente = each.key }
  }
  ```

  Em `oficina-lambda-auth`, que usa workspaces, `Ambiente = terraform.workspace` em `default_tags` está correto.

  Isso habilita o **Cost Explorer agrupado por `Repositorio` e `Ambiente`** — útil no vídeo e essencial para controlar gasto.

  **2. Contrato de parâmetros SSM** — quem publica o quê:

  O prefixo separa o que é **compartilhado** entre ambientes do que é **por ambiente** — reflexo direto da [estratégia de ambientes](#estratégia-de-ambientes-leia-antes-de-escrever-hcl).

  | Parâmetro | Publicado por | Consumido por |
  |---|---|---|
  | `/oficina/shared/vpc/id` | infra-k8s | infra-db, lambda-auth |
  | `/oficina/shared/vpc/subnets_privadas` (StringList) | infra-k8s | infra-db, lambda-auth |
  | `/oficina/shared/eks/cluster_name` | infra-k8s | app (CD) |
  | `/oficina/shared/eks/node_sg_id` | infra-k8s | infra-db (ingress 5432) |
  | `/oficina/shared/lambda/sg_id` | infra-k8s | infra-db (ingress 5432), lambda-auth |
  | `/oficina/{env}/apigw/id` | infra-k8s | lambda-auth |
  | `/oficina/{env}/apigw/url` | infra-k8s | app (README, Postman) |
  | `/oficina/{env}/apigw/authorizer_id` | lambda-auth | infra-k8s (2º apply) |
  | `/oficina/{env}/alb/target_group_arn` | infra-k8s | app (TargetGroupBinding) |
  | `/oficina/{env}/db/endpoint` | infra-db | app, lambda-auth |
  | `/oficina/{env}/db/secret_arn` | infra-db | app, lambda-auth |
  | `/oficina/{env}/jwt/secret_arn` | infra-k8s | app, lambda-auth |

  Publicar (no repo produtor):

  ```hcl
  resource "aws_ssm_parameter" "vpc_id" {
    name  = "/oficina/shared/vpc/id"
    type  = "String"
    value = module.vpc.vpc_id
  }
  ```

  Consumir (no repo consumidor):

  ```hcl
  data "aws_ssm_parameter" "vpc_id" {
    name = "/oficina/shared/vpc/id"
  }
  # uso: data.aws_ssm_parameter.vpc_id.value
  ```

  > **Por que SSM e não `terraform_remote_state`?** `terraform_remote_state` exige que o repo consumidor tenha permissão de leitura no **state inteiro** do produtor — que contém segredos e todos os atributos de todos os recursos. Com SSM você expõe apenas o que decidiu expor, e a permissão IAM é granular por path (`arn:aws:ssm:*:*:parameter/oficina/prod/*`). É a mesma diferença entre exportar uma API pública e dar acesso ao banco inteiro. Registre isso como **ADR-011**.

- **Critérios de aceite:**
  - `default_tags` idêntico nos 3 repos de infra (com `Repositorio` variando).
  - Tabela de contrato SSM revisada e commitada no README de cada repo de infra.
- **Estimativa:** P
- **Depende de:** F3-0.1

---

### F3-0.4 — OIDC entre GitHub Actions e AWS (deploy sem segredos de longa duração)

- **Descrição:** Os workflows precisam autenticar na AWS. Guardar `AWS_ACCESS_KEY_ID`/`SECRET` como secret do GitHub é o caminho fácil e errado — a chave não expira, vaza em logs e não tem escopo. Com OIDC, o GitHub troca um token de identidade efêmero por credenciais AWS de curta duração, restritas ao repositório **e à branch**.

- **Como fazer:**

  No `bootstrap/` (F3-0.1), acrescente:

  ```hcl
  # Provedor OIDC do GitHub — um por conta AWS
  resource "aws_iam_openid_connect_provider" "github" {
    url             = "https://token.actions.githubusercontent.com"
    client_id_list  = ["sts.amazonaws.com"]
    thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
  }

  locals {
    org   = "ProblemaTheu"
    repos = ["oficina-app", "oficina-lambda-auth", "oficina-infra-k8s", "oficina-infra-db"]
  }

  # Uma role por repositório — princípio do menor privilégio
  resource "aws_iam_role" "github" {
    for_each = toset(local.repos)
    name     = "gha-${each.value}"

    assume_role_policy = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Effect    = "Allow"
        Principal = { Federated = aws_iam_openid_connect_provider.github.arn }
        Action    = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          # Só as branches de deploy podem assumir a role.
          # Uma feature branch NÃO consegue aplicar Terraform em produção.
          StringLike = {
            "token.actions.githubusercontent.com:sub" = [
              "repo:${local.org}/${each.value}:ref:refs/heads/main",
              "repo:${local.org}/${each.value}:environment:prod",
              # formato com IDs imutáveis — ver aviso abaixo
              "repo:${local.org}@*/${each.value}@*:ref:refs/heads/main",
              "repo:${local.org}@*/${each.value}@*:environment:prod",
            ]
          }
        }
      }]
    })
  }
  ```

  A política anexada a cada role varia: os repos de infra precisam de permissão ampla (EKS, EC2, RDS, IAM); o repo da app precisa apenas de `eks:DescribeCluster` + leitura de SSM. Comece com `AdministratorAccess` **apenas** nos repos de infra para destravar, e abra uma tarefa de *hardening* (restringir por tag) antes da entrega — e **registre honestamente** essa escolha no README, avaliadores valorizam mais a consciência do trade-off do que uma política perfeita.

  No workflow:

  ```yaml
  permissions:
    contents: read
    id-token: write   # ← sem isso o OIDC não funciona

  steps:
    - uses: aws-actions/configure-aws-credentials@v4
      with:
        role-to-assume: arn:aws:iam::706215605178:role/gha-oficina-infra-k8s
        aws-region: us-east-1
  ```

  > 🚨 **O `sub` não é o dos tutoriais.** O GitHub emite *immutable subject claims*, com os IDs numéricos do dono e do repositório embutidos — por isso as quatro entradas acima cobrem os dois formatos. Com só o formato clássico, o erro é `Not authorized to perform sts:AssumeRoleWithWebIdentity`, sem dizer qual condição falhou. O diagnóstico (ler as claims em vez de adivinhar) está em [execucao.md](execucao.md#oidc-o-sub-não-é-o-dos-tutoriais).

- **Critérios de aceite:**
  - Provedor OIDC criado; 4 roles com `sub` restrito a `main`/`homolog`.
  - Um workflow de teste roda `aws sts get-caller-identity` com sucesso e sem nenhum secret de credencial AWS no repositório.
  - Tentativa de assumir a role a partir de uma `feature/*` **falha**.
- **Estimativa:** M
- **Depende de:** F3-0.1

---

### F3-0.5 — Contas, limites de serviço e ferramentas do time

- **Descrição:** Tarefa chata e sem código, mas **bloqueante e com prazo de terceiros**. Três dos itens abaixo dependem de aprovação da AWS ou de outra empresa e levam de horas a dias — descobrir isso na Sprint 4 custa a entrega.

- **Como fazer:**

  **1. Contas e credenciais** — faça tudo na Sprint 0:

  | O quê | Onde | Prazo | Guardar |
  |---|---|---|---|
  | Conta AWS com método de pagamento | aws.amazon.com | imediato | ID da conta (12 dígitos) |
  | Conta New Relic (free tier) | newrelic.com/signup | imediato | **license key** (ingestão), **user key** (Terraform) e **account ID** — são três coisas diferentes |
  | SonarCloud: projeto para `lambda-auth` | sonarcloud.io | imediato | `SONAR_TOKEN` |
  | Saída do sandbox do SES | AWS Console → SES | **até 24 h** | — |
  | Aumento de quota de vCPU (ver abaixo) | AWS Service Quotas | **horas a 2 dias** | — |

  **2. ⚠️ O limite de vCPU que pega conta nova.** Contas AWS recentes vêm com a quota *"Running On-Demand Standard (A, C, D, H, I, M, R, T, Z) instances"* em **apenas 5 vCPUs**. Dois nós `t3.medium` já consomem 4 vCPUs. Quando o Cluster Autoscaler tentar subir o 3º nó durante o teste de carga (F3-4.8), o EC2 devolve `VcpuLimitExceeded` — o pod fica `Pending` para sempre e a demonstração de escalabilidade morre na frente da câmera.

  ```bash
  # conferir a quota atual
  aws service-quotas get-service-quota \
    --service-code ec2 --quota-code L-1216C47A --region us-east-1

  # pedir aumento para 32 vCPUs (folga confortável)
  aws service-quotas request-service-quota-increase \
    --service-code ec2 --quota-code L-1216C47A --desired-value 32 --region us-east-1
  ```

  Peça na Sprint 0. A aprovação costuma sair em algumas horas, mas pode levar dois dias.

  **3. Ferramentas locais** — todo mundo do time com as mesmas versões evita "na minha máquina funciona":

  ```bash
  brew install awscli terraform kubectl kustomize helm gh k6 jq gitleaks
  brew install golangci-lint
  go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

  aws --version && terraform version && kubectl version --client && helm version
  ```

  Fixe a versão do Terraform (`.terraform-version` ou `TF_VERSION` no workflow, já em F3-1.4): versões diferentes de Terraform escrevem *state* incompatível, e a primeira pessoa a rodar uma versão mais nova impede as outras de rodar.

  **4. Decisões ainda em aberto** que precisam de resposta do time antes da Sprint 1:

  - [x] ~~Nomes finais dos repositórios~~ — **decidido:** renomear para `oficina-app`, `oficina-lambda-auth`, `oficina-infra-k8s`, `oficina-infra-db`. Procedimento em [execucao.md](execucao.md#o-que-o-rename-tocou)
  - [ ] E-mail que receberá alertas do New Relic e do AWS Budgets (`var.email_alertas`)
  - [ ] Remetente do SES (`var.email_remetente`)
  - [ ] Dois ambientes ou só produção (impacta custo — ver [roadmap](roadmap.md#orçamento-e-janelas-de-provisionamento))
  - [ ] Quem é dono de cada repositório (ver divisão sugerida no roadmap)

  **5. Transforme este backlog em issues.** As 43 tarefas têm IDs estáveis justamente para virarem issues rastreáveis:

  ```bash
  gh issue create --repo ProblemaTheu/oficina-app \
    --title "F3-5.1 — Revisão do modelo relacional" \
    --body "Ver docs/planejamentos/fase-3/backlog.md#f3-51--revisão-do-modelo-relacional"
  ```

- **Critérios de aceite:**
  - Todas as contas criadas e chaves guardadas em local seguro (não em arquivo do repo).
  - Quota de vCPU **aprovada** (não apenas solicitada) antes da Sprint 2.
  - Saída do sandbox do SES solicitada.
  - Todo o time com as ferramentas instaladas e a mesma versão de Terraform.
  - As 5 decisões em aberto respondidas e registradas.
- **Estimativa:** P (mas com prazos de terceiros — comece no dia 1)
- **Depende de:** —

---

## E1 — Split em 4 repositórios + CI/CD

### F3-1.1 — Criar os repositórios e migrar o conteúdo

- **Descrição:** Separar o monorepo em quatro, preservando o histórico do repo atual como repo da aplicação.

- **Como fazer:**

  **1. O repo atual vira o repo da aplicação.** Renomeie no GitHub (`Settings → Repository name`) de `tech-challenge-1` para `oficina-app`. O GitHub mantém redirect do nome antigo, então nada quebra. Atualize o `module` no `go.mod`:

  ```bash
  # go.mod: github.com/problematheu/tech-challenge-1 → .../oficina-app
  go mod edit -module github.com/ProblemaTheu/oficina-app
  grep -rl "problematheu/tech-challenge-1" --include="*.go" . \
    | xargs sed -i '' 's|problematheu/tech-challenge-1|ProblemaTheu/oficina-app|g'
  go build ./... && go test ./...
  ```

  > ✅ **Decidido: renomear.** Os quatro repositórios ficam com nomenclatura consistente. O passo a passo com os 8 pontos de ajuste interno (module path, `ci.yml`, Docker Hub, Trivy, READMEs) está em [execucao.md](execucao.md#o-que-o-rename-tocou).

  **2. Extrair o Terraform para os repos de infra, preservando histórico** (opcional, mas elegante):

  ```bash
  brew install git-filter-repo

  # infra-k8s
  git clone https://github.com/ProblemaTheu/oficina-app.git oficina-infra-k8s
  cd oficina-infra-k8s
  git filter-repo --path infra/ --path k8s/ --path-rename infra/:terraform/
  git remote add origin https://github.com/ProblemaTheu/oficina-infra-k8s.git
  git push -u origin main
  ```

  Depois reorganize à mão (o filter-repo só recorta; a arquitetura nova você escreve). Se o histórico não importar, `mkdir` + copiar arquivos é perfeitamente legítimo e mais rápido.

  **3. Remover do repo da aplicação o que migrou:**

  ```bash
  cd oficina-app
  git rm -r infra/                      # Terraform sai
  # k8s/ FICA: os manifestos são artefato da aplicação e versionam junto com ela
  git commit -m "chore: mover Terraform para oficina-infra-k8s (F3-1.1)"
  ```

  **Divisão final de responsabilidade:**

  | Repo | Contém | Não contém |
  |---|---|---|
  | `oficina-app` | Go, OpenAPI, Dockerfile, `k8s/` (Kustomize), Postman, docs arquiteturais | Terraform |
  | `oficina-lambda-auth` | Go (2 handlers), `terraform/` (funções + rotas no Gateway) | VPC, cluster |
  | `oficina-infra-k8s` | `terraform/` (VPC, EKS, API Gateway, VPC Link, ALB, Secrets, New Relic Helm) | código de aplicação |
  | `oficina-infra-db` | `terraform/` (RDS, subnet group, SG, credenciais) | rede (lê da SSM) |

  > **Por que `k8s/` fica na aplicação e não no `infra-k8s`?** Porque o manifesto muda junto com o código: um campo novo no ConfigMap, uma probe nova, um `resources` ajustado. Se morar no repo de infra, toda mudança de app vira PR cruzado em dois repos. `infra-k8s` provisiona o **cluster**; `app` declara **como a aplicação roda nele**. Registre como **ADR-012**.

- **Critérios de aceite:**
  - Os 4 repositórios existem, com conteúdo coerente com a tabela acima.
  - `go build ./... && go test ./...` verde no repo da aplicação após o split.
  - Nenhum repo contém `.tfstate` ou `.env` com segredo real.
  - `soat-architecture` adicionado como colaborador nos 4 (pode ser feito ao final, mas anote agora).
- **Estimativa:** M
- **Depende de:** F3-0.1

---

### F3-1.2 — Proteção de branch nos 4 repositórios

> ⏱️ **[Corte 10](plano-10-dias.md#os-cortes)** — `homolog` fica protegida e com CI, mas **sem deploy**; só `main` implanta. As regras de proteção do enunciado seguem integralmente demonstradas.

- **Descrição:** Requisito explícito do enunciado: `main` protegida sem commits diretos, PR obrigatório para merge.

- **Como fazer:**

  Via UI: `Settings → Rules → Rulesets → New branch ruleset`. Via CLI (mais rápido e reprodutível para 4 repos):

  ```bash
  for repo in oficina-app oficina-lambda-auth oficina-infra-k8s oficina-infra-db; do
    gh api -X POST "repos/ProblemaTheu/$repo/rulesets" \
      -H "Accept: application/vnd.github+json" \
      --input - <<'JSON'
  {
    "name": "protecao-main",
    "target": "branch",
    "enforcement": "active",
    "conditions": { "ref_name": { "include": ["refs/heads/main", "refs/heads/homolog"], "exclude": [] } },
    "rules": [
      { "type": "deletion" },
      { "type": "non_fast_forward" },
      { "type": "pull_request",
        "parameters": {
          "required_approving_review_count": 1,
          "dismiss_stale_reviews_on_push": true,
          "require_code_owner_review": false,
          "require_last_push_approval": false,
          "required_review_thread_resolution": true
        }
      },
      { "type": "required_status_checks",
        "parameters": {
          "strict_required_status_checks_policy": true,
          "required_status_checks": [{ "context": "Build e testes" }, { "context": "Lint" }]
        }
      }
    ]
  }
  JSON
  done
  ```

  > ⚠️ Se o time tem **uma pessoa só**, `required_approving_review_count: 1` te bloqueia (você não aprova o próprio PR). Use `0` e mantenha os *status checks* obrigatórios — ou peça a um colega para aprovar. Não desligue a regra: no vídeo você vai querer mostrar o PR sendo barrado.
  >
  > Os nomes em `required_status_checks` são os **nomes dos jobs** (`name:` no YAML), não os do arquivo. No repo atual são exatamente `Build e testes` e `Lint`.

  **Prova para o vídeo** — grave esta tentativa falhando:

  ```bash
  git checkout main && echo "teste" >> README.md && git commit -am "direto"
  git push origin main
  # remote: error: GH013: Repository rule violations found
  ```

- **Critérios de aceite:**
  - Push direto em `main` e `homolog` é rejeitado nos 4 repos.
  - PR sem CI verde não pode ser mergeado.
  - Screenshot/gravação da rejeição guardada para o vídeo.
- **Estimativa:** P
- **Depende de:** F3-1.1

---

### F3-1.3 — Ambientes do GitHub e segredos

> ⏱️ **[Corte 10](plano-10-dias.md#os-cortes)** — crie apenas o *environment* `prod`, com reviewer obrigatório (rende a cena de aprovação no vídeo).

- **Descrição:** Criar os *environments* `homolog` e `prod` em cada repo, com os segredos e variáveis que os workflows consomem — e *required reviewers* em `prod`, para que o deploy em produção seja uma decisão consciente.

- **Como fazer:**

  ```bash
  for repo in oficina-app oficina-lambda-auth oficina-infra-k8s oficina-infra-db; do
    gh api -X PUT "repos/ProblemaTheu/$repo/environments/homolog"
    gh api -X PUT "repos/ProblemaTheu/$repo/environments/prod" \
      -f 'reviewers[][type]=User' -F "reviewers[][id]=$(gh api user -q .id)"
  done
  ```

  Variáveis e segredos por repo:

  | Nome | Tipo | Repos | Valor |
  |---|---|---|---|
  | `AWS_ROLE_ARN` | variable | todos | `arn:aws:iam::<conta>:role/gha-<repo>` |
  | `AWS_REGION` | variable | todos | `us-east-1` |
  | `TF_BUCKET` | variable | infra-* , lambda | nome do bucket do bootstrap |
  | `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` | secret | app | já existem |
  | `NEW_RELIC_LICENSE_KEY` | secret | infra-k8s, lambda | do New Relic |
  | `NEW_RELIC_ACCOUNT_ID` | variable | infra-k8s | do New Relic |
  | `SONAR_TOKEN` | secret | app | já existe |

  ```bash
  gh secret set NEW_RELIC_LICENSE_KEY --repo ProblemaTheu/oficina-infra-k8s --env prod
  gh variable set AWS_REGION --repo ProblemaTheu/oficina-app --body us-east-1
  ```

  > **Nunca** coloque a senha do banco como secret do GitHub. Ela é gerada pelo Terraform e vive no Secrets Manager; a aplicação a lê de lá. Segredo que passa por CI é segredo que vaza em log.

- **Critérios de aceite:** Environments criados nos 4 repos; `prod` com reviewer obrigatório; segredos configurados e nenhum valor sensível no código.
- **Estimativa:** P
- **Depende de:** F3-0.4, F3-1.1

---

### F3-1.4 — CI/CD dos repositórios de infraestrutura

> ⏱️ **[Cortes 10 e 14](plano-10-dias.md#os-cortes)** — `apply` só na `main` (`push: branches: [main]`): sem ambiente de homologação, um push em `homolog` aplicaria contra um ambiente inexistente e pode deixar recurso órfão cobrando. Remova também o step *Selecionar workspace*, a variável `USA_WORKSPACES` e a lógica `ref_name == 'main' && 'prod' || 'homolog'`.

- **Descrição:** Terraform em CI segue um padrão consagrado: **`plan` em PR (comentado no PR), `apply` em merge**. Nunca `apply` automático a partir de um PR — um PR malicioso destruiria a infra.

- **Como fazer:**

  `.github/workflows/terraform.yml` (idêntico em `infra-k8s`, `infra-db` e no diretório `terraform/` do `lambda-auth`):

  ```yaml
  name: Terraform

  on:
    pull_request:
      branches: [homolog, main]
    push:
      branches: [homolog, main]

  permissions:
    contents: read
    id-token: write
    pull-requests: write   # para comentar o plan no PR

  env:
    TF_VERSION: '1.15.7'   # DEVE ser igual ao seu terraform local — ver nota abaixo
    AWS_REGION: us-east-1

  jobs:
    terraform:
      name: Terraform
      runs-on: ubuntu-latest
      # Em push, o ambiente vem da branch; em PR, sempre homolog (só faz plan)
      environment: ${{ github.ref_name == 'main' && 'prod' || 'homolog' }}
      defaults:
        run:
          working-directory: terraform
      steps:
        - uses: actions/checkout@v4

        - uses: hashicorp/setup-terraform@v3
          with:
            terraform_version: ${{ env.TF_VERSION }}

        - uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: ${{ vars.AWS_ROLE_ARN }}
            aws-region: ${{ env.AWS_REGION }}

        - name: Format check
          run: terraform fmt -check -recursive

        - name: Init
          run: terraform init -input=false

        # SOMENTE no repo oficina-lambda-auth. Em infra-k8s e infra-db o
        # state é único (os ambientes saem de for_each) — remova este step lá.
        - name: Selecionar workspace
          if: vars.USA_WORKSPACES == 'true'
          run: |
            WS="${{ github.ref_name == 'main' && 'prod' || 'homolog' }}"
            terraform workspace select "$WS" || terraform workspace new "$WS"

        - name: Validate
          run: terraform validate

        # Análise estática de segurança da infra — vale ponto na avaliação
        - name: tfsec
          uses: aquasecurity/tfsec-action@v1.0.3
          with:
            working_directory: terraform
            soft_fail: true

        - name: Plan
          id: plan
          run: terraform plan -input=false -no-color -out=tfplan | tee plan.txt

        - name: Comentar o plan no PR
          if: github.event_name == 'pull_request'
          uses: actions/github-script@v7
          with:
            script: |
              const fs = require('fs');
              // GitHub limita comentários a 65536 caracteres
              let plan = fs.readFileSync('terraform/plan.txt', 'utf8');
              if (plan.length > 60000) plan = plan.slice(-60000);
              await github.rest.issues.createComment({
                issue_number: context.issue.number,
                owner: context.repo.owner,
                repo: context.repo.repo,
                body: `### 📋 Terraform Plan\n<details><summary>ver plan</summary>\n\n\`\`\`hcl\n${plan}\n\`\`\`\n</details>`
              });

        - name: Apply
          if: github.event_name == 'push'
          run: terraform apply -input=false -auto-approve tfplan

        - name: Job Summary
          if: always()
          run: |
            {
              echo "# 🏗️ Terraform — ${{ github.ref_name == 'main' && 'PRODUÇÃO' || 'homologação' }}"
              echo
              echo "Workspace: \`${{ github.ref_name == 'main' && 'prod' || 'homolog' }}\`"
              echo "Ação: \`${{ github.event_name == 'push' && 'apply' || 'plan (somente)' }}\`"
            } >> "$GITHUB_STEP_SUMMARY"
  ```

  > ⚠️ **`TF_VERSION` do CI tem que ser igual ao seu Terraform local.** O state guarda a versão que o escreveu, e uma versão mais antiga **se recusa** a lê-lo: `state snapshot was created by Terraform vX, which is newer than current vY`. Se você aplicar da sua máquina com 1.15.7 e o CI rodar 1.9.8, o pipeline quebra e só volta quando você atualizar o CI. Confira com `terraform version` e fixe o mesmo número aqui e num `.terraform-version` na raiz.

  **Dois detalhes que costumam quebrar:**

  1. `terraform plan -out=tfplan` seguido de `apply tfplan` em jobs **diferentes** não funciona sem passar o arquivo por artifact. Aqui está tudo no mesmo job, então funciona — e é mais seguro (o `apply` aplica exatamente o que foi planejado).
  2. O `environment: prod` faz o job **pausar esperando aprovação** (F3-1.3) antes de qualquer step. É exatamente o comportamento desejado e rende uma boa cena no vídeo.

- **Critérios de aceite:**
  - PR em `infra-k8s` comenta o plan automaticamente.
  - Merge em `homolog` aplica no workspace `homolog` sem intervenção.
  - Merge em `main` aguarda aprovação e então aplica em `prod`.
  - `terraform fmt -check` e `tfsec` rodam.
- **Estimativa:** M
- **Depende de:** F3-0.4, F3-1.3

---

### F3-1.5 — CD da aplicação (build → registry → EKS)


> ⏱️ **[Corte 1](plano-10-dias.md#os-cortes)** — sem `TargetGroupBinding`, o passo *Resolver placeholders* só substitui `${DB_HOST}`: tire `$TARGET_GROUP_ARN` do `envsubst` e o `target-group-binding.yaml` do overlay.**

- **Descrição:** Evoluir o `cd.yml` atual (que já publica no Docker Hub e tem o job de deploy pronto porém desabilitado) para deploy real, com ambiente por branch.

- **Como fazer:**

  ```yaml
  name: CD

  on:
    push:
      branches: [homolog, main]
    workflow_dispatch: {}

  concurrency:
    group: cd-${{ github.ref }}
    cancel-in-progress: false

  permissions:
    contents: read
    id-token: write

  jobs:
    build-and-push:
      name: Build e push da imagem
      runs-on: ubuntu-latest
      outputs:
        tag: ${{ steps.meta.outputs.tag }}
      steps:
        - uses: actions/checkout@v4
        - id: meta
          run: echo "tag=${GITHUB_SHA::7}" >> "$GITHUB_OUTPUT"
        - uses: docker/login-action@v3
          with:
            username: ${{ secrets.DOCKERHUB_USERNAME }}
            password: ${{ secrets.DOCKERHUB_TOKEN }}
        - uses: docker/setup-buildx-action@v3
        - uses: docker/build-push-action@v6
          with:
            context: .
            push: true
            platforms: linux/amd64
            tags: |
              docker.io/problematheu/oficina-api:${{ steps.meta.outputs.tag }}
              docker.io/problematheu/oficina-api:${{ github.ref_name }}
            cache-from: type=gha
            cache-to: type=gha,mode=max

    deploy:
      name: Deploy no EKS
      needs: build-and-push
      runs-on: ubuntu-latest
      environment: ${{ github.ref_name == 'main' && 'prod' || 'homolog' }}
      env:
        ENV: ${{ github.ref_name == 'main' && 'prod' || 'homolog' }}
      steps:
        - uses: actions/checkout@v4

        - uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: ${{ vars.AWS_ROLE_ARN }}
            aws-region: ${{ vars.AWS_REGION }}

        # Todos os dados de infra vêm da SSM — o repo da app não precisa
        # saber nada além dos paths. Atenção ao prefixo: o cluster é
        # COMPARTILHADO (/shared/), o resto é por ambiente (/$ENV/).
        - name: Ler parâmetros da infraestrutura
          run: |
            ler() { aws ssm get-parameter --name "$1" --query Parameter.Value --output text; }
            {
              echo "CLUSTER=$(ler /oficina/shared/eks/cluster_name)"
              echo "TARGET_GROUP_ARN=$(ler /oficina/$ENV/alb/target_group_arn)"
              echo "DB_HOST=$(ler /oficina/$ENV/db/endpoint)"
              echo "GATEWAY_URL=$(ler /oficina/$ENV/apigw/url)"
            } >> "$GITHUB_ENV"

        - name: kubeconfig
          run: aws eks update-kubeconfig --name "$CLUSTER" --region "${{ vars.AWS_REGION }}"

        # Resolve os valores dinâmicos nos manifestos (ARN do target group,
        # endpoint do RDS). Sem isto o TargetGroupBinding sobe com um ARN
        # placeholder e os pods nunca são registrados no ALB.
        - name: Resolver placeholders dos manifestos
          run: |
            for f in k8s/overlays/$ENV/*.yaml; do
              envsubst '$TARGET_GROUP_ARN $DB_HOST $GATEWAY_URL' < "$f" > "$f.tmp" && mv "$f.tmp" "$f"
            done
            ! grep -rn 'SUBSTITUIR_PELO_' k8s/overlays/$ENV || { echo "placeholder não resolvido"; exit 1; }

        - name: Aplicar manifestos
          run: |
            cd k8s/overlays/$ENV
            kustomize edit set image \
              docker.io/problematheu/oficina-api=docker.io/problematheu/oficina-api:${{ needs.build-and-push.outputs.tag }}
            kubectl apply -k .

        - name: Aguardar rollout
          run: kubectl -n oficina-$ENV rollout status deployment/api --timeout=300s

        # Se o rollout falhar, volta para a revisão anterior automaticamente
        - name: Rollback em caso de falha
          if: failure()
          run: kubectl -n oficina-$ENV rollout undo deployment/api

        # Marca o deploy no New Relic — a linha vertical no gráfico que
        # mostra "a latência subiu depois deste deploy"
        - name: Registrar deploy no New Relic
          if: success()
          run: |
            curl -sS -X POST "https://api.newrelic.com/v2/applications/${{ vars.NEW_RELIC_APP_ID }}/deployments.json" \
              -H "Api-Key: ${{ secrets.NEW_RELIC_API_KEY }}" \
              -H "Content-Type: application/json" \
              -d "{\"deployment\":{\"revision\":\"${{ needs.build-and-push.outputs.tag }}\",\"user\":\"${{ github.actor }}\"}}"
  ```

  Crie os overlays `k8s/overlays/homolog` e `k8s/overlays/prod` (o atual `overlays/aws` vira base deles), cada um com seu `namespace:` e seu patch de ConfigMap.

  > **Rate limit do Docker Hub:** contas gratuitas têm limite de pulls anônimos por IP. Nós do EKS puxando `latest` sem credencial podem tomar `toomanyrequests` justo na demo. Duas saídas: criar um `imagePullSecret` com a conta do Docker Hub (grátis, 5 min de trabalho) ou migrar para **ECR** (também no free tier, 500 MB). Recomendo o `imagePullSecret` — menos mudanças.

- **Critérios de aceite:**
  - Push em `homolog` faz deploy automático em `oficina-homolog`.
  - Push em `main` aguarda aprovação e faz deploy em `oficina-prod`.
  - Rollout falho reverte sozinho.
  - Marcador de deploy aparece no New Relic.
- **Estimativa:** M
- **Depende de:** F3-1.4, F3-3.3

---

### F3-1.6 — CI/CD do repositório da Lambda

> ⏱️ **[Cortes 10 e 14](plano-10-dias.md#os-cortes)** — sem workspaces (remova o `terraform workspace select`) e deploy só na `main`. Em `terraform/`, `local.env = "prod"` fixo.

- **Descrição:** Build do binário Go para Lambda, empacotamento e deploy via Terraform.

- **Como fazer:**

  ```yaml
  name: CI/CD Lambda

  on:
    pull_request:
      branches: [homolog, main]
    push:
      branches: [homolog, main]

  permissions:
    contents: read
    id-token: write

  jobs:
    test:
      name: Build e testes
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with: { go-version-file: go.mod, cache: true }
        - run: go vet ./...
        - run: go test -race -covermode=atomic -coverprofile=coverage.out ./...
        - uses: golangci/golangci-lint-action@v8

    deploy:
      name: Deploy
      needs: test
      if: github.event_name == 'push'
      runs-on: ubuntu-latest
      environment: ${{ github.ref_name == 'main' && 'prod' || 'homolog' }}
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with: { go-version-file: go.mod }

        # provided.al2023 + arm64 (Graviton): ~20% mais barato e mais rápido.
        # O binário PRECISA se chamar "bootstrap" nesse runtime.
        - name: Compilar as duas funções
          run: |
            for fn in auth-token auth-authorizer; do
              CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
                go build -trimpath -ldflags="-s -w" -o dist/$fn/bootstrap ./cmd/$fn
              (cd dist/$fn && zip -q ../$fn.zip bootstrap)
            done

        - uses: aws-actions/configure-aws-credentials@v4
          with:
            role-to-assume: ${{ vars.AWS_ROLE_ARN }}
            aws-region: ${{ vars.AWS_REGION }}

        - uses: hashicorp/setup-terraform@v3
        - working-directory: terraform
          run: |
            terraform init -input=false
            WS="${{ github.ref_name == 'main' && 'prod' || 'homolog' }}"
            terraform workspace select "$WS" || terraform workspace new "$WS"
            terraform apply -input=false -auto-approve

        - name: Smoke test do endpoint
          run: |
            URL=$(aws ssm get-parameter --name "/oficina/${{ github.ref_name == 'main' && 'prod' || 'homolog' }}/apigw/url" --query Parameter.Value --output text)
            code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$URL/auth/token" \
              -H 'Content-Type: application/json' -d '{"cpf":"00000000000"}')
            # CPF sintaticamente inválido deve responder 400 — prova que a função subiu
            [ "$code" = "400" ] || { echo "esperado 400, veio $code"; exit 1; }
  ```

  > **Por que o Terraform faz o deploy do código e não `aws lambda update-function-code`?** Porque o `aws_lambda_function` tem `source_code_hash`: se o zip não mudou, o Terraform não redeploya; se mudou, ele atualiza e o state reflete a realidade. Misturar as duas abordagens gera drift permanente no plan.

- **Critérios de aceite:** PR roda testes e lint; push em `homolog`/`main` compila, aplica o Terraform e o smoke test passa.
- **Estimativa:** M
- **Depende de:** F3-1.4, F3-2.1

---

### F3-1.7 — Suíte de segurança nos quatro repositórios

> ⏱️ **[Cortes 8 e 15](plano-10-dias.md#os-cortes)** — o enunciado não pede análise de segurança. Faça só o `gitleaks` (5 min) e a varredura histórica. Pule `tfsec`, `checkov` e Sonar nos repos novos; o `security.yml` do `oficina-app` continua rodando.

- **Descrição:** O repositório atual tem `.github/workflows/security.yml` com govulncheck, gosec, Trivy e Sonar — resultado da Fase 2. Ao dividir em quatro repos, **três nascem sem nenhuma verificação de segurança**, justo os que passam a manipular IAM, security groups e segredos. Além disso, dois riscos novos aparecem: segredo commitado (quatro repos, quatro chances) e má configuração de infraestrutura.

- **Como fazer:**

  **1. Todos os repos — segredos vazados.** Adicione ao workflow de CI:

  ```yaml
  - name: gitleaks
    uses: gitleaks/gitleaks-action@v2
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  ```

  Rode também **uma vez sobre o histórico completo** dos quatro repos logo após o split — o repo atual teve `.env` e `.tfstate` versionados em algum momento:

  ```bash
  gitleaks detect --source . --log-opts="--all --full-history" --report-path leaks.json
  ```

  Se aparecer credencial real, **não basta remover o arquivo**: rotacione o segredo. Um commit antigo continua acessível por SHA mesmo após force-push.

  **2. Repos com Go (`app` e `lambda-auth`)** — copie o `security.yml` existente, ajustando os paths:

  | Ferramenta | O que pega |
  |---|---|
  | `govulncheck` | CVEs nas dependências que o seu código **realmente alcança** (menos ruído que um scanner genérico) |
  | `gosec` | padrões inseguros no código (SQL concatenado, `math/rand` para segredo, TLS desabilitado) |
  | `Trivy` | vulnerabilidades na imagem final |
  | `SonarCloud` | qualidade + cobertura; crie um projeto novo para o `lambda-auth` |

  **3. Repos de Terraform** — além do `tfsec` já incluído em F3-1.4, acrescente `checkov`, que cobre um conjunto diferente de regras:

  ```yaml
  - name: checkov
    uses: bridgecrewio/checkov-action@master
    with:
      directory: terraform
      framework: terraform
      soft_fail: true    # comece observando; endureça depois de tratar os achados
      output_format: cli,sarif
      output_file_path: console,checkov.sarif

  - name: Publicar no code scanning
    uses: github/codeql-action/upload-sarif@v3
    with: { sarif_file: checkov.sarif }
  ```

  Achados que estas ferramentas **vão** apontar no nosso desenho, e a resposta esperada de cada um:

  | Achado | Resposta |
  |---|---|
  | `aws_security_group` com egress `0.0.0.0/0` (SG da Lambda) | aceitar e documentar: a Lambda precisa alcançar Secrets Manager e RDS; restringir por prefix list é a evolução |
  | RDS sem Multi-AZ | aceitar em homologação; ligar em prod se o orçamento permitir |
  | `skip_final_snapshot = true` | aceitar em homologação; falso em prod (já está) |
  | Bucket de state sem replicação | aceitar |
  | Log group sem KMS | tratar se o custo permitir |

  > `soft_fail: true` no começo é deliberado: uma suíte que quebra o build no primeiro dia é desligada no segundo. Rode observando, trate o que importa, **depois** torne bloqueante. E registre no README quais achados foram aceitos conscientemente — um avaliador respeita muito mais "sabemos deste risco e eis o porquê" do que um relatório verde por omissão.

  **4. Dockerfile "quando aplicável"** (exigência literal do enunciado): o repo `app` tem o seu; `lambda-auth` empacota em zip (`provided.al2023`), que é mais leve e rápido que imagem de container — **documente essa escolha no README** para não parecer omissão. Os repos de infra não têm código executável e portanto não têm Dockerfile; diga isso explicitamente no README de cada um.

- **Critérios de aceite:**
  - `gitleaks` rodando nos 4 repos e varredura histórica feita, com evidência de que nada vazou (ou de que o que vazou foi rotacionado).
  - govulncheck, gosec, Trivy e Sonar ativos nos 2 repos Go.
  - tfsec + checkov nos 3 repos com Terraform, publicando no code scanning.
  - README de cada repo lista os achados aceitos e o motivo.
- **Estimativa:** M
- **Depende de:** F3-1.1, F3-1.4

---

## E2 — Autenticação serverless por CPF

> Coração da fase. O enunciado pede três coisas da Function: **validar o CPF**, **consultar existência e status do cliente** e **gerar e devolver um JWT**. Fazemos isso com duas funções: `auth-token` (emite) e `auth-authorizer` (valida na borda).

### F3-2.1 — Lambda `auth-token`: validar CPF, consultar cliente e emitir JWT

- **Descrição:** Function Go que recebe um CPF, valida os dígitos verificadores, consulta a tabela `clientes` no RDS e devolve um JWT HS256 conforme o contrato de F3-0.2.

- **Como fazer:**

  **Estrutura do repositório `oficina-lambda-auth`:**

  ```
  cmd/
    auth-token/main.go          ← handler HTTP (API Gateway v2)
    auth-authorizer/main.go     ← handler de authorizer
  internal/
    cpf/cpf.go                  ← validação (sem dependência externa)
    cpf/cpf_test.go
    cliente/repository.go       ← consulta ao Postgres
    token/jwt.go                ← emissão e validação
    segredo/secrets.go          ← Secrets Manager com cache
  terraform/                    ← funções, IAM, rotas no Gateway
  ```

  **1. Validação de CPF** (`internal/cpf/cpf.go`) — algoritmo dos dígitos verificadores, sem biblioteca:

  ```go
  // Package cpf valida CPFs brasileiros pelo algoritmo de dígitos verificadores.
  package cpf

  import (
      "errors"
      "strings"
      "unicode"
  )

  var ErrInvalido = errors.New("cpf inválido")

  // Normalizar remove máscara e devolve apenas os 11 dígitos.
  func Normalizar(entrada string) string {
      var b strings.Builder
      for _, r := range entrada {
          if unicode.IsDigit(r) {
              b.WriteRune(r)
          }
      }
      return b.String()
  }

  // Validar confere tamanho, repetição e os dois dígitos verificadores.
  // Devolve o CPF normalizado quando válido.
  func Validar(entrada string) (string, error) {
      d := Normalizar(entrada)
      if len(d) != 11 {
          return "", ErrInvalido
      }
      // CPFs de dígito repetido (111.111.111-11) passam no cálculo dos
      // verificadores, mas são inválidos por convenção da Receita.
      if strings.Count(d, string(d[0])) == 11 {
          return "", ErrInvalido
      }
      if digito(d, 9) != int(d[9]-'0') || digito(d, 10) != int(d[10]-'0') {
          return "", ErrInvalido
      }
      return d, nil
  }

  // digito calcula o verificador da posição n (9 = primeiro, 10 = segundo).
  func digito(d string, n int) int {
      soma, peso := 0, n+1
      for i := 0; i < n; i++ {
          soma += int(d[i]-'0') * peso
          peso--
      }
      resto := soma * 10 % 11
      if resto == 10 {
          return 0
      }
      return resto
  }
  ```

  > O repo da aplicação já tem `internal/domain/valueobject/document.go` fazendo validação de CPF/CNPJ. **Não importe um repo do outro** — são deploys independentes. Copie a lógica (são 40 linhas) e mantenha os testes nos dois lados. Duplicação entre serviços com ciclo de vida independente é aceitável; acoplamento entre eles não é. Registre como **ADR-013**.

  **2. Segredo com cache** (`internal/segredo/secrets.go`) — buscar o segredo a cada invocação custa ~30 ms e dinheiro:

  ```go
  package segredo

  import (
      "context"
      "sync"

      "github.com/aws/aws-sdk-go-v2/config"
      "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
  )

  var (
      mu     sync.Mutex
      cache  = map[string]string{}
  )

  // Buscar devolve o valor do segredo, memorizado para toda a vida do
  // container Lambda (invocações quentes reaproveitam).
  func Buscar(ctx context.Context, nome string) (string, error) {
      mu.Lock()
      defer mu.Unlock()
      if v, ok := cache[nome]; ok {
          return v, nil
      }
      cfg, err := config.LoadDefaultConfig(ctx)
      if err != nil {
          return "", err
      }
      out, err := secretsmanager.NewFromConfig(cfg).GetSecretValue(ctx,
          &secretsmanager.GetSecretValueInput{SecretId: &nome})
      if err != nil {
          return "", err
      }
      cache[nome] = *out.SecretString
      return cache[nome], nil
  }
  ```

  **3. Handler** (`cmd/auth-token/main.go`):

  ```go
  package main

  import (
      "context"
      "database/sql"
      "encoding/json"
      "log/slog"
      "os"
      "time"

      "github.com/aws/aws-lambda-go/events"
      "github.com/aws/aws-lambda-go/lambda"
      "github.com/golang-jwt/jwt/v5"
      _ "github.com/lib/pq"

      "github.com/ProblemaTheu/oficina-lambda-auth/internal/cpf"
      "github.com/ProblemaTheu/oficina-lambda-auth/internal/segredo"
  )

  // db é criado FORA do handler: containers Lambda são reutilizados entre
  // invocações, então a conexão sobrevive e evita o custo de handshake.
  var db *sql.DB

  func init() {
      slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
      var err error
      db, err = sql.Open("postgres", os.Getenv("DB_DSN"))
      if err != nil {
          slog.Error("falha ao abrir conexão", "error", err)
          os.Exit(1)
      }
      // Lambda é single-thread por invocação: mais de 1 conexão é desperdício
      // e multiplica o consumo de conexões do RDS pelo número de containers.
      db.SetMaxOpenConns(1)
      db.SetMaxIdleConns(1)
      db.SetConnMaxIdleTime(5 * time.Minute)
  }

  type requisicao struct {
      CPF string `json:"cpf"`
  }

  type resposta struct {
      AccessToken string `json:"access_token"`
      TokenType   string `json:"token_type"`
      ExpiresIn   int    `json:"expires_in"`
  }

  const validade = time.Hour

  func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
      var body requisicao
      if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
          return erro(400, "payload_invalido", "corpo da requisição não é um JSON válido")
      }

      // 1 — validar o CPF
      documento, err := cpf.Validar(body.CPF)
      if err != nil {
          slog.WarnContext(ctx, "cpf inválido")   // NUNCA logar o CPF: é dado pessoal (LGPD)
          return erro(400, "cpf_invalido", "CPF informado é inválido")
      }

      // 2 — consultar existência e status do cliente
      var id, nome, status string
      err = db.QueryRowContext(ctx,
          `SELECT id::text, nome, status FROM clientes WHERE cpf_cnpj_digitos = $1`,
          documento,
      ).Scan(&id, &nome, &status)

      switch {
      case err == sql.ErrNoRows:
          return erro(404, "cliente_nao_encontrado", "não há cliente cadastrado com este CPF")
      case err != nil:
          slog.ErrorContext(ctx, "falha ao consultar cliente", "error", err)
          return erro(500, "erro_interno", "não foi possível processar a autenticação")
      case status != "ativo":
          slog.WarnContext(ctx, "cliente inativo", "cliente_id", id, "status", status)
          return erro(403, "cliente_inativo", "cadastro do cliente não está ativo")
      }

      // 3 — gerar e devolver o token
      segredoJWT, err := segredo.Buscar(ctx, os.Getenv("JWT_SECRET_NAME"))
      if err != nil {
          slog.ErrorContext(ctx, "falha ao obter segredo", "error", err)
          return erro(500, "erro_interno", "não foi possível processar a autenticação")
      }

      agora := time.Now()
      token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
          "sub":  id,
          "cpf":  documento,
          "nome": nome,
          "tipo": "cliente",
          "iss":  "oficina-auth-lambda",
          "aud":  "oficina-api",
          "iat":  agora.Unix(),
          "exp":  agora.Add(validade).Unix(),
      })
      assinado, err := token.SignedString([]byte(segredoJWT))
      if err != nil {
          slog.ErrorContext(ctx, "falha ao assinar token", "error", err)
          return erro(500, "erro_interno", "não foi possível processar a autenticação")
      }

      slog.InfoContext(ctx, "token emitido", "cliente_id", id)
      return json200(resposta{AccessToken: assinado, TokenType: "Bearer", ExpiresIn: int(validade.Seconds())})
  }

  func main() { lambda.Start(handler) }
  ```

  (`erro` e `json200` são helpers triviais que serializam o corpo e setam `Content-Type`.)

  **4. Testes** — a validação de CPF é lógica pura e merece tabela de casos:

  ```go
  func TestValidar(t *testing.T) {
      casos := []struct{ nome, entrada, esperado string; erro bool }{
          {"válido com máscara", "529.982.247-25", "52998224725", false},
          {"válido sem máscara", "52998224725", "52998224725", false},
          {"dígito verificador errado", "52998224726", "", true},
          {"todos iguais", "11111111111", "", true},
          {"curto demais", "1234567890", "", true},
          {"com letras", "5299822472a", "", true},
          {"vazio", "", "", true},
      }
      for _, c := range casos {
          t.Run(c.nome, func(t *testing.T) { /* ... */ })
      }
  }
  ```

  > **Ponto de atenção — Lambda + RDS:** cada container Lambda quente mantém uma conexão. Com 100 execuções concorrentes você abre 100 conexões, e uma `db.t3.micro` suporta ~87. A solução de produção é o **RDS Proxy** (pooling gerenciado, ~US$ 0,015/h por vCPU). Para o volume desta demonstração a conexão direta basta; **documente o trade-off na RFC-003** — a banca vai perguntar.

- **Critérios de aceite:**
  - CPF válido de cliente ativo → 200 com JWT decodificável e claims conforme F3-0.2.
  - CPF sintaticamente inválido → 400 `cpf_invalido`.
  - CPF válido sem cadastro → 404 `cliente_nao_encontrado`.
  - Cliente com `status <> 'ativo'` → 403 `cliente_inativo`.
  - Nenhum log contém o CPF completo.
  - Cobertura do pacote `cpf` ≥ 95%.
- **Estimativa:** G
- **Depende de:** F3-0.2, F3-5.1 (coluna `status` e `cpf_cnpj_digitos`)

---

### F3-2.2 — Lambda `auth-authorizer`: validação do token na borda

- **Descrição:** O API Gateway HTTP API tem um JWT Authorizer nativo, mas ele **só aceita emissores OIDC com JWKS público** (Cognito, Auth0…) — não valida HS256. Como a decisão foi HS256 com segredo compartilhado, a validação na borda precisa de um authorizer customizado.

- **Como fazer:**

  ```go
  package main

  import (
      "context"
      "log/slog"
      "os"
      "strings"

      "github.com/aws/aws-lambda-go/events"
      "github.com/aws/aws-lambda-go/lambda"
      "github.com/golang-jwt/jwt/v5"

      "github.com/ProblemaTheu/oficina-lambda-auth/internal/segredo"
  )

  // Formato simples: responde apenas allow/deny + contexto.
  // Requer "AuthorizerPayloadFormatVersion 2.0" e EnableSimpleResponses no Terraform.
  type resposta struct {
      IsAuthorized bool              `json:"isAuthorized"`
      Context      map[string]string `json:"context,omitempty"`
  }

  var negado = resposta{IsAuthorized: false}

  func handler(ctx context.Context, ev events.APIGatewayV2CustomAuthorizerV2Request) (resposta, error) {
      cabecalho := ev.Headers["authorization"] // headers chegam sempre em minúsculas
      if !strings.HasPrefix(cabecalho, "Bearer ") {
          return negado, nil
      }

      chave, err := segredo.Buscar(ctx, os.Getenv("JWT_SECRET_NAME"))
      if err != nil {
          slog.ErrorContext(ctx, "falha ao obter segredo", "error", err)
          return negado, nil
      }

      token, err := jwt.Parse(
          strings.TrimPrefix(cabecalho, "Bearer "),
          func(t *jwt.Token) (any, error) { return []byte(chave), nil },
          jwt.WithValidMethods([]string{"HS256"}), // impede o ataque alg=none
          jwt.WithAudience("oficina-api"),
          jwt.WithIssuedAt(),
      )
      if err != nil || !token.Valid {
          slog.WarnContext(ctx, "token rejeitado", "error", err)
          return negado, nil
      }

      claims, ok := token.Claims.(jwt.MapClaims)
      if !ok {
          return negado, nil
      }

      // O contexto é repassado à integração como header/variável — a
      // aplicação pode confiar nele sem reparsear o token.
      ctxOut := map[string]string{}
      for _, k := range []string{"sub", "tipo", "cpf", "papel"} {
          if v, ok := claims[k].(string); ok {
              ctxOut[k] = v
          }
      }
      return resposta{IsAuthorized: true, Context: ctxOut}, nil
  }

  func main() { lambda.Start(handler) }
  ```

  **Cache do authorizer:** configure `authorizer_result_ttl_in_seconds = 300` no Terraform, com `identity_sources = ["$request.header.Authorization"]`. Assim o mesmo token não invoca a Lambda a cada requisição — cai de ~2000 invocações/min para ~20 num teste de carga. **Cuidado:** com cache, revogar um token leva até 5 min para surtir efeito. Para esta aplicação é aceitável; documente.

  > **Vale a pena ter o authorizer se a aplicação já valida o JWT?** Sim, e por dois motivos concretos: (1) requisição não autorizada nem chega a consumir CPU do cluster — proteção contra abuso e economia de escala; (2) o enunciado pede explicitamente "proteger rotas sensíveis da aplicação" **no gateway**. A validação na aplicação permanece como defesa em profundidade, porque o ALB é alcançável de dentro da VPC.

- **Critérios de aceite:**
  - Requisição sem `Authorization` a uma rota protegida → 401 devolvido pelo **gateway**, sem log de acesso no pod.
  - Token expirado → 401.
  - Token assinado com outro segredo → 401.
  - Token válido → 200, e o log do pod mostra o `sub` recebido via contexto do authorizer.
- **Estimativa:** M
- **Depende de:** F3-2.1

---

### F3-2.3 — Terraform das funções e das rotas no API Gateway

- **Descrição:** Provisionar as duas Lambdas, seus papéis IAM, a configuração de VPC (para alcançar o RDS privado) e registrá-las como rotas/authorizer no API Gateway criado pelo `infra-k8s`.

- **Como fazer:**

  ```hcl
  # terraform/main.tf (repo oficina-lambda-auth)
  locals {
    env    = terraform.workspace
    prefix = "oficina-${local.env}"
  }

  # ── Dados vindos dos outros repos, via SSM ────────────────────────────────
  # compartilhados entre ambientes
  data "aws_ssm_parameter" "subnets"      { name = "/oficina/shared/vpc/subnets_privadas" }
  data "aws_ssm_parameter" "lambda_sg"    { name = "/oficina/shared/lambda/sg_id" }
  # por ambiente (local.env = workspace; este repo USA workspaces)
  data "aws_ssm_parameter" "apigw_id"     { name = "/oficina/${local.env}/apigw/id" }
  data "aws_ssm_parameter" "db_endpoint"  { name = "/oficina/${local.env}/db/endpoint" }
  data "aws_ssm_parameter" "db_secret"    { name = "/oficina/${local.env}/db/secret_arn" }
  data "aws_ssm_parameter" "jwt_secret"   { name = "/oficina/${local.env}/jwt/secret_arn" }

  data "aws_secretsmanager_secret_version" "db" {
    secret_id = data.aws_ssm_parameter.db_secret.value
  }

  # ── IAM ───────────────────────────────────────────────────────────────────
  resource "aws_iam_role" "lambda" {
    name = "${local.prefix}-lambda-auth"
    assume_role_policy = jsonencode({
      Version   = "2012-10-17"
      Statement = [{
        Effect    = "Allow"
        Action    = "sts:AssumeRole"
        Principal = { Service = "lambda.amazonaws.com" }
      }]
    })
  }

  # Necessário para a Lambda criar ENIs nas subnets privadas
  resource "aws_iam_role_policy_attachment" "vpc" {
    role       = aws_iam_role.lambda.name
    policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
  }

  resource "aws_iam_role_policy" "segredos" {
    role = aws_iam_role.lambda.id
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = [data.aws_ssm_parameter.jwt_secret.value, data.aws_ssm_parameter.db_secret.value]
      }]
    })
  }

  # ── Funções ───────────────────────────────────────────────────────────────
  locals {
    funcoes = {
      "auth-token"      = { timeout = 10, memoria = 256 }
      "auth-authorizer" = { timeout = 5,  memoria = 128 }
    }
  }

  resource "aws_lambda_function" "fn" {
    for_each = local.funcoes

    function_name = "${local.prefix}-${each.key}"
    role          = aws_iam_role.lambda.arn
    filename      = "${path.module}/../dist/${each.key}.zip"
    # Redeploya somente quando o binário muda de fato
    source_code_hash = filebase64sha256("${path.module}/../dist/${each.key}.zip")

    runtime       = "provided.al2023"   # runtime custom para Go
    handler       = "bootstrap"
    architectures = ["arm64"]           # Graviton: mais barato

    timeout     = each.value.timeout
    memory_size = each.value.memoria

    # A Lambda precisa estar na VPC para alcançar o RDS privado.
    # Custo: cold start um pouco maior (criação de ENI, hoje bem mais rápido
    # que antigamente graças ao Hyperplane).
    vpc_config {
      subnet_ids         = split(",", data.aws_ssm_parameter.subnets.value)
      security_group_ids = [data.aws_ssm_parameter.lambda_sg.value]
    }

    environment {
      variables = {
        JWT_SECRET_NAME = data.aws_ssm_parameter.jwt_secret.value
        DB_DSN = format(
          "postgres://%s:%s@%s/%s?sslmode=require",
          jsondecode(data.aws_secretsmanager_secret_version.db.secret_string)["username"],
          jsondecode(data.aws_secretsmanager_secret_version.db.secret_string)["password"],
          data.aws_ssm_parameter.db_endpoint.value,
          "oficina_${local.env}",
        )
        # Instrumentação New Relic (F3-4.4)
        NEW_RELIC_LAMBDA_HANDLER = "bootstrap"
        NEW_RELIC_ACCOUNT_ID     = var.new_relic_account_id
      }
    }

    tracing_config { mode = "Active" }   # X-Ray, útil para o trace ponta a ponta
  }

  resource "aws_cloudwatch_log_group" "fn" {
    for_each          = local.funcoes
    name              = "/aws/lambda/${local.prefix}-${each.key}"
    retention_in_days = 14   # sem isso, log fica para sempre e vira custo
  }

  # ── Rota pública POST /auth/token ─────────────────────────────────────────
  resource "aws_apigatewayv2_integration" "token" {
    api_id                 = data.aws_ssm_parameter.apigw_id.value
    integration_type       = "AWS_PROXY"
    integration_uri        = aws_lambda_function.fn["auth-token"].invoke_arn
    payload_format_version = "2.0"
  }

  resource "aws_apigatewayv2_route" "token" {
    api_id    = data.aws_ssm_parameter.apigw_id.value
    route_key = "POST /auth/token"
    target    = "integrations/${aws_apigatewayv2_integration.token.id}"
    # sem authorization_type → rota pública, é onde o token nasce
  }

  resource "aws_lambda_permission" "token" {
    statement_id  = "AllowAPIGatewayInvoke"
    action        = "lambda:InvokeFunction"
    function_name = aws_lambda_function.fn["auth-token"].function_name
    principal     = "apigateway.amazonaws.com"
    source_arn    = "${data.aws_apigatewayv2_api.principal.execution_arn}/*/*"
  }

  data "aws_apigatewayv2_api" "principal" {
    api_id = data.aws_ssm_parameter.apigw_id.value
  }

  # ── Authorizer usado pelas rotas protegidas ───────────────────────────────
  resource "aws_apigatewayv2_authorizer" "jwt" {
    api_id                            = data.aws_ssm_parameter.apigw_id.value
    name                              = "${local.prefix}-jwt"
    authorizer_type                   = "REQUEST"
    authorizer_uri                    = aws_lambda_function.fn["auth-authorizer"].invoke_arn
    authorizer_payload_format_version = "2.0"
    enable_simple_responses           = true
    identity_sources                  = ["$request.header.Authorization"]
    authorizer_result_ttl_in_seconds  = 300
  }

  resource "aws_lambda_permission" "authorizer" {
    statement_id  = "AllowAPIGatewayInvokeAuthorizer"
    action        = "lambda:InvokeFunction"
    function_name = aws_lambda_function.fn["auth-authorizer"].function_name
    principal     = "apigateway.amazonaws.com"
    source_arn    = "${data.aws_apigatewayv2_api.principal.execution_arn}/authorizers/${aws_apigatewayv2_authorizer.jwt.id}"
  }

  # Publica o ID do authorizer para o infra-k8s amarrar nas rotas /v1/*
  resource "aws_ssm_parameter" "authorizer_id" {
    name  = "/oficina/${local.env}/apigw/authorizer_id"
    type  = "String"
    value = aws_apigatewayv2_authorizer.jwt.id
  }
  ```

  > **Ordem de aplicação:** `infra-k8s` cria o Gateway (sem as rotas `/v1/*` protegidas) → `lambda-auth` cria o authorizer e publica o ID → `infra-k8s` roda de novo e amarra o authorizer nas rotas `/v1/*`. Dois `apply` no `infra-k8s` na primeira vez. É a única costura assimétrica da arquitetura; documente no README do `infra-k8s` para ninguém tropeçar.

- **Critérios de aceite:** `terraform apply` cria as 2 funções, as permissões, a rota e o authorizer; `curl -X POST $URL/auth/token` responde.
- **Estimativa:** G
- **Depende de:** F3-2.2, F3-3.4

---

### F3-2.4 — Adequar a aplicação ao novo contrato de token

- **Descrição:** A aplicação precisa (a) aceitar tokens emitidos pela Lambda, (b) distinguir `cliente` de `usuario` e (c) emitir seus próprios tokens já no formato novo.

- **Como fazer:**

  **1. Emitir com os claims novos** — em `internal/application/usecase/auth_usecase.go`, dentro de `Login`:

  ```go
  claims := jwt.MapClaims{
      "sub":   usuario.ID.String(),
      "email": usuario.Email,
      "nome":  usuario.Nome,
      "papel": nomePapel,           // NOVO — requer buscar o papel no login
      "tipo":  "usuario",           // NOVO
      "iss":   "oficina-api",       // NOVO
      "aud":   "oficina-api",       // NOVO
      "iat":   now.Unix(),
      "exp":   now.Add(jwtExpiresIn).Unix(),
  }
  ```

  **2. Validar `aud` e expor o tipo** — em `internal/infra/http/middleware/jwt.go`:

  ```go
  token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
      if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
          return nil, jwt.ErrSignatureInvalid
      }
      return secret, nil
  },
      jwt.WithValidMethods([]string{"HS256"}),
      jwt.WithAudience("oficina-api"),   // NOVO — aceita ambos os emissores
  )
  ```

  **3. Middleware de autorização por tipo** — novo arquivo `internal/infra/http/middleware/autorizacao.go`:

  ```go
  // ExigirTipo bloqueia a requisição quando o claim "tipo" do token não está
  // entre os permitidos. Complementa o JWT: autenticar é saber quem é;
  // autorizar é saber o que pode.
  func ExigirTipo(permitidos ...string) func(http.Handler) http.Handler {
      permitido := make(map[string]bool, len(permitidos))
      for _, t := range permitidos {
          permitido[t] = true
      }
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              claims, ok := r.Context().Value(ClaimsContextKey).(jwt.MapClaims)
              if !ok {
                  escreverErro(w, http.StatusUnauthorized, "nao_autorizado", "token ausente")
                  return
              }
              tipo, _ := claims["tipo"].(string)
              if !permitido[tipo] {
                  escreverErro(w, http.StatusForbidden, "acesso_negado",
                      "este token não tem permissão para esta operação")
                  return
              }
              next.ServeHTTP(w, r)
          })
      }
  }
  ```

  E no `cmd/api/main.go`, separe os grupos de rota:

  ```go
  r.Group(func(r chi.Router) {
      r.Use(apimiddleware.Correlacao())      // F3-4.1
      r.Use(apimiddleware.AssinaturaWebhook())
      r.Use(apimiddleware.JWT())
      r.Use(apimiddleware.ExigirTipo("usuario", "cliente"))
      api.HandlerFromMuxWithBaseURL(strictHandler, r, "/v1")
  })
  ```

  > **Granularidade por rota:** o `oapi-codegen` monta todas as rotas de uma vez, então aplicar `ExigirTipo("usuario")` só em algumas exige ou (a) um segundo `r.Group` com `HandlerFromMuxWithBaseURL` parcial, ou (b) a verificação dentro dos handlers que precisam. **Recomendo (b)**: um helper `exigirUsuario(ctx)` chamado no início dos handlers de escrita. Menos mágica, mais explícito, e o teste fica trivial.

  **4. Filtrar OS por cliente** — quando `tipo == "cliente"`, `GET /v1/work-orders` deve devolver apenas as OS cujo `cliente_id == sub`. Isso é o que torna a rota "protegida por CPF" de verdade e não só autenticada.

  **5. Regenerar o contrato** — adicione ao `docs/openapi.yaml` o novo `securityScheme` e a rota do Gateway como `servers`, e rode:

  ```bash
  go generate ./internal/infra/http/api/...
  ```

- **Critérios de aceite:**
  - Token da Lambda (`tipo=cliente`) é aceito pela aplicação e acessa `GET /v1/work-orders`, vendo **apenas** as próprias OS.
  - Token de cliente em `POST /v1/work-orders` → 403.
  - Token de funcionário continua funcionando em tudo que funcionava antes.
  - `aud` inválido → 401.
  - Testes cobrindo cada linha da tabela de F3-0.2.
- **Estimativa:** G
- **Depende de:** F3-0.2, F3-2.1

---

### F3-2.5 — Segredo compartilhado e remoção do fallback inseguro


> ⏱️ **[Corte 4](plano-10-dias.md#os-cortes)** — sem External Secrets Operator — o Terraform cria o `Secret` do Kubernetes direto:
> ```hcl
> resource "kubernetes_secret" "oficina" {
>   for_each = local.ambientes
>   metadata { name = "oficina-secrets"; namespace = "oficina-${each.key}" }
>   data = {
>     JWT_SECRET  = random_password.jwt[each.key].result
>     DB_PASSWORD = random_password.db[each.key].result
>   }
> }
> ```
> O `envFrom` do Deployment não muda. O que se perde é a rotação automática — rotacionar passa a exigir um novo `apply`.**

- **Descrição:** `jwtSecret()` cai num literal `"changeme-insecure-default-secret"` quando `JWT_SECRET` não está setado — em dois arquivos. Em produção isso significa que qualquer pessoa com acesso ao repositório forja tokens válidos. Além disso, aplicação e Lambdas precisam do **mesmo** segredo, vindo do Secrets Manager.

- **Como fazer:**

  **1. Gerar e guardar o segredo** (no `infra-k8s`):

  ```hcl
  resource "random_password" "jwt" {
    for_each = local.ambientes
    length   = 64
    special  = false   # evita problemas de escape em env vars
  }

  # Um segredo POR AMBIENTE: vazar o de homologação não compromete produção
  resource "aws_secretsmanager_secret" "jwt" {
    for_each                = local.ambientes
    name                    = "oficina/${each.key}/jwt-secret"
    recovery_window_in_days = 0   # permite recriar no mesmo nome durante o desafio
  }

  resource "aws_secretsmanager_secret_version" "jwt" {
    for_each      = local.ambientes
    secret_id     = aws_secretsmanager_secret.jwt[each.key].id
    secret_string = random_password.jwt[each.key].result
  }

  resource "aws_ssm_parameter" "jwt_secret_arn" {
    for_each = local.ambientes
    name     = "/oficina/${each.key}/jwt/secret_arn"
    type     = "String"
    value    = aws_secretsmanager_secret.jwt[each.key].arn
  }
  ```

  **2. Levar o segredo para dentro do cluster** com o **External Secrets Operator** — assim o Secret do Kubernetes é gerado a partir do Secrets Manager, sem ninguém copiar valor à mão:

  ```yaml
  # k8s/base/external-secret.yaml (repo da aplicação)
  apiVersion: external-secrets.io/v1beta1
  kind: ExternalSecret
  metadata:
    name: oficina-secrets
  spec:
    refreshInterval: 1h
    secretStoreRef:
      name: aws-secrets
      kind: ClusterSecretStore
    target:
      name: oficina-secrets     # nome que o Deployment já consome via envFrom
      creationPolicy: Owner
    data:
      - secretKey: JWT_SECRET
        remoteRef: { key: oficina/prod/jwt-secret }
      - secretKey: DB_USER
        remoteRef: { key: oficina/prod/db, property: username }
      - secretKey: DB_PASSWORD
        remoteRef: { key: oficina/prod/db, property: password }
  ```

  O operator é instalado pelo `infra-k8s` via Helm, com IRSA dando permissão de `secretsmanager:GetSecretValue`.

  **3. Falhar rápido quando o segredo faltar** — substitua as duas cópias de `jwtSecret()`:

  ```go
  // jwtSecret devolve o segredo de assinatura. A ausência da variável é um
  // erro de configuração fatal: subir com um segredo padrão conhecido
  // permitiria forjar tokens.
  func jwtSecret() []byte {
      s := os.Getenv("JWT_SECRET")
      if len(s) < 32 {
          panic("JWT_SECRET ausente ou com menos de 32 caracteres")
      }
      return []byte(s)
  }
  ```

  Extraia para um único pacote (`internal/infra/config`) para não ter duas cópias divergindo. Nos testes, use `t.Setenv("JWT_SECRET", strings.Repeat("x", 32))`.

  > `panic` no boot é intencional e correto aqui: o pod entra em `CrashLoopBackOff`, a readiness nunca passa, o rollout falha e o deploy anterior continua servindo. Falhar barulhentamente é melhor do que subir inseguro em silêncio.

- **Critérios de aceite:**
  - Nenhuma ocorrência de `changeme` no código (`grep -r changeme .` vazio).
  - Aplicação sem `JWT_SECRET` não sobe.
  - Token emitido pela Lambda é aceito pela aplicação — prova de que ambos leem o mesmo segredo.
  - Rotacionar o segredo no Secrets Manager reflete no cluster em ≤ 1 h (ou imediatamente com `kubectl annotate`).
- **Estimativa:** M
- **Depende de:** F3-3.3

---

### F3-2.6 — Atualizar OpenAPI, Postman e a documentação de autenticação

- **Descrição:** O contrato precisa refletir o novo fluxo, e a coleção Postman é entregável explícito ("Link para o Swagger/Postman das APIs" em cada README).

- **Como fazer:**
  - Em `docs/openapi.yaml`: adicionar `servers` com a URL do API Gateway de cada ambiente; documentar `POST /auth/token` (mesmo estando fora do `/v1`, como rota do gateway); descrever no `securitySchemes` que o Bearer aceita dois emissores e explicar o claim `tipo`.
  - Na coleção Postman: nova pasta **Auth por CPF** com a requisição de token e um *test script* que salva o token na variável de coleção:

    ```javascript
    // aba Tests da requisição POST /auth/token
    const body = pm.response.json();
    pm.test("token emitido", () => pm.expect(body.access_token).to.be.a("string"));
    pm.collectionVariables.set("clienteToken", body.access_token);
    ```

  - Adicionar variáveis `{{gatewayUrl}}`, `{{cpf}}` e `{{clienteToken}}`; apontar as requisições existentes para `{{gatewayUrl}}` em vez de `localhost:8080`.
  - Regenerar: `go generate ./internal/infra/http/api/...`.

- **Critérios de aceite:** OpenAPI válido e regenerado sem drift; coleção Postman executa o fluxo completo (token por CPF → consulta de OS) contra o Gateway; README linkando ambos.
- **Estimativa:** M
- **Depende de:** F3-2.4

---

## E3 — Infraestrutura AWS com Terraform

> Base de partida: `infra/environments/aws/` da Fase 2 já tem VPC + EKS + RDS funcionais. Este épico **reaproveita** esse código, distribui entre dois repositórios e acrescenta o que falta: API Gateway, VPC Link, ALB interno, Secrets Manager, addons do cluster e os parâmetros SSM.

### Estratégia de ambientes (leia antes de escrever HCL)

> ⏱️ **[Cortes 10 e 14](plano-10-dias.md#os-cortes)** — há apenas o ambiente `prod` e **sem `for_each`**: escreva os recursos direto, com `prod` no nome. Todo o HCL a seguir usa `for_each` porque descreve o desenho de dois ambientes — traduza `module.rds[each.key]` para `module.rds` e `${each.key}` para `prod`.
>
> Leia a seção mesmo assim: o raciocínio de compartilhado × por-ambiente é o conteúdo da **ADR-016**, e a banca avalia a decisão, não o `for_each`.

O enunciado exige "deploy automático das branches de homologação e produção". A leitura ingênua — um workspace do Terraform por ambiente — **duplica a infraestrutura inteira**: duas VPCs, dois clusters EKS, dois NAT Gateways. São ~US$ 195/mês virando ~US$ 390/mês, sem nenhum ganho na avaliação.

O que separamos e o que compartilhamos:

| Recurso | Compartilhado | Por ambiente | Por quê |
|---|---|---|---|
| VPC, subnets, NAT | ✅ | | +US$ 32/mês por NAT duplicado; isolamento de rede não é requisito |
| Cluster EKS, nós, addons | ✅ | | +US$ 73/mês só de control plane |
| ALB interno, VPC Link | ✅ | | um ALB roteia para os dois ambientes por listener |
| ESO, New Relic agent, autoscaler | ✅ | | são do cluster, não da aplicação |
| **Namespace** | | ✅ | isolamento de workload, quotas e RBAC |
| **Target group + listener** | | ✅ | cada namespace tem seus próprios pods |
| **API Gateway** | | ✅ | URLs distintas, throttling e authorizer independentes |
| **Instância RDS** | | ✅ | dado de homologação não pode encostar em produção |
| **Segredo JWT** | | ✅ | vazar o de homolog não compromete produção |

**Como isso vira código:** `infra-k8s` e `infra-db` usam **um state único, sem workspaces**, com os recursos por ambiente saindo de `for_each = local.ambientes`. Ler `each.key` é muito mais explícito do que depender de qual workspace estava selecionado quando o CI rodou — a causa mais comum de "apliquei em produção sem querer".

```hcl
# padrão usado em todo infra-k8s e infra-db
locals { ambientes = toset(["homolog", "prod"]) }
```

**E o "deploy automático por branch"?** Ele é demonstrado onde de fato importa e onde o enunciado o cobra: em `oficina-app` e `oficina-lambda-auth`, cujo **código** muda a cada PR — merge em `homolog` implanta no namespace/stage de homologação, merge em `main` implanta em produção. Os repositórios de infra não fazem deploy de código: fazem `plan` em PR e `apply` no merge da `main`, que é o padrão correto para IaC. Registre isso na **RFC-004** — é uma decisão consciente, não um atalho, e explicá-la bem vale mais do que duplicar a fatura.

> ⚠️ **Duas instâncias RDS pequenas em vez de uma com dois databases.** Seria mais barato ter uma instância e dois bancos, mas criar o segundo database exige um `CREATE DATABASE` contra um RDS que está em **subnet privada** — e o runner do GitHub Actions está fora da VPC. As saídas seriam um runner self-hosted ou um Job no cluster; nenhuma vale a economia de US$ 12/mês. Duas `db.t4g.micro` resolvem com Terraform puro.

### F3-3.1 — Repositório `oficina-infra-db`: RDS PostgreSQL gerenciado

- **Descrição:** Banco gerenciado em subnets privadas, com credenciais no Secrets Manager e acesso liberado apenas para os nós do EKS e para as Lambdas.

- **Como fazer:**

  ```hcl
  # terraform/main.tf — state único; o ambiente é uma dimensão via for_each
  locals { ambientes = toset(["homolog", "prod"]) }

  data "aws_ssm_parameter" "vpc_id"    { name = "/oficina/shared/vpc/id" }
  data "aws_ssm_parameter" "subnets"   { name = "/oficina/shared/vpc/subnets_privadas" }
  data "aws_ssm_parameter" "node_sg"   { name = "/oficina/shared/eks/node_sg_id" }
  data "aws_ssm_parameter" "lambda_sg" { name = "/oficina/shared/lambda/sg_id" }

  resource "random_password" "db" {
    for_each = local.ambientes
    length   = 32
    special  = false  # a senha vai para uma DSN de URL; especiais exigem escape
  }

  # SG compartilhado: as regras de acesso são idênticas nos dois ambientes
  module "sg" {
    source  = "terraform-aws-modules/security-group/aws"
    version = "~> 5.1"

    name   = "oficina-rds"
    vpc_id = data.aws_ssm_parameter.vpc_id.value

    # 5432 aberto SOMENTE para quem precisa. Nada de 0.0.0.0/0.
    ingress_with_source_security_group_id = [
      { from_port = 5432, to_port = 5432, protocol = "tcp",
        source_security_group_id = data.aws_ssm_parameter.node_sg.value,
        description = "EKS worker nodes" },
      { from_port = 5432, to_port = 5432, protocol = "tcp",
        source_security_group_id = data.aws_ssm_parameter.lambda_sg.value,
        description = "Lambda auth" },
    ]
  }

  module "rds" {
    source   = "terraform-aws-modules/rds/aws"
    version  = "~> 6.7"
    for_each = local.ambientes

    identifier = "oficina-${each.key}"

    engine               = "postgres"
    engine_version       = "15.7"
    family               = "postgres15"
    major_engine_version = "15"
    instance_class       = each.key == "prod" ? "db.t4g.small" : "db.t4g.micro"

    allocated_storage     = 20
    max_allocated_storage = 100      # autoscaling de storage — evita disco cheio na demo
    storage_encrypted     = true

    db_name  = "oficina_${each.key}"
    username = "oficina"
    password = random_password.db[each.key].result
    port     = 5432

    manage_master_user_password = false

    multi_az               = false   # ligue em prod se o orçamento permitir
    publicly_accessible    = false   # NUNCA true
    create_db_subnet_group = true
    subnet_ids             = split(",", data.aws_ssm_parameter.subnets.value)
    vpc_security_group_ids = [module.sg.security_group_id]

    backup_retention_period = each.key == "prod" ? 7 : 1
    deletion_protection     = each.key == "prod"
    skip_final_snapshot     = each.key != "prod"

    # Observabilidade nativa — complementa o New Relic
    performance_insights_enabled    = true
    performance_insights_retention_period = 7
    enabled_cloudwatch_logs_exports = ["postgresql"]

    # Parâmetro útil para o dashboard: registra queries lentas (> 1 s)
    parameters = [
      { name = "log_min_duration_statement", value = "1000" }
    ]
  }

  # ── Credenciais no Secrets Manager ────────────────────────────────────────
  resource "aws_secretsmanager_secret" "db" {
    for_each                = local.ambientes
    name                    = "oficina/${each.key}/db"
    recovery_window_in_days = 0
  }

  resource "aws_secretsmanager_secret_version" "db" {
    for_each  = local.ambientes
    secret_id = aws_secretsmanager_secret.db[each.key].id
    secret_string = jsonencode({
      username = module.rds[each.key].db_instance_username
      password = random_password.db[each.key].result
      host     = module.rds[each.key].db_instance_address
      port     = 5432
      dbname   = "oficina_${each.key}"
    })
  }

  # ── Contrato SSM ──────────────────────────────────────────────────────────
  resource "aws_ssm_parameter" "endpoint" {
    for_each = local.ambientes
    name     = "/oficina/${each.key}/db/endpoint"
    type     = "String"
    value    = module.rds[each.key].db_instance_endpoint
  }

  resource "aws_ssm_parameter" "secret_arn" {
    for_each = local.ambientes
    name     = "/oficina/${each.key}/db/secret_arn"
    type     = "String"
    value    = aws_secretsmanager_secret.db[each.key].arn
  }
  ```

  **Diferenças em relação ao código da Fase 2 e o porquê:**

  | Fase 2 | Fase 3 | Motivo |
  |---|---|---|
  | `db.t3.micro` | `db.t4g.micro/small` | Graviton: ~20% mais barato, mesma performance |
  | senha só no state | senha no Secrets Manager | state não é cofre |
  | sem `storage_encrypted` | criptografado | requisito básico de segurança; ativar depois exige recriar |
  | sem backup | 7 dias em prod | banco gerenciado sem backup não é gerenciado |
  | VPC criada no mesmo módulo | VPC lida da SSM | separação de repositórios |
  | sem Performance Insights | ativado | dá o "gargalo em tempo real" que o enunciado pede |

  > **Sobre as migrations:** continuam rodando no boot da aplicação (golang-migrate com advisory lock do Postgres — já implementado e testado na Fase 2). Não crie um Job separado; o lock garante que só um pod aplique, mesmo com 5 réplicas subindo juntas.

- **Critérios de aceite:**
  - `terraform apply` cria o RDS em subnets privadas, criptografado, inacessível da internet.
  - Segredo criado com `username`, `password`, `host`, `port`, `dbname`.
  - Parâmetros SSM publicados.
  - `psql` a partir de um pod do cluster conecta; a partir da internet, não.
- **Estimativa:** M
- **Depende de:** F3-3.2 (VPC e SGs)

---

### F3-3.2 — Repositório `oficina-infra-k8s`: VPC e rede


> ⏱️ **[Corte 2](plano-10-dias.md#os-cortes)** — sem NAT Gateway — `enable_nat_gateway = false` e nós em subnet pública (`subnet_ids = module.vpc.public_subnets` no node group). Economiza US$ 8 e ~2 h de setup.**

- **Descrição:** A rede é a fundação de tudo. Migrar o módulo VPC da Fase 2, acrescentar o SG das Lambdas e publicar tudo na SSM.

- **Como fazer:**

  ```hcl
  locals {
    ambientes = toset(["homolog", "prod"])
    cidr      = "10.0.0.0/16"
    azs       = slice(data.aws_availability_zones.disponiveis.names, 0, 2)
  }

  data "aws_availability_zones" "disponiveis" { state = "available" }

  # VPC ÚNICA, compartilhada pelos dois ambientes (ver Estratégia de ambientes)
  module "vpc" {
    source  = "terraform-aws-modules/vpc/aws"
    version = "~> 5.8"

    name = "oficina"
    cidr = local.cidr
    azs  = local.azs

    private_subnets = [for i, _ in local.azs : cidrsubnet(local.cidr, 8, i)]
    public_subnets  = [for i, _ in local.azs : cidrsubnet(local.cidr, 8, i + 10)]

    enable_nat_gateway   = true
    single_nat_gateway   = true    # ~US$ 32/mês por NAT; um só para o desafio
    enable_dns_hostnames = true

    public_subnet_tags  = { "kubernetes.io/role/elb" = 1 }
    private_subnet_tags = { "kubernetes.io/role/internal-elb" = 1 }
  }

  # SG para as Lambdas na VPC (o repo lambda-auth apenas consome o ID)
  resource "aws_security_group" "lambda" {
    name   = "oficina-lambda"
    vpc_id = module.vpc.vpc_id
    egress {
      from_port   = 0
      to_port     = 0
      protocol    = "-1"
      cidr_blocks = ["0.0.0.0/0"]   # precisa sair para o Secrets Manager e o RDS
    }
  }

  resource "aws_ssm_parameter" "vpc_id" {
    name  = "/oficina/shared/vpc/id"
    type  = "String"
    value = module.vpc.vpc_id
  }

  resource "aws_ssm_parameter" "subnets_privadas" {
    name  = "/oficina/shared/vpc/subnets_privadas"
    type  = "StringList"
    value = join(",", module.vpc.private_subnets)
  }

  resource "aws_ssm_parameter" "lambda_sg" {
    name  = "/oficina/shared/lambda/sg_id"
    type  = "String"
    value = aws_security_group.lambda.id
  }
  ```

  > **Economia relevante:** um NAT Gateway custa ~US$ 32/mês + tráfego, e é o item mais caro depois do EKS. Se as Lambdas só precisam falar com RDS e Secrets Manager (ambos dentro da AWS), você pode **remover o NAT** e usar **VPC Endpoints** (`secretsmanager`, `ssm`, `logs`, `ecr.api`, `ecr.dkr`, `s3` gateway). Endpoints de interface custam ~US$ 7/mês cada, então com 3–4 já compensa... mas dá mais trabalho. Para o desafio, mantenha o NAT único e **destrua o ambiente após a gravação**.

- **Critérios de aceite:** VPC com 2 AZs, subnets públicas/privadas tagueadas para o EKS, NAT único, SG das Lambdas e 3 parâmetros SSM publicados.
- **Estimativa:** P
- **Depende de:** F3-0.1

---

### F3-3.3 — Cluster EKS com escalabilidade e addons


> ⏱️ **[Corte 3](plano-10-dias.md#os-cortes)** — sem Cluster Autoscaler — 2 nós `t3.small` fixos e HPA de 2 a 6 pods. Os `access_entries` e o `apply -target=module.eks` continuam **obrigatórios**.**

- **Descrição:** Cluster gerenciado com node group escalável, `metrics-server` (para o HPA), AWS Load Balancer Controller (para o `TargetGroupBinding`), External Secrets Operator e o agente do New Relic.

- **Como fazer:**

  ```hcl
  module "eks" {
    source  = "terraform-aws-modules/eks/aws"
    version = "~> 20.24"

    cluster_name    = "oficina"    # UM cluster para os dois ambientes
    cluster_version = var.eks_versao   # default = "1.36" (ver aviso abaixo)

    vpc_id     = module.vpc.vpc_id
    subnet_ids = module.vpc.private_subnets

    cluster_endpoint_public_access           = true   # kubectl do CI e da sua máquina
    enable_cluster_creator_admin_permissions = true

    # Addons gerenciados — versão resolvida automaticamente pela AWS
    cluster_addons = {
      coredns                = { most_recent = true }
      kube-proxy             = { most_recent = true }
      vpc-cni                = { most_recent = true }
      eks-pod-identity-agent = { most_recent = true }
      metrics-server         = { most_recent = true }   # ← o HPA depende disso
    }

    eks_managed_node_groups = {
      default = {
        instance_types = ["t3.medium"]
        min_size       = 2
        max_size       = 4      # espaço para o Cluster Autoscaler
        desired_size   = 2
        # Rótulo usado por nodeSelector/afinidade se precisar segregar carga
        labels = { workload = "geral" }
      }
    }
  }
  ```

  **🚨 Antes de tudo: escolha uma versão do Kubernetes em _standard support_.**

  O EKS cobra **US$ 0,10/h** por cluster em *standard support* e **US$ 0,60/h** em *extended support* — **seis vezes mais**. Uma versão sai do standard support cerca de 14 meses após o lançamento, e a transição é **automática e silenciosa**: o cluster continua funcionando, só a fatura muda. Nos 9 dias do plano isso é a diferença entre **US$ 21,60 e ~US$ 130**, transformando o item mais caro do projeto no dobro de tudo o mais somado.

  Descubra as versões válidas **hoje** e pegue a mais recente com `"STANDARD_SUPPORT"`:

  ```bash
  aws eks describe-cluster-versions --region us-east-1 \
    --query 'clusterVersions[?versionStatus==`STANDARD_SUPPORT`].[clusterVersion,endOfStandardSupportDate]' \
    --output table
  ```

  ```hcl
  variable "eks_versao" {
    description = "Versão do Kubernetes no EKS (manter em standard support)"
    type        = string
    default     = "1.36"   # standard support até 01/08/2027
  }
  ```

  Confira a data de fim do suporte antes de reaproveitar este plano em outra época — o comando e o histórico da consulta estão em [execucao.md](execucao.md#eks-versão-antiga-custa-6-mais). Mantenha o `kubectl` a no máximo uma *minor* do cluster.

  **⚠️ Duas armadilhas que travam o time por um dia inteiro cada.**

  **(a) Quem cria o cluster é o único que consegue usá-lo.** `enable_cluster_creator_admin_permissions = true` dá acesso administrativo a **quem rodou o apply** — que no CI é a role OIDC do GitHub, não os usuários do time. Resultado: `kubectl get pods` na máquina de vocês responde `error: You must be logged in to the server (Unauthorized)`, e não há como depurar nada. Declare os acessos explicitamente:

  ```hcl
  module "eks" {
    # ...
    authentication_mode = "API_AND_CONFIG_MAP"

    access_entries = {
      # uma entrada por pessoa do time
      maria = {
        principal_arn = "arn:aws:iam::123456789012:user/maria"
        policy_associations = {
          admin = {
            policy_arn   = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"
            access_scope = { type = "cluster" }
          }
        }
      }
      # a role do CD só precisa implantar, não administrar o cluster
      cd = {
        principal_arn = "arn:aws:iam::123456789012:role/gha-oficina-app"
        policy_associations = {
          edit = {
            policy_arn   = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSEditPolicy"
            access_scope = { type = "namespace", namespaces = ["oficina-homolog", "oficina-prod"] }
          }
        }
      }
    }
  }
  ```

  Repare no escopo da role do CD: `Edit` restrito aos dois namespaces, não `ClusterAdmin`. Um pipeline comprometido não deve poder apagar o cluster.

  **(b) Providers `kubernetes`/`helm` no mesmo apply que cria o cluster.** É o problema clássico: a configuração do provider depende de atributos (`cluster_endpoint`, CA) de um recurso que ainda não existe no momento do `plan`. O erro é `Provider configuration not known at plan time` ou, pior, um apply que passa e um `destroy` que falha depois.

  ```hcl
  provider "kubernetes" {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    # exec: token gerado na hora, sem credencial estática no state
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
    }
  }

  provider "helm" {
    kubernetes {
      host                   = module.eks.cluster_endpoint
      cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
      exec {
        api_version = "client.authentication.k8s.io/v1beta1"
        command     = "aws"
        args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
      }
    }
  }
  ```

  Isso resolve a autenticação, **mas não o primeiro apply**. Escolha uma:

  | Abordagem | Como | Quando usar |
  |---|---|---|
  | Apply em duas etapas | `terraform apply -target=module.eks` e depois `terraform apply` | mais simples; documente no README e no workflow (um step condicional na primeira execução) |
  | Dois root modules | `terraform/cluster/` e `terraform/addons/`, o segundo lendo o primeiro por SSM | mais limpo, mais arquivos, dois states |

  Recomendo a **primeira** para o escopo do desafio — mas registre a escolha, porque quem provisionar do zero vai esbarrar nela.

  **Escalabilidade em dois níveis** — vale explicar no vídeo, porque o enunciado pede "cluster com escalabilidade":

  | Nível | Componente | O que escala | Gatilho |
  |---|---|---|---|
  | Pod | **HPA** (já existe em `k8s/base/hpa.yaml`) | réplicas da aplicação (2 → 5) | CPU média > 50% |
  | Nó | **Cluster Autoscaler** ou **Karpenter** | instâncias EC2 (2 → 4) | pods em `Pending` por falta de recurso |

  Sem o segundo nível, o HPA cria pods que ficam `Pending` para sempre quando os nós lotam. Instale o Cluster Autoscaler via Helm:

  ```hcl
  resource "helm_release" "cluster_autoscaler" {
    name       = "cluster-autoscaler"
    repository = "https://kubernetes.github.io/autoscaler"
    chart      = "cluster-autoscaler"
    namespace  = "kube-system"

    set {
      name  = "autoDiscovery.clusterName"
      value = module.eks.cluster_name
    }
    set {
      name  = "awsRegion"
      value = var.regiao
    }
    set {
      name  = "rbac.serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
      value = module.autoscaler_irsa.iam_role_arn
    }
  }
  ```

  **Demais addons via Helm** (mesmo padrão): `aws-load-balancer-controller`, `external-secrets`, `newrelic-bundle` (F3-4.3). Todos precisam de **IRSA** (IAM Roles for Service Accounts) — use o módulo `terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks`, que já tem políticas prontas por addon.

  Publique na SSM:

  ```hcl
  resource "aws_ssm_parameter" "cluster_name" {
    name  = "/oficina/shared/eks/cluster_name"
    type  = "String"
    value = module.eks.cluster_name
  }

  resource "aws_ssm_parameter" "node_sg" {
    name  = "/oficina/shared/eks/node_sg_id"
    type  = "String"
    value = module.eks.node_security_group_id
  }
  ```

  Crie também os dois namespaces:

  ```hcl
  resource "kubernetes_namespace" "app" {
    for_each = local.ambientes
    metadata {
      name   = "oficina-${each.key}"
      labels = { ambiente = each.key }
    }
  }

  # ⏱️ Corte 16: PULE a ResourceQuota — ela existe para um ambiente não
  # engolir o outro, e com um namespace só não protege de nada.
  #
  # Quota por ambiente: impede que homologação consuma o cluster inteiro
  # e deixe produção sem recurso para escalar.
  resource "kubernetes_resource_quota" "app" {
    for_each = local.ambientes
    metadata {
      name      = "quota"
      namespace = kubernetes_namespace.app[each.key].metadata[0].name
    }
    spec {
      hard = {
        "requests.cpu"    = each.key == "prod" ? "2" : "1"
        "requests.memory" = each.key == "prod" ? "4Gi" : "2Gi"
        "pods"            = each.key == "prod" ? "12" : "6"
      }
    }
  }
  ```

  > **Aviso de tempo:** `terraform apply` de um EKS do zero leva **15–20 minutos** (o control plane sozinho é ~10 min). Não é travamento. E o `destroy` leva outros ~15 min, então planeje a agenda da demonstração.

- **Critérios de aceite:**
  - `aws eks update-kubeconfig` + `kubectl get nodes` mostra 2 nós `Ready` **a partir da máquina de cada pessoa do time**, não só de quem rodou o apply.
  - `kubectl top nodes` funciona (prova que o metrics-server está de pé).
  - Addons instalados e `Running`.
  - Parâmetros SSM publicados.
- **Estimativa:** G
- **Depende de:** F3-3.2

---

### F3-3.4 — API Gateway HTTP API por ambiente

- **Descrição:** Criar um Gateway por ambiente, com access logs e throttling.

  > **Por que uma API por ambiente e não uma API com dois stages?** Porque no HTTP API (v2) os *stages* compartilham as mesmas rotas e integrações — não existe *stage variable* que permita apontar o mesmo `route_key` para backends diferentes (isso só existe no REST API v1). Como homologação e produção precisam de integrações distintas (listeners diferentes do ALB) e authorizers distintos (segredos JWT diferentes), duas APIs são o desenho correto. Cada uma usa o stage `$default`, que também deixa a URL limpa, sem `/prod` no caminho.

- **Como fazer:**

  ```hcl
  # Um Gateway por ambiente: URLs distintas, throttling e authorizer próprios
  resource "aws_apigatewayv2_api" "principal" {
    for_each      = local.ambientes
    name          = "oficina-${each.key}"
    protocol_type = "HTTP"

    cors_configuration {
      allow_origins = ["*"]                       # restrinja se houver front conhecido
      allow_methods = ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]
      allow_headers = ["authorization", "content-type", "x-signature", "x-correlation-id"]
      max_age       = 300
    }
  }

  resource "aws_cloudwatch_log_group" "apigw" {
    for_each          = local.ambientes
    name              = "/aws/apigw/oficina-${each.key}"
    retention_in_days = 14
  }

  resource "aws_apigatewayv2_stage" "principal" {
    for_each    = local.ambientes
    api_id      = aws_apigatewayv2_api.principal[each.key].id
    name        = "$default"   # sem prefixo de stage na URL; o ambiente já é a API
    auto_deploy = true

    access_log_settings {
      destination_arn = aws_cloudwatch_log_group.apigw[each.key].arn
      # JSON estruturado: casa com o requisito de "logs estruturados com correlação"
      format = jsonencode({
        requestId      = "$context.requestId"
        correlationId  = "$context.requestId"
        ip             = "$context.identity.sourceIp"
        requestTime    = "$context.requestTime"
        routeKey       = "$context.routeKey"
        status         = "$context.status"
        responseLength = "$context.responseLength"
        latency        = "$context.responseLatency"
        integrationErr = "$context.integration.error"
        authorizerErr  = "$context.authorizer.error"
      })
    }

    default_route_settings {
      # Proteção contra abuso — e material para o dashboard de erros 429
      throttling_burst_limit   = each.key == "prod" ? 200 : 50
      throttling_rate_limit    = each.key == "prod" ? 100 : 25
      detailed_metrics_enabled = true
    }
  }

  resource "aws_ssm_parameter" "apigw_id" {
    for_each = local.ambientes
    name     = "/oficina/${each.key}/apigw/id"
    type     = "String"
    value    = aws_apigatewayv2_api.principal[each.key].id
  }

  resource "aws_ssm_parameter" "apigw_url" {
    for_each = local.ambientes
    name     = "/oficina/${each.key}/apigw/url"
    type     = "String"
    value    = aws_apigatewayv2_stage.principal[each.key].invoke_url
  }
  ```

  > ⚠️ **Variável de `$context` escrita errada não dá erro** — nem no `terraform validate`, nem no deploy. Ela simplesmente vira string vazia no log, e você só descobre quando precisa investigar um incidente. Confira cada nome na [lista oficial de variáveis de `$context`](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-logging-variables.html) e valide olhando um log real no CloudWatch antes de considerar a tarefa pronta.
  >
  > Para correlacionar de ponta a ponta, propague o `x-correlation-id` do cliente quando existir: adicione `correlationIdCliente = "$context.request.header.x-correlation-id"` ao formato e faça a aplicação reaproveitar esse header (F3-4.1). Assim o mesmo identificador aparece no log do Gateway, no log do pod e no trace do New Relic.

- **Critérios de aceite:** Gateway criado; `curl $URL` responde (404 até haver rotas); access logs aparecendo no CloudWatch em JSON; SSM publicado.
- **Estimativa:** M
- **Depende de:** F3-3.2

---

### F3-3.5 — ALB interno, Target Group e VPC Link


> ⏱️ **[Corte 1](plano-10-dias.md#os-cortes)** — tarefa cortada. Use `Service type: LoadBalancer` e integre o API Gateway direto no NLB via `HTTP_PROXY`, exigindo um header compartilhado. Leia esta tarefa mesmo assim — o *porquê* dela é o conteúdo da ADR que justifica o corte.**

- **Descrição:** Este é o ponto mais sutil da arquitetura. O API Gateway vive **fora** da sua VPC; o cluster está em **subnets privadas**. A ponte é um VPC Link apontando para um Load Balancer interno. O desafio: quem cria o ALB?

- **Como fazer:**

  **O problema da dependência circular.** O caminho "natural" seria o Ingress do Kubernetes criar o ALB (via AWS Load Balancer Controller) e o Terraform referenciá-lo. Mas aí o `terraform apply` do `infra-k8s` **precisaria** que a aplicação já estivesse implantada — e a aplicação precisa da infra. Ciclo.

  **A solução: `TargetGroupBinding`.** O Terraform cria ALB + Target Group (recursos AWS puros, sem depender de nada no cluster). A aplicação declara um CR `TargetGroupBinding` que instrui o AWS Load Balancer Controller a **registrar os pods** naquele Target Group já existente. A dependência passa a fluir em uma direção só.

  ```hcl
  # Target group do tipo "ip" — os pods do EKS têm IP próprio na VPC (CNI da AWS).
  # Um por ambiente: cada namespace registra seus próprios pods.
  resource "aws_lb_target_group" "api" {
    for_each    = local.ambientes
    name        = "oficina-${each.key}-api"
    port        = 8080
    protocol    = "HTTP"
    target_type = "ip"
    vpc_id      = module.vpc.vpc_id

    health_check {
      path                = "/health/ready"   # já implementado na aplicação
      matcher             = "200"
      interval            = 15
      healthy_threshold   = 2
      unhealthy_threshold = 3
    }

    # Sem isso, o rolling update derruba conexões em voo
    deregistration_delay = 30
  }

  # UM ALB para os dois ambientes; a separação acontece por listener/porta
  resource "aws_lb" "interno" {
    name               = "oficina-alb"
    internal           = true                    # ← não exposto na internet
    load_balancer_type = "application"
    subnets            = module.vpc.private_subnets
    security_groups    = [aws_security_group.alb.id]
  }

  locals {
    # porta do listener por ambiente — o Gateway de cada ambiente aponta para a sua
    porta_listener = { prod = 80, homolog = 8080 }
  }

  resource "aws_lb_listener" "http" {
    for_each          = local.ambientes
    load_balancer_arn = aws_lb.interno.arn
    port              = local.porta_listener[each.key]
    protocol          = "HTTP"
    default_action {
      type             = "forward"
      target_group_arn = aws_lb_target_group.api[each.key].arn
    }
  }

  # ── VPC Link: a ponte do Gateway para dentro da VPC ───────────────────────
  # Um só: o VPC Link é uma ENI em subnets, não tem noção de ambiente.
  resource "aws_apigatewayv2_vpc_link" "principal" {
    name               = "oficina"
    subnet_ids         = module.vpc.private_subnets
    security_group_ids = [aws_security_group.vpclink.id]
  }

  resource "aws_apigatewayv2_integration" "api" {
    for_each           = local.ambientes
    api_id             = aws_apigatewayv2_api.principal[each.key].id
    integration_type   = "HTTP_PROXY"
    integration_uri    = aws_lb_listener.http[each.key].arn
    integration_method = "ANY"
    connection_type    = "VPC_LINK"
    connection_id      = aws_apigatewayv2_vpc_link.principal.id

    # Propaga o contexto do authorizer para a aplicação como headers
    request_parameters = {
      "append:header.x-cliente-id" = "$context.authorizer.sub"
      "append:header.x-token-tipo" = "$context.authorizer.tipo"
    }
  }

  data "aws_ssm_parameter" "authorizer_id" {
    for_each = local.ambientes
    name     = "/oficina/${each.key}/apigw/authorizer_id"   # publicado pelo lambda-auth
  }

  # Rota curinga protegida pelo authorizer da Lambda
  resource "aws_apigatewayv2_route" "api" {
    for_each           = local.ambientes
    api_id             = aws_apigatewayv2_api.principal[each.key].id
    route_key          = "ANY /v1/{proxy+}"
    target             = "integrations/${aws_apigatewayv2_integration.api[each.key].id}"
    authorization_type = "CUSTOM"
    authorizer_id      = data.aws_ssm_parameter.authorizer_id[each.key].value
  }

  # Rotas públicas — sem authorizer, a aplicação cuida da validação.
  # for_each aninhado: produto cartesiano ambiente × rota, achatado em um mapa.
  locals {
    rotas_publicas = [
      "POST /v1/auth/login",
      "POST /v1/webhooks/budget-response",   # protegida por HMAC
      "GET /health/ready",
    ]
    rotas_publicas_por_ambiente = {
      for par in setproduct(tolist(local.ambientes), local.rotas_publicas) :
      "${par[0]}|${par[1]}" => { ambiente = par[0], rota = par[1] }
    }
  }

  resource "aws_apigatewayv2_route" "publicas" {
    for_each  = local.rotas_publicas_por_ambiente
    api_id    = aws_apigatewayv2_api.principal[each.value.ambiente].id
    route_key = each.value.rota
    target    = "integrations/${aws_apigatewayv2_integration.api[each.value.ambiente].id}"
  }

  resource "aws_ssm_parameter" "tg_arn" {
    for_each = local.ambientes
    name     = "/oficina/${each.key}/alb/target_group_arn"
    type     = "String"
    value    = aws_lb_target_group.api[each.key].arn
  }
  ```

  E no **repo da aplicação**, `k8s/overlays/prod/target-group-binding.yaml`:

  ```yaml
  apiVersion: elbv2.k8s.aws/v1beta1
  kind: TargetGroupBinding
  metadata:
    name: api
    namespace: oficina-prod
  spec:
    # Resolvido pelo passo "Resolver placeholders" do CD (envsubst),
    # que lê /oficina/{env}/alb/target_group_arn da SSM.
    targetGroupARN: ${TARGET_GROUP_ARN}
    serviceRef:
      name: api
      port: 80
    targetType: ip
  ```

  > **Regras de security group que costumam faltar** (e geram um 503 misterioso no Gateway) — lembre que agora há **duas portas de listener**, 80 (prod) e 8080 (homolog):
  > 1. SG do **VPC Link** → egress para o SG do ALB nas portas 80 e 8080.
  > 2. SG do **ALB** → ingress a partir do SG do VPC Link nas portas 80 e 8080.
  > 3. SG do **ALB** → egress para o SG dos **nós** na porta 8080 (porta do container).
  > 4. SG dos **nós** → ingress a partir do SG do ALB na porta 8080.
  >
  > Faltando a (4), o health check do target group falha e todos os alvos ficam `unhealthy`. Debug: `aws elbv2 describe-target-health --target-group-arn <arn>`.
  >
  > **Alternativa mais simples**, se o VPC Link consumir tempo demais: ALB **internet-facing** com um `aws_lb_listener_rule` exigindo um header secreto que só o Gateway envia. Funciona, atende o requisito de gateway, mas é menos elegante — e a banca pode questionar. Use como plano B, documentado.

- **Critérios de aceite:**
  - `curl $GATEWAY_URL/v1/work-orders` sem token → 401 (do authorizer).
  - Com token válido → 200 vindo do pod.
  - `describe-target-health` mostra os pods como `healthy`.
  - Escalar o Deployment registra automaticamente os novos pods no target group.
- **Estimativa:** G
- **Depende de:** F3-3.3, F3-3.4, F3-2.3

---

### F3-3.6 — Overlays de Kubernetes por ambiente

- **Descrição:** Substituir o `k8s/overlays/aws` (com placeholders `SUBSTITUIR_PELO_*`) por dois overlays reais.

- **Como fazer:**

  ```
  k8s/
    base/                       ← inalterado (deployment, service, hpa, configmap)
      external-secret.yaml      ← NOVO (F3-2.5)
    overlays/
      local/                    ← mantido (kind + postgres + mailpit)
      homolog/
        kustomization.yaml
        configmap-patch.yaml
        target-group-binding.yaml
      prod/
        kustomization.yaml
        configmap-patch.yaml
        target-group-binding.yaml
        hpa-patch.yaml          ← prod escala mais: min 3, max 10
  ```

  ```yaml
  # k8s/overlays/prod/kustomization.yaml
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization

  namespace: oficina-prod

  resources:
    - ../../base
    - target-group-binding.yaml

  patches:
    - path: configmap-patch.yaml
    - path: hpa-patch.yaml

  # A tag é sobrescrita pelo CD (kustomize edit set image)
  images:
    - name: docker.io/problematheu/oficina-api
      newTag: latest

  # Muda a cada deploy → força rollout mesmo sem mudança de imagem
  labels:
    - pairs:
        app.kubernetes.io/instance: prod
  ```

  ```yaml
  # k8s/overlays/prod/configmap-patch.yaml
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: oficina-config
  data:
    DB_HOST: ${DB_HOST}                                        # envsubst, da SSM
    DB_NAME: oficina_prod
    NOTIFIER: smtp
    SMTP_HOST: email-smtp.us-east-1.amazonaws.com              # SES
    SMTP_PORT: "587"
    NEW_RELIC_APP_NAME: oficina-api-prod
    LOG_LEVEL: info
    LOG_FORMAT: json
  ```

  > **Valores dinâmicos (endpoint do RDS, ARN do target group) em YAML estático.** Três opções: (a) o CD faz `envsubst` lendo da SSM antes do `kubectl apply` — simples e explícito; (b) `kustomize edit set` no pipeline; (c) External Secrets também para ConfigMap. Adotamos **(a)** — é o passo "Resolver placeholders" de [F3-1.5](#f3-15--cd-da-aplicação-build--registry--eks), com cinco linhas de workflow e uma guarda que falha o deploy se algum placeholder ficar por resolver.
  >
  > ⚠️ Passe a lista explícita de variáveis ao `envsubst` (`envsubst '$TARGET_GROUP_ARN $DB_HOST'`). Sem ela, o `envsubst` substitui **toda** `$VAR` do arquivo — e YAML de Kubernetes tem `$(...)` e cifrões legítimos que seriam apagados silenciosamente.

- **Critérios de aceite:** `kubectl apply -k k8s/overlays/homolog` e `.../prod` sobem em namespaces distintos, com valores corretos e nenhum placeholder.
- **Estimativa:** M
- **Depende de:** F3-3.5

---

### F3-3.7 — Notificação por e-mail em produção (Amazon SES)


> ⏱️ **[Corte 5](plano-10-dias.md#os-cortes)** — tarefa cortada: `NOTIFIER=log` em produção e e-mail demonstrado no ambiente local com Mailpit. Elimina 3 h e a espera de até 24 h pela saída do sandbox do SES. Registre em ADR e diga no vídeo.

- **Descrição:** A Fase 2 entregou notificação de status por e-mail com um `SMTPNotifier` apontando para o Mailpit local. Em produção o `ConfigMap` está com `SMTP_HOST: SUBSTITUIR_PELO_SMTP_REAL`. Se isso for para a nuvem como está, **o envio falha silenciosamente** — o design é assíncrono e não derruba a transição de status, então ninguém percebe até alguém perguntar por que o cliente não recebeu nada. Uma funcionalidade entregue na fase anterior regredindo na fase seguinte é exatamente o tipo de coisa que a banca nota.

- **Como fazer:**

  ```hcl
  # Identidade verificada — sem isso o SES recusa qualquer envio
  resource "aws_ses_email_identity" "remetente" {
    email = var.email_remetente     # ex.: oficina@seudominio.com
  }

  # Usuário IAM dedicado ao envio SMTP
  resource "aws_iam_user" "ses" {
    name = "oficina-ses-smtp"
  }

  resource "aws_iam_user_policy" "ses" {
    user = aws_iam_user.ses.name
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Effect   = "Allow"
        Action   = ["ses:SendRawEmail", "ses:SendEmail"]
        Resource = "*"
      }]
    })
  }

  resource "aws_iam_access_key" "ses" {
    user = aws_iam_user.ses.name
  }

  # A senha SMTP do SES NÃO é a secret key: é derivada dela por HMAC.
  # O provider expõe o valor já convertido — não tente calcular à mão.
  resource "aws_secretsmanager_secret" "smtp" {
    for_each                = local.ambientes
    name                    = "oficina/${each.key}/smtp"
    recovery_window_in_days = 0
  }

  resource "aws_secretsmanager_secret_version" "smtp" {
    for_each  = local.ambientes
    secret_id = aws_secretsmanager_secret.smtp[each.key].id
    secret_string = jsonencode({
      SMTP_USER = aws_iam_access_key.ses.id
      SMTP_PASS = aws_iam_access_key.ses.ses_smtp_password_v4
    })
  }
  ```

  E acrescente as duas chaves ao `ExternalSecret` de F3-2.5.

  > ⚠️ **A armadilha do sandbox do SES.** Toda conta nova começa em *sandbox*: só envia **para endereços verificados**, no máximo 200 mensagens/dia. Se você descobrir isso na hora de gravar o vídeo, perdeu a cena. Duas opções:
  >
  > 1. **Verifique os endereços de destino** que aparecerão na demonstração (o e-mail do cliente de teste). Leva 2 minutos e resolve para o vídeo.
  > 2. **Peça saída do sandbox** em *SES → Account dashboard → Request production access*. A aprovação leva **até 24 h** — faça isso na Sprint 2, não na 5.
  >
  > Faça a (1) sempre, e a (2) por garantia.

  **Alternativa se o SES for atrito demais:** manter `NOTIFIER=log` em produção e demonstrar o e-mail no ambiente local com Mailpit, deixando explícito no README que a integração SMTP é a mesma e só muda o provedor. É honesto e defensável — o que não é aceitável é o `SUBSTITUIR_PELO_SMTP_REAL` chegar na entrega.

- **Critérios de aceite:**
  - Uma transição de status em produção envia e-mail de verdade, recebido na caixa de entrada.
  - Credenciais SMTP no Secrets Manager, nunca no ConfigMap.
  - Nenhum `SUBSTITUIR_PELO_*` restante nos manifestos (`grep -rn SUBSTITUIR k8s/` vazio).
  - Falha de envio é registrada como `IntegracaoEvent` com `resultado: falha` (F3-4.5) e **não** quebra a transição.
- **Estimativa:** M
- **Depende de:** F3-2.5, F3-3.6

---

## E4 — Observabilidade com New Relic

> O enunciado é específico e checa item por item: **latência das APIs**, **CPU/memória do Kubernetes**, **healthchecks e uptime**, **alertas para falhas no processamento de OS**, **logs estruturados JSON com correlação**, e dashboards com **volume diário de OS**, **tempo médio por status** e **erros de integração**. Cada um vira uma tarefa abaixo.

### F3-4.1 — Logs estruturados JSON com correlação de requisições

- **Descrição:** Hoje `cmd/api/main.go` usa `log.Printf` (texto puro) e os use cases usam `slog` com o handler default, que também é texto. Não há nada que ligue o log de um handler ao log do use case da mesma requisição. Sem isso, o requisito "logs estruturados (JSON), incluindo correlação entre requisições" não está atendido — e depurar em produção com 5 réplicas é impossível.

- **Como fazer:**

  **1. Handler JSON global** (`cmd/api/main.go`, primeira coisa no `main`):

  ```go
  func configurarLog() {
      nivel := slog.LevelInfo
      if os.Getenv("LOG_LEVEL") == "debug" {
          nivel = slog.LevelDebug
      }
      h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
          Level: nivel,
          ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
              // "time" → "timestamp" e "msg" → "message": nomes que o
              // parser do New Relic reconhece sem configuração extra
              switch a.Key {
              case slog.TimeKey:
                  a.Key = "timestamp"
              case slog.MessageKey:
                  a.Key = "message"
              }
              return a
          },
      })
      slog.SetDefault(slog.New(h).With(
          "service.name", os.Getenv("NEW_RELIC_APP_NAME"),
          "hostname", os.Getenv("HOSTNAME"),   // K8s injeta o nome do pod
      ))
      // Redireciona o log padrão (usado por libs terceiras) para o slog
      slog.SetLogLoggerLevel(slog.LevelInfo)
  }
  ```

  E **troque todos os `log.Printf`/`log.Fatalf`** de `main.go` por `slog.Info`/`slog.Error` + `os.Exit(1)`. São ~8 ocorrências.

  **2. Middleware de correlação** (`internal/infra/http/middleware/correlacao.go`):

  ```go
  package middleware

  import (
      "context"
      "log/slog"
      "net/http"

      "github.com/google/uuid"
  )

  type ctxLoggerKey struct{}

  const HeaderCorrelacao = "X-Correlation-Id"

  // Correlacao garante que toda requisição tenha um identificador único,
  // reaproveitando o que veio do cliente/gateway quando existir, e coloca
  // um *slog.Logger já decorado no contexto.
  func Correlacao() func(http.Handler) http.Handler {
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              id := r.Header.Get(HeaderCorrelacao)
              if id == "" {
                  id = uuid.NewString()
              }
              // Devolver o header permite ao cliente citar o ID ao abrir chamado
              w.Header().Set(HeaderCorrelacao, id)

              logger := slog.Default().With(
                  "correlation_id", id,
                  "http.method", r.Method,
                  "http.route", r.URL.Path,
              )
              ctx := context.WithValue(r.Context(), ctxLoggerKey{}, logger)
              next.ServeHTTP(w, r.WithContext(ctx))
          })
      }
  }

  // Log devolve o logger da requisição, ou o default fora de um contexto HTTP.
  func Log(ctx context.Context) *slog.Logger {
      if l, ok := ctx.Value(ctxLoggerKey{}).(*slog.Logger); ok {
          return l
      }
      return slog.Default()
  }
  ```

  **3. Usar nos use cases.** Os use cases hoje chamam `slog.InfoContext(ctx, ...)` — o `ctx` é passado mas **o logger não sai dele**, então a correlação se perde. Troque por:

  ```go
  // antes
  slog.InfoContext(ctx, "login: tentativa de autenticação", "email", input.Email)
  // depois
  middleware.Log(ctx).InfoContext(ctx, "login: tentativa de autenticação", "email", input.Email)
  ```

  > Importar `infra/http/middleware` dentro de `application/usecase` **viola a regra de dependência** que a F2-0.2 corrigiu com esforço. Faça direito: crie `internal/application/logging` com a função `Log(ctx)` e a chave de contexto; a infra escreve nela, a aplicação lê. A camada de dentro define a porta, a de fora implementa. Custa 20 linhas e mantém a arquitetura íntegra — e a banca **vai** olhar isso.

  **4. Correlacionar com o trace do APM.** Com o agente New Relic ativo (F3-4.2) e `nrslog`, cada linha de log ganha `trace.id` e `span.id` automaticamente, e a UI passa a mostrar "logs in context" dentro do trace. É o que transforma "temos logs" em "temos observabilidade".

  Resultado esperado de uma linha de log:

  ```json
  {
    "timestamp": "2026-09-01T18:22:04.113Z",
    "level": "INFO",
    "message": "criar OS: concluído",
    "service.name": "oficina-api-prod",
    "hostname": "api-7d9c8b5f4-x2klm",
    "correlation_id": "b1f2...",
    "http.method": "POST",
    "http.route": "/v1/work-orders",
    "os_id": "9f1c...",
    "trace.id": "4bf92f3577b34da6",
    "span.id": "00f067aa0ba902b7"
  }
  ```

- **Critérios de aceite:**
  - `kubectl logs` mostra **apenas** JSON válido (`kubectl logs deploy/api | jq .` sem erro).
  - Uma requisição gera linhas com o mesmo `correlation_id` no handler e no use case.
  - `X-Correlation-Id` enviado pelo cliente é reaproveitado, não substituído.
  - Nenhum `log.Printf` restante (`grep -rn '"log"' cmd/ internal/` vazio).
- **Estimativa:** M
- **Depende de:** —

---

### F3-4.2 — APM da aplicação Go

> ⏱️ **[Corte 11](plano-10-dias.md#os-cortes)** — pule a troca do driver por `nrpq`. O APM cobre a latência das APIs, que é o requisito; o span de query é diagnóstico extra.

- **Descrição:** Instrumentar a aplicação para reportar transações, latência, throughput, erros e traces distribuídos.

- **Como fazer:**

  ```bash
  go get github.com/newrelic/go-agent/v3
  go get github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog
  ```

  ```go
  // cmd/api/main.go
  nrApp, err := newrelic.NewApplication(
      newrelic.ConfigAppName(os.Getenv("NEW_RELIC_APP_NAME")),
      newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
      newrelic.ConfigDistributedTracerEnabled(true),
      newrelic.ConfigAppLogForwardingEnabled(true),   // envia os logs junto com o trace
      // Sem license key (dev local) o agente fica inerte em vez de derrubar a app
      newrelic.ConfigEnabled(os.Getenv("NEW_RELIC_LICENSE_KEY") != ""),
  )
  if err != nil {
      slog.Error("new relic: falha ao inicializar", "error", err)
  }
  ```

  Middleware para o chi:

  ```go
  // internal/infra/http/middleware/apm.go
  func APM(app *newrelic.Application) func(http.Handler) http.Handler {
      return func(next http.Handler) http.Handler {
          return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
              if app == nil {
                  next.ServeHTTP(w, r)
                  return
              }
              // Nome da transação = padrão da rota, não o path concreto.
              // "/v1/work-orders/{id}" e não "/v1/work-orders/9f1c-..." —
              // caso contrário cada UUID vira uma transação diferente e o
              // APM fica inútil (cardinalidade explosiva).
              nome := r.Method + " " + r.URL.Path
              if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
                  nome = r.Method + " " + rc.RoutePattern()
              }
              txn := app.StartTransaction(nome)
              defer txn.End()
              txn.SetWebRequestHTTP(r)
              w = txn.SetWebResponse(w)
              next.ServeHTTP(w, newrelic.RequestWithTransactionContext(r, txn))
          })
      }
  }
  ```

  Instrumentar o banco — troque o driver por `nrpq` (wrapper de `lib/pq`) em `internal/infra/database/database.go`:

  ```go
  import _ "github.com/newrelic/go-agent/v3/integrations/nrpq"
  db, err := sql.Open("nrpostgres", dsn)   // era "postgres"
  ```

  Com isso cada query vira um span dentro do trace, e você consegue responder "a latência subiu por causa do banco ou da aplicação?" — que é literalmente o "detectar gargalos em tempo real" do enunciado.

  Ligue o `nrslog` ao handler JSON de F3-4.1 para injetar `trace.id`/`span.id`:

  ```go
  h := nrslog.WrapHandler(nrApp, slog.NewJSONHandler(os.Stdout, opts))
  slog.SetDefault(slog.New(h))
  ```

  > A API do `nrslog` mudou entre versões (houve `nrslog.JSONHandler`, depois `WrapHandler`). Confira o exemplo do pacote na versão que o `go.mod` resolver — o `go doc github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog` responde em segundos.

  Ordem dos middlewares (importa):

  ```go
  r.Use(apimiddleware.Correlacao())   // 1º: todo log já sai correlacionado
  r.Use(apimiddleware.APM(nrApp))     // 2º: a transação envolve o resto
  r.Use(apimiddleware.AssinaturaWebhook())
  r.Use(apimiddleware.JWT())
  ```

- **Critérios de aceite:**
  - A aplicação aparece em **APM & Services** no New Relic com throughput e response time.
  - Transações agrupadas por padrão de rota (nenhuma transação com UUID no nome).
  - Traces mostram spans de banco de dados.
  - Logs aparecem em "Logs in context" dentro de um trace.
  - Rodar local sem `NEW_RELIC_LICENSE_KEY` não quebra nada.
- **Estimativa:** M
- **Depende de:** F3-4.1

---

### F3-4.3 — Monitoramento do cluster Kubernetes

- **Descrição:** CPU, memória, estado dos pods, eventos e coleta dos logs de todos os containers.

- **Como fazer:**

  Instale o `nri-bundle` via Helm no `infra-k8s` (um único chart traz infra agent, kube-state-metrics, coletor de logs, Prometheus agent e, opcionalmente, Pixie):

  ```hcl
  resource "helm_release" "newrelic" {
    name             = "newrelic-bundle"
    repository       = "https://helm-charts.newrelic.com"
    chart            = "nri-bundle"
    namespace        = "newrelic"
    create_namespace = true

    set_sensitive {
      name  = "global.licenseKey"
      value = var.new_relic_license_key
    }
    set {
      name  = "global.cluster"
      value = module.eks.cluster_name
    }
    set {
      name  = "newrelic-infrastructure.privileged"
      value = "true"
    }
    set {
      name  = "kube-state-metrics.enabled"
      value = "true"
    }
    # Coleta o stdout de todos os pods — é aqui que o JSON de F3-4.1 é ingerido
    set {
      name  = "newrelic-logging.enabled"
      value = "true"
    }
    set {
      name  = "kubeEvents.enabled"
      value = "true"      # OOMKilled, Failed scheduling, etc.
    }
    # Pixie dá tracing automático sem instrumentar, mas consome bastante
    # recurso do cluster. Deixe desligado em nós t3.medium.
    set {
      name  = "pixie-chart.enabled"
      value = "false"
    }
  }
  ```

  > ⚠️ O `newrelic-logging` (Fluent Bit) coleta **tudo**. Com o free tier de 100 GB/mês, logs em `debug` de 5 pods podem estourar a cota em dias. Mantenha `LOG_LEVEL=info` em produção e, se necessário, exclua namespaces ruidosos via `newrelic-logging.fluentBit.config.filters`. Monitore o consumo em *Administration → Data management*.

  Depois de instalado, o New Relic mostra em **Kubernetes → Cluster explorer**: nós, pods, CPU/memória por container, e a correlação automática entre o pod e a aplicação do APM (mesma `service.name`).

- **Critérios de aceite:**
  - Cluster visível no Kubernetes explorer com os 2 nós e os pods da aplicação.
  - Gráficos de CPU e memória por container populados.
  - Logs dos pods aparecem em **Logs**, parseados como JSON (campos pesquisáveis, não texto).
  - Um `kubectl delete pod` gera evento visível no New Relic.
- **Estimativa:** M
- **Depende de:** F3-3.3, F3-4.1

---

### F3-4.4 — Observabilidade das Lambdas

- **Descrição:** As funções de autenticação também precisam de latência, erros e cold starts monitorados — é o caminho crítico do login.

- **Como fazer:**

  Caminho mais simples e suficiente: **log forwarding do CloudWatch para o New Relic**. O `nri-bundle` não cobre Lambda; use a integração de logs:

  1. New Relic UI → *Integrations & Agents* → **AWS Lambda** → seguir o assistente, que cria uma Lambda de forwarding (`newrelic-log-ingestion`).
  2. Assine o log group das suas funções nela:

  ```hcl
  resource "aws_cloudwatch_log_subscription_filter" "nr" {
    for_each        = local.funcoes
    name            = "newrelic-${each.key}"
    log_group_name  = aws_cloudwatch_log_group.fn[each.key].name
    filter_pattern  = ""
    destination_arn = var.newrelic_log_ingestion_arn
  }
  ```

  Como o handler já escreve JSON via `slog` (F3-2.1), os campos chegam estruturados e pesquisáveis.

  Caminho mais completo (APM real, com distributed tracing ligando Gateway → Lambda → RDS): adicionar a **layer** `NewRelicLambdaExtension` e as variáveis `NEW_RELIC_LAMBDA_HANDLER`, `NEW_RELIC_ACCOUNT_ID`. Vale se sobrar tempo; o `newrelic-lambda` CLI automatiza (`newrelic-lambda integrations install` + `newrelic-lambda subscriptions install`).

- **Critérios de aceite:** Logs das duas funções pesquisáveis no New Relic com campos estruturados; duração e erros visíveis; cold start identificável.
- **Estimativa:** M
- **Depende de:** F3-2.3, F3-4.3

---

### F3-4.5 — Eventos de negócio para os dashboards

- **Descrição:** Latência e CPU vêm de graça com o agente. Mas "volume diário de ordens de serviço", "tempo médio de execução por status" e "erros nas integrações" são **métricas de negócio** — só existem se a aplicação emitir. É o item que mais diferencia uma entrega boa de uma mediana.

- **Como fazer:**

  Defina uma porta na aplicação (mesma lógica do `Notifier` da Fase 2, que já existe e funciona bem):

  ```go
  // internal/application/usecase/ports.go
  // Metricas registra eventos de negócio para a plataforma de observabilidade.
  // Implementações não devem bloquear nem falhar a operação de domínio.
  type Metricas interface {
      RegistrarEvento(ctx context.Context, nome string, atributos map[string]any)
  }
  ```

  Implementação em `internal/infra/observability/newrelic_metricas.go`:

  ```go
  type NewRelicMetricas struct{ app *newrelic.Application }

  func (m *NewRelicMetricas) RegistrarEvento(ctx context.Context, nome string, attrs map[string]any) {
      if m.app == nil {
          return
      }
      // Anexa o trace atual, ligando o evento de negócio à requisição
      if txn := newrelic.FromContext(ctx); txn != nil {
          md := txn.GetTraceMetadata()
          attrs["trace.id"] = md.TraceID
      }
      m.app.RecordCustomEvent(nome, attrs)
  }
  ```

  E uma `NoopMetricas` para testes e ambiente local.

  **Onde emitir** (em `ordem_servico_usecase.go`):

  ```go
  // Ao criar uma OS
  uc.metricas.RegistrarEvento(ctx, "OrdemServicoEvent", map[string]any{
      "evento":    "criada",
      "resultado": "sucesso",
      "os_id":     os.ID.String(),
      "numero":    os.Numero,
      "valor":     os.ValorTotal,
  })

  // Ao falhar (estoque insuficiente, transição inválida, erro de banco)
  uc.metricas.RegistrarEvento(ctx, "OrdemServicoEvent", map[string]any{
      "evento":    "criada",
      "resultado": "falha",
      "motivo":    codigoDoErro(err),   // "estoque_insuficiente", "transicao_invalida"...
  })

  // A cada transição de status — aqui está o "tempo médio por status"
  uc.metricas.RegistrarEvento(ctx, "OrdemServicoEvent", map[string]any{
      "evento":           "transicao",
      "resultado":        "sucesso",
      "os_id":            os.ID.String(),
      "status_anterior":  anterior,
      "status_novo":      novo,
      // duração que a OS passou no status anterior
      "duracao_segundos": time.Since(entradaNoStatusAnterior).Seconds(),
  })

  // Falhas de integração (webhook e e-mail)
  uc.metricas.RegistrarEvento(ctx, "IntegracaoEvent", map[string]any{
      "integracao": "webhook_orcamento",   // ou "email"
      "resultado":  "falha",
      "motivo":     "assinatura_invalida",
  })
  ```

  > **Por que `RecordCustomEvent` e não uma métrica numérica?** Porque evento é um registro com atributos: você consulta por qualquer dimensão depois (`FACET motivo`, `FACET status_novo`, `WHERE valor > 1000`) sem ter previsto a pergunta. Uma métrica pré-agregada só responde o que você decidiu medir antes. Para volume e distribuição, evento é a escolha certa.
  >
  > **Cuidado com dado pessoal:** nunca coloque CPF, e-mail ou nome nos atributos. `cliente_id` (UUID) basta e é anonimizado.

- **Critérios de aceite:**
  - `SELECT count(*) FROM OrdemServicoEvent SINCE 1 hour ago` retorna dados após exercitar a API.
  - Eventos de falha carregam `motivo` preenchido.
  - `duracao_segundos` presente nas transições.
  - Testes unitários usam `NoopMetricas` e continuam verdes.
- **Estimativa:** G
- **Depende de:** F3-4.2

---

### F3-4.6 — Dashboards

> ⏱️ **[Cortes 12 e 16](plano-10-dias.md#os-cortes)** — monte o dashboard **pela UI**, não por Terraform: as consultas NRQL abaixo são as mesmas, cole cada uma num widget. Mantenha **um** monitor Synthetic (requisito de uptime) e ignore o bloco de IaC.

- **Descrição:** Montar o painel que o enunciado descreve. Construa **como código** (Terraform, provider `newrelic`) no repo `infra-k8s` — assim o dashboard não se perde e demonstra maturidade.

- **Como fazer:**

  As consultas NRQL, prontas para colar:

  | Painel | NRQL |
  |---|---|
  | **Volume diário de OS** | `SELECT count(*) FROM OrdemServicoEvent WHERE evento = 'criada' AND resultado = 'sucesso' TIMESERIES 1 day SINCE 30 days ago` |
  | **Tempo médio por status** (min) | `SELECT average(duracao_segundos)/60 AS 'minutos' FROM OrdemServicoEvent WHERE evento = 'transicao' FACET status_anterior SINCE 7 days ago` |
  | **Erros e falhas nas integrações** | `SELECT count(*) FROM IntegracaoEvent WHERE resultado = 'falha' FACET integracao, motivo TIMESERIES SINCE 24 hours ago` |
  | **Falhas no processamento de OS** | `SELECT count(*) FROM OrdemServicoEvent WHERE resultado = 'falha' FACET motivo SINCE 24 hours ago` |
  | **Latência das APIs (p50/p95/p99)** | `SELECT percentile(duration, 50, 95, 99) FROM Transaction WHERE appName = 'oficina-api-prod' TIMESERIES SINCE 6 hours ago` |
  | **Endpoints mais lentos** | `SELECT average(duration) FROM Transaction WHERE appName = 'oficina-api-prod' FACET name SINCE 1 hour ago LIMIT 10` |
  | **Taxa de erro HTTP** | `SELECT percentage(count(*), WHERE error IS true) FROM Transaction WHERE appName = 'oficina-api-prod' TIMESERIES` |
  | **CPU dos pods** | `SELECT average(cpuUsedCores) FROM K8sContainerSample WHERE containerName = 'api' FACET podName TIMESERIES SINCE 1 hour ago` |
  | **Memória dos pods** | `SELECT average(memoryWorkingSetBytes)/1e6 AS 'MB' FROM K8sContainerSample WHERE containerName = 'api' FACET podName TIMESERIES` |
  | **Réplicas (prova do HPA)** | `SELECT uniqueCount(podName) FROM K8sContainerSample WHERE containerName = 'api' TIMESERIES SINCE 1 hour ago` |
  | **Healthcheck / uptime** | `SELECT percentage(count(*), WHERE result = 'SUCCESS') FROM SyntheticCheck WHERE monitorName = 'oficina-health' SINCE 24 hours ago` |
  | **Latência da autenticação (Lambda)** | `SELECT average(duration) FROM AwsLambdaInvocation WHERE entityName LIKE '%auth-token%' TIMESERIES` |

  Terraform:

  ```hcl
  provider "newrelic" {
    account_id = var.new_relic_account_id
    api_key    = var.new_relic_api_key    # user key, não a license key
    region     = "US"
  }

  resource "newrelic_one_dashboard" "oficina" {
    for_each    = local.ambientes
    name        = "Oficina — ${each.key}"
    permissions = "public_read_only"

    page {
      name = "Negócio"

      widget_line {
        title  = "Volume diário de ordens de serviço"
        row    = 1
        column = 1
        width  = 6
        height = 3
        nrql_query {
          query = "SELECT count(*) FROM OrdemServicoEvent WHERE evento = 'criada' AND resultado = 'sucesso' TIMESERIES 1 day SINCE 30 days ago"
        }
      }

      widget_bar {
        title  = "Tempo médio por status (minutos)"
        row    = 1
        column = 7
        width  = 6
        height = 3
        nrql_query {
          query = "SELECT average(duracao_segundos)/60 FROM OrdemServicoEvent WHERE evento = 'transicao' FACET status_anterior SINCE 7 days ago"
        }
      }
      # ... demais widgets
    }

    page {
      name = "Infraestrutura e APIs"
      # latência, CPU, memória, réplicas
    }
  }
  ```

  **Synthetics** para o uptime (também via Terraform):

  ```hcl
  resource "newrelic_synthetics_monitor" "health" {
    name             = "oficina-health"
    type             = "SIMPLE"
    uri              = "${var.gateway_url}/health/ready"
    period           = "EVERY_5_MINUTES"
    status           = "ENABLED"
    locations_public = ["AWS_US_EAST_1", "AWS_SA_EAST_1"]
    validation_string = "UP"
  }
  ```

- **Critérios de aceite:**
  - Dashboard com no mínimo os 3 painéis que o enunciado nomeia + latência + CPU/memória + uptime.
  - Todos os painéis com dados reais (não vazios) no momento da gravação.
  - Dashboard versionado como código no `infra-k8s`.
  - Link do dashboard (permalink público ou read-only) no README.
- **Estimativa:** G
- **Depende de:** F3-4.3, F3-4.5

---

### F3-4.7 — Alertas

> ⏱️ **[Corte 13](plano-10-dias.md#os-cortes)** — apenas a condição **"Falhas no processamento de ordens de serviço"**, a única que o enunciado nomeia. Crie pela UI, com destino de e-mail, e **dispare de verdade** antes de gravar.

- **Descrição:** Requisito explícito: "alertas para falhas no processamento de ordens de serviço". Um alerta que nunca disparou não vale nada — teste antes da entrega.

- **Como fazer:**

  ```hcl
  resource "newrelic_alert_policy" "oficina" {
    for_each            = local.ambientes
    name                = "Oficina — ${each.key}"
    incident_preference = "PER_CONDITION"
  }

  # O alerta que o enunciado pede nominalmente
  resource "newrelic_nrql_alert_condition" "falha_os" {
    policy_id                    = newrelic_alert_policy.oficina.id
    name                         = "Falhas no processamento de ordens de serviço"
    type                         = "static"
    enabled                      = true
    aggregation_window           = 60
    aggregation_method           = "event_flow"
    aggregation_delay            = 120
    violation_time_limit_seconds = 3600

    nrql {
      query = "SELECT count(*) FROM OrdemServicoEvent WHERE resultado = 'falha'"
    }

    critical {
      operator              = "above"
      threshold             = 3        # mais de 3 falhas
      threshold_duration    = 300      # em 5 minutos
      threshold_occurrences = "ALL"
    }
  }

  resource "newrelic_nrql_alert_condition" "latencia" {
    policy_id = newrelic_alert_policy.oficina.id
    name      = "Latência p95 acima de 1s"
    type      = "static"
    nrql { query = "SELECT percentile(duration, 95) FROM Transaction WHERE appName = 'oficina-api-${each.key}'" }
    critical {
      operator           = "above"
      threshold          = 1
      threshold_duration = 300
      threshold_occurrences = "ALL"
    }
  }

  resource "newrelic_nrql_alert_condition" "pods_indisponiveis" {
    policy_id = newrelic_alert_policy.oficina.id
    name      = "Menos de 2 pods da API disponíveis"
    type      = "static"
    nrql { query = "SELECT uniqueCount(podName) FROM K8sContainerSample WHERE containerName = 'api'" }
    critical {
      operator           = "below"
      threshold          = 2
      threshold_duration = 300
      threshold_occurrences = "ALL"
    }
  }

  # Para onde a notificação vai
  resource "newrelic_notification_destination" "email" {
    name = "time-oficina"
    type = "EMAIL"
    property {
      key   = "email"
      value = var.email_alertas
    }
  }
  ```

  Some a isso um alerta sobre o monitor Synthetic (uptime) e um sobre a taxa de erro da Lambda `auth-token`.

  > **Teste o alerta de verdade.** Force falhas: chame `POST /v1/work-orders` com um `cliente_id` inexistente 5 vezes em um minuto e observe o incidente abrir e o e-mail chegar. Grave isso — é uma das cenas mais convincentes do vídeo, e prova que a cadeia inteira (evento → NRQL → condição → notificação) funciona.

- **Critérios de aceite:**
  - Política com no mínimo 3 condições, uma delas de falha no processamento de OS.
  - Destino de notificação configurado e **comprovadamente recebendo**.
  - Um incidente real aberto e resolvido, com print ou gravação.
  - Alertas versionados como código.
- **Estimativa:** M
- **Depende de:** F3-4.5, F3-4.6

---

### F3-4.8 — Teste de carga e validação do autoescalonamento

- **Descrição:** Sem tráfego, o HPA fica em 2 réplicas, os dashboards ficam vazios e não há o que mostrar no vídeo. Esta tarefa gera os dados que provam os requisitos de **escalabilidade** e de **monitoramento em tempo real** — e é o que transforma a demonstração de "olha o gráfico" em "olha o gráfico reagindo ao que acabei de fazer".

- **Como fazer:**

  **1. Gere um token e prepare o script.** Use `k6`, que produz métricas melhores que o `hey` e roda em container:

  ```javascript
  // scripts/carga.js
  import http from 'k6/http';
  import { check, sleep } from 'k6';

  const URL   = __ENV.GATEWAY_URL;
  const TOKEN = __ENV.TOKEN;

  export const options = {
    stages: [
      { duration: '1m', target: 20 },   // sobe devagar
      { duration: '3m', target: 80 },   // patamar: é aqui que o HPA reage
      { duration: '1m', target: 0 },    // desce, para ver o scale-down
    ],
    thresholds: {
      http_req_duration: ['p(95)<1500'],
      http_req_failed:   ['rate<0.01'],
    },
  };

  export default function () {
    const params = { headers: { Authorization: `Bearer ${TOKEN}` } };
    const res = http.get(`${URL}/v1/work-orders`, params);
    check(res, { 'status 200': (r) => r.status === 200 });
    sleep(0.5);
  }
  ```

  ```bash
  TOKEN=$(curl -s -X POST "$GATEWAY_URL/auth/token" \
    -H 'Content-Type: application/json' -d '{"cpf":"52998224725"}' | jq -r .access_token)

  docker run --rm -i -e GATEWAY_URL="$GATEWAY_URL" -e TOKEN="$TOKEN" \
    grafana/k6 run - < scripts/carga.js
  ```

  **2. Observe em três telas ao mesmo tempo** (é esta a cena do vídeo):

  ```bash
  kubectl -n oficina-prod get hpa api -w      # réplicas subindo de 2 → 4 → 5
  kubectl -n oficina-prod top pods            # CPU real por pod
  kubectl -n oficina-prod get pods -w         # pods novos ficando Ready
  ```

  **3. Gere também dados de negócio**, não só tráfego de leitura. Um script que cria OS e faz transições ao longo de alguns dias (ou com timestamps retroativos no banco) é o que popula "volume diário" e "tempo médio por status" — um gráfico com um único ponto não demonstra nada.

  ```bash
  # rode algumas vezes ao longo dos dias anteriores à gravação
  for i in $(seq 1 30); do
    curl -s -X POST "$GATEWAY_URL/v1/work-orders" -H "Authorization: Bearer $TOKEN_FUNCIONARIO" \
      -H 'Content-Type: application/json' -d @scripts/os-exemplo.json > /dev/null
  done
  ```

  **4. Provoque falhas de propósito** para popular o painel de erros e disparar o alerta de F3-4.7: chame com `cliente_id` inexistente, peça peça sem estoque, mande webhook com assinatura errada.

  > **Ponto de atenção:** o `authorizer_result_ttl_in_seconds = 300` faz o mesmo token ser cacheado no Gateway. Isso é ótimo para custo, mas significa que o teste de carga **não** exercita a Lambda authorizer. Se quiser medir o caminho de autenticação, gere um token novo a cada iteração ou baixe o TTL temporariamente.
  >
  > **Cuidado com o HPA e o `metrics-server`:** se `kubectl top pods` não responder, o HPA fica com `<unknown>` no target e nunca escala. Valide isso **antes** de rodar a carga, não durante.

- **Critérios de aceite:**
  - HPA sai de 2 para ≥ 4 réplicas sob carga e volta a 2 depois (respeitando o `stabilizationWindowSeconds`).
  - `p95 < 1,5 s` e taxa de erro < 1% durante o patamar.
  - Dashboards de volume diário e tempo médio por status com **múltiplos pontos**, não um só.
  - Painel de erros populado e alerta de falha de OS disparado.
  - Gravação da tela com HPA e New Relic lado a lado guardada para o vídeo.
- **Estimativa:** M
- **Depende de:** F3-4.6, F3-3.5

---

## E5 — Modelagem de banco de dados

> Enunciado: *"Melhorar e documentar a modelagem do banco de dados, garantindo consistência e performance"* + *"Justificativa formal para a escolha do banco de dados e ajustes no modelo relacional, com diagramas ER e explicação dos relacionamentos"*. Há **duas** entregas aqui: mexer no modelo (E5) e documentá-lo (E6).

### F3-5.1 — Revisão do modelo relacional

- **Descrição:** Aplicar as correções levantadas na revisão: índices ausentes, coluna `status` do cliente, CPF normalizado, timezone e constraints de integridade.

- **Como fazer:**

  Crie `internal/infra/database/migrations/000005_fase3_modelagem.up.sql` (e o `.down.sql` correspondente — golang-migrate exige o par):

  ```sql
  -- ═══════════════════════════════════════════════════════════════════
  -- 1. Status do cliente — exigido pela Lambda de autenticação
  -- ═══════════════════════════════════════════════════════════════════
  ALTER TABLE clientes
    ADD COLUMN status varchar(20) NOT NULL DEFAULT 'ativo';

  ALTER TABLE clientes
    ADD CONSTRAINT chk_clientes_status
    CHECK (status IN ('ativo', 'inativo', 'bloqueado'));

  COMMENT ON COLUMN clientes.status IS
    'Situação do cadastro. Somente clientes ativos obtêm token de autenticação.';

  -- ═══════════════════════════════════════════════════════════════════
  -- 2. CPF/CNPJ normalizado — a Lambda consulta por dígitos, sem máscara
  --    Coluna GERADA: o banco mantém sincronizada, é impossível divergir
  -- ═══════════════════════════════════════════════════════════════════
  ALTER TABLE clientes
    ADD COLUMN cpf_cnpj_digitos varchar(14)
    GENERATED ALWAYS AS (regexp_replace(cpf_cnpj, '[^0-9]', '', 'g')) STORED;

  CREATE UNIQUE INDEX ux_clientes_cpf_digitos ON clientes (cpf_cnpj_digitos);

  COMMENT ON COLUMN clientes.cpf_cnpj_digitos IS
    'CPF/CNPJ somente dígitos, derivado de cpf_cnpj. Usado no login por CPF.';

  -- ═══════════════════════════════════════════════════════════════════
  -- 3. Índices nas foreign keys
  --    No PostgreSQL, FK NÃO cria índice automaticamente (ao contrário do
  --    MySQL). Sem eles, todo JOIN e todo DELETE em cascata varre a tabela.
  -- ═══════════════════════════════════════════════════════════════════
  CREATE INDEX ix_veiculos_cliente          ON veiculos (cliente_id);
  CREATE INDEX ix_os_cliente                ON ordens_servico (cliente_id);
  CREATE INDEX ix_os_veiculo                ON ordens_servico (veiculo_id);
  CREATE INDEX ix_os_responsavel            ON ordens_servico (usuario_responsavel_id);
  CREATE INDEX ix_itens_os_servicos_os      ON itens_os_servicos (os_id);
  CREATE INDEX ix_itens_os_servicos_servico ON itens_os_servicos (servico_id);
  CREATE INDEX ix_itens_os_pecas_os         ON itens_os_pecas (os_id);
  CREATE INDEX ix_itens_os_pecas_peca       ON itens_os_pecas (peca_id);
  CREATE INDEX ix_historicos_os             ON historicos_status (os_id, alterado_em DESC);
  CREATE INDEX ix_usuarios_papel            ON usuarios (papel_id);

  -- ═══════════════════════════════════════════════════════════════════
  -- 4. Índice composto para a listagem de OS (a query mais executada)
  --    A F2-2.1 ordena por prioridade de status e depois criado_em ASC.
  -- ═══════════════════════════════════════════════════════════════════
  CREATE INDEX ix_os_status_criado ON ordens_servico (status_id, criado_em ASC);

  -- ═══════════════════════════════════════════════════════════════════
  -- 5. Timezone — timestamp sem timezone é uma bomba-relógio.
  --    O pod roda em UTC, o RDS em UTC, mas o dashboard e o relatório de
  --    "volume DIÁRIO" são em America/Sao_Paulo: 3h de deslocamento.
  --    timestamptz guarda o instante absoluto; a conversão vira apresentação.
  -- ═══════════════════════════════════════════════════════════════════
  ALTER TABLE clientes
    ALTER COLUMN criado_em     TYPE timestamptz USING criado_em     AT TIME ZONE 'UTC',
    ALTER COLUMN atualizado_em TYPE timestamptz USING atualizado_em AT TIME ZONE 'UTC';

  ALTER TABLE ordens_servico
    ALTER COLUMN criado_em     TYPE timestamptz USING criado_em     AT TIME ZONE 'UTC',
    ALTER COLUMN atualizado_em TYPE timestamptz USING atualizado_em AT TIME ZONE 'UTC',
    ALTER COLUMN aprovado_em   TYPE timestamptz USING aprovado_em   AT TIME ZONE 'UTC',
    ALTER COLUMN reprovado_em  TYPE timestamptz USING reprovado_em  AT TIME ZONE 'UTC',
    ALTER COLUMN iniciado_em   TYPE timestamptz USING iniciado_em   AT TIME ZONE 'UTC',
    ALTER COLUMN finalizado_em TYPE timestamptz USING finalizado_em AT TIME ZONE 'UTC',
    ALTER COLUMN entregue_em   TYPE timestamptz USING entregue_em   AT TIME ZONE 'UTC';

  ALTER TABLE historicos_status
    ALTER COLUMN alterado_em TYPE timestamptz USING alterado_em AT TIME ZONE 'UTC';

  -- ═══════════════════════════════════════════════════════════════════
  -- 6. Consistência: valores que não fazem sentido negativos
  -- ═══════════════════════════════════════════════════════════════════
  ALTER TABLE pecas
    ADD CONSTRAINT chk_pecas_estoque_nao_negativo CHECK (estoque_atual >= 0),
    ADD CONSTRAINT chk_pecas_preco_nao_negativo   CHECK (preco >= 0);

  ALTER TABLE itens_os_pecas
    ADD CONSTRAINT chk_itens_quantidade_positiva CHECK (quantidade > 0);

  ALTER TABLE ordens_servico
    ADD CONSTRAINT chk_os_valor_nao_negativo CHECK (valor_total >= 0);

  -- Uma OS não pode ter a mesma peça lançada duas vezes: some na quantidade
  CREATE UNIQUE INDEX ux_itens_os_pecas ON itens_os_pecas (os_id, peca_id);
  CREATE UNIQUE INDEX ux_itens_os_servicos ON itens_os_servicos (os_id, servico_id);
  ```

  **Armadilhas conhecidas:**

  1. **`CREATE INDEX CONCURRENTLY` não funciona aqui.** Ele não pode rodar dentro de uma transação, e o driver Postgres do golang-migrate envolve cada migration em uma. Com tabelas pequenas, o `CREATE INDEX` normal leva milissegundos — sem problema. Se algum dia a base crescer, faça esse tipo de índice fora do migrate.
  2. **`ALTER COLUMN ... TYPE` reescreve a tabela** e pega `ACCESS EXCLUSIVE LOCK`. Em uma base de demonstração, instantâneo. Em produção real, exigiria estratégia de coluna nova + backfill — mencione isso na RFC-002, mostra que você sabe a diferença.
  3. **A coluna gerada exige PostgreSQL 12+.** Estamos no 15, ok. Se algum `INSERT` do código tentar escrever em `cpf_cnpj_digitos`, o Postgres rejeita — confira os repositórios.
  4. **`ux_itens_os_pecas` pode falhar** se a base de seed já tiver duplicatas. Rode antes: `SELECT os_id, peca_id, count(*) FROM itens_os_pecas GROUP BY 1,2 HAVING count(*) > 1;`

  **Comprovação de ganho (material excelente para o vídeo):**

  ```sql
  EXPLAIN (ANALYZE, BUFFERS)
  SELECT o.*, s.nome_status FROM ordens_servico o
  JOIN status_ordens s ON s.id = o.status_id
  WHERE s.nome_status NOT IN ('finalizada','entregue')
  ORDER BY o.criado_em ASC LIMIT 20;
  ```

  Rode **antes** e **depois** da migration e guarde as duas saídas: `Seq Scan` → `Index Scan`, com a diferença de `execution time`. Isso é a "justificativa formal com garantia de performance" saindo do discurso para o número.

  **Ajustes no código que a migration exige:**
  - `internal/domain/entity/client.go`: adicionar `Status string`.
  - `internal/infra/repository/client_repository.go`: incluir `status` nos `SELECT`/`INSERT`/`UPDATE`.
  - `docs/openapi.yaml`: campo `status` no schema `Cliente` (enum) e regenerar.
  - Migration de seed: garantir que os clientes existentes fiquem `ativo` (o `DEFAULT` já cuida) e criar **um cliente inativo** no seed, para demonstrar o 403 no vídeo.

- **Critérios de aceite:**
  - `migrate up` e `migrate down` funcionam nos dois sentidos.
  - `EXPLAIN ANALYZE` da listagem mostra uso de índice.
  - Cliente inativo não obtém token (403 na Lambda).
  - CPF com e sem máscara resolvem para o mesmo cliente.
  - Testes de integração (`scripts/test-integration.sh`) continuam verdes.
- **Estimativa:** G
- **Depende de:** —

---

### F3-5.3 — Seeds e credenciais em ambiente de nuvem

- **Descrição:** As migrations `000002` e `000004` inserem dados de seed — inclusive **três usuários com senhas conhecidas e publicadas no README** (`admin@oficina.com` / `Admin@123`). Isso é adequado para desenvolvimento local e **inaceitável em um ambiente exposto à internet por um API Gateway**. Como as migrations rodam no boot em qualquer ambiente, esses usuários vão para produção automaticamente. Ao mesmo tempo, a demonstração **precisa** de dados: um cliente ativo e um inativo com CPFs válidos.

- **Como fazer:**

  **1. Separe seed de estrutura.** Migrations de schema rodam em todo lugar; seed de demonstração, não. Duas abordagens:

  - **Recomendada:** manter as migrations de seed, mas com senhas **não determinísticas** em produção. Troque o hash fixo por um placeholder e force a troca no primeiro login, ou crie o admin fora da migration, via um comando administrativo executado uma vez:

    ```bash
    kubectl -n oficina-prod run seed-admin --rm -i --restart=Never \
      --image=docker.io/problematheu/oficina-api:latest \
      --env="DB_HOST=..." -- /app/api seed-admin --email=admin@oficina.com
    # imprime uma senha aleatória uma única vez, e sai
    ```

  - **Alternativa mais simples:** manter a migration como está e, logo após o primeiro deploy em produção, **rodar um `UPDATE` trocando os hashes** por senhas geradas, guardadas no Secrets Manager. Menos elegante, mas resolve em 5 minutos.

  Em qualquer caso: **remova as senhas do README** ou deixe claríssimo que valem apenas para o ambiente local com `docker compose`.

  **2. Crie a migration de dados de demonstração** (`000006_seed_clientes_demo`), com CPFs que **passam** na validação de dígitos verificadores — CPF inventado não serve, a Lambda vai rejeitar antes de consultar o banco:

  ```sql
  INSERT INTO clientes (id, nome, cpf_cnpj, email, telefone, status) VALUES
    (gen_random_uuid(), 'Maria Silva',   '529.982.247-25', 'maria@exemplo.com',  '11999990001', 'ativo'),
    (gen_random_uuid(), 'Carlos Pereira','111.444.777-35', 'carlos@exemplo.com', '11999990002', 'inativo')
  ON CONFLICT (cpf_cnpj) DO NOTHING;
  ```

  Confira cada CPF antes de commitar — se o dígito verificador estiver errado, o fluxo do vídeo quebra ao vivo:

  ```bash
  # validador rápido, usando o pacote que você acabou de escrever
  go run ./cmd/validar-cpf 52998224725
  ```

  O cliente `inativo` é o que demonstra o requisito "consultar a existência **e o status** do cliente" — sem ele, o 403 não tem como ser mostrado.

  **3. Dados históricos para os dashboards.** "Volume diário de OS" com 30 dias de janela precisa de OS espalhadas no tempo. Um script de seed que insere OS com `criado_em` retroativo resolve:

  ```sql
  INSERT INTO ordens_servico (id, cliente_id, veiculo_id, status_id, numero, criado_em, finalizado_em)
  SELECT gen_random_uuid(), c.id, v.id, s.id,
         'OS-2026-' || lpad(g::text, 5, '0'),
         now() - (g || ' days')::interval,
         now() - (g || ' days')::interval + '4 hours'::interval
  FROM generate_series(1, 30) g
  CROSS JOIN LATERAL (SELECT id FROM clientes LIMIT 1) c
  CROSS JOIN LATERAL (SELECT id FROM veiculos LIMIT 1) v
  CROSS JOIN LATERAL (SELECT id FROM status_ordens WHERE nome_status = 'entregue') s;
  ```

  > Isso popula o banco, mas **não** gera os `OrdemServicoEvent` do New Relic (que só nascem quando o use case roda). Para os dashboards, prefira exercitar a API de verdade ao longo dos dias que antecedem a gravação — ver [F3-4.8](#f3-48--teste-de-carga-e-validação-do-autoescalonamento). Use o SQL apenas para os relatórios que leem do banco.

  **4. Rollback de migration em produção.** `migrate down` em produção apaga dados. Antes de qualquer migration destrutiva no RDS: snapshot manual (`aws rds create-db-snapshot`), e prefira migrations aditivas (adicionar coluna, nunca remover na mesma versão). A migration de F3-5.1 é aditiva exceto pelos `ALTER COLUMN TYPE` — que são reversíveis, mas reescrevem a tabela.

- **Critérios de aceite:**
  - Nenhuma senha de produção conhecida publicamente; README deixa claro o escopo local das credenciais de exemplo.
  - Existem um cliente **ativo** e um **inativo**, com CPFs que passam na validação.
  - Base de produção com OS suficientes para os dashboards não ficarem vazios.
  - Procedimento de snapshot antes de migration documentado no README do `app`.
- **Estimativa:** M
- **Depende de:** F3-5.1, F3-2.1

---

### F3-5.2 — Documentar o modelo e justificar a escolha do banco

- **Descrição:** Produzir a **RFC-002** com a justificativa formal e o diagrama ER com os relacionamentos explicados.

- **Como fazer:**

  Boa parte já existe: `docs/architecture-decisions.md` traz a ADR-001 (PostgreSQL) com alternativas comparadas e quatro argumentos sólidos. Reaproveite **como base** e eleve a RFC com o que a ADR não tem:

  1. **Requisitos não-funcionais quantificados** — volume esperado de OS/dia, concorrência, retenção, RPO/RTO.
  2. **Por que gerenciado (RDS) e não auto-hospedado** — backup automático, patching, Multi-AZ, Performance Insights; o custo de operar Postgres em pod (PVC, backup, upgrade) não se paga.
  3. **Comparação com as alternativas gerenciadas da AWS** — Aurora Serverless v2 (escala melhor, custa mais e tem cold start), DynamoDB (descartado: o domínio é relacional, com JOINs e transações multi-tabela), RDS MySQL.
  4. **Ajustes no modelo** — a lista de F3-5.1, cada um com o problema que resolve.
  5. **Diagrama ER** — o de [arquitetura.md](arquitetura.md#3-modelo-entidade-relacionamento-alvo).
  6. **Explicação dos relacionamentos** — a tabela de [arquitetura.md](arquitetura.md#explicação-dos-relacionamentos-para-a-rfc-002), com destaque para as duas decisões que a banca costuma questionar: **preço copiado** nos itens da OS (imutabilidade histórica) e **`cliente_id` na OS** mesmo existindo em `veiculos` (o titular na abertura precisa ser preservado).
  7. **Evidência de performance** — os dois `EXPLAIN ANALYZE`.

  Para gerar o ER a partir do banco real (mais confiável que desenhar à mão):

  ```bash
  # SchemaSpy: HTML completo com ER e relacionamentos
  docker run -v "$PWD/docs/er:/output" --network host schemaspy/schemaspy:latest \
    -t pgsql11 -host localhost -port 5433 -db tech_challenge_db \
    -u postgres -p "$POSTGRES_PASSWORD" -s public
  ```

  Mantenha também a versão Mermaid no Markdown (versionável e legível em diff).

- **Critérios de aceite:** `docs/rfcs/RFC-002-banco-de-dados.md` completa, com ER, tabela de relacionamentos, comparativo de alternativas e os `EXPLAIN` de antes/depois; linkada no README.
- **Estimativa:** M
- **Depende de:** F3-5.1

---

## E6 — Documentação arquitetural

> Entregável avaliado diretamente. O enunciado lista cinco itens: componentes, sequência, RFCs, ADRs e justificativa do banco. Nenhum é opcional.

### F3-6.1 — Diagramas de componentes e de sequência

- **Descrição:** Os diagramas já estão desenhados em [arquitetura.md](arquitetura.md) — falta promovê-los para a documentação oficial e exportá-los como imagem para o PDF.

- **Como fazer:**
  - Copiar as seções 1 e 2 de `arquitetura.md` para `docs/arquitetura.md` no repo da aplicação, e o trecho relevante para o README de cada repo (o enunciado pede "diagrama da arquitetura específica daquele repositório" — o do `infra-db` mostra RDS/SG/subnets, não o cluster inteiro).
  - Revisar os diagramas contra o que **realmente** foi construído ao fim do E3 — plano e realidade divergem, e um diagrama desatualizado desconta ponto.
  - Exportar em PNG/SVG via [mermaid.live](https://mermaid.live) para o PDF do portal.

- **Critérios de aceite:** Diagrama de componentes com nuvem, APIs, banco e monitoramento; dois diagramas de sequência (autenticação e abertura de OS); ambos fiéis à implementação; exportados como imagem.
- **Estimativa:** M
- **Depende de:** E2, E3, E4

---

### F3-6.2 — RFCs

- **Descrição:** O enunciado quer RFCs para **decisões técnicas relevantes** (dá os exemplos: escolha da nuvem, do banco e da estratégia de autenticação). RFC é um documento de **proposta discutida**: contexto, opções, trade-offs, decisão, consequências e plano de reversão.

- **Como fazer:**

  Crie `docs/rfcs/` com um template e no mínimo estas quatro:

  | RFC | Título | Conteúdo essencial |
  |---|---|---|
  | **RFC-001** | Escolha da nuvem | AWS × GCP × Azure. Critérios: EKS/GKE/AKS, custo de gateway serverless, free tier, familiaridade do time, e o fato de a Fase 2 já ter Terraform de AWS. Consequências: lock-in em API Gateway e Lambda; mitigação: a aplicação continua um container portável. |
  | **RFC-002** | Banco de dados | F3-5.2. |
  | **RFC-003** | Estratégia de autenticação | O núcleo. Ver detalhamento abaixo. |
  | **RFC-004** | Divisão em quatro repositórios | Monorepo × polirepo; o que ganha (deploy independente, blast radius menor, CI mais rápido) e o que perde (mudança atômica cross-repo, duplicação de código, orquestração de ordem de apply); como o contrato SSM mitiga. |

  Template (`docs/rfcs/TEMPLATE.md`):

  ```markdown
  # RFC-00X — <Título>

  | | |
  |---|---|
  | **Status** | Proposta \| Aceita \| Substituída por RFC-00Y |
  | **Autores** | |
  | **Data** | AAAA-MM-DD |
  | **Decisores** | |

  ## 1. Contexto e problema
  ## 2. Requisitos e restrições
  ## 3. Opções consideradas
  ### 3.1 Opção A — ...   (prós, contras, custo estimado)
  ## 4. Decisão
  ## 5. Consequências
  ### 5.1 Positivas  ### 5.2 Negativas  ### 5.3 Riscos e mitigações
  ## 6. Plano de reversão
  ## 7. Referências
  ```

  **O que a RFC-003 precisa cobrir** (é a que a banca vai ler com mais atenção):
  - Por que **JWT stateless** e não sessão em Redis.
  - Por que **HS256 com segredo compartilhado** e não RS256/JWKS — e sob que condição a decisão se inverteria (mais de um consumidor do token, ou o segredo precisando ser distribuído fora do nosso controle).
  - Por que **Lambda Authorizer** e não o JWT Authorizer nativo (não valida HS256) e não Cognito (custo de migrar a base de usuários; o enunciado pede uma Function autoral).
  - Por que **CPF sem senha** é aceitável aqui: o cliente consulta apenas as **próprias** OS, dado de baixa sensibilidade; e por que **não seria** aceitável para operações de escrita ou financeiras — deixando explícito que a evolução natural é OTP por SMS/e-mail. Reconhecer a limitação vale mais do que fingir que ela não existe.
  - **Cache do authorizer (300 s)** e a janela de revogação que ele cria.
  - **Coexistência** dos dois emissores e como o claim `tipo` evita escalonamento de privilégio.

- **Critérios de aceite:** 4 RFCs completas seguindo o template, com opções e trade-offs reais (não só a opção escolhida), linkadas no README.
- **Estimativa:** G
- **Depende de:** E2, E3

---

### F3-6.3 — ADRs

- **Descrição:** `docs/architecture-decisions.md` já tem ADRs da Fase 1 (banco, router…). Estender com as decisões **permanentes** da Fase 3. O enunciado dá os exemplos "padrão de comunicação" e "uso de HPA".

- **Como fazer:**

  Mantenha o formato existente (Contexto → Alternativas → Decisão → Consequências) e acrescente:

  | ADR | Decisão |
  |---|---|
  | ADR-008 | **Padrão de comunicação: REST síncrono** — por quê, e sob que carga migraríamos para eventos (SQS/EventBridge) na notificação e no webhook |
  | ADR-009 | **HPA por CPU a 50%, 2–5 réplicas** — por que CPU e não requests/s ou métrica customizada; por que `minReplicas: 2` (disponibilidade durante rolling update); relação com o Cluster Autoscaler |
  | ADR-010 | **Dois emissores de token com claim `tipo`** — cliente × funcionário |
  | ADR-011 | **SSM Parameter Store como contrato entre repos** — em vez de `terraform_remote_state` |
  | ADR-012 | **Manifestos Kubernetes no repo da aplicação** — não no repo de infra |
  | ADR-013 | **Duplicar a validação de CPF** entre app e Lambda — em vez de biblioteca compartilhada |
  | ADR-014 | **`TargetGroupBinding` em vez de Ingress** — para quebrar a dependência circular |
  | ADR-015 | **Migrations no boot da aplicação** com advisory lock — em vez de Job/init container |

  > **RFC × ADR, na prática:** RFC é a **discussão** (várias opções, ainda em aberto, pode ser rejeitada); ADR é o **registro** de uma decisão já tomada e seu porquê. Uma RFC aceita costuma gerar uma ou mais ADRs. Explicitar essa distinção no início do documento mostra domínio do processo.

- **Critérios de aceite:** No mínimo 8 ADRs novas, no formato existente, incluindo obrigatoriamente comunicação e HPA.
- **Estimativa:** M
- **Depende de:** E2, E3, E4

---

### F3-6.4 — READMEs dos quatro repositórios

- **Descrição:** O enunciado é literal sobre o conteúdo: propósito, tecnologias, passos de execução e deploy, diagrama específico daquele repo e link para Swagger/Postman.

- **Como fazer:**

  Estrutura mínima, idêntica nos quatro:

  ```markdown
  # <nome do repo>
  > Uma frase dizendo o que este repositório faz e o que ele NÃO faz.

  ## Papel na arquitetura
  <diagrama Mermaid específico + link para o diagrama geral>

  ## Tecnologias
  <tabela nome | versão | uso>

  ## Pré-requisitos
  ## Execução local
  ## Deploy
  <o que dispara, para onde vai, como acompanhar>

  ## Variáveis e segredos
  ## Contrato com os outros repositórios
  <o que este repo publica na SSM e o que ele consome>

  ## Links
  - Repositórios irmãos: ...
  - Swagger / Postman: ...
  - Dashboard New Relic: ...
  - Deploy ativo: <URL do API Gateway>
  ```

  O README atual da aplicação tem 32 KB e é muito bom — **não jogue fora**. Atualize as seções de arquitetura, acrescente a autenticação por CPF, o link do Gateway e o do dashboard, e mova o histórico da Fase 2 para uma seção recolhida.

  Checklist específico exigido: **"Links para os deploys ativos (se aplicável)"** — coloque a URL do API Gateway de produção no topo dos quatro READMEs.

- **Critérios de aceite:** Os 4 READMEs com todas as seções; diagramas próprios; links de Swagger/Postman e do deploy ativo; alguém de fora consegue subir o projeto seguindo apenas o README.
- **Estimativa:** G
- **Depende de:** F3-6.1

---

## E7 — Entrega

### F3-7.1 — Vídeo de demonstração (≤ 15 min)

- **Descrição:** Seis itens obrigatórios de demonstração em 15 minutos — sem roteiro e sem ensaio, não cabe.

- **Como fazer:** roteiro completo, com cronometragem, em [entrega.md](entrega.md#roteiro-do-vídeo).

- **Critérios de aceite:** Vídeo publicado (YouTube/Vimeo, público ou não listado), ≤ 15 min, demonstrando os seis itens do enunciado.
- **Estimativa:** M
- **Depende de:** tudo

---

### F3-7.2 — PDF do portal e compartilhamento dos repositórios

- **Descrição:** Documento único com links dos 4 repos, do vídeo, das documentações e a confirmação do `soat-architecture`.

- **Como fazer:** conteúdo e checklist em [entrega.md](entrega.md#conteúdo-do-pdf).

- **Critérios de aceite:** PDF submetido; `soat-architecture` com acesso confirmado nos **4** repositórios (print do aceite do convite).
- **Estimativa:** P
- **Depende de:** F3-7.1

---

### F3-7.3 — Controle de custo e teardown

- **Descrição:** A infraestrutura desta fase custa dinheiro real e continua cobrando depois que o vídeo foi gravado. Esta tarefa evita a fatura desagradável.

- **Como fazer:**

  **Antes de tudo** — crie o orçamento no primeiro dia, não no último:

  ```hcl
  resource "aws_budgets_budget" "mensal" {
    name         = "oficina-mensal"
    budget_type  = "COST"
    limit_amount = "50"
    limit_unit   = "USD"
    time_unit    = "MONTHLY"

    notification {
      comparison_operator        = "GREATER_THAN"
      threshold                  = 50          # 50% do orçamento
      threshold_type             = "PERCENTAGE"
      notification_type          = "FORECASTED"
      subscriber_email_addresses = [var.email_alertas]
    }
    notification {
      comparison_operator        = "GREATER_THAN"
      threshold                  = 90
      threshold_type             = "PERCENTAGE"
      notification_type          = "ACTUAL"
      subscriber_email_addresses = [var.email_alertas]
    }
  }
  ```

  **Estimativa de custo (us-east-1, 24×7):**

  Com a [infraestrutura compartilhada](#estratégia-de-ambientes-leia-antes-de-escrever-hcl) — um cluster, uma VPC, um ALB atendendo os dois ambientes:

  | Recurso | Qtd | Custo/mês | Observação |
  |---|---|---|---|
  | EKS control plane | 1 | ~US$ 73 | o maior item; duplicar ambientes dobraria isto |
  | EC2 `t3.medium` | 2 | ~US$ 60 | nós do cluster, compartilhados |
  | NAT Gateway | 1 | ~US$ 32 + tráfego | |
  | RDS `db.t4g` | 2 | ~US$ 30 | `micro` em homolog, `small` em prod |
  | ALB interno | 1 | ~US$ 16 | dois listeners, um por ambiente |
  | API Gateway | 2 | ~US$ 1 | US$ 1,00 por milhão de requisições |
  | Lambda | 4 | ~US$ 0 | free tier cobre o volume da demo |
  | Secrets Manager | ~6 | ~US$ 2,40 | US$ 0,40 por segredo |
  | **Total** | | **~US$ 215/mês** | ≈ **US$ 7,20/dia** |

  > ⏱️ Com os cortes do [plano de 10 dias](plano-10-dias.md#custo-real-da-nuvem) — sem NAT, sem ALB, nós `t3.small` e um ambiente: **~US$ 41 pelos 9 dias**.

  > Com workspaces duplicando VPC, EKS e NAT, o mesmo desenho custaria **~US$ 390/mês**. A decisão de arquitetura de ambientes vale, sozinha, quase metade da fatura.

  **Estratégia recomendada:** provisionar → validar → gravar → **destruir**. Deixe o Terraform aplicado apenas nos dias de trabalho ativo. Um ciclo de 5 dias custa ~US$ 36.

  **Ordem do destroy** (inversa à do apply, senão trava):

  ```bash
  # 1. Aplicação (libera os pods registrados no target group)
  kubectl delete -k k8s/overlays/prod
  # 2. Lambdas e rotas do Gateway
  cd oficina-lambda-auth/terraform && terraform destroy
  # 3. Banco
  cd ../../oficina-infra-db/terraform && terraform destroy
  # 4. Cluster, gateway e rede (o mais demorado, ~15 min)
  cd ../../oficina-infra-k8s/terraform && terraform destroy
  ```

  **Depois do destroy, confira o que sobrou** — Terraform não remove o que não criou:

  ```bash
  aws ec2 describe-volumes --filters Name=status,Values=available          # EBS órfãos (PVC)
  aws ec2 describe-addresses --query 'Addresses[?AssociationId==null]'      # Elastic IPs soltos
  aws elbv2 describe-load-balancers                                         # ALB criado por Ingress
  aws logs describe-log-groups --query 'logGroups[].logGroupName'           # log groups
  ```

  > Se o `destroy` do `infra-k8s` travar removendo a VPC, quase sempre é um ENI de Lambda, um ALB criado pelo Ingress ou um security group criado pelo Load Balancer Controller — nenhum deles está no state. Remova-os à mão e rode o destroy de novo.

  **Para a entrega, com o ambiente destruído:** o enunciado pede "links para os deploys ativos (**se aplicável**)". Deixe claro no README que o ambiente é provisionado sob demanda e que o vídeo comprova o funcionamento, com o `terraform apply` reproduzível em ~25 min. É a postura honesta e nenhum avaliador razoável espera que um aluno mantenha um EKS ligado indefinidamente.

- **Critérios de aceite:** Budget criado e notificando; procedimento de teardown documentado no README do `infra-k8s`; após o destroy, `Cost Explorer` mostra o gasto caindo a ~zero; nenhum recurso órfão.
- **Estimativa:** P
- **Depende de:** E3
