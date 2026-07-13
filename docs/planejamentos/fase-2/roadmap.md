# Roadmap — Tech Challenge Fase 2

Sequência de execução sugerida, dependências e riscos. As tarefas referenciam os IDs de [backlog.md](backlog.md).

---

## Princípios de sequenciamento

1. **Estabilizar a base antes de evoluir** — correções de dívida técnica (E0) primeiro, pois afetam a arquitetura avaliada.
2. **Código antes de infra pesada** — as APIs precisam estar prontas para serem empacotadas e demonstradas.
3. **Infra incremental** — Docker → Kubernetes local → CI/CD → Terraform → (AWS opcional).
4. **Documentação e vídeo por último**, quando o ambiente já está estável.

---

## Sprints sugeridas

### Sprint 1 — Fundação e código (E0 + E1 + início E2)
Objetivo: base limpa e regras de negócio novas prontas.

- F2-0.2 — Corrigir dependência de camadas *(desbloqueia o resto do código)*
- F2-0.1 — Tratar `/cancel`
- F2-1.1 — Clean Code + golangci-lint
- F2-1.2 — Padronizar erros nos handlers
- F2-2.1 — Nova listagem de OS

**Saída:** código refatorado, listagem conforme enunciado, testes verdes.

### Sprint 2 — APIs de integração (fim E2 + E1 testes)
Objetivo: webhook e notificações.

- F2-2.2 — Webhook de aprovação/recusa
- F2-2.3 — Notificação por e-mail
- F2-2.4 — Atualizar OpenAPI + regenerar
- F2-1.3 — Testes dos fluxos críticos
- F2-1.4 — (Opcional) Testes de integração

**Saída:** contrato atualizado, integrações funcionando com mailer local.

### Sprint 3 — Containerização e Kubernetes (E3 + E4)
Objetivo: aplicação rodando em cluster local com escala.

- F2-3.1 / F2-3.2 — Dockerfile e compose revisados (+ MailHog)
- F2-4.1 → F2-4.6 — Namespace, ConfigMap/Secret, Deployment, Service, HPA, Postgres
- F2-4.7 — (Opcional) Job de migrations

**Saída:** `kubectl apply` sobe tudo; HPA escala sob carga.

### Sprint 4 — CI/CD e IaC (E6 + E5)
Objetivo: automação de ponta a ponta.

- F2-6.1 — CI (build/test/lint)
- F2-6.2 — Build/push Docker Hub
- F2-6.4 — Secrets da pipeline
- F2-6.3 — Deploy no cluster
- F2-5.1 / F2-5.2 — Terraform base + cluster local
- F2-5.4 — Doc Terraform
- F2-5.3 — (Opcional/futuro) Módulo AWS

**Saída:** pipeline completa; infra provisionável por código.

### Sprint 5 — Documentação e entrega (E7)
Objetivo: fechar entregáveis.

- F2-7.1 — README com arquitetura
- F2-7.2 — Diagrama de arquitetura
- F2-7.3 — Coleção de APIs
- F2-7.4 — Vídeo demonstrativo
- F2-7.5 — Entrega no portal + compartilhar repo

**Saída:** entrega submetida.

---

## Grafo de dependências (essencial)

```
F2-0.2 ──► F2-2.1 ──► F2-2.4 ──► F2-7.3
   │                    ▲
   └──► F2-2.2 ─────────┘
F2-0.1
F2-2.3 ──► F2-3.2
F2-1.1 ──► F2-6.1 ──► F2-6.2 ──► F2-6.3 ──► (vídeo)
F2-3.1 ──► F2-6.2
E4 (k8s) ─────────────► F2-6.3
F2-5.1 ──► F2-5.2 ──► F2-5.4
Todos ────────────────► F2-7.4 ──► F2-7.5
```

---

## Caminho crítico

`F2-0.2 → F2-2.1/F2-2.2 → F2-2.4 → E4 (Deployment+HPA) → F2-6.2 → F2-6.3 → F2-7.4 → F2-7.5`

Atrasos nesse caminho atrasam a entrega. Itens fora dele (AWS, testes de integração, Job de migrations) são paralelizáveis ou opcionais.

---

## Riscos e mitigações

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Deploy K8s a partir de runner hospedado não alcança cluster local | Alto | Usar self-hosted runner OU demonstrar deploy manual no vídeo; deixar o job pronto para EKS |
| `metrics-server` no kind exige configuração TLS extra | Médio | Documentar flags; testar HPA cedo (Sprint 3) |
| Provedor de e-mail real indisponível/custo | Médio | Mailer fake (MailHog) para local e vídeo; provedor real opcional |
| Módulo AWS gera custo inesperado | Médio | Deixar `aws` opt-in, nunca aplicado por padrão; `destroy` documentado |
| "Exclusão lógica" mal interpretada | Baixo | Confirmado como filtro na listagem, não deleção física; sem `deleted_at` a menos que se decida o contrário |
| Escopo de refatoração crescer demais | Médio | Limitar E1 aos fluxos críticos; não reescrever o que funciona |

---

## Divisão sugerida por integrante (se em grupo)

- **Pessoa A (Backend/domínio):** E0, E1, E2.
- **Pessoa B (Infra/DevOps):** E3, E4, E5.
- **Pessoa C (CI/CD + Doc):** E6, E7, integração e vídeo.

Sincronizar nos pontos de contrato (OpenAPI) e nos manifestos que consomem variáveis do código.
