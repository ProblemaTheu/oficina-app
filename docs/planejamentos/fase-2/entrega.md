# Entrega no portal do aluno — checklist e conteúdo do PDF

O PDF da Fase 2 pede: *"PDF contendo o link do repositório github compartilhado
com o usuário `soat-architecture`; desenho da arquitetura com os recursos
escolhidos e link do vídeo (com até 15 minutos de duração) apresentando a
solução desenvolvida."*

## Checklist antes de submeter

- [ ] Branch `docs/planejamento-fase-2` merged na `main`
- [ ] Repositório compartilhado com o usuário **soat-architecture**
      (Settings → Collaborators → Add people) e acesso conferido
- [ ] Vídeo gravado (ver [roteiro-video.md](roteiro-video.md)) e publicado
      (YouTube/Vimeo, público ou não listado, ≤ 15 min)
- [ ] Link do vídeo adicionado no README (tabela "Onde está cada coisa")
- [ ] Diagrama de arquitetura exportado como imagem para o PDF
      (o Mermaid do README pode ser exportado em https://mermaid.live)
- [ ] PDF gerado com o conteúdo abaixo e submetido no portal

## Conteúdo do PDF

---

**Tech Challenge — Fase 2 — [Nome do grupo / integrantes]**

**Repositório GitHub** (compartilhado com `soat-architecture`):
https://github.com/problematheu/tech-challenge-1

**Vídeo demonstrativo** (até 15 min):
`[COLAR LINK DO YOUTUBE/VIMEO]`

**Desenho da arquitetura**:
`[COLAR IMAGEM exportada do diagrama "Arquitetura da solução" do README]`

Recursos escolhidos (resumo):
- **Aplicação**: Go 1.26, Clean Architecture, API-first (OpenAPI + oapi-codegen);
- **Kubernetes**: manifestos Kustomize em `/k8s` — Deployment (2 réplicas),
  Service, ConfigMap/Secret, HPA (CPU 50%, 2–5 réplicas), Postgres StatefulSet
  (local) e overlay preparado para EKS + RDS;
- **IaC**: Terraform em `/infra` — cluster kind + metrics-server (local) e
  VPC/EKS/RDS (AWS, opt-in);
- **CI/CD**: GitHub Actions — build/testes/lint em PRs, testes de integração
  com Postgres real e suíte de segurança (govulncheck, gosec, Trivy, Sonar);
- **Integrações**: webhook de aprovação de orçamento autenticado por HMAC e
  notificação de status por e-mail (SMTP/Mailpit).

---
