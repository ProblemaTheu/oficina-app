# Planejamento — Tech Challenge Fase 3

> Elevar a API de Oficina Mecânica a **nível de operação corporativa**: autenticação serverless por CPF atrás de um API Gateway, quatro repositórios com CI/CD independente, infraestrutura 100% em Terraform na AWS, observabilidade com New Relic e documentação arquitetural formal (RFCs, ADRs, diagramas de componentes/sequência/ER).

Este diretório traduz o enunciado (`pdfs/13SOAT - Fase 3 - Tech Challenge.pdf`) em épicos e tarefas executáveis, com critérios de aceite e **passo a passo técnico** — de modo que qualquer pessoa do time execute sem depender de contexto verbal.

---

## Índice

| Documento | Conteúdo |
|-----------|----------|
| [README.md](README.md) | Este arquivo — resumo executivo, decisões, mapa de entregáveis, épicos e DoD |
| [arquitetura.md](arquitetura.md) | Arquitetura-alvo: diagramas de componentes, sequência, ER e topologia de repositórios |
| [backlog.md](backlog.md) | Backlog detalhado — **é aqui que está o "como fazer"**, com código e comandos |
| **[plano-10-dias.md](plano-10-dias.md)** | ⏱️ **Plano de execução real** — 1 pessoa, 01/09→10/09, ~42 h. Os 9 cortes, cronograma diário e ordem de sacrifício. **Comece por aqui.** |
| [roadmap.md](roadmap.md) | Sprints, grafo de dependências, ordem de `terraform apply`, riscos e custo (referência do "jeito certo com tempo") |
| [entrega.md](entrega.md) | Checklist de entrega, roteiro do vídeo e conteúdo do PDF do portal |

---

## Resumo executivo

A Fase 2 entregou uma aplicação Go em Clean Architecture rodando em Kubernetes local (kind), com Terraform, CI/CD no GitHub Actions, webhook HMAC e notificação por e-mail. **Tudo em um único repositório e sem nuvem real.**

A Fase 3 muda o eixo do trabalho: sai do "faz funcionar" e entra no "opera de verdade". São cinco frentes:

1. **Autenticação serverless + API Gateway** — o cliente se autentica por **CPF** em uma Lambda que consulta a base e emite um JWT; o API Gateway passa a ser a única porta de entrada e valida o token antes de encaminhar para o cluster.
2. **Quatro repositórios** — a aplicação, a Lambda, a infra de Kubernetes e a infra de banco passam a viver separados, cada um com seu CI/CD, `main` protegida e deploy automático de homologação e produção.
3. **Nuvem real (AWS)** — EKS com HPA, RDS PostgreSQL gerenciado, API Gateway e Lambda, tudo provisionado por Terraform com backend remoto compartilhado.
4. **Observabilidade** — New Relic: APM na aplicação e na Lambda, métricas de CPU/memória do cluster, logs JSON correlacionados por `trace.id`, dashboards de negócio e alertas.
5. **Documentação arquitetural formal** — diagrama de componentes, diagramas de sequência, RFCs, ADRs e justificativa do banco com modelo ER revisado.

> **A regra de ouro desta fase:** nada do que já funciona precisa ser reescrito. A aplicação Go muda pouco — ganha validação do token do Lambda, logs estruturados e instrumentação. O grosso do trabalho é **infraestrutura, automação e documentação**.

---

## Decisões técnicas já tomadas

| Tema | Decisão | Racional |
|------|---------|----------|
| **Nuvem** | **AWS**, conta pessoal / Free Tier | A Fase 2 já deixou `infra/environments/aws` esboçado (VPC + EKS + RDS). Conta pessoal permite criar IAM Roles e OIDC — necessários para o deploy sem segredos de longa duração. |
| **Observabilidade** | **New Relic** | Free tier **permanente** (100 GB/mês, 1 usuário full). Datadog só oferece trial de 14 dias — risco de expirar antes da gravação do vídeo. Agente Go oficial + integração Kubernetes via Helm + extensão para Lambda. |
| **Divisão de repositórios** | Repo atual **vira o repo da aplicação e é renomeado** `tech-challenge-1` → `oficina-app`; 3 repos novos são criados (`oficina-lambda-auth`, `oficina-infra-k8s`, `oficina-infra-db`) | Preserva histórico, CI e secrets já existentes, e deixa os quatro repositórios com nomenclatura consistente na entrega. O rename tem 8 pontos de ajuste interno — procedimento em [plano-10-dias](plano-10-dias.md#renomear-tech-challenge-1--oficina-app). O Terraform migra para os repos de infra. |
| **Gateway** | **AWS API Gateway HTTP API** (v2) | Mais barato (~1/3 do REST API), latência menor, suporte nativo a JWT/Lambda Authorizer e VPC Link para alcançar o cluster privado. |
| **Assinatura do JWT** | **HS256 com segredo compartilhado** no AWS Secrets Manager | A aplicação já valida HS256 (`internal/infra/http/middleware/jwt.go`). Evita reescrever o middleware. Evolução para RS256/JWKS fica registrada em ADR. |
| **Validação no gateway** | **Lambda Authorizer (REQUEST)** | O JWT Authorizer nativo do API Gateway exige um emissor OIDC com JWKS público — incompatível com HS256. O authorizer customizado é a forma correta de validar HS256 na borda. |
| **Ambiente de homologação** | ⏱️ **Dispensado** por orientação no fórum da disciplina — apenas `prod` é provisionado; a branch `homolog` permanece com CI, sem deploy. **Guarde o print/link do post e cite no README dos 4 repos** | O PDF do enunciado pede deploy automático das duas branches; a dispensa veio do fórum. Sem a evidência registrada, a divergência conta contra. Ver [corte nº 10](plano-10-dias.md#corte-nº-10-em-detalhe--como-fazer). |
| **Ambientes** | **Infra compartilhada + recursos por ambiente**, via `for_each` sobre `local.ambientes`. Com a dispensa acima, `local.ambientes = toset(["prod"])` — o `for_each` fica, com um elemento | Mantém o desenho correto e reversível em uma linha. Se os dois ambientes voltassem, VPC, cluster e balanceador continuariam compartilhados: duplicá-los custaria +US$ 73/mês só de control plane. Ver [Estratégia de ambientes](backlog.md#estratégia-de-ambientes-leia-antes-de-escrever-hcl). |
| **Workspaces do Terraform** | **Nenhum.** Os três repos com Terraform usam o workspace `default` | Workspace serve para separar ambientes; com um ambiente só, ele vira apenas uma forma nova de aplicar no lugar errado. (No desenho de dois ambientes, `lambda-auth` usaria workspaces e os repos de infra não — ver [F3-0.1](backlog.md#f3-01--backend-remoto-do-terraform-s3-com-lock-nativo).) |
| **Ligação API Gateway → EKS** | **VPC Link + ALB interno criado pelo Terraform** + `TargetGroupBinding` no cluster — ⏱️ **simplificado para NLB público + header compartilhado no [plano de 10 dias](plano-10-dias.md#os-9-cortes)** | Elimina a dependência circular (o Gateway precisaria do ARN de um ALB que só existe depois do deploy da app). Ver [F3-3.5](backlog.md#f3-35--alb-interno-target-group-e-vpc-link). |
| **Comunicação entre repos de infra** | **AWS SSM Parameter Store** (não `terraform_remote_state`) | Acoplamento fraco: cada repo publica seus outputs como parâmetros e lê os dos outros por nome, sem precisar de acesso ao state alheio nem de permissão no bucket do outro time. |
| **Autenticação de cliente × usuário interno** | **Coexistem** — `POST /v1/auth/login` (funcionário, e-mail+senha) continua; o fluxo por CPF é do **cliente final** | São atores distintos. O claim `tipo` no JWT (`cliente` \| `usuario`) discrimina o escopo de acesso. Ver ADR-010. |

---

## Mapa de entregáveis (enunciado → artefato)

| Requisito do enunciado | Artefato | Repositório | Épico |
|---|---|---|---|
| Implementar um API Gateway | `aws_apigatewayv2_api` — uma API por ambiente, stage `$default` | infra-k8s | E3 |
| Proteger rotas sensíveis com autenticação via CPF | Lambda Authorizer + claim `tipo=cliente` no middleware | lambda-auth + app | E2 |
| Function serverless: validar CPF | `internal/cpf` (validador de dígitos verificadores) | lambda-auth | E2 |
| Function serverless: consultar existência e **status** do cliente | Query em `clientes` + coluna `status` (nova) | lambda-auth + app | E2, E5 |
| Function serverless: gerar e devolver JWT | `POST /auth/token` → JWT HS256 | lambda-auth | E2 |
| 4 repositórios separados com CI/CD | 4 repos no GitHub | todos | E1 |
| `main` protegida, PR obrigatório | Rulesets do GitHub | todos | E1 |
| Deploy automático de homologação e produção | Workflows `cd-homolog.yml` / `cd-prod.yml` | todos | E1 |
| Banco de dados gerenciado | RDS PostgreSQL 15 Multi-AZ opcional | infra-db | E3 |
| Cluster Kubernetes com escalabilidade | EKS + HPA + Cluster Autoscaler | infra-k8s | E3 |
| Terraform para provisionamento | `infra-k8s` e `infra-db` inteiros | infra-k8s, infra-db | E3 |
| Integração Datadog/New Relic | Agente Go, `nri-bundle` via Helm, layer da Lambda | app, infra-k8s, lambda | E4 |
| Monitorar latência, CPU/memória, healthchecks, uptime | Dashboards + Synthetics | app | E4 |
| Alertas para falhas no processamento de OS | Alert Condition NRQL | app | E4 |
| Logs estruturados JSON com correlação | `slog` + `nrslog` + `X-Correlation-Id` | app | E4 |
| Dashboards (volume diário de OS, tempo médio por status, erros) | Custom events `OrdemServicoEvent` + NRQL | app | E4 |
| Diagrama de Componentes | [arquitetura.md](arquitetura.md#1-diagrama-de-componentes) | app (README) | E6 |
| Diagrama de Sequência (auth e abertura de OS) | [arquitetura.md](arquitetura.md#2-diagramas-de-sequência) | app (README) | E6 |
| RFCs | `docs/rfcs/RFC-00X-*.md` | app | E6 |
| ADRs | `docs/architecture-decisions.md` (estender) | app | E6 |
| Justificativa do banco + ER + relacionamentos | `docs/rfcs/RFC-002-banco-de-dados.md` + [arquitetura.md](arquitetura.md#3-modelo-entidade-relacionamento-alvo) | app | E5, E6 |
| README por repo (propósito, tecnologias, execução, diagrama, link Swagger/Postman) | 4 × `README.md` | todos | E6 |
| Dockerfiles (quando aplicável) | `app`: Dockerfile; `lambda-auth`: zip `provided.al2023` (documentado); infra: N/A | todos | E1 |
| Vídeo ≤ 15 min | Link no README + PDF | — | E7 |
| PDF com links dos 4 repos, vídeo, docs e `soat-architecture` | [entrega.md](entrega.md) | — | E7 |

---

## Épicos

| ID | Épico | Objetivo | Prioridade |
|----|-------|----------|------------|
| **E0** | Fundação e contratos | Contas e quotas com prazo de terceiros, backend remoto do Terraform, convenções de nomes/tags, contrato do JWT e OIDC com a AWS | Bloqueante |
| **E1** | Split em 4 repositórios + CI/CD | Criar os repos, migrar conteúdo, proteger `main`, pipelines de homolog/prod e suíte de segurança nos quatro | Alta |
| **E2** | Autenticação serverless por CPF | Lambda de token, Lambda Authorizer, ajustes no middleware da aplicação | Alta |
| **E3** | Infraestrutura AWS (Terraform) | VPC, EKS, RDS, API Gateway, VPC Link, ALB interno, Secrets Manager | Alta |
| **E4** | Observabilidade (New Relic) | APM, infra do cluster, logs JSON correlacionados, dashboards e alertas | Alta |
| **E5** | Modelagem de banco de dados | Índices, `status` do cliente, `timestamptz`, constraints e documentação ER | Média |
| **E6** | Documentação arquitetural | Componentes, sequência, RFCs, ADRs e os 4 READMEs | Alta |
| **E7** | Entrega | Vídeo, PDF do portal, `soat-architecture`, teardown de custo | Alta |

Detalhamento completo, com comandos e código, em [backlog.md](backlog.md).

---

## Achados da revisão do estado atual

Levantamento feito sobre o código da Fase 2 — cada item vira tarefa concreta:

1. **O banco não tem um único índice secundário.** As migrations `000001`–`000004` criam 11 tabelas e **nenhum `CREATE INDEX`**. Nem as foreign keys (`veiculos.cliente_id`, `ordens_servico.cliente_id`, `historicos_status.os_id`, `itens_os_pecas.os_id`…) estão indexadas — no PostgreSQL FKs **não** ganham índice automático, ao contrário do MySQL. Isso significa *sequential scan* em toda listagem de OS e em todo `JOIN`. É o achado mais forte para o requisito "melhorar e documentar a modelagem do banco, garantindo consistência e **performance**". → **F3-5.1**

2. **Não existe `status` no cliente.** O enunciado pede que a Lambda "consulte a existência **e o status** do cliente na base". A tabela `clientes` tem apenas `id, nome, cpf_cnpj, email, telefone, criado_em, atualizado_em`. → **F3-5.1**

3. **`cpf_cnpj` é `varchar(20)` sem normalização.** Se o cadastro aceita `123.456.789-09` e a Lambda consulta `12345678909`, o login falha silenciosamente. Precisa de coluna normalizada (só dígitos) com índice único. → **F3-5.1**

4. **Todos os timestamps são `timestamp` sem timezone.** Com a aplicação no cluster em UTC e o RDS em UTC, mas relatórios e dashboards em America/Sao_Paulo, o "volume diário de OS" do dashboard sairá deslocado em 3 h. → **F3-5.1**

5. **Logs não são estruturados.** `cmd/api/main.go` usa `log.Printf` (texto puro) e os use cases usam `slog` com o handler **default de texto**. O enunciado exige **JSON com correlação entre requisições**. → **F3-4.1**

6. **Não há request/correlation ID.** Nenhum middleware gera ou propaga um identificador de requisição; é impossível correlacionar log de handler, use case e repositório. → **F3-4.1**

7. **O JWT não tem `iss`, `aud` nem `tipo`.** Os claims atuais são `sub, email, nome, iat, exp`. Com dois emissores (aplicação e Lambda) e dois tipos de sujeito (funcionário e cliente), o token precisa se identificar. → **F3-0.2**

8. **`jwtSecret()` tem um fallback inseguro hardcoded** (`"changeme-insecure-default-secret"`) em dois lugares (`auth_usecase.go` e `middleware/jwt.go`). Em produção isso é uma falha de segurança: se a variável não for injetada, a API aceita tokens forjados por qualquer um que leia o repositório. → **F3-2.5**

9. **O `docker-compose.yml` e o overlay `k8s/overlays/aws` têm placeholders `SUBSTITUIR_PELO_*`.** Precisam virar valores reais vindos do Terraform via SSM/External Secrets. → **F3-3.6**

10. **Os três repositórios novos nascem sem nenhuma verificação de segurança.** O `security.yml` (govulncheck, gosec, Trivy, Sonar) existe só no repo atual — justamente os repos que passam a manipular IAM, security groups e segredos ficariam descobertos. Falta também detecção de segredo commitado, com quatro repos multiplicando a chance. → **F3-1.7**

11. **As migrations de seed criam usuários com senha publicada no README** (`admin@oficina.com` / `Admin@123`). Como as migrations rodam no boot em qualquer ambiente, esses usuários vão para produção — que estará exposta pela internet através do API Gateway. → **F3-5.3**

12. **A notificação por e-mail regride em produção.** O `SMTPNotifier` da Fase 2 aponta para `SUBSTITUIR_PELO_SMTP_REAL`; como o envio é assíncrono e não derruba a transição, a falha é **silenciosa**. Uma funcionalidade da fase anterior quebrando na seguinte é o tipo de coisa que a banca nota. → **F3-3.7**

13. **O CD atual publica no Docker Hub.** Continua válido (decisão da Fase 2), mas o EKS puxando de um registry público sem credencial está sujeito a rate limit do Docker Hub. Avaliar `imagePullSecrets` ou migrar para ECR. → **F3-1.5**

---

## Critérios de aceite globais (Definition of Done da fase)

A Fase 3 só é considerada entregue quando:

- [ ] Existem **4 repositórios** no GitHub, cada um com `README.md`, CI/CD funcional e `main` protegida com PR obrigatório.
- [ ] `POST {api_gateway_url}/auth/token` com um CPF válido de cliente **ativo** devolve um JWT; CPF inválido → 400; inexistente → 404; inativo → 403.
- [ ] Uma rota protegida da aplicação **só** responde quando chamada através do API Gateway com o token da Lambda; chamada sem token → 401 **no gateway**, antes de chegar ao cluster.
- [ ] `terraform apply` nos repos `infra-db` e `infra-k8s` provisiona VPC, EKS, RDS, API Gateway e VPC Link do zero, e `terraform destroy` remove tudo.
- [ ] O HPA escala a aplicação sob carga no EKS (2 → ≥4 réplicas), comprovado por `kubectl get hpa -w` e pelo gráfico de réplicas no New Relic.
- [ ] Nenhum segredo real versionado nos 4 repos (`gitleaks` limpo, inclusive no histórico) e suíte de segurança ativa em todos.
- [ ] Nenhum placeholder `SUBSTITUIR_PELO_*` sobrevive nos manifestos; a notificação por e-mail funciona (ou a limitação está documentada).
- [ ] Nenhuma credencial de produção com senha conhecida publicamente.
- [ ] O New Relic exibe: APM da aplicação e da Lambda, CPU/memória dos pods, logs JSON com `trace.id`, e um dashboard com volume diário de OS, tempo médio por status e erros de integração.
- [ ] Existe pelo menos **um alerta** configurado para falha no processamento de ordens de serviço, com política de notificação testada.
- [ ] `docs/` contém diagrama de componentes, diagramas de sequência (autenticação e abertura de OS), no mínimo **3 RFCs** e **ADRs** atualizados.
- [ ] O modelo ER está documentado com os relacionamentos explicados e a justificativa formal da escolha do banco.
- [ ] Vídeo ≤ 15 min publicado demonstrando: autenticação por CPF, pipeline rodando, deploy automatizado, consumo de API protegida, dashboard ao vivo e logs/traces.
- [ ] PDF submetido no portal com os links e a confirmação do `soat-architecture` nos **4** repositórios.
- [ ] Cada pessoa do time consegue rodar `kubectl get nodes` contra o cluster (não só a role do CI).
- [ ] Custo sob controle: `terraform destroy` executado ou orçamento monitorado no AWS Budgets após a gravação.
