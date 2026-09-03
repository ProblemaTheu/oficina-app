# Diário de execução — Fase 3

> Registro do que foi feito, valores reais do ambiente e achados técnicos que custaram tempo. O [plano](plano-10-dias.md) diz o que fazer; este documento diz o que já aconteceu.

---

## Valores do ambiente

| Item | Valor |
|---|---|
| Conta AWS | `706215605178` · usuário IAM `admin-cli` · região `us-east-1` |
| Bucket de state | `oficina-tfstate-706215605178` (versionado, AES256, lock nativo) |
| Roles OIDC | `gha-oficina-app`, `gha-oficina-lambda-auth`, `gha-oficina-infra-k8s`, `gha-oficina-infra-db` |
| Conta New Relic | `8465573` |
| EKS | versão **1.36** (standard support até 01/08/2027) |
| Repositórios | `ProblemaTheu/{oficina-app, oficina-lambda-auth, oficina-infra-k8s, oficina-infra-db}` |
| Budget | US$ 60/mês · alertas em 50% previsto e 90% real |

---

## 01–02/09 — Dia 1 concluído

**AWS:** conta configurada, quota de vCPU 5 → 32 solicitada, Budget criado, bootstrap aplicado (13 recursos: bucket de state, provedor OIDC e 4 roles com `sub` restrito a `main`/`environment:prod`).

**Repositórios:** `tech-challenge-1` renomeado para `oficina-app`; os 3 repos novos criados, semeados com README no formato do enunciado e publicados. Branch `homolog` nos 4.

**CI/CD:** variables `AWS_ROLE_ARN`, `AWS_REGION` e `TF_BUCKET` nos 4; secrets do New Relic e Docker Hub. Smoke test de OIDC executado com sucesso.

**New Relic:** conta criada e ambas as chaves validadas contra a API — envio de evento e consulta NRQL de volta, o que já prova o pipeline de que os dashboards dependem.

**Qualidade:** `go build`, `go vet`, `go test -race` e `golangci-lint` verdes após o rename. `gitleaks` nos 58 commits do histórico: 3 achados, todos falso positivo, suprimidos em `.gitleaks.toml`.

### O que o rename tocou

Oito pontos que o redirect do GitHub **não** cobre: module path (36 arquivos Go), `ci.yml` (prefixo do módulo no Job Summary), imagem no Docker Hub (`cd.yml`, `deployment.yaml`, overlay local), tags do Trivy no `security.yml`, default de `EKS_CLUSTER_NAME`, `README.md`, `k8s/README.md`, `infra/README.md`, `scripts/k8s-local-deploy.sh` e o `git remote`.

---

## 02/09 — Dia 2 concluído

Nuvem de pé. VPC sem NAT, EKS 1.36 com 2 nós `t3.small` em subnet pública, 5 addons, namespace `oficina-prod` e RDS `db.t4g.micro`.

**Entregável verificado:** `kubectl get nodes` responde da máquina local com 2 nós `Ready`, e `kubectl top nodes` funciona — o `metrics-server` está de pé, que é do que o HPA depende.

### Ajustes em relação ao backlog

| Escrito no backlog | O que foi feito | Por quê |
|---|---|---|
| `vpc ~> 5.8`, `eks ~> 20.24`, `rds ~> 6.7`, `security-group ~> 5.1` | `vpc ~> 6.7`, `eks ~> 21.25`, recursos diretos no banco | Os módulos antigos não aceitam o AWS provider 6.x, que o bootstrap já usa |
| `module "rds"` | `aws_db_instance` e mais 5 recursos diretos | O módulo v7 trocou `password` por `password_wo`; são ~90 linhas todas usadas, mais barato escrever que estudar |
| `enable_cluster_creator_admin_permissions = true` | `false`, com 3 `access_entries` explícitas | O "creator" no CI é a role OIDC, não você — `kubectl` local responderia `Unauthorized` |
| versão do node group herdada do cluster | `kubernetes_version` explícito no node group | Herdada, ela deixa um `count` do submódulo desconhecido em plan e **quebra qualquer `terraform import`** |
| Performance Insights, parameter group e export de log no RDS | só Performance Insights | O log group fica órfão no destroy; o PI já entrega a leitura de gargalo que o enunciado pede |

### Valores novos do ambiente

| Item | Valor |
|---|---|
| VPC | `vpc-0856fe2ed483471b4` — 2 AZs, sem NAT |
| Cluster | `oficina` · EKS 1.36 · 2× `t3.small` em subnet pública |
| SG dos nós | `sg-0e310511548743205` |
| SG das Lambdas | `sg-053ea581fe5097f3a` |
| Namespace | `oficina-prod` |
| Quota de vCPU | 8 (o pedido de 32 segue `CASE_OPENED` — não bloqueou: 2× `t3.small` = 4 vCPUs) |

### Runbook de destruição

`destroy.sh` nos dois repos de infra, escrito **antes** do primeiro apply. Ordem obrigatória: `infra-db` primeiro, `infra-k8s` depois — o SG do RDS referencia o SG dos nós.

O script apaga os Services do namespace antes do `terraform destroy`, porque **o NLB é criado pelo Kubernetes, não pelo Terraform**: deixá-lo de pé faz o destroy da VPC falhar com `DependencyViolation` e rende um load balancer órfão de US$ 0,54/dia. No fim ele lista VPCs, RDS, load balancers e chaves KMS em `PendingDeletion`.

---

## 03/09 — Dia 3 concluído

Aplicação na nuvem. Migrations 000005 e 000006 aplicadas no RDS pelo boot do pod, overlay `prod` implantado no EKS e API pública.

**Entregável verificado:** `curl http://<lb>/health/ready` responde `{"status":"UP","components":{"db":{"status":"UP"}}}` pela internet, e a **OS-2026-00001** foi criada para a Maria Silva.

| Item | Valor |
|---|---|
| Load balancer | `adf501e528a5c4a999da812ef487526f-556148891.us-east-1.elb.amazonaws.com` (Classic, internet-facing) |
| Imagem | `problematheu/oficina-api:latest` — publicada pelo CD via `workflow_dispatch` |
| Cliente ativo | Maria Silva · `529.982.247-25` · `770e8400-…-440001` |
| Cliente inativo | Carlos Pereira · `111.444.777-35` · `770e8400-…-440002` |
| Segredos da app | `oficina/prod/app` no Secrets Manager (`jwt_secret`, `webhook_secret`) |

O `JWT_SECRET` já nasceu no Secrets Manager em vez de gerado no cluster, porque no dia 5 ele vira segredo **compartilhado** com a Lambda (F3-2.5) — assim não precisa rotacionar depois.


---

## 03/09 — Dia 4 concluído

As duas Lambdas escritas, compiladas e testadas. O `oficina-lambda-auth` saiu de "só README" para o repositório completo, menos o Terraform, que é do dia 5.

| Pacote | Conteúdo |
|---|---|
| `internal/cpf` | dígitos verificadores sem dependência externa · **95,2%** de cobertura, 14 casos |
| `internal/token` | contrato de claims (F3-0.2) compartilhado pelos dois handlers |
| `internal/segredo` | Secrets Manager com cache por container, lendo campo de JSON |
| `cmd/auth-token` | 400 CPF inválido · 404 sem cadastro · 403 inativo · 200 com JWT |
| `cmd/auth-authorizer` | assinatura, `exp` e `aud` na borda, com resposta simples |
| `Makefile` | `dist/*.zip` com binário `bootstrap`, linux/arm64 |

**O segredo já casa com o que existe.** O `scripts/prod-secret.sh` do dia 3 criou `oficina/prod/app` no formato `{"jwt_secret": ..., "webhook_secret": ...}`. As Lambdas leem o campo `jwt_secret` desse mesmo segredo, então o HS256 assina e verifica com a mesma chave dos dois lados sem nenhum passo de sincronização.

**A DSN da Lambda ganha `sslmode=require` se vier sem.** Não é elegância: é a mesma armadilha do RDS que custou meia hora no dia 3, e a mensagem de erro aponta para o lugar errado.

**O que foi testado no authorizer.** A decisão de aceitar ou negar saiu do handler para uma função pura, testada contra token assinado com outro segredo, expirado, de outra audiência, sem audiência, sem o prefixo `Bearer`, e — o ataque clássico — com o header trocado para `alg=none`. Sem `jwt.WithValidMethods`, esse último passa.


---

## 03/09 — Dia 5 concluído

O fluxo ponta a ponta pelo API Gateway está de pé. **URL pública:** `https://950h33a7r7.execute-api.us-east-1.amazonaws.com`

| Verificação | Resultado |
|---|---|
| `POST /auth/token` cliente ativo | 200 + JWT com as claims do contrato |
| cliente inativo | 403 `cliente_inativo` |
| CPF inválido | 400 `cpf_invalido` |
| CPF válido sem cadastro | 404 `cliente_nao_encontrado` |
| `/v1/work-orders` sem token | 401 — barrado no Gateway, não chega ao cluster |
| token adulterado | 403 |
| token da Lambda | 200 — prova que app e Lambda leem o mesmo segredo |
| `/health/ready`, `/v1/auth/login` | 200, públicas |

### A costura assimétrica foi eliminada

O backlog mandava aplicar o `infra-k8s` **duas vezes** na primeira subida: ele criaria as rotas `/v1/*`, que precisam do `authorizer_id` publicado pelo `lambda-auth`, que por sua vez precisa do Gateway já existir. O próprio backlog chamava isso de *"a única costura assimétrica da arquitetura"*.

A causa é que uma rota protegida depende de **duas** coisas de repositórios diferentes: a integração e o authorizer. Passando as rotas para o repositório que já é dono do authorizer, a dependência vira linear:

`bootstrap → infra-k8s → infra-db → app → lambda-auth`

O `infra-k8s` é aplicado uma vez. O custo é que a integração `HTTP_PROXY` para o balanceador vive no `lambda-auth`, que não é dono do cluster — anotado como trade-off na ADR.

### F3-2.4 — o contrato fechado

A aplicação já **aceitava** o token da Lambda (mesmo segredo, mesma assinatura), mas tratava cliente e funcionário como a mesma coisa. Três lacunas, da menos para a mais grave, todas fechadas:

| Antes | Agora |
|---|---|
| `aud` não validado — token assinado para outro sistema com a mesma chave valeria aqui | middleware exige `aud=oficina-api`; o login passou a emitir `papel`, `tipo`, `iss` e `aud` |
| token de **cliente** alcançava `POST /v1/work-orders` e todas as operações internas | 27 handlers exigem `tipo=usuario` |
| `GET /v1/work-orders` devolvia **todas** as OS a qualquer cliente | filtro imposto a partir do `sub`, e OS alheia por ID responde 404 |

O segundo merece nota: **o authorizer do API Gateway não impede isso.** Ele valida a assinatura e para por aí — e o token do cliente é perfeitamente assinado. A autorização por tipo é trabalho da aplicação, não da borda.

Provado em produção, com uma OS para cada cliente:

| Quem pede | O que vê |
|---|---|
| funcionário | 2 OS |
| cliente Maria | 1 OS — só a dela |
| Maria forçando `cliente_id` do Carlos no parâmetro | 1 OS — o filtro é imposto, não aceito |
| Maria lendo a OS do Carlos por ID | 404 |

O 404 é deliberado: um 403 confirmaria que aquela OS existe.

> ⚠️ **Efeito colateral aceito:** tokens emitidos antes deste deploy não têm `aud` e passam a receber 401. Validade de 8 h, então o efeito se esgota sozinho. Token sem o claim `tipo` continua sendo tratado como funcionário justamente para não deslogar todo mundo no instante do deploy.


---

## 03/09 — Coleção do Postman e um achado de segurança

A coleção foi apontada para o API Gateway (`gatewayUrl`), ganhou a pasta de **autenticação por CPF** com os quatro desfechos da Lambda, uma pasta de **proteção da borda** (401 sem token, 403 com token adulterado) e scripts que capturam `token`, `osId`, `servicoId` e `pecaId` sozinhos — não é mais preciso colar UUID à mão.

Simulando as 30 requisições contra o ambiente, uma divergiu: `POST /v1/auth/register` devolvia 401 pelo Gateway. Investigando, o problema era o contrário do que parecia.

### Registro anônimo criava administrador

A rota estava na lista de públicas da aplicação desde a Fase 1 e aceita o papel desejado no corpo. Com o cluster local isso era discutível; com o `Service type: LoadBalancer` da Fase 3, o balanceador é alcançável pela internet e a rota virou escalonamento de privilégio anônimo. Confirmado no ambiente, sem token nenhum:

```
POST http://<lb>/v1/auth/register  {"papel":"administrador"}
201 {"papel":"administrador", ...}
```

A conta foi removida do banco. A rota saiu das públicas e passou a exigir **papel de administrador** — só `tipo=usuario` não bastaria, porque qualquer funcionário poderia se promover.

### O buraco maior continua aberto

Fechar a rota não fecha o problema de fundo: **o balanceador é público, e quem o alcança direto pula o API Gateway e o authorizer inteiro.** Toda a proteção da borda vale apenas para quem entra pela porta da frente.

É exatamente o que o *header compartilhado* do corte 1 resolve — o Gateway injeta um header secreto na integração e a aplicação recusa requisição sem ele. Falta implementar:

1. `gateway_secret` no segredo `oficina/prod/app`;
2. `request_parameters` nas integrações do `lambda-auth` injetando o header;
3. middleware na aplicação recusando quem não o traz, com `/health/*` isento (o kubelet sonda o pod direto, sem passar pelo balanceador).


---

## Achados que custaram tempo

### OIDC: o `sub` não é o dos tutoriais

`Not authorized to perform sts:AssumeRoleWithWebIdentity`, sem indicar qual condição falhou. O GitHub emite **immutable subject claims**, com os IDs numéricos embutidos:

```
repo:ProblemaTheu@86577215/oficina-infra-k8s@1354226544:ref:refs/heads/main
```

Existe justamente para renomear repositório não quebrar a trust policy. A policy passou a cobrir os dois formatos.

**Não adivinhe o `sub` — leia-o** com um step que decodifica o payload (só as claims, nunca o token):

```yaml
- name: Claims do token OIDC
  run: |
    TOKEN=$(curl -sH "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
      "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=sts.amazonaws.com" | jq -r .value)
    PAYLOAD=$(echo "$TOKEN" | cut -d. -f2)
    PAYLOAD="${PAYLOAD//-/+}"; PAYLOAD="${PAYLOAD//_//}"
    while [ $(( ${#PAYLOAD} % 4 )) -ne 0 ]; do PAYLOAD="${PAYLOAD}="; done
    echo "$PAYLOAD" | base64 -d | jq '{sub, aud, repository, ref}'
```

### EKS: versão antiga custa 6× mais

`1.31` já está em *extended support*: **US$ 0,60/h** em vez de US$ 0,10. Nos 9 dias, ~US$ 130 contra US$ 21,60 — sem erro nenhum, só a fatura diferente. Confira antes de fixar:

```bash
aws eks describe-cluster-versions --region us-east-1 \
  --query 'clusterVersions[?versionStatus==`STANDARD_SUPPORT`].[clusterVersion,endOfStandardSupportDate]' \
  --output table
```

### New Relic: três chaves diferentes

License key tem 40 caracteres e termina em `NRAL`; user key começa com `NRAK-`; account ID é numérico. Formato errado dá `authentication required`, que parece erro de configuração. Valide em 10 segundos:

```bash
# user key
curl -s https://api.newrelic.com/graphql -H "Api-Key: $USER_KEY" \
  -H 'Content-Type: application/json' -d '{"query":"{ actor { accounts { id name } } }"}'

# license key — envia um evento de verdade
curl -s -X POST "https://insights-collector.newrelic.com/v1/accounts/8465573/events" \
  -H "Api-Key: $LICENSE_KEY" -H 'Content-Type: application/json' \
  -d '[{"eventType":"SetupFase3","etapa":"teste"}]'
```

Grave sem passar por chat ou histórico de shell: `gh secret set NOME --repo dono/repo < arquivo`.

### Terraform ≥ 1.10 dispensa o DynamoDB

O backend S3 tem lock nativo (`use_lockfile = true`); `dynamodb_table` está deprecado. Um recurso a menos para criar, pagar e destruir.

### zsh come `:r` em interpolação

`"arn:aws:iam::$ACC:role/..."` vira `...706215605178ole/...` — no zsh, `$VAR:r` é o modificador que remove extensão. Use sempre `${VAR}` quando houver `:` logo depois.

### gofmt e a ordem dos imports

Trocar `problematheu` por `ProblemaTheu` desordena todos os blocos de import: maiúscula ordena antes de minúscula em ASCII. `gofmt -w` resolve, mas o lint acusa antes.

### Falso alarme registrado

Suspeitei que o `.golangci.yml` estivesse no formato v1 e quebraria com o golangci-lint 2.x. Já estava em `version: "2"` e roda com 0 issues. Verificar custou 30 segundos; supor teria custado uma tarefa inventada.

### Apply interrompido deixa recurso órfão

Um `terraform apply` morto no meio (Ctrl-C, prompt recusado) **não desfaz o que já criou**. O state só é gravado ao fim de cada operação, então o cluster EKS ficou `ACTIVE` na AWS e ausente do state — cobrando, invisível para o `destroy`. Junto ficou um lock preso no S3.

A recuperação, na ordem:

```bash
terraform force-unlock -force <ID-do-lock>   # o ID vem na mensagem de erro
terraform state list                          # compare com o que existe na AWS
terraform import 'module.eks.aws_eks_cluster.this[0]' oficina
terraform plan                                # exija "0 to change, 0 to destroy"
```

O `import` só passou depois de fixar `kubernetes_version` no node group (ver tabela do dia 2). E antes de importar, confira o que o recurso vivo tem que a config não pede: o cluster nasceu com criptografia KMS de secrets e **a AWS não permite desligá-la depois** — a config voltou a pedir a chave, porque recriar custaria 25 min para economizar US$ 0,23.

### `-target` não arrasta o que ninguém referencia

`terraform apply -target=module.eks` criou VPC e subnets, mas **não** o Internet Gateway nem as route tables — nenhum recurso do EKS aponta para eles. Os nós subiram com IP público e sem rota para fora, não baixaram imagem nenhuma, e o node group morreu em:

```
NodeCreationFailure: Instances failed to join the kubernetes cluster
```

Essa mensagem tem meia dúzia de causas possíveis e não indica nenhuma. O diagnóstico que resolve em 2 min:

```bash
aws ec2 describe-internet-gateways --filters Name=attachment.vpc-id,Values=<vpc-id>
aws ec2 describe-route-tables --filters Name=vpc-id,Values=<vpc-id> \
  --query 'RouteTables[].Routes[].DestinationCidrBlock'
```

Sem `0.0.0.0/0` na route table das subnets dos nós, é isso. `apply -target=module.vpc` antes de tudo evita o problema.

### A conta não cria NLB nem ALB — só Classic

`Service type: LoadBalancer` com a anotação `aws-load-balancer-type: nlb` ficou 6 minutos em `<pending>`. O motivo só aparece em `kubectl describe svc`:

```
OperationNotPermitted: This AWS account currently does not support creating
load balancers. For more information, please contact AWS Support.
```

Não é quota (`describe-account-limits` mostra 50) nem configuração: é **restrição de conta nova**, e vale para toda a API **ELBv2** — NLB e ALB. A API clássica **não** é afetada: removida a anotação de tipo, o mesmo Service subiu um CLB em segundos.

Consequência prática: nenhuma. O corte 1 aponta o API Gateway para o DNS público do balanceador, e `HTTP_PROXY` não distingue CLB de NLB. Custa US$ 0,05/dia a mais. Abrir caso no Support só valeria a pena se ALB voltasse ao escopo.

> Se aparecer `<pending>` em Service `LoadBalancer`, **leia `kubectl describe svc` antes de mexer em subnet, tag ou security group** — o controller escreve a causa real nos eventos.

### RDS 15 recusa conexão sem TLS

`rds.force_ssl=1` é o padrão do parameter group do PostgreSQL 15. A aplicação subia em `CrashLoopBackOff` com:

```
pq: no pg_hba.conf entry for host "10.0.11.26", user "oficina",
database "oficina", no encryption (28000)
```

A mensagem cita `pg_hba.conf` e parece problema de permissão ou de security group — e é de criptografia. A DSN tinha `sslmode=disable` fixo no código; virou `DB_SSLMODE`, com `require` em produção e `disable` como default para não mexer no ambiente local.

### A imagem do Docker Hub não existia

O rename do dia 1 trocou o nome para `problematheu/oficina-api` no `cd.yml` e nos manifestos, mas ninguém criou o repositório no Docker Hub — o publicado ainda era `problematheu/tech-challenge-api`. E o Docker local está logado como `nukeer`, não como `problematheu`, então não dava para publicar da máquina.

Saída: `gh workflow run cd.yml --ref feature/fase-3`. O `cd.yml` tem `workflow_dispatch` e o job de deploy só roda com `vars.DEPLOY_ENABLED == 'true'`, que não existe — então o pipeline publica a imagem com as credenciais que já estão nos secrets e não tenta implantar nada.

---

## Pendências

| Pendência | Depende de | Prazo |
|---|---|---|
| **Branch protection nos 4 repos** | `ProblemaTheu` conceder **Admin** | 07/09 |
| **Environment `prod` com reviewer** | idem | 07/09 |
| **`soat-architecture` nos 3 repos novos** | idem | antes da entrega |
| Quota de vCPU aprovada | AWS (solicitado, `PENDING`) | antes do dia 2 |
| `SONAR_PROJECT_KEY` aponta para `tech-challenge-1` | *Update key* no SonarCloud **antes** de trocar a variable | 07/09 |
| Rotacionar as chaves do New Relic | passaram pelo chat | depois da entrega |

**O que `Write` já permite** (testado): push, branches, secrets e variables de repositório. **Só ruleset e environment exigem `admin`** — ruleset responde `404`, environment responde `403 Must have admin rights`.
