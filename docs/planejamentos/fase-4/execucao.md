# Diário de execução — Fase 4

> Registro do que foi feito, valores reais do ambiente e achados técnicos que custaram tempo. O [plano](plano-execucao.md) diz o que fazer; este documento diz o que já aconteceu. Regra: **tudo que custou mais de 30 minutos entra aqui.**

---

## Valores herdados da Fase 3

Confirmar no D1 — o ambiente pode ter sido destruído após a entrega da Fase 3.

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
| Repo Billing | `ProblemaTheu/oficina-billing-service` |
| Repo Execution | `ProblemaTheu/oficina-execution-service` |
| Aplicação Mercado Pago | `[id]` · vendedor de teste `[usuario]` · comprador de teste `[usuario]` |
| Segredo do Billing | `oficina/prod/billing` (`db_password`, `mp_access_token`, `mp_webhook_secret`, `webhook_secret`) |
| Token interno | `oficina/prod/internal` |
| Filas | `[URLs]` |
| Tabela DynamoDB | `execution-tarefas` |
| RDS Billing | `[endpoint]` |
| CLBs | OS `[dns]` · Billing `[dns]` · Execution `[dns]` |
| URL do webhook MP | `https://<gateway>/billing/webhooks/mercadopago` |

---

## Pendências herdadas da Fase 3

| Pendência | Status |
|---|---|
| Branch protection nos repos existentes | ✅ ativa (confirmado em 31/10) |
| Mesmo ruleset nos 2 repos novos | ⬜ |
| Environment `prod` com reviewer | ⬜ |
| `soat-architecture` nos repos | ⬜ |
| Chave do Sonar `tech-challenge-1` → `oficina-app` | ⬜ |
| SonarCloud automático no `infra-k8s` | ⬜ |
| Rotacionar as chaves do New Relic | ⬜ |
| `OrdemServicoEvent` chegando ao New Relic (conferir) | ⬜ |

---

## Registro

<!-- Modelo de entrada:

## DD/MM — Dn concluído

**Entregável verificado:** ...

### Ajustes em relação ao backlog
| Escrito no backlog | O que foi feito | Por quê |
|---|---|---|

### Achados que custaram tempo
-->
