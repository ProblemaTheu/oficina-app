# Plano de execução — 10 dias, uma pessoa, 01/09 → 10/09

> **Este documento substitui o [roadmap.md](roadmap.md) como plano de execução.** O roadmap continua válido como referência do "jeito certo com tempo"; aqui está o que cabe na realidade: **~40 horas, uma pessoa, orçamento apertado**.

---

## A conta que não fecha (e o que fazer sobre ela)

| | |
|---|---|
| Capacidade real | 7 dias × 3h + 3 dias (sáb, dom, feriado 07/09) × 7h ≈ **42 h** |
| Ambiente de homologação | **dispensado** por orientação no fórum da disciplina (corte nº 10) |
| Escopo do [backlog.md](backlog.md) | 43 tarefas ≈ **130 h** |
| Déficit | **~88 h** |

Não existe reorganização de cronograma que resolva 88 horas. O que existe é **cortar profundidade sem cortar requisito** — e é isso que este plano faz. Cada corte abaixo mantém o item do enunciado atendido, mas com a implementação mais barata que ainda demonstra o conceito.

> **Princípio que guia tudo aqui:** um trade-off consciente e documentado vale mais na avaliação do que uma implementação sofisticada pela metade. Toda vez que este plano corta algo, ele diz **como justificar o corte na entrega**. Faça isso — é o que separa "não deu tempo" de "decidimos assim porque".

---

## Os 9 cortes

| # | Corte | Economia | Como justificar na entrega |
|---|---|---|---|
| 1 | **VPC Link + ALB interno + `TargetGroupBinding`** → `Service type: LoadBalancer` (NLB público) e API Gateway fazendo `HTTP_PROXY` para ele, protegido por header secreto | **~6 h** + elimina o risco nº 1 do projeto | ADR: "o VPC Link é a topologia correta para produção; adotamos a integração direta com header compartilhado pelo prazo, e o caminho de evolução está documentado" |
| 2 | **NAT Gateway** → nós em subnet pública com IP público | ~2 h + **US$ 8** | ADR: nós públicos com SG restritivo; em produção real iriam para subnet privada com NAT |
| 3 | **Cluster Autoscaler** → 2 nós `t3.small` fixos, HPA escalando pods 2→6 dentro da capacidade | ~2 h | O enunciado pede "cluster com escalabilidade" — o HPA entrega isso. Cite o autoscaler como evolução na ADR-009 |
| 4 | **External Secrets Operator** → `kubernetes_secret` criado pelo Terraform lendo o Secrets Manager | ~2 h | Mesmo resultado funcional; o ESO agrega rotação automática, que não é requisito |
| 5 | **Amazon SES** → `NOTIFIER=log` em prod; e-mail demonstrado no ambiente local com Mailpit (que já funciona desde a Fase 2) | ~3 h + mata o risco do sandbox (aprovação de até 24 h) | README: "a integração SMTP é a mesma; muda apenas o provedor. Demonstrada localmente" |
| 6 | **Expansão de testes** → manter a suíte da Fase 2 verde; testes novos só para validação de CPF e authorizer | ~4 h | A Fase 3 não pede cobertura; a Fase 2 já entregou 85% nos use cases |
| 7 | **`git filter-repo`** → copiar arquivos para os repos novos | ~2 h | Ninguém avalia histórico de commit dos repos de infra |
| 8 | **checkov + Sonar/Trivy no repo da Lambda** → apenas `gitleaks` (todos) e `tfsec` (infra) | ~2 h | Segurança presente em todos os repos, com profundidade proporcional |
| 9 | **k6 com estágios** → loop `curl`/`hey` de 3 minutos | ~1 h | Só precisa provar que o HPA reage |
| 10 | **Ambiente de homologação** → apenas `prod` (autorizado no fórum da disciplina) | **~3 h** + US$ 4 | Cite o fórum no README; a branch `homolog` continua existindo com CI completo |

**Total economizado: ~27 h.** Escopo restante: **~36 h** em ~42 h disponíveis — **~6 h de folga**, a primeira que este plano tem.

### Corte nº 10 em detalhe — como fazer

O fórum da disciplina liberou dispensar o ambiente de homologação. É o único corte que **reduz escopo sem assumir risco técnico**: os outros nove trocam robustez por tempo; este simplesmente remove trabalho.

**No Terraform** (`infra-k8s` e `infra-db`), uma linha:

```hcl
locals { ambientes = toset(["prod"]) }   # era ["homolog", "prod"]
```

Mantenha o `for_each` em vez de remover — custa nada, e voltar atrás é trocar uma linha. Isso elimina: a segunda instância RDS (e sua espera de provisionamento), o segundo API Gateway com rotas e authorizer, o segundo namespace, o segundo overlay, o segundo conjunto de segredos e o segundo listener.

**Nos workflows**, `homolog` deixa de implantar e passa a ser branch de integração com CI completo:

```yaml
# ci.yml — roda em PR para qualquer branch protegida
on:
  pull_request:
    branches: [homolog, main]

# cd.yml — só main implanta
on:
  push:
    branches: [main]
```

**Por que manter a branch `homolog`:** ela ainda demonstra o fluxo `feature → homolog → main` com PR obrigatório e CI bloqueante, que é o que as *regras de proteção* do enunciado cobram. Você perde o ambiente, não o processo.

**⚠️ Dois efeitos colaterais que precisam de ação — senão viram bug:**

**(a) Tire `homolog` do gatilho de `push` dos workflows de Terraform.** O template de [F3-1.4](backlog.md#f3-14--cicd-dos-repositórios-de-infraestrutura) dispara `apply` em push para `homolog` **e** `main`. Sem ambiente de homologação, um merge em `homolog` rodaria `terraform apply` contra um ambiente inexistente — no melhor caso falha, no pior cria recursos órfãos que você paga sem saber:

```yaml
on:
  pull_request:
    branches: [homolog, main]   # plan + CI: mantém as duas
  push:
    branches: [main]            # apply: SÓ main
```

E remova a lógica `github.ref_name == 'main' && 'prod' || 'homolog'` de todos os workflows — o ambiente agora é sempre `prod`.

**(b) Os workspaces do Terraform perdem o propósito — em todos os repos.** A justificativa para o `lambda-auth` usar workspaces era "cada ambiente tem seu deploy independente". Com um ambiente só, não há o que separar. Elimine:

- o step *Selecionar workspace* e a variável `USA_WORKSPACES` de [F3-1.4](backlog.md#f3-14--cicd-dos-repositórios-de-infraestrutura) e [F3-1.6](backlog.md#f3-16--cicd-do-repositório-da-lambda);
- em [F3-2.3](backlog.md#f3-23--terraform-das-funções-e-das-rotas-no-api-gateway), troque `local.env = terraform.workspace` por `local.env = "prod"`.

Os três repos passam a usar o workspace `default`. Além de economizar tempo, isso mata a classe de erro mais cara de todas: aplicar no workspace errado sem perceber.

> ⚠️ **Guarde a evidência.** O PDF do enunciado diz "deploy automático das branches de homologação e produção"; a dispensa veio do fórum. Salve o print ou o link do post e **cite no README** dos quatro repositórios: *"ambiente de homologação dispensado conforme orientação no fórum da disciplina em [data] — [link]"*. Se o avaliador for pelo PDF, essa linha é a sua defesa. Mencione também no vídeo, em uma frase.

**Onde gastar as ~6 h de folga:** em **nada**. Este plano não tinha buffer algum e o sábado 05/09 é o dia crítico. Resista à tentação de reinstalar o VPC Link ou o SES — se sobrar tempo de verdade na quarta 09/09, use para **ensaiar o vídeo uma segunda vez**, que é o entregável de maior peso e o menos ensaiado.

---

## Custo real da nuvem

Com os cortes 1–3 (sem NAT, sem ALB, nós `t3.small`), ligando em **02/09** e destruindo em **10/09** — 9 dias, 24 h por dia:

| Recurso | Cálculo | Total 9 dias |
|---|---|---|
| EKS control plane | US$ 0,10/h × 216 h | **US$ 21,60** |
| ↳ ⚠️ se a versão do K8s estiver em *extended support* | US$ 0,60/h × 216 h | *US$ 129,60* — ver [F3-3.3](backlog.md#f3-33--cluster-eks-com-escalabilidade-e-addons) |
| 2× EC2 `t3.small` | US$ 0,0208/h × 2 × 216 h | US$ 9,00 |
| NLB | US$ 0,0225/h × 216 h | US$ 4,90 |
| 1× RDS `db.t4g.micro` | US$ 0,016/h × 216 h | US$ 3,45 |
| API Gateway + Lambda + Secrets | volume da demo | ~US$ 2,00 |
| **Total** | | **≈ US$ 41** |

**Onde o Free Tier ajuda** (conta AWS nova, primeiros 12 meses):

- **RDS**: 750 h/mês de `db.t4g.micro` + 20 GB → com um ambiente só, **a instância inteira sai de graça** (−US$ 3,45). Confira em *Billing → Free Tier* que está sendo aplicado.
- **EC2**: 750 h/mês de `t3.micro` → um dos nós pode ser `t3.micro` e sair de graça (−US$ 4,50), mas 1 GB de RAM é apertado para a aplicação + DaemonSet do New Relic. **Não arrisque** — o nó ficar sem memória no dia da gravação custa mais caro que US$ 4,50.
- **Lambda, API Gateway, CloudWatch**: cobertos pelo volume da demonstração.

**Piso realista: ~US$ 37,50** (com o Free Tier do RDS aplicado). Desse total, **US$ 21,60 é o control plane do EKS** — que é requisito do enunciado ("Cluster Kubernetes com escalabilidade"), não escolha de arquitetura. Não há como entregar a fase sem pagar isso.

> ❌ **Não tente economizar destruindo e recriando todo dia.** Um ciclo `destroy` + `apply` do EKS leva ~35 minutos de relógio. Fazer isso 8 vezes queima **4,5 horas** do seu orçamento de 42 h para economizar ~US$ 20. Você está trocando a hora mais escassa do projeto por o valor de um lanche.
>
> ✅ **Ligue em 02/09 e deixe ligado.** Além do tempo, isso resolve o problema dos dashboards: "volume diário de OS" precisa de dias acumulando dados. Ligando dia 2, você chega no dia 10 com 8 pontos no gráfico. Ligando dia 8, tem um ponto — e o painel não demonstra nada.

**Antes de qualquer `apply`:** crie o AWS Budget de US$ 60 com alerta em 50% ([F3-7.3](backlog.md#f3-73--controle-de-custo-e-teardown)). São 5 minutos e é a sua rede de segurança.

---

## Cronograma dia a dia

Cada dia tem um **entregável verificável**. Se o dia acabar sem ele, você está atrasado — consulte a [ordem de sacrifício](#ordem-de-sacrifício-se-atrasar) no mesmo dia, não no dia 9.

### Passo 0 — Máquina e credenciais (~30 min, antes do dia 1)

**Ferramentas** (macOS/Homebrew) — sem elas nada do resto roda:

```bash
brew install awscli go kustomize helm gh gitleaks golangci-lint terraform kubectl jq
```

**Credenciais AWS** — tem um ovo-e-galinha: não dá para criar usuário IAM pela CLI sem já ter credencial. Então o começo é pelo console:

1. Conta em [aws.amazon.com](https://aws.amazon.com) (pede cartão; cobrança por uso)
2. Console → **IAM** → *Users* → *Create user* → `<seu-nome>-cli` → *Attach policies directly* → **`AdministratorAccess`**
3. No usuário criado → *Security credentials* → *Create access key* → **Command Line Interface (CLI)**. O *secret* aparece **uma única vez** — guarde
4. `aws configure` → access key, secret, região **`us-east-1`**, formato **`json`**

> Não use as chaves da conta **root**: elas não têm como ser restringidas nem revogadas seletivamente, e vazam com o mesmo descuido que qualquer outra.

**Confirme antes de seguir:**

```bash
aws sts get-caller-identity     # deve devolver Account e Arn
terraform version               # anote: este número vai no TF_VERSION do CI
```

✅ **Entregável:** `aws sts get-caller-identity` responde, e você anotou a versão do Terraform.

---

### 🔴 Ter 01/09 — 3 h — Fundação (sem custo de nuvem)

**Faça nos primeiros 30 minutos, antes de qualquer outra coisa:**

```bash
# 1. Quota de vCPU — a aprovação leva de horas a 2 dias e SEM ela o HPA não escala
aws service-quotas request-service-quota-increase \
  --service-code ec2 --quota-code L-1216C47A --desired-value 32 --region us-east-1

# 2. AWS Budget — sua rede de segurança
aws budgets create-budget --account-id <SUA_CONTA> --budget \
  '{"BudgetName":"oficina","BudgetLimit":{"Amount":"60","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"}'
```

E crie a conta New Relic (guarde **license key**, **user key** e **account ID** — são três coisas diferentes).

Depois:
- [F3-1.1](backlog.md#f3-11--criar-os-repositórios-e-migrar-o-conteúdo) — criar os 3 repos novos (copiando arquivos, sem `filter-repo`) e **renomear `tech-challenge-1` → `oficina-app`** (ver procedimento abaixo)
- [F3-1.2](backlog.md#f3-12--proteção-de-branch-nos-4-repositórios) — branch protection nos 4 (`required_approving_review_count: 0`, você está sozinho) e criar a branch `homolog`
- ✅ [F3-0.1](backlog.md#f3-01--backend-remoto-do-terraform-s3-com-lock-nativo) + [F3-0.4](backlog.md#f3-04--oidc-entre-github-actions-e-aws-deploy-sem-segredos-de-longa-duração) — **bootstrap concluído** (S3 + OIDC + 4 roles). Sem DynamoDB: o Terraform 1.15 usa lock nativo do S3

✅ **Entregável:** `aws sts get-caller-identity` rodando num workflow do GitHub, sem nenhum secret de credencial AWS.

#### Renomear `tech-challenge-1` → `oficina-app`

Decisão do time: os quatro repositórios ficam com nomes consistentes. O GitHub mantém redirect do nome antigo, então links externos e o `git push` continuam funcionando — mas **oito coisas dentro do repositório não seguem o redirect** e precisam ser trocadas na mão. Reserve ~40 min.

**1. Renomeie no GitHub:** `Settings → General → Repository name` → `oficina-app`.

**2. Atualize o remote local:**

```bash
git remote set-url origin https://github.com/ProblemaTheu/oficina-app.git
```

**3. Module path do Go — 36 arquivos:**

```bash
go mod edit -module github.com/ProblemaTheu/oficina-app
grep -rl "problematheu/tech-challenge-1" --include="*.go" . \
  | xargs sed -i '' 's|problematheu/tech-challenge-1|ProblemaTheu/oficina-app|g'
go build ./... && go test ./... && golangci-lint run
```

**4. ⚠️ `ci.yml:110` — a que ninguém percebe.** O Job Summary corta o prefixo do módulo com uma string literal:

```python
nome_curto = pkg.replace("github.com/problematheu/tech-challenge-1/", "")
```

Isso **não quebra o build**: o summary passa a mostrar o caminho completo de cada pacote e ninguém liga a causa ao rename. Troque pelo path novo — ou, melhor, torne imune:

```python
nome_curto = pkg.split("/", 3)[-1]   # descarta host/org/repo, sobra o pacote
```

**5. Imagem no Docker Hub** — `tech-challenge-api` → `oficina-api` (o [CD do plano](backlog.md#f3-15--cd-da-aplicação-build--registry--eks) já usa o nome novo). Crie o repositório no Docker Hub **antes** do primeiro push, senão o job falha com `denied: requested access to the resource is denied`. Arquivos: `cd.yml` (4 ocorrências), `k8s/base/deployment.yaml`, `k8s/overlays/local/kustomization.yaml`.

**6. `security.yml`** — tag local da imagem no scan do Trivy: `tech-challenge-1:` → `oficina-app:` (3 ocorrências, linhas 197/203/213).

**7. `cd.yml:106`** — o default `EKS_CLUSTER_NAME || 'tech-challenge-eks'`. O plano usa `oficina`, e o CD novo lê o nome da SSM — então esse fallback some junto na reescrita do workflow.

**8. Documentação:** os dois blocos `git clone` do `README.md` (linhas 126 e 236) e o `k8s/README.md` (`docker build -t` / `kind load`).

**Confira que não sobrou nada:**

```bash
grep -rn "tech-challenge-1\|tech-challenge-api" \
  --exclude-dir=.git --exclude-dir=docs/planejamentos . || echo "limpo"
```

> `docs/planejamentos/fase-2/` **deve** manter as referências antigas — é registro histórico do que foi entregue naquela fase, não configuração ativa.
>
> **SonarCloud:** o `sonar-project.properties` não tem `projectKey` (vem do workflow), mas o SonarCloud indexa por repositório. Depois do rename, confira em *Administration → Update key* se o projeto continua ligado — senão a análise passa a criar um projeto novo e você perde o histórico.

---

### 🔴 Qua 02/09 — 3 h — Nuvem de pé (custo começa)

- [F3-3.2](backlog.md#f3-32--repositório-oficina-infra-k8s-vpc-e-rede) — VPC **sem NAT**, nós em subnet pública
- [F3-3.3](backlog.md#f3-33--cluster-eks-com-escalabilidade-e-addons) — EKS **1.36**, **2 nós `t3.small`**, addon `metrics-server`, **`access_entries` com seu usuário IAM** (senão você não usa `kubectl`), namespaces
- [F3-3.1](backlog.md#f3-31--repositório-oficina-infra-db-rds-postgresql-gerenciado) — RDS, **uma instância** (`local.ambientes = toset(["prod"])` — corte nº 10)

⚠️ **Primeiro apply**: rode `terraform apply -target=module.eks` antes do apply completo, senão o provider `kubernetes` falha (ver F3-3.3). E reserve os ~20 min que o EKS leva — não é travamento.

✅ **Entregável:** `kubectl get nodes` mostra 2 nós `Ready` **da sua máquina** (não só do CI).

> **2 nós `t3.small` bastam?** Sim, e a conta é simples: 2 × 2 GB ≈ 3,2 GB alocáveis. O HPA no teto (6 pods × 128 Mi) pede 768 Mi; DaemonSet do New Relic ~400 Mi; `kube-system` ~400 Mi. Sobra folga. Em CPU: 6 × 100m = 600m contra ~3,8 vCPU alocáveis. E 3 × `t3.small` (US$ 13,48) sairia mais caro que 2 × `t3.medium` seria útil — o HPA escala **pods**, não nós, então nó extra não melhora a demonstração.

---

### 🔴 Qui 03/09 — 3 h — Aplicação rodando na nuvem

- [F3-5.1](backlog.md#f3-51--revisão-do-modelo-relacional) — migration de modelagem (índices, `status`, `cpf_cnpj_digitos`, `timestamptz`)
- [F3-5.3](backlog.md#f3-53--seeds-e-credenciais-em-ambiente-de-nuvem) — seed com cliente **ativo** e **inativo** (CPFs válidos!)
- Overlay `prod` + `Service type: LoadBalancer` + deploy manual (`kubectl apply -k`)

✅ **Entregável:** `curl http://<nlb>/health/ready` responde `UP`, e a primeira OS é criada. **A partir de hoje o gráfico de volume diário começa a existir.**

> Antes de dormir, deixe rodando um gerador simples de tráfego — os dashboards precisam de dados acumulados:
> ```bash
> while true; do curl -s http://<nlb>/health/ready > /dev/null; sleep 60; done
> ```

---

### 🟡 Sex 04/09 — 3 h — Lambda de autenticação (código)

- [F3-2.1](backlog.md#f3-21--lambda-auth-token-validar-cpf-consultar-cliente-e-emitir-jwt) — `auth-token`: validação de CPF, consulta ao cliente, emissão do JWT
- [F3-2.2](backlog.md#f3-22--lambda-auth-authorizer-validação-do-token-na-borda) — `auth-authorizer`
- Testes de tabela do pacote `cpf` (é a única coisa que vale testar aqui)

✅ **Entregável:** `go test ./...` verde nos dois handlers.

---

### 🟢 Sáb 05/09 — 7 h — **O dia mais importante**: fluxo ponta a ponta

- [F3-2.3](backlog.md#f3-23--terraform-das-funções-e-das-rotas-no-api-gateway) — Terraform das Lambdas, IAM, VPC config
- [F3-3.4](backlog.md#f3-34--api-gateway-http-api-por-ambiente) — API Gateway + rota `/auth/token` + authorizer
- Integração `HTTP_PROXY` para o NLB (corte nº 1) + rotas `/v1/*` protegidas
- [F3-2.4](backlog.md#f3-24--adequar-a-aplicação-ao-novo-contrato-de-token) + [F3-2.5](backlog.md#f3-25--segredo-compartilhado-e-remoção-do-fallback-inseguro) — claims `tipo`/`iss`/`aud` na app, segredo compartilhado

✅ **Entregável — este é o marco crítico da fase:**
```bash
TOKEN=$(curl -s -X POST $GW/auth/token -d '{"cpf":"529.982.247-25"}' | jq -r .access_token)
curl -s -o /dev/null -w '%{http_code}' $GW/v1/work-orders                       # 401
curl -s -o /dev/null -w '%{http_code}' $GW/v1/work-orders -H "Authorization: Bearer $TOKEN"  # 200
```
**Se sábado terminar sem isso funcionando, pare e releia a [ordem de sacrifício](#ordem-de-sacrifício-se-atrasar) antes de continuar.**

---

### 🟢 Dom 06/09 — 7 h — Observabilidade

- [F3-4.1](backlog.md#f3-41--logs-estruturados-json-com-correlação-de-requisições) — logs JSON + correlação
- [F3-4.2](backlog.md#f3-42--apm-da-aplicação-go) — agente New Relic + `nrpq` no banco
- [F3-4.5](backlog.md#f3-45--eventos-de-negócio-para-os-dashboards) — eventos `OrdemServicoEvent` e `IntegracaoEvent`
- [F3-4.3](backlog.md#f3-43--monitoramento-do-cluster-kubernetes) — `nri-bundle` via Helm (1 h, é só um `helm_release`)
- Deploy e geração de OS reais

✅ **Entregável:** APM recebendo dados, logs JSON com `trace.id` no New Relic, CPU/memória dos pods visíveis.

---

### 🟢 Seg 07/09 (feriado) — 7 h — CI/CD e dashboards

- [F3-1.5](backlog.md#f3-15--cd-da-aplicação-build--registry--eks) — CD da aplicação (**só `main` → prod**; `homolog` roda apenas CI)
- [F3-1.6](backlog.md#f3-16--cicd-do-repositório-da-lambda) — CI/CD da Lambda
- [F3-1.4](backlog.md#f3-14--cicd-dos-repositórios-de-infraestrutura) — workflow de Terraform (plan em PR, apply em merge)
- [F3-1.7](backlog.md#f3-17--suíte-de-segurança-nos-quatro-repositórios) — apenas `gitleaks` nos 4 + `tfsec` nos de infra
- [F3-4.6](backlog.md#f3-46--dashboards) — dashboard (**pela UI do New Relic, não por Terraform** — economiza 2 h e o resultado visual é o mesmo)
- [F3-4.7](backlog.md#f3-47--alertas) — 1 alerta: falha no processamento de OS, **e dispare-o de verdade**
- [F3-4.8](backlog.md#f3-48--teste-de-carga-e-validação-do-autoescalonamento) — carga simples, provar HPA 2→4+

✅ **Entregável:** merge em `homolog` implanta sozinho; dashboard com dados; alerta disparado e e-mail recebido.

---

### 🔵 Ter 08/09 — 3 h — Documentação

Este é o dia de **melhor retorno por hora do projeto inteiro**: a argumentação já está escrita em [README.md](README.md), [arquitetura.md](arquitetura.md) e [backlog.md](backlog.md). Você está transcrevendo e ajustando, não pensando do zero.

- [F3-6.2](backlog.md#f3-62--rfcs) — 4 RFCs (nuvem, banco, autenticação, repositórios) — ~40 min cada
- [F3-6.3](backlog.md#f3-63--adrs) — 8 ADRs, **incluindo as dos cortes deste plano** (VPC Link, NAT, autoscaler, SES)
- [F3-6.1](backlog.md#f3-61--diagramas-de-componentes-e-de-sequência) — diagramas revisados contra o que você **realmente** construiu
- [F3-5.2](backlog.md#f3-52--documentar-o-modelo-e-justificar-a-escolha-do-banco) — RFC-002 com ER e os dois `EXPLAIN ANALYZE`

✅ **Entregável:** `docs/rfcs/` com 4 arquivos, ADRs estendidas, diagramas fiéis.

---

### 🔵 Qua 09/09 — 3 h — READMEs e ensaio

- [F3-6.4](backlog.md#f3-64--readmes-dos-quatro-repositórios) — 4 READMEs (o da aplicação já existe e é bom — só atualize)
- [F3-2.6](backlog.md#f3-26--atualizar-openapi-postman-e-a-documentação-de-autenticação) — Postman com a pasta de auth por CPF apontando para o Gateway
- **Ensaio cronometrado do vídeo**, com o [roteiro](entrega.md#roteiro-do-vídeo) aberto

✅ **Entregável:** ensaio completo em ≤ 15 min, e a lista do que travou nele.

> Não pule o ensaio. Ele é o que impede a gravação de virar 6 tomadas na quinta-feira.

---

### 🏁 Qui 10/09 — 3 h — Gravação e entrega

- Gravar por blocos (nunca tomada única) e editar
- [F3-7.2](backlog.md#f3-72--pdf-do-portal-e-compartilhamento-dos-repositórios) — PDF + `soat-architecture` nos 4 repos
- Submeter
- **Só depois de submeter:** [F3-7.3](backlog.md#f3-73--controle-de-custo-e-teardown) — `terraform destroy` e checagem de recursos órfãos

---

## Ordem de sacrifício (se atrasar)

Quando um dia terminar sem o entregável, corte **de cima para baixo** — nesta ordem, o dano à nota é crescente:

> O antigo item nº 1 desta lista era "sacrificar o ambiente de homologação". Ele **já foi executado** (corte nº 10) — a primeira alavanca de emergência foi gasta antes de começar. O que sobrou é mais caro.

| # | Sacrifique | Impacto | O que dizer na entrega |
|---|---|---|---|
| 1 | Painéis extras do dashboard (mantenha os 3 nomeados no enunciado) | Baixo | — |
| 2 | Alertas 2 e 3 (mantenha o de falha de OS) | Baixo | O enunciado nomeia só esse |
| 3 | READMEs dos repos de infra reduzidos ao mínimo | Médio | — |
| 4 | Segundo diagrama de sequência | Médio | O enunciado pede dois — entregue ao menos o da autenticação |
| 5 | Lambda authorizer (Gateway sem autorização na borda; app valida o JWT) | **Alto** | Perde "proteger rotas sensíveis **no gateway**" — evite até o limite |

**Nunca sacrifique**, em nenhuma hipótese — sem estes a entrega não é avaliável:

- 4 repositórios com `main` protegida e CI/CD rodando
- Autenticação por CPF funcionando ponta a ponta pelo API Gateway (incluindo o caso do **cliente inativo → 403**)
- Aplicação no EKS com HPA
- RDS gerenciado provisionado por Terraform
- New Relic com APM, logs JSON e ao menos 1 dashboard e 1 alerta
- RFCs, ADRs, ER e diagramas
- **Vídeo e PDF**

> Se na quarta-feira 09/09 faltar tempo, **corte funcionalidade, nunca o vídeo**. Uma solução 80% pronta com vídeo e documentação boa é avaliável; uma solução 100% pronta sem vídeo não é entrega.

---

## Três regras para os 10 dias

1. **Time-box de 45 minutos por bug de infraestrutura.** Estourou? Aplique o corte correspondente e documente como ADR. O sábado inteiro pode evaporar em uma regra de security group.
2. **Commit e push todo dia**, mesmo incompleto. O histórico de commits diários é evidência de processo, e protege contra perder trabalho.
3. **Anote as decisões enquanto decide**, em um `docs/rfcs/rascunho.md`. Terça 08/09 você vai transcrever, não relembrar — e é a diferença entre 3 h e 6 h de documentação.
