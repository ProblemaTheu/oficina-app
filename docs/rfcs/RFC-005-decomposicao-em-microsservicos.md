# RFC-005 — Decomposição em microsserviços

| Campo | Valor |
|---|---|
| **Status** | Aceita |
| **Autores** | Equipe SOAT — Tech Challenge Fase 4 |
| **Relacionadas** | [RFC-006](RFC-006-saga-e-consistencia-distribuida.md), [ADR-009](../architecture-decisions.md#adr-009--banco-por-serviço-postgresql-os-billing-e-dynamodb-execution), [ADR-013](../architecture-decisions.md#adr-013--código-de-mensageria-duplicado-nos-repositórios) |

## Contexto

Até a Fase 3 a oficina era um monólito modular (`oficina-app`) com um único PostgreSQL. A Fase 4 pede **no mínimo três microsserviços** — OS, pagamento/orçamento e execução —, cada um com **banco próprio** (ao menos um relacional e um NoSQL), **sem acesso ao banco de outro serviço**, comunicando-se por API ou mensageria, com repositório, CI/CD e cobertura ≥ 80% próprios.

Os bounded contexts mapeados na Fase 1 ([bounded-contexts.md](../bounded-contexts.md)) já separavam **Atendimento/OS**, **Orçamento e Pagamento** e **Execução**. A dúvida não era *se* cortar, mas **onde** cada dado e cada regra fica.

## Decisão

### 1. Três serviços, alinhados aos contextos

| Serviço | Repositório | Fica com | Banco |
|---|---|---|---|
| **OS Service** | `oficina-app` | Clientes, veículos, usuários, catálogo de serviços e peças, **estoque com reserva**, ciclo de vida da OS, histórico, **orquestrador da saga** | PostgreSQL (o da Fase 3) |
| **Billing Service** | `oficina-billing` | Orçamento (geração, aprovação, recusa), cobrança no Mercado Pago, verificação e estorno | PostgreSQL próprio |
| **Execution Service** | `oficina-execution` | Fila de diagnóstico e reparo, API do mecânico | DynamoDB |

### 2. O estoque fica no OS Service

Estoque não ganhou serviço próprio. A reserva das peças acontece na **abertura da OS** e a baixa no **início do reparo** — os dois eventos já pertencem ao ciclo da OS. Separar o estoque obrigaria mais uma etapa remota (com compensação) para algo que hoje é um `UPDATE` na mesma transação. A reserva (`pecas.estoque_reservado`) foi introduzida justamente para que a saga possa liberar o que não foi usado.

### 3. O repositório do OS mantém o nome `oficina-app`

Renomear quebraria o histórico das Fases 1–3, os links do avaliador e a integração com Sonar e Docker Hub. O README esclarece que `oficina-app` é o OS Service.

### 4. Contratos explícitos, sem biblioteca compartilhada

Os serviços se conhecem só pelo **envelope de mensagem** e pelo **catálogo de tipos** ([arquitetura §5](../planejamentos/fase-4/arquitetura.md#5-contrato-de-mensagens)) e pelo JWT comum. O código de mensageria é copiado em cada repositório (ADR-013).

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| **4–5 serviços** (Estoque, Catálogo e Notificação separados) | Mais etapas remotas e compensações sem ganho de autonomia: estoque e catálogo mudam junto com a OS; notificação é efeito colateral sem estado relevante |
| **Billing dividido em Orçamento e Pagamento** | Orçamento aprovado e cobrança formam um único ciclo com o mesmo dono; separar criaria uma saga dentro da saga |
| **Orquestrador como serviço próprio** | Mais um deploy e mais um banco; o orquestrador precisa gravar o estado da saga **na mesma transação** que muda a OS — fora do OS Service isso exigiria mais uma etapa distribuída |
| **Execution em PostgreSQL** | Cumpriria o funcional, mas o requisito pede ao menos um NoSQL; a fila é acesso por chave e por estado, encaixe natural de DynamoDB (ADR-009) |

## Consequências

- **Positivas:** cada serviço evolui, escala e é implantado sozinho; o banco de cada um é privado (credencial e role próprias); falha do Billing não derruba a abertura de OS — a saga espera.
- **Negativas:** consistência passa a ser **eventual** e exige saga, outbox e inbox (RFC-006); debugar exige trace distribuído (New Relic atravessando as filas); três pipelines para manter.
- **Evolução:** se o estoque ganhar regras próprias (compras, fornecedores), ele é o próximo candidato a serviço — a reserva já tem a semântica de compensação pronta.
