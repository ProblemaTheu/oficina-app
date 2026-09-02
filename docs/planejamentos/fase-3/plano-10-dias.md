# Plano de execução — 01/09 → 10/09

> **Este é o documento de trabalho.** O [roadmap](roadmap.md) descreve o sequenciamento ideal com prazo folgado; aqui está o que cabe na realidade: **uma pessoa, ~42 h, orçamento apertado**. O que já foi feito e os valores do ambiente estão em [execucao.md](execucao.md).

| | |
|---|---|
| Capacidade | 7 dias × 3 h + 3 dias (sáb, dom, feriado 07/09) × 7 h ≈ **42 h** |
| Escopo do [backlog](backlog.md) completo | 43 tarefas ≈ **130 h** |
| Escopo após os cortes | **~29 h** — folga de ~13 h |
| Ambiente de homologação | dispensado por orientação no fórum da disciplina |

Não existe cronograma que resolva 88 horas de déficit. O que existe é **cortar profundidade sem cortar requisito** — cada corte abaixo mantém o item do enunciado atendido pela implementação mais simples que ainda o demonstra.

> **Princípio:** um trade-off consciente e documentado vale mais na avaliação do que uma implementação sofisticada pela metade. Todo corte aqui vem com o que dizer na entrega. Faça isso — é o que separa "não deu tempo" de "decidimos assim porque".

---

## 🚧 Bloqueio ativo

`Mendeszx` tem **Write**, não **Admin**, nos quatro repositórios (o dono é `ProblemaTheu`). Isso trava **branch protection**, **environment `prod`** e **adicionar o `soat-architecture`** — os dois primeiros são requisitos explícitos do enunciado, o terceiro é exigência de entrega.

**Resolve:** em cada repo, *Settings → Collaborators → `Mendeszx` → Role: **Admin***. É a mesma tela onde o Write já foi concedido.

**Prazo: 07/09.** Depois disso começa a consumir a folga. Detalhes e o que já funciona com Write em [execucao.md](execucao.md#pendências).

---

## Os cortes

Nenhum remove requisito do enunciado. Os de 1 a 9 trocam robustez por tempo e viram ADR; os de 11 a 16 apenas retiram trabalho que ninguém avalia.

| # | Corte | Economia | Como justificar |
|---|---|---|---|
| 1 | **VPC Link + ALB interno + `TargetGroupBinding`** → `Service type: LoadBalancer` (NLB) e API Gateway com `HTTP_PROXY` + header compartilhado | **6 h** | ADR: topologia correta para produção; adotamos a integração direta pelo prazo, com o caminho de evolução documentado |
| 2 | **NAT Gateway** → nós em subnet pública | 2 h + US$ 8 | ADR: SG restritivo; em produção real iriam para subnet privada |
| 3 | **Cluster Autoscaler** → 2 nós `t3.small` fixos, HPA de 2 a 6 pods | 2 h | O enunciado pede "cluster com escalabilidade" — o HPA entrega. Autoscaler como evolução na ADR-009 |
| 4 | **External Secrets Operator** → `kubernetes_secret` criado pelo Terraform | 2 h | Mesmo resultado; o ESO agrega rotação automática, que não é requisito |
| 5 | **Amazon SES** → `NOTIFIER=log` em prod, e-mail demonstrado no local com Mailpit | 3 h | "A integração SMTP é a mesma; muda o provedor." Também elimina a espera de até 24 h pela saída do sandbox |
| 6 | **Expansão de testes** → manter a suíte da Fase 2 verde; novos só para CPF e authorizer | 4 h | A Fase 3 não pede cobertura; a Fase 2 já entregou ~85% nos use cases |
| 7 | **`git filter-repo`** → copiar arquivos | 2 h | Ninguém avalia histórico de commit de repositório de infra |
| 8 | **Segurança dos repos novos** → só `gitleaks` | 2 h | O enunciado **não menciona** análise de segurança. O `gitleaks` fica porque custa 5 min |
| 9 | **k6 com estágios** → loop `hey`/`curl` de 3 min | 1 h | Só precisa provar que o HPA reage |
| 10 | ~~Ambiente de homologação~~ | 3 h + US$ 4 | ✅ **já aplicado** — ver [detalhe](#corte-nº-10--ambiente-único) |
| 11 | **Instrumentação `nrpq` do banco** | 1 h | Pede-se latência das APIs, não breakdown por camada |
| 12 | **Dashboards e alertas pela UI**, não por Terraform | 2 h | "Expor dashboards" é o requisito; versioná-los é elegância |
| 13 | **Só o alerta de falha de OS** | 0,5 h | É o único que o enunciado nomeia |
| 14 | **Sem `for_each` de ambientes** — recursos diretos | 1 h | Com um ambiente, `for_each` obriga a indexar tudo com `["prod"]` |
| 15 | **Sem `tfsec`, `checkov` e Sonar nos repos novos** | 1,5 h | Mesma razão do corte 8 |
| 16 | **Sem `ResourceQuota` e sem Synthetics além de um** | 0,5 h | Quota com um namespace não protege de nada |

### ⚠️ O que não cortar

**O Lambda authorizer.** É a tentação óbvia (~3 h): o middleware JWT da aplicação já valida o token desde a Fase 1, então as rotas ficariam protegidas.

Mas a seção do enunciado se chama **"Autenticação e API Gateway"** e lista os dois juntos. Sem o authorizer, o Gateway roteia mas não protege — a parte mais visível da fase. Com 13 h de folga, 3 h no item mais avaliado é o melhor uso de tempo do plano. Ele segue em último lugar na [ordem de sacrifício](#ordem-de-sacrifício-se-atrasar): se atrasar de verdade, corte; antes disso, não.

**Os diagramas de sequência.** Já estão escritos em [arquitetura.md](arquitetura.md#2-diagramas-de-sequência) — o trabalho é revisá-los contra o que foi construído. Cortar economiza ~30 min e custa um requisito nomeado (*"para o fluxo de autenticação **e** abertura de ordens de serviço"*).

### Corte nº 10 — ambiente único

O fórum liberou dispensar homologação. No Terraform, isso significa `prod` fixo e **sem `for_each`** (corte 14): escreva os recursos direto, com `prod` no nome.

Nos workflows, `homolog` deixa de implantar e vira branch de integração com CI completo:

```yaml
# ci.yml — roda em PR para as duas branches protegidas
on:
  pull_request:
    branches: [homolog, main]

# cd.yml e terraform.yml — só main implanta
on:
  push:
    branches: [main]
```

⚠️ **Tirar `homolog` do gatilho de `push` não é opcional:** sem ambiente de homologação, um merge ali rodaria `terraform apply` contra um ambiente inexistente — no melhor caso falha, no pior cria recursos órfãos que você paga sem saber.

**Mantenha a branch.** O fluxo `feature → homolog → main` com PR e CI bloqueante continua demonstrado; some o ambiente, não o processo.

> **Guarde a evidência.** O PDF do enunciado pede deploy automático das duas branches; a dispensa veio do fórum. Salve o print ou link e cite nos 4 READMEs e no PDF: *"ambiente de homologação dispensado conforme orientação no fórum da disciplina em [data] — [link]"*. Mencione também em uma frase no vídeo.

---

## Requisito → implementação mínima

Confira antes de acrescentar qualquer coisa ao escopo. Se não está na coluna da esquerda, não entra.

| O enunciado pede | Mínimo que atende |
|---|---|
| API Gateway | `aws_apigatewayv2_api` + rotas + integração `HTTP_PROXY` |
| Rotas protegidas por CPF | `auth-token` emite o JWT; `auth-authorizer` valida no Gateway |
| Function: valida CPF, consulta existência **e status**, gera JWT | `auth-token` (~150 linhas de Go) |
| 4 repos com CI/CD e deploy automático | 1 workflow por repo |
| `main` protegida, PR obrigatório | Ruleset — **bloqueado por admin** |
| Banco gerenciado | 1 RDS `db.t4g.micro` |
| Cluster K8s com escalabilidade | EKS + HPA por CPU (já existe em `k8s/base/hpa.yaml`) |
| Terraform | os 3 repos de infra |
| Latência das APIs | agente APM do New Relic |
| CPU/memória do K8s | `nri-bundle` via Helm |
| Healthchecks e uptime | `/health/ready` (já existe) + 1 monitor Synthetic |
| Alertas de falha de OS | 1 NRQL alert condition |
| Logs JSON com correlação | handler `slog` JSON + middleware de correlation id |
| Volume diário · tempo médio por status · erros de integração | 2 custom events + 3 painéis NRQL |
| Diagrama de componentes · 2 de sequência | **já escritos** em [arquitetura.md](arquitetura.md) — só revisar |
| RFCs · ADRs | 4 RFCs de 1 página + 8 ADRs curtas — a argumentação já está no [README](README.md) |
| Justificativa do banco + ER | RFC-002 + o ER de [arquitetura.md](arquitetura.md#3-modelo-entidade-relacionamento-alvo) |
| 4 READMEs | **3 já escritos**; falta atualizar o do `oficina-app` |
| Vídeo ≤ 15 min · PDF | [entrega.md](entrega.md) |

---

## Custo real da nuvem

Ligando em **02/09** e destruindo após a entrega — 9 dias, 24 h por dia, com os cortes 1–3 aplicados:

| Recurso | Cálculo | Total |
|---|---|---|
| EKS control plane | US$ 0,10/h × 216 h | **US$ 21,60** |
| ↳ ⚠️ se a versão do K8s estiver em *extended support* | US$ 0,60/h × 216 h | *US$ 129,60* |
| 2× EC2 `t3.small` | US$ 0,0208/h × 2 × 216 h | US$ 8,99 |
| NLB | US$ 0,0225/h × 216 h | US$ 4,86 |
| 1× RDS `db.t4g.micro` | US$ 0,016/h × 216 h | US$ 3,46 |
| API Gateway + Lambda + Secrets | volume da demo | ~US$ 2,00 |
| **Total** | | **≈ US$ 41** |

Com o Free Tier do RDS (750 h/mês de `db.t4g.micro`), a instância sai de graça: **piso ~US$ 37,50**. Desse total, **US$ 21,60 é o control plane do EKS** — requisito do enunciado, não escolha de arquitetura.

> ❌ **Não destrua e recrie todo dia.** Um ciclo `destroy` + `apply` do EKS leva ~35 min. Oito ciclos queimam **4,5 h** do seu recurso mais escasso para economizar ~US$ 20.
>
> ✅ **Deixe ligado.** Além do tempo, resolve os dashboards: "volume diário de OS" precisa de dias acumulando dados. Ligando dia 2, você chega no dia 10 com 8 pontos no gráfico; ligando dia 8, com um ponto.

---

## Cronograma

Cada dia tem um **entregável verificável**. Se o dia acabar sem ele, consulte a [ordem de sacrifício](#ordem-de-sacrifício-se-atrasar) **naquele dia**, não no dia 9.

### ✅ Passo 0 e Ter 01/09 — concluídos

Ferramentas, credenciais AWS, quota, Budget, bootstrap, rename, os 4 repositórios publicados, variables e secrets, OIDC validado e conta New Relic. Registro completo em [execucao.md](execucao.md).

Resta apenas o que depende de `admin` — ver [bloqueio ativo](#bloqueio-ativo).

### 🔴 Qua 02/09 — 3 h — Nuvem de pé

- [F3-3.2](backlog.md#f3-32--repositório-oficina-infra-k8s-vpc-e-rede) — VPC **sem NAT**, nós em subnet pública
- [F3-3.3](backlog.md#f3-33--cluster-eks-com-escalabilidade-e-addons) — EKS **1.36**, 2 nós `t3.small`, addon `metrics-server`, **`access_entries` com seu usuário IAM** (senão você não usa `kubectl`), namespace
- [F3-3.1](backlog.md#f3-31--repositório-oficina-infra-db-rds-postgresql-gerenciado) — RDS, uma instância

⚠️ **Primeiro apply:** rode `terraform apply -target=module.eks` antes do apply completo, senão o provider `kubernetes` falha. Reserve os ~20 min que o EKS leva — não é travamento.

✅ **Entregável:** `kubectl get nodes` mostra 2 nós `Ready` **da sua máquina**, não só do CI.

> **2 nós bastam?** 2 × 2 GB ≈ 3,2 GB alocáveis. O HPA no teto (6 pods × 128 Mi) pede 768 Mi; New Relic ~400 Mi; `kube-system` ~400 Mi. Sobra folga. O HPA escala **pods**, não nós — um terceiro nó custaria US$ 4,50 e não melhoraria a demonstração.

### 🔴 Qui 03/09 — 3 h — Aplicação na nuvem

- [F3-5.1](backlog.md#f3-51--revisão-do-modelo-relacional) — migration de modelagem (índices, `status`, `cpf_cnpj_digitos`, `timestamptz`)
- [F3-5.3](backlog.md#f3-53--seeds-e-credenciais-em-ambiente-de-nuvem) — seed com cliente **ativo** e **inativo** (CPFs válidos!)
- Overlay `prod` + `Service type: LoadBalancer` + deploy manual (`kubectl apply -k`)

✅ **Entregável:** `curl http://<nlb>/health/ready` responde `UP` e a primeira OS é criada. **A partir de hoje o gráfico de volume diário começa a existir.**

> Deixe rodando um gerador simples de tráfego — os dashboards precisam de dados acumulados:
> ```bash
> while true; do curl -s http://<nlb>/health/ready > /dev/null; sleep 60; done
> ```

### 🟡 Sex 04/09 — 3 h — Lambda de autenticação

- [F3-2.1](backlog.md#f3-21--lambda-auth-token-validar-cpf-consultar-cliente-e-emitir-jwt) — `auth-token`
- [F3-2.2](backlog.md#f3-22--lambda-auth-authorizer-validação-do-token-na-borda) — `auth-authorizer`
- Testes de tabela do pacote `cpf` — a única coisa que vale testar aqui

✅ **Entregável:** `go test ./...` verde nos dois handlers.

### 🟢 Sáb 05/09 — 7 h — **O dia crítico**: fluxo ponta a ponta

- [F3-2.3](backlog.md#f3-23--terraform-das-funções-e-das-rotas-no-api-gateway) — Terraform das Lambdas, IAM, VPC config
- [F3-3.4](backlog.md#f3-34--api-gateway-http-api-por-ambiente) — API Gateway + rota `/auth/token` + authorizer
- Integração `HTTP_PROXY` para o NLB (corte 1) + rotas `/v1/*` protegidas
- [F3-2.4](backlog.md#f3-24--adequar-a-aplicação-ao-novo-contrato-de-token) + [F3-2.5](backlog.md#f3-25--segredo-compartilhado-e-remoção-do-fallback-inseguro) — claims `tipo`/`iss`/`aud`, segredo compartilhado

✅ **Entregável:**
```bash
TOKEN=$(curl -s -X POST $GW/auth/token -d '{"cpf":"529.982.247-25"}' | jq -r .access_token)
curl -s -o /dev/null -w '%{http_code}' $GW/v1/work-orders                                   # 401
curl -s -o /dev/null -w '%{http_code}' $GW/v1/work-orders -H "Authorization: Bearer $TOKEN"  # 200
```
**Se sábado terminar sem isso, pare e releia a ordem de sacrifício antes de continuar.**

### 🟢 Dom 06/09 — 7 h — Observabilidade

- [F3-4.1](backlog.md#f3-41--logs-estruturados-json-com-correlação-de-requisições) — logs JSON + correlação
- [F3-4.2](backlog.md#f3-42--apm-da-aplicação-go) — agente New Relic (sem `nrpq`)
- [F3-4.5](backlog.md#f3-45--eventos-de-negócio-para-os-dashboards) — eventos `OrdemServicoEvent` e `IntegracaoEvent`
- [F3-4.3](backlog.md#f3-43--monitoramento-do-cluster-kubernetes) — `nri-bundle` via Helm
- Deploy e geração de OS reais

✅ **Entregável:** APM recebendo dados, logs JSON com `trace.id`, CPU/memória dos pods visíveis.

### 🟢 Seg 07/09 (feriado) — 7 h — CI/CD e dashboards

- [F3-1.5](backlog.md#f3-15--cd-da-aplicação-build--registry--eks) — CD da aplicação
- [F3-1.6](backlog.md#f3-16--cicd-do-repositório-da-lambda) — CI/CD da Lambda
- [F3-1.4](backlog.md#f3-14--cicd-dos-repositórios-de-infraestrutura) — workflow de Terraform (plan em PR, apply no merge)
- [F3-4.6](backlog.md#f3-46--dashboards) — dashboard **pela UI**
- [F3-4.7](backlog.md#f3-47--alertas) — 1 alerta, **disparado de verdade**
- [F3-4.8](backlog.md#f3-48--teste-de-carga-e-validação-do-autoescalonamento) — carga simples, provar HPA 2→4+

✅ **Entregável:** merge na `main` implanta sozinho; dashboard com dados; alerta disparado e e-mail recebido.

### 🔵 Ter 08/09 — 3 h — Documentação

Melhor retorno por hora do plano: a argumentação já está escrita em [README.md](README.md), [arquitetura.md](arquitetura.md) e [backlog.md](backlog.md). É transcrição, não invenção.

- [F3-6.2](backlog.md#f3-62--rfcs) — 4 RFCs (~40 min cada)
- [F3-6.3](backlog.md#f3-63--adrs) — 8 ADRs, **incluindo as dos cortes deste plano**
- [F3-6.1](backlog.md#f3-61--diagramas-de-componentes-e-de-sequência) — diagramas revisados contra o que foi construído
- [F3-5.2](backlog.md#f3-52--documentar-o-modelo-e-justificar-a-escolha-do-banco) — RFC-002 com ER e os dois `EXPLAIN ANALYZE`

### 🔵 Qua 09/09 — 3 h — READMEs e ensaio

- [F3-6.4](backlog.md#f3-64--readmes-dos-quatro-repositórios) — atualizar o do `oficina-app`; os outros 3 já existem
- [F3-2.6](backlog.md#f3-26--atualizar-openapi-postman-e-a-documentação-de-autenticação) — Postman apontando para o Gateway
- **Ensaio cronometrado do vídeo**, com o [roteiro](entrega.md#roteiro-do-vídeo) aberto

> Não pule o ensaio. É o que impede a gravação de virar 6 tomadas na quinta.

### 🏁 Qui 10/09 — 3 h — Gravação e entrega

- Gravar por blocos e editar
- [F3-7.2](backlog.md#f3-72--pdf-do-portal-e-compartilhamento-dos-repositórios) — PDF + `soat-architecture`
- Submeter
- **Só depois de submeter:** [F3-7.3](backlog.md#f3-73--controle-de-custo-e-teardown) — `terraform destroy` e checagem de órfãos

---

## Ordem de sacrifício (se atrasar)

Corte **de cima para baixo** — o dano é crescente.

> O antigo item nº 1 era o ambiente de homologação. Já foi gasto (corte 10) antes de começar: a primeira alavanca de emergência não existe mais.

| # | Sacrifique | Impacto |
|---|---|---|
| 1 | Painéis extras do dashboard (mantendo os 3 nomeados) | Baixo |
| 2 | Alertas 2 e 3 (mantendo o de falha de OS) | Baixo |
| 3 | Instrumentação `nrpq` do banco | Baixo |
| 4 | READMEs de infra reduzidos ao mínimo | Médio |
| 5 | RFCs curtas em vez de completas | Médio |
| 6 | Lambda authorizer | **Alto** — ver [o que não cortar](#o-que-não-cortar) |

**Nunca sacrifique** — sem isto a entrega não é avaliável:

- 4 repositórios com `main` protegida e CI/CD rodando
- Autenticação por CPF ponta a ponta pelo Gateway, incluindo **cliente inativo → 403**
- Aplicação no EKS com HPA
- RDS gerenciado provisionado por Terraform
- New Relic com APM, logs JSON, 1 dashboard e 1 alerta
- RFCs, ADRs, ER e diagramas
- **Vídeo e PDF**

> Se na quarta faltar tempo, **corte funcionalidade, nunca o vídeo**. Solução 80% pronta com vídeo e documentação boa é avaliável; solução 100% pronta sem vídeo não é entrega.

---

## Três regras

1. **Time-box de 45 minutos por bug de infraestrutura.** Estourou? Aplique o corte correspondente e documente como ADR. O sábado inteiro evapora em uma regra de security group.
2. **Commit e push todo dia**, mesmo incompleto. É evidência de processo e protege contra perder trabalho.
3. **Anote as decisões enquanto decide**, num `docs/rfcs/rascunho.md`. Terça você transcreve, não relembra — é a diferença entre 3 h e 6 h de documentação.
