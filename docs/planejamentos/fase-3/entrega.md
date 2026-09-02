# Entrega — Fase 3

Checklist de fechamento, roteiro do vídeo e conteúdo do PDF do portal.

---

## Checklist antes de submeter

### Repositórios
- [ ] Os **4** repositórios existem e estão públicos (ou com o `soat-architecture` convidado)
- [ ] `soat-architecture` adicionado como colaborador nos 4 e **convite aceito** (print guardado)
- [ ] `main` protegida nos 4, com PR obrigatório — comprovado por uma tentativa de push rejeitada
- [ ] Print/link do post do fórum que dispensa o ambiente de homologação, citado nos 4 READMEs e no PDF
- [ ] CI verde no último commit de cada `main`
- [ ] Nenhum segredo real versionado (`gitleaks detect` limpo nos 4)
- [ ] Nenhum `.tfstate` versionado
- [ ] `README.md` completo em cada repo, com propósito, tecnologias, execução, deploy, diagrama próprio e links

### Funcionalidade
- [ ] `POST {gateway}/auth/token` com CPF de cliente ativo → 200 com JWT
- [ ] CPF inválido → 400 · inexistente → 404 · cliente inativo → 403
- [ ] Rota protegida sem token → **401 no gateway**
- [ ] Rota protegida com token do Lambda → 200
- [ ] Login de funcionário continua funcionando
- [ ] Webhook de orçamento com HMAC continua funcionando
- [ ] Notificação de status por e-mail demonstrada (em nuvem via SES, **ou** no ambiente local com Mailpit — com a escolha registrada em ADR)
- [ ] HPA escala sob carga (2 → ≥4 réplicas) e volta a 2 depois

### Observabilidade
- [ ] APM da aplicação recebendo dados
- [ ] Cluster visível no Kubernetes explorer, com CPU e memória
- [ ] Logs em JSON, com `correlation_id` e `trace.id`
- [ ] Dashboard com volume diário de OS, tempo médio por status, erros de integração, latência, CPU/memória e uptime
- [ ] Alerta de falha no processamento de OS **disparado e recebido** ao menos uma vez

### Documentação
- [ ] Diagrama de componentes (nuvem, APIs, banco, monitoramento)
- [ ] Diagrama de sequência da autenticação
- [ ] Diagrama de sequência da abertura de OS
- [ ] ≥ 4 RFCs (nuvem, banco, autenticação, divisão de repos)
- [ ] ≥ 8 ADRs novas, incluindo padrão de comunicação e HPA
- [ ] Justificativa formal do banco + ER + explicação dos relacionamentos
- [ ] Link para Swagger/OpenAPI e coleção Postman em cada README

### Encerramento
- [ ] Vídeo publicado, ≤ 15 min, com os 6 itens demonstrados
- [ ] PDF submetido no portal
- [ ] `terraform destroy` executado e recursos órfãos verificados
- [ ] AWS Budgets confirmando queda do gasto

---

## Roteiro do vídeo

**Duração-alvo: 14 minutos.** Grave por blocos e edite — tentar fazer em uma tomada única sempre estoura o tempo.

> ⏱️ **Executando o [plano de 10 dias](plano-10-dias.md)?** O ambiente de homologação foi dispensado (corte nº 10). No bloco de CI/CD, demonstre o deploy automático em produção e **diga em uma frase** que homologação foi dispensada por orientação do fórum — em vez de deixar o avaliador notar a ausência sozinho. Confira também a [ordem de sacrifício](plano-10-dias.md#ordem-de-sacrifício-se-atrasar) antes de gravar.

### Preparação (antes de ligar a gravação)

- [ ] Ambiente `prod` provisionado e **aquecido** (chame `/auth/token` uma vez para eliminar o cold start)
- [ ] Postman aberto com a coleção e as variáveis preenchidas
- [ ] Dashboard do New Relic aberto em uma aba, com dados das últimas horas
- [ ] Terminal com fonte grande (≥ 16 pt) e prompt limpo
- [ ] Um cliente **ativo** e um **inativo** no banco, CPFs anotados
- [ ] Uma PR já preparada, pronta para merge (não crie do zero na gravação)
- [ ] Script de carga pronto (`hey -z 120s -c 50 ...`)
- [ ] Notificações do sistema desligadas

### Blocos

| Tempo | Bloco | O que mostrar |
|---|---|---|
| **0:00–1:00** | Abertura e arquitetura | Nome, integrantes, o problema da Fase 3. Diagrama de componentes na tela: "cliente entra pelo API Gateway, o Lambda autentica por CPF, o cluster EKS roda a aplicação, o RDS persiste, o New Relic observa." Não leia o diagrama inteiro — mostre o caminho de uma requisição. |
| **1:00–2:00** | Os 4 repositórios | Tela dividida ou navegação rápida: os 4 repos no GitHub, o que cada um contém. Mostrar `Settings → Rules` com `main` protegida. **Tente um `git push origin main` e mostre a rejeição.** |
| **2:00–4:00** | **Autenticação por CPF** | No Postman: `POST /auth/token` com CPF válido → token. Decodificar em jwt.io mostrando os claims (`tipo: cliente`, `iss`, `exp`). Depois os três erros: CPF inválido → 400, inexistente → 404, **cliente inativo → 403**. Mostrar o código da Lambda por 15 s — validação de CPF e consulta ao banco. |
| **4:00–5:30** | **Consumo das APIs protegidas** | `GET /v1/work-orders` **sem** token → 401. Enfatizar: *"esse 401 veio do API Gateway, a requisição nem chegou ao cluster"* — e mostrar o log do authorizer no New Relic. Com token → 200, retornando só as OS daquele cliente. Depois, token de funcionário criando uma OS. |
| **5:30–8:00** | **Pipeline CI/CD e deploy automatizado** | Mergear a PR preparada. Acompanhar o Actions ao vivo: testes, lint, build da imagem, push, `kubectl apply`, `rollout status`. Mostrar o `environment: prod` **aguardando aprovação** e aprovar. Em paralelo, `kubectl get pods -w` mostrando o rolling update. Fechar mostrando o marcador de deploy no New Relic. |
| **8:00–10:00** | **Infraestrutura como código** | `terraform state list` nos dois repos de infra, mostrando o volume de recursos gerenciados. Mostrar o comentário automático de `plan` em uma PR. `kubectl get nodes` e `kubectl get hpa`. Rodar a carga (`hey`) e mostrar o HPA subindo de 2 para 4/5 réplicas em tempo real. |
| **10:00–13:00** | **Dashboard e análise ao vivo** | Dashboard do New Relic com dados reais. Percorrer: volume diário de OS; tempo médio por status; erros de integração; latência p95; CPU/memória dos pods (**com o pico da carga que acabou de rodar** — é a melhor cena do vídeo, pois conecta ação e observação); contagem de réplicas subindo. Abrir um **trace distribuído**: Gateway → aplicação → query no banco, apontando onde está o tempo. Abrir **logs in context**: JSON com `correlation_id` e `trace.id`. Provocar falhas de OS e mostrar o **incidente do alerta abrindo** e o e-mail chegando. |
| **13:00–14:00** | Documentação e encerramento | Passar rápido por RFCs, ADRs, diagrama de sequência e ER. Recapitular os requisitos atendidos. Agradecer. |

**Erros que custam pontos:**
- Ler o código linha a linha — mostre o comportamento, não a sintaxe.
- Passar 3 minutos esperando o `terraform apply`. Grave o apply antes e mostre o resultado.
- Dashboard vazio. Gere tráfego **horas antes** para os painéis diários terem dados.
- Esquecer de demonstrar o **cliente inativo** — é o "consultar o status do cliente" do enunciado, explicitamente pedido.
- Estourar 15 minutos. O corte é rígido.

---

## Conteúdo do PDF

> Documento único a submeter no Portal do Aluno.

---

**Tech Challenge — Fase 3 — 13SOAT**
**Grupo:** *[nomes e RMs dos integrantes]*

### 1. Repositórios

| # | Repositório | Descrição | Link |
|---|---|---|---|
| 1 | `oficina-lambda-auth` | Functions serverless de autenticação por CPF (emissão de token e authorizer) | `https://github.com/ProblemaTheu/oficina-lambda-auth` |
| 2 | `oficina-infra-k8s` | Terraform: VPC, EKS, API Gateway, VPC Link, ALB, Secrets, New Relic | `https://github.com/ProblemaTheu/oficina-infra-k8s` |
| 3 | `oficina-infra-db` | Terraform: RDS PostgreSQL gerenciado | `https://github.com/ProblemaTheu/oficina-infra-db` |
| 4 | `oficina-app` | Aplicação principal em Go + manifestos Kubernetes | `https://github.com/ProblemaTheu/tech-challenge-1` |

✅ **Usuário `soat-architecture` adicionado como colaborador nos 4 repositórios.**

### 2. Vídeo de demonstração

`[COLAR LINK DO YOUTUBE/VIMEO]` — duração: `[XX:XX]`

### 3. Documentações

| Documento | Link |
|---|---|
| Diagrama de componentes | `.../oficina-app/blob/main/docs/arquitetura.md#componentes` |
| Diagramas de sequência | `.../oficina-app/blob/main/docs/arquitetura.md#sequencia` |
| RFC-001 — Escolha da nuvem | `.../docs/rfcs/RFC-001-nuvem.md` |
| RFC-002 — Banco de dados (justificativa + ER) | `.../docs/rfcs/RFC-002-banco-de-dados.md` |
| RFC-003 — Estratégia de autenticação | `.../docs/rfcs/RFC-003-autenticacao.md` |
| RFC-004 — Divisão em quatro repositórios | `.../docs/rfcs/RFC-004-repositorios.md` |
| ADRs | `.../docs/architecture-decisions.md` |
| OpenAPI / Swagger | `.../docs/openapi.yaml` |
| Coleção Postman | `.../docs/postman_collection.json` |
| Dashboard New Relic | `[permalink read-only]` |

### 4. Arquitetura

`[INSERIR IMAGEM do diagrama de componentes, exportada de mermaid.live]`

**Recursos utilizados (AWS, us-east-1):**

> 🚨 **Confira item a item antes de gerar o PDF.** Esta lista precisa espelhar **o que você realmente construiu**. O avaliador abre o repositório: descrever VPC Link, NAT Gateway ou External Secrets num PDF cujo Terraform não os contém não é otimismo, é informação incorreta na entrega — e custa mais caro do que assumir a simplificação. A lista abaixo já reflete a [variante do plano de 10 dias](plano-10-dias.md#os-9-cortes); se você implementou o desenho completo, troque pelos itens entre parênteses.

- **Entrada:** API Gateway HTTP API (uma API por ambiente), access logs em JSON e throttling por ambiente
- **Autenticação:** duas AWS Lambda em Go (`provided.al2023`, arm64) — emissão de JWT por CPF e authorizer de borda no Gateway
- **Computação:** Amazon EKS 1.31, node group gerenciado (3 × `t3.small`), HPA por CPU a 50% (2–6 réplicas) *(desenho completo: 2–4 × `t3.medium` + Cluster Autoscaler)*
- **Rede:** VPC com 2 AZs; nós em subnet pública; exposição via `Service type: LoadBalancer` (NLB) com header compartilhado exigido pelo API Gateway *(desenho completo: NAT Gateway + VPC Link + ALB interno com `TargetGroupBinding`)*
- **Dados:** Amazon RDS PostgreSQL 15.7 (`db.t4g.micro`), criptografado, em subnet privada, com Performance Insights
- **Segredos:** AWS Secrets Manager, com o `Secret` do Kubernetes criado pelo Terraform *(desenho completo: External Secrets Operator)*; SSM Parameter Store como contrato entre os repositórios
- **Observabilidade:** New Relic — APM Go, `nri-bundle` no cluster, logs JSON correlacionados por `trace.id`, dashboard e alerta de falha no processamento de OS
- **Notificações:** porta `Notifier` com implementações SMTP e log; e-mail demonstrado no ambiente local (Mailpit) *(desenho completo: Amazon SES em produção)*
- **IaC:** Terraform 1.9 com backend S3 + lock em DynamoDB; ambientes por `for_each` sobre infraestrutura compartilhada
- **CI/CD:** GitHub Actions com autenticação OIDC (sem chaves de longa duração), `plan` em PR e `apply` em merge, deploy automático da `main` para produção
- **Ambientes:** apenas produção. O ambiente de homologação foi dispensado conforme orientação no fórum da disciplina em `[DATA]` (`[LINK DO POST]`); a branch `homolog` permanece no fluxo com PR obrigatório e CI bloqueante

**Simplificações assumidas e registradas em ADR** *(remova as que não se aplicarem)*: integração direta Gateway→NLB no lugar de VPC Link; nós em subnet pública sem NAT; HPA sem Cluster Autoscaler; `Secret` provisionado pelo Terraform no lugar do External Secrets Operator; notificação por e-mail demonstrada localmente. Cada uma está justificada em `docs/architecture-decisions.md` com o caminho de evolução para produção.

### 5. Observação sobre o ambiente

O ambiente é provisionado sob demanda a partir do Terraform (`terraform apply` completo em ~25 minutos) e destruído após as demonstrações, por controle de custo. O vídeo comprova o funcionamento end-to-end de toda a solução.

---
