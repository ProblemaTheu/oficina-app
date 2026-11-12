# Observabilidade da saga (Fase 4 — F4-7.2 a F4-7.4)

## `SagaEvent`

Custom event do New Relic emitido pelo OS Service (orquestrador) **depois do commit** de cada transição. É um evento por etapa registrada em `sagas_os_etapas`, mais o T1 na abertura da OS. Respostas duplicadas não emitem nada.

| Atributo | Exemplo | Observação |
|---|---|---|
| `saga_id`, `os_id` | UUIDs | Mesmas chaves dos logs: `saga_id:<id>` filtra Logs e eventos |
| `estado_anterior` → `estado` | `AGUARDANDO_PAGAMENTO` → `COMPENSANDO` | `estado_anterior` ausente na abertura |
| `etapa` | `T5` | T1–T7 (arquitetura §3.3) |
| `resultado` | `ok` · `falha` · `compensada` · `ignorada` | |
| `motivo` | `pagamento_recusado: cc_rejected_other_reason` | Em `falha`. Em `ignorada`: `resposta_inesperada` |
| `tipo_mensagem` | `PagamentoRecusado` | Resposta que causou a transição |
| `duracao_etapa_segundos` | `42.1` | Espera desde a transição anterior (só na 1ª etapa da transição) |
| `terminou` | `true` | **Só** no evento que levou a saga a `CONCLUIDA` ou `COMPENSADA` |
| `duracao_total_segundos` | `318.0` | Só quando `terminou` |
| `trace.id`, `span.id` | | Liga o evento ao trace distribuído |

Não há CPF, e-mail nem valor pago (mesma regra de LGPD do `OrdemServicoEvent`).

**Por que `terminou` e não `estado = 'COMPENSADA'`:** uma transição terminal registra várias etapas (ex.: T2 e T1 compensadas na mesma resposta). Além disso, respostas atrasadas chegam com a saga já terminal e viram `ignorada`. Contar por `estado` inflaria os desfechos; `terminou IS true` conta cada saga uma vez.

## Dashboard

[`dashboard-saga.json`](dashboard-saga.json): New Relic → **Dashboards → Import dashboard** → colar o JSON. A conta `8465573` já está nas queries; troque se for outra.

| Painel | Pergunta que responde |
|---|---|
| Sagas por desfecho | Quantas concluíram e quantas compensaram, ao longo do tempo |
| Taxa de compensação | % das sagas terminadas que compensaram |
| Duração ponta a ponta | p50/p95 da abertura à conclusão |
| Onde as sagas falham | Etapa × motivo das falhas |
| Tempo médio por etapa | Onde a saga espera (aprovação humana e pagamento dominam) |
| Transições recentes | Tabela para achar o `saga_id` e seguir nos Logs e no trace |
| Latência p95 / erros por serviço | Os 3 APMs (`oficina-api-prod`, `oficina-billing-prod`, `oficina-execution-prod`) |

## Alerta de compensação (F4-7.4)

**Alerts → Alert conditions → New alert condition → NRQL**:

```sql
SELECT percentage(count(*), WHERE estado = 'COMPENSADA')
FROM SagaEvent WHERE terminou IS true
```

| Campo | Valor |
|---|---|
| Window duration | 15 minutos (sliding window de 5 min) |
| Threshold | **Critical: above 30** for at least 5 minutes |
| Fill data gaps | None (sem saga terminada não há sinal) |
| Notification | Workflow com o destino de e-mail da equipe |

Para disparar na demo: 4 OS seguidas com placa `FAL*` (o Execution responde `ExecucaoFalhou`, e a saga estorna e compensa).

A DLQ de cada fila tem alarme no CloudWatch + SNS, no `infra/` de cada serviço.
