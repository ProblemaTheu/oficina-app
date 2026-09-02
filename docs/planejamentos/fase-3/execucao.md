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
