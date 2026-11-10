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
| Repo Billing | `ProblemaTheu/oficina-billing-service` |
| Repo Execution | `ProblemaTheu/oficina-execution-service` |
| Aplicação Mercado Pago | `7172532218475199` (Checkout Pro · API de Orders · conta do vendedor de teste) · vendedor de teste `TESTUSER38172746988478263` (id `3832864978`) · comprador de teste id `3832647366` |
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

**Situação encontrada:** em ter 10/11 (D6 do plano original de 7 dias) nada do D0–D5 estava feito — os repos `oficina-billing-service` e `oficina-execution-service` não existiam, não havia migration `000007` nem código de saga, e este diário estava vazio. O PR do planejamento (#18) só foi mergeado em 09/11.

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
