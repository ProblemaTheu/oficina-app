# RFC-001 — Escolha da nuvem: AWS

| Campo | Valor |
|---|---|
| **Status** | Aceita |
| **Autores** | Equipe SOAT — Tech Challenge Fase 3 |
| **Relacionadas** | [RFC-002](RFC-002-banco-de-dados.md), [RFC-003](RFC-003-autenticacao.md), [RFC-004](RFC-004-estrategia-de-ambientes.md) |

## Contexto

A Fase 3 exige mover a aplicação de um Kubernetes local (kind, na Fase 2) para
uma **nuvem real**, com os seguintes requisitos do enunciado:

- API Gateway como única porta de entrada;
- função serverless para autenticação por CPF;
- banco de dados **gerenciado**;
- cluster Kubernetes com **escalabilidade** (autoescalonamento);
- todo o provisionamento em **Terraform**, com deploy automático por CI/CD;
- deploy sem segredos de longa duração no GitHub (OIDC).

A escolha da nuvem condiciona todas as demais decisões (gateway, serverless,
banco, observabilidade) e precisa ser feita antes de qualquer código de
infraestrutura.

## Decisão

Adotar a **AWS** como provedor de nuvem único, em conta pessoal / Free Tier, na
região **us-east-1**.

Serviços utilizados:

| Necessidade | Serviço AWS |
|---|---|
| Porta de entrada / roteamento | API Gateway HTTP API (v2) |
| Autenticação serverless por CPF | AWS Lambda (Go, `provided.al2023`, arm64) |
| Banco de dados gerenciado | Amazon RDS PostgreSQL |
| Cluster Kubernetes com escala | Amazon EKS + HPA (metrics-server) |
| Segredos | AWS Secrets Manager |
| Contrato entre repositórios de infra | AWS SSM Parameter Store |
| CI/CD sem segredo de longa duração | IAM OIDC + GitHub Actions |
| Logs de borda | CloudWatch Logs (Lambda e API Gateway) |

## Alternativas consideradas

| Alternativa | Prós | Contras | Veredito |
|---|---|---|---|
| **AWS** | A Fase 2 já deixou `infra/environments/aws` esboçado (VPC + EKS + RDS); ecossistema serverless + gateway + IAM/OIDC maduro; RDS e EKS gerenciados; documentação e integração oficial com New Relic | Curva de VPC/EKS/IAM mais íngreme; custo de control plane do EKS | **Escolhida** |
| GCP (GKE + Cloud SQL + API Gateway) | GKE Autopilot simplifica o cluster | Retrabalho total sobre o esboço da Fase 2; Cloud Functions/API Gateway menos integrados ao fluxo de Lambda Authorizer HS256 | Descartada |
| Azure (AKS + Azure Database + APIM) | Boa integração corporativa | Sem base herdada; APIM tem custo/tier mínimo maior para o cenário | Descartada |

## Justificativa

1. **Reaproveitamento.** A Fase 2 já esboçou VPC + EKS + RDS em `infra/environments/aws`. Continuar na AWS preserva esse trabalho e o conhecimento da equipe.
2. **OIDC sem segredo de longa duração.** Conta pessoal permite criar IAM Roles e o *identity provider* OIDC do GitHub — requisito para o deploy automático sem `AWS_SECRET_ACCESS_KEY` versionado. Adotamos a divisão de duas roles: uma *read-only* para `terraform plan` em PR e uma de escrita para `apply` em `main`.
3. **Serverless + gateway integrados.** O par API Gateway HTTP API + Lambda Authorizer é o caminho nativo para validar o JWF na borda (ver [RFC-003](RFC-003-autenticacao.md)).
4. **Free Tier viável.** O ambiente completo cabe em conta pessoal com custo controlado (~US$ 5/dia com os cortes do plano de 10 dias), monitorado por AWS Budgets e destruído após a gravação.

## Consequências

**Positivas**
- Uma única conta/nuvem simplifica IAM, billing e teardown.
- SSM Parameter Store dá acoplamento fraco entre os 4 repositórios (ver [RFC-004](RFC-004-estrategia-de-ambientes.md)).

**Negativas / riscos assumidos**
- **Quota de vCPU de conta nova** (5 vCPUs) pode travar o 3º nó do EKS e a demonstração de escala — mitigado pedindo aumento para 32 vCPUs no dia 1.
- Custo do control plane do EKS é fixo enquanto o cluster existe — mitigado por janelas de provisionamento e teardown.
- **Lock-in** moderado (Lambda Authorizer, SSM, Secrets Manager). Aceitável para o escopo acadêmico; a lógica de domínio permanece portável (Go + PostgreSQL).

## Referências

- Planejamento: [`README.md` — Decisões técnicas já tomadas](../planejamentos/fase-3/README.md)
- [`roadmap.md` — Orçamento e janelas de provisionamento](../planejamentos/fase-3/roadmap.md)
- Enunciado: `docs/planejamentos/fase-3/pdfs/13SOAT - Fase 3 - Tech Challenge.pdf`
