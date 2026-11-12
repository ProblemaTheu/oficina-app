# Diário de execução — Fase 4

> Registro do que foi feito, valores reais do ambiente e achados técnicos que custaram tempo. O [plano](plano-execucao.md) diz o que fazer; este documento diz o que já aconteceu. Regra: **tudo que custou mais de 30 minutos entra aqui.**

---

## Valores herdados da Fase 3

Confirmar no D1 (ter 10/11) — o ambiente pode ter sido destruído após a entrega da Fase 3.

| Item | Valor |
|---|---|
| Conta AWS | `706215605178` · usuário IAM `admin-cli` · região `us-east-1` |
| Bucket de state | `oficina-tfstate-706215605178` (versionado, lock nativo) |
| Roles OIDC | `gha-oficina-app`, `gha-oficina-lambda-auth`, `gha-oficina-infra-k8s`, `gha-oficina-infra-db` (+ `-plan`) |
| Conta New Relic | `8465573` — **chaves a rotacionar** |
| EKS | `oficina`, versão 1.36, 2 × `t3.small` em subnet pública |
| Namespace do OS | `oficina-prod` |
| Segredo da app | `oficina/prod/app` (`jwt_secret`, `webhook_secret`) |
| Imagem do OS | `problematheu/oficina-api` |
| Restrição da conta | Não cria ELBv2 (NLB/ALB) — só Classic Load Balancer |
| Quota de vCPU | 8 (pedido de 32 estava `CASE_OPENED`) |
| Budget | US$ 60/mês |

## Valores novos da Fase 4

| Item | Valor |
|---|---|
| Conta AWS (nova) | `584481982243` · usuário IAM `admin-cli` · membro da organization `o-7hzscs8ddo` (management `564612156833`) · caso de suporte **179445505200032** (restrição de conta nova) |
| Bucket de state | `oficina-tfstate-584481982243` |
| Repo Billing | `ProblemaTheu/oficina-billing` |
| Repo Execution | `ProblemaTheu/oficina-execution` |
| Aplicação Mercado Pago | `7172532218475199` (Checkout Pro · API de Orders · conta do vendedor de teste) · vendedor de teste `TESTUSER38172746988478263` (id `3832864978`) · comprador de teste id `3832647366` |
| Segredo do Billing | `oficina/prod/billing` (`db_password`, `mp_access_token`, `mp_webhook_secret`, `webhook_secret`) |
| Token interno | `oficina/prod/internal` |
| Filas | `os-saga-replies.fifo`, `billing-commands.fifo`, `execution-commands.fifo` (+ `-dlq.fifo`) — URLs na SSM `/oficina/prod/{os,billing,execution}/fila-*-url` |
| Tabela DynamoDB | `execution-tarefas` |
| RDS Billing | `[endpoint]` |
| CLBs | OS `[dns]` · Billing `[dns]` · Execution `[dns]` |
| URL do webhook MP | `https://<gateway>/billing/webhooks/mercadopago` |

---

## Pendências herdadas da Fase 3

| Pendência | Status |
|---|---|
| Branch protection nos repos existentes | ✅ ativa (confirmado em 31/10) |
| Repos novos criados (`oficina-billing`, `oficina-execution` — sem o sufixo `-service`) | ✅ 10/11 14:38, **vazios** (sem commit, sem `main`) |
| Mesmo ruleset nos 2 repos novos (+ `required_status_checks`) | ⬜ — só depois do 1º commit na `main` |
| Roles OIDC `gha-oficina-billing` / `gha-oficina-execution` | ✅ [infra-k8s#3](https://github.com/ProblemaTheu/oficina-infra-k8s/pull/3) (bootstrap reaplicado na conta nova) |
| 3º nó, access entries e namespaces `billing-prod` / `execution-prod` | ⬜ [infra-k8s#5](https://github.com/ProblemaTheu/oficina-infra-k8s/pull/5) — plan ok (86 to add); merge depois da quota de vCPU |
| Environment `prod` com reviewer | ⬜ |
| `soat-architecture` nos repos | ⬜ |
| Chave do Sonar `tech-challenge-1` → `oficina-app` | ⬜ |
| SonarCloud automático no `infra-k8s` | ⬜ |
| Rotacionar as chaves do New Relic | ⬜ |
| `OrdemServicoEvent` chegando ao New Relic (conferir) | ⬜ |

---

## Registro

## 12/11 — Deploy pronto no código; conta AWS nova restrita

**Entregável verificado:** os 3 serviços com `k8s/` (base + overlay prod), `infra/` próprio e CD independente (testes com gate de 80% → imagem → deploy no EKS → smoke → rollback), mergeados e com o CD verde no `main` (deploy pulado por `DEPLOY_ENABLED`): imagens `problematheu/oficina-billing:9df3b43` e `problematheu/oficina-execution:3c65f67` publicadas. Lambda sem acesso ao banco do OS (F4-1.5) e `SagaEvent` (F4-7.2) implementados. Push direto na `main` rejeitado nos 3 repos de serviço (`docs/evidencias/`).

| PR | Conteúdo |
|---|---|
| app [#30](https://github.com/ProblemaTheu/oficina-app/pull/30) · billing [#4](https://github.com/ProblemaTheu/oficina-billing/pull/4) · execution [#5](https://github.com/ProblemaTheu/oficina-execution/pull/5) | F4-6.1–6.3: `infra/` (filas + DLQ + alarme, RDS do Billing, DynamoDB, Pod Identity, SSM), manifestos, CD |
| app [#31](https://github.com/ProblemaTheu/oficina-app/pull/31) · lambda-auth [#6](https://github.com/ProblemaTheu/oficina-lambda-auth/pull/6) | F4-1.5: `GET /internal/clientes/por-cpf/{cpf}`; Lambda fora da VPC, sem DSN; rotas `/billing/*` e `/execution/*` no Gateway |
| app [#32](https://github.com/ProblemaTheu/oficina-app/pull/32) | F4-7.2/7.3: `SagaEvent` + `observability/dashboard-saga.json` + NRQL do alerta |
| infra-k8s [#5](https://github.com/ProblemaTheu/oficina-infra-k8s/pull/5) · infra-db [#4](https://github.com/ProblemaTheu/oficina-infra-db/pull/4) · lambda-auth [#5](https://github.com/ProblemaTheu/oficina-lambda-auth/pull/5) | Recriados a partir de branches do repo (substituem os PRs de fork) |

### Ajustes em relação ao backlog
| Escrito no backlog | O que foi feito | Por quê |
|---|---|---|
| `SagaEvent` com `sagaId`, `estado`… e painéis por `estado IN ('CONCLUIDA','COMPENSADA')` | Atributos em snake_case (como `OrdemServicoEvent` e os logs) e `terminou = true` só na transição terminal | A última compensação registra 2 etapas na mesma transição e respostas atrasadas chegam com a saga já terminal: contar por `estado` inflava os desfechos |
| DNS do CLB do OS fixo em `backend_url` (lambda-auth) e endpoint do RDS fixo no overlay prod | CLBs descobertos pela tag `kubernetes.io/service-name`; `DB_HOST` e filas lidos da SSM pelo CD | Os dois valores eram da conta antiga e mudam a cada recriação |
| Execution com 1 réplica | 1 réplica **e** estratégia `Recreate` | O surge de um rolling update também colocaria 2 relays sem líder ao mesmo tempo |
| SNS do alarme da DLQ sem detalhe | SNS cifrado com **CMK própria** | A chave gerenciada `aws/sns` não permite o CloudWatch publicar (alarme falharia em silêncio); sem cifrar, Sonar e Trivy reprovam |

### Achados que custaram tempo
- **Conta nova restrita pela AWS:** quota de vCPU On-Demand = **0** (padrão 5), `RequestServiceQuotaIncrease` e Lambda com `AccessDenied` mesmo para `AdministratorAccess`. Não é SCP (`simulate-principal-policy` mostra `AllowedByOrganizations: true`). A API de suporte exige plano pago → caso aberto pelo console (*Account and billing*).
- **PR de fork não roda `terraform plan`:** workflows de `pull_request` vindos de fork não recebem as *variables* do repo nem `id-token: write` — o plan falhava com `Input required and not supplied: aws-region`, com qualquer valor de variável. Resolvido com Write nos repos de infra e PRs a partir de branches do próprio repo. Os plans autenticam com as roles `-plan` da conta nova (infra-k8s: `86 to add`; infra-db e lambda-auth falham só por SSM que o infra-k8s ainda vai criar).
- **gitleaks varre todos os commits do PR:** placeholder `troque-me-com-pelo-menos-32-caracteres` foi pego por `generic-api-key` (entropia); corrigir num commit novo não basta — os branches foram reescritos.
- **Teste de relay instável** nos 3 repos (`publicadas 28 de 50`): 50 ciclos fixos com 3 réplicas disputando a liderança; em runner lento quase todo ciclo volta sem publicar. Agora drena a outbox com prazo.
- **`soat-architecture`** aparece como colaborador só no `oficina-app` (404 nos outros 5) — conferir convites pendentes (exige admin).

## 11/11 — Código da saga e documentação concluídos (conta AWS perdida)

**Entregável verificado:** saga completa nos 3 serviços reais, no `docker-compose.saga.yml`, nos 4 cenários — caminho feliz (estoque 10→8), falha no reparo com estorno (`refunded`, reserva liberada), orçamento recusado e cartão recusado (order `canceled`, nova tentativa bloqueada). Verificado por: BDD (`go test -tags bdd`, 4 cenários, também no CI), `scripts/demo-saga.sh todos` e a coleção Postman (`docs/postman/fase-4-saga.json`, Postman CLI, 0 falhas).

| Repositório | Entregue | Cobertura unitária |
|---|---|---|
| `oficina-app` | Mensageria (outbox/relay/consumer), migration `000007`, `CriarOS` transacional, orquestrador (100%), BDD | gate ≥ 80% mantido |
| `oficina-billing` | Orçamento, Orders API do MP, webhook com `x-signature`, reconciliador, estorno, stub do MP | 97,9% |
| `oficina-execution` | Fila em DynamoDB single-table, API do mecânico, falha simulada | 98,8% |

Documentação: seção Fase 4 + "Saga Pattern — por que orquestrada" no README, RFC-005, RFC-006, ADRs 009–016, `docs/arquitetura.md` e `docs/openapi.yaml` nos dois serviços novos.

### Ajustes em relação ao backlog
| Escrito no backlog | O que foi feito | Por quê |
|---|---|---|
| Deploy na AWS ao fim de cada trilha | **Bloqueado**: a conta AWS foi encerrada ao fim do crédito gratuito | Toda a infra (E0/E6) será refeita numa conta nova; o CD do `oficina-app` só faz deploy com `DEPLOY_ENABLED=true` |
| `infra-k8s` #3 e #4 | Ficam obsoletos (apontam para a conta antiga) | Serão refeitos com o novo account ID |
| ADR-015 (Lambda via REST) e ADR-016 (Pod Identity) implementadas | Registradas como **aceitas, implementação pendente** | Dependem da conta nova |
| `GET /v1/work-orders/{id}` com bloco `saga` | Não implementado; o estado é consultado em `sagas_os`/`sagas_os_etapas` | Corte de escopo; evolução |

### Achados que custaram tempo
- Healthcheck do stub do MP em porta errada no compose (corrigido em #26).
- Quality gate do Sonar em projeto novo (`None` na `main`) reprovava com `qualitygate.wait=true` → espera só em PR.
- `DeduzirEstoquePecas` legado ignorava a reserva e violava o `CHECK` → passa a comparar com o disponível.

## 10/11 — Mercado Pago validado

**Entregável verificado:** cobrança, consulta e estorno do Checkout Pro funcionando por `curl` com credenciais de teste (order `ORDTST01M73RYFVR9787NCYPB515912X`: `processed` → `refunded`).

### Ajustes em relação ao backlog
| Escrito no backlog | O que foi feito | Por quê |
|---|---|---|
| Checkout Pro via **API de Preferences** | Checkout Pro via **API de Orders** (`/v1/orders`) | `POST /v1/payments/{id}/refunds` responde `401 Unauthorized use of live credentials` para pagamentos de conta de teste — com qualquer token testado. Com `/v1/orders/{id}/refund` o estorno funciona |
| Usar as credenciais de produção do vendedor de teste | Usar as **credenciais de teste** de uma aplicação na conta pessoal | É o que a doc atual manda; o token termina com o id do vendedor de teste vinculado, o que é esperado |
| Pagar em janela anônima | Pagar em **outro navegador / perfil Convidado** | Janelas anônimas do Chrome compartilham sessão: o checkout vê o vendedor pagando a si mesmo ("A transação não aceita este meio de pagamento") |

### Achados que custaram tempo (~1h30)
- Estorno: Preferences/Payments bloqueia com conta de teste; Orders funciona. Idempotência confirmada (mesma chave → mesmo `REF...`); chave nova em order estornada → `409 order_already_refunded`.
- Os tokens de teste aparecem no histórico de conversa — **renovar as credenciais** no painel antes da gravação.
- **Webhook (ngrok → receptor local):** simulações de `order` e `payment` entregues e com `x-signature` **válido** pela regra do F4-2.4; `ts` em **segundos**; query `?data.id=<id>&type=order`. Evento correto: **Order (Mercado Pago)**.
- **Cartão recusado:** order fica `action_required / waiting_retry` e aceita nova tentativa (até 20) — `retries.allowed: false` é ignorado. **`POST /v1/orders/{id}/cancel`** → `canceled` e o checkout bloqueia ("O que você queria pagar não está disponível"). Billing cancela a order antes de emitir `PagamentoRecusado`.
- A rede corporativa bloqueia `*.ngrok-free.dev` **de saída** (TLS reset), mas o MP chega ao túnel normalmente.
- **Orders de teste não disparam webhook.** A aplicação só tem configuração de webhook de produção (`webhooks?env=production`, sem seletor Teste/Produção apesar da doc) e *Notificações enviadas* mostra **nenhuma** para os 5 pagamentos `ORDTST...`; os *Logs* da aplicação mostram todas as nossas chamadas. **Decisão:** reconciliador no Billing (consulta pendentes a cada 10 s, F4-2.4 passo 7) como mecanismo principal; webhook validado por simulação e mantido como acelerador.
- Aplicação definitiva: `7172532218475199`, criada na conta do **vendedor de teste**, tipo **API de Orders**.

## 10/11 — Replanejamento

**Situação encontrada:** em ter 10/11 (D6 do plano original de 7 dias) nada do D0–D5 estava feito — os repos `oficina-billing` e `oficina-execution` não existiam, não havia migration `000007` nem código de saga, e este diário estava vazio. O PR do planejamento (#18) só foi mergeado em 09/11.

**Decisão:** [plano de 4 dias](plano-execucao.md) (ter 10/11 → sex 13/11), submissão na sexta até 18 h, sábado e domingo como reserva. Novos cortes 7′ e 18–25; o estorno do pagamento tardio passou a **não cortável**.

### Ajustes de desenho feitos na revisão
| Escrito antes | Corrigido para | Por quê |
|---|---|---|
| Webhook do MP grava `notificacoes_mp` e responde 200; se o `GET /payments` falhar, 500 | `GET` primeiro; grava tudo numa transação só depois; idempotência por `mp_payment_id` + transição de status | O reenvio do MP batia na PK de `x_request_id` e o pagamento nunca era processado |
| Relay do DynamoDB sem líder, Execution com HPA 1–3 | Execution (e Billing) com 1 réplica fixa | Duas réplicas podiam publicar mensagens da mesma saga fora de ordem |
| `INSERT INTO status_ordens (id, nome) ... ON CONFLICT (nome)` | `nome_status` + `descricao`, id fixo | A coluna real é `nome_status`; a migration não rodaria |
| `reserva_estado` só no critério de aceite | Coluna em `itens_os_pecas` no DDL | Idempotência de `Liberar`/`Baixar` dependia dela |
| `sagas_os_etapas.resultado` sem `ignorada` | `CHECK` inclui `ignorada` | O orquestrador grava `ignorada` para respostas inesperadas |
| Estado `INICIADA`; T6/T7 desalinhados; `DevolverEstoque` | Saga nasce em `DIAGNOSTICO_PENDENTE`; T6 = início do reparo (pivô); sem `DevolverEstoque` | Estado sem transição de saída; compensação nunca alcançada |
| Relay para no primeiro erro, para sempre | `falhou_em` após 10 tentativas | Uma linha venenosa travava todas as sagas |
| `/internal/*` "não exposto" | Alcançável pelo CLB público, protegido por token | Precisão na ADR-015 |

<!-- Modelo de entrada:

## DD/MM — Dn concluído

**Entregável verificado:** ...

### Ajustes em relação ao backlog
| Escrito no backlog | O que foi feito | Por quê |
|---|---|---|

### Achados que custaram tempo
-->
