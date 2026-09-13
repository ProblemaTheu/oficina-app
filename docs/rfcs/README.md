# RFCs — Tech Challenge Fase 3

Este diretório reúne as **Request for Comments** (RFCs) da Fase 3: as decisões
técnicas de maior impacto, registradas com o contexto, as alternativas
consideradas e as consequências assumidas — no momento em que foram tomadas.

> **RFC × ADR.** As RFCs discutem decisões amplas e com trade-offs de negócio/custo
> (que nuvem, que banco, que estratégia de autenticação, quantos ambientes). Os
> [ADRs](../architecture-decisions.md) registram decisões pontuais de arquitetura de
> software. Onde há sobreposição, a RFC referencia o ADR correspondente.

| RFC | Título | Status |
|-----|--------|--------|
| [RFC-001](RFC-001-escolha-da-nuvem.md) | Escolha da nuvem: AWS | Aceita |
| [RFC-002](RFC-002-banco-de-dados.md) | Banco de dados gerenciado e modelo relacional | Aceita |
| [RFC-003](RFC-003-autenticacao.md) | Estratégia de autenticação: CPF serverless, API Gateway e JWT | Aceita |
| [RFC-004](RFC-004-estrategia-de-ambientes.md) | Estratégia de ambientes e CI/CD | Aceita |

## Convenções

- **Status possíveis:** Rascunho · Em discussão · Aceita · Substituída · Descartada.
- Uma RFC aceita não é reescrita: se a decisão muda, cria-se uma nova RFC que
  **substitui** a anterior (registrando o número).
- As decisões aqui sintetizam o planejamento detalhado em
  [`docs/planejamentos/fase-3/`](../planejamentos/fase-3/).
