# Planejamento — Tech Challenge Fase 2

> Evolução da API de Oficina Mecânica para garantir **qualidade, resiliência e escalabilidade**, incorporando práticas modernas de infraestrutura e automação.

Este diretório contém o planejamento completo de execução da Fase 2. Ele traduz os requisitos do enunciado (`pdfs/14SOAT - Fase 2 - Tech challenge.pdf`) em épicos e tarefas acionáveis, com critérios de aceite e sugestões técnicas, de modo que qualquer pessoa do time possa executar sem depender de contexto verbal.

---

## Índice

| Documento | Conteúdo |
|-----------|----------|
| [README.md](README.md) | Este arquivo — visão geral, resumo executivo, achados e critérios globais |
| [backlog.md](backlog.md) | Backlog detalhado: épicos, tarefas, critérios de aceite e estimativas |
| [roadmap.md](roadmap.md) | Ordenação sugerida em sprints, grafo de dependências e riscos |

---

## Resumo executivo

A Fase 1 entregou uma API REST em Go funcional, com Clean Architecture, PostgreSQL, JWT, Docker e pipeline de segurança. A Fase 2 **não recria** o produto: ela **evolui** a base existente em quatro frentes:

1. **Evolução da aplicação** — refatoração (Clean Code/Clean Architecture), novas regras nas APIs de Ordem de Serviço e testes dos fluxos críticos.
2. **Infraestrutura** — containerização revisada, orquestração com Kubernetes (Deployments, Services, ConfigMaps, Secrets, HPA) e Infraestrutura como Código (Terraform).
3. **Automação** — pipeline de CI/CD completa: build, testes, imagem Docker, deploy no cluster e do banco.
4. **Documentação e entrega** — README com desenho de arquitetura, coleção de APIs, vídeo demonstrativo e PDF de entrega.

### Decisões técnicas já tomadas

| Tema | Decisão | Racional |
|------|---------|----------|
| **Kubernetes** | Local **agora** (kind/minikube); AWS (EKS + RDS) preparado para o **futuro** | Reduz custo/complexidade inicial; manifestos portáveis e Terraform modularizado permitem promover para cloud sem reescrever |
| **Aprovação de orçamento** | **Webhook** (endpoint inbound) | Desacoplamento e suporte a alto volume de requisições externas |
| **Registry de imagens** | **Docker Hub** | Conta já existente |
| **Notificação ao cliente** | **Email** disparado na mudança de status | Requisito do enunciado ("atualização de status via ferramenta como e-mail") |

---

## Mapa de entregáveis (enunciado → artefato)

| Requisito do enunciado | Artefato no repositório | Épico |
|------------------------|-------------------------|-------|
| Refatoração Clean Code / Clean Architecture | `internal/**` refatorado | E1 |
| Testes automatizados dos fluxos críticos | `**/*_test.go` | E1 |
| Abertura de OS | `POST /v1/work-orders` (revisar) | E2 |
| Consulta de status da OS | `GET /v1/work-orders/{id}/status` (revisar) | E2 |
| Aprovação de orçamento por notificação externa | `POST /v1/webhooks/...` (novo webhook) | E2 |
| Listagem de OS ordenada e com exclusão lógica | `GET /v1/work-orders` (nova regra) | E2 |
| Atualização de status via e-mail | Serviço de notificação por e-mail | E2 |
| Dockerfile e docker-compose revisados | `Dockerfile`, `docker-compose.yml` | E3 |
| Manifestos Kubernetes | `/k8s` | E4 |
| Scripts Terraform | `/infra` | E5 |
| Pipeline CI/CD | `.github/workflows/` | E6 |
| README com arquitetura e instruções | `README.md`, `docs/` | E7 |
| Coleção de APIs | `docs/postman_collection.json` / OpenAPI | E7 |
| Vídeo demonstrativo (≤15 min) | Link no README + PDF | E7 |
| PDF de entrega + repo compartilhado com `soat-architecture` | Entrega no portal | E7 |

---

## Épicos

| ID | Épico | Objetivo | Prioridade |
|----|-------|----------|------------|
| **E0** | Correções e dívida técnica | Sanar divergências entre doc e código antes de evoluir | Alta |
| **E1** | Refatoração e testes | Clean Code, Clean Architecture e cobertura dos fluxos críticos | Alta |
| **E2** | Evolução das APIs de OS | Listagem ordenada, webhook de aprovação e notificação por e-mail | Alta |
| **E3** | Containerização | Revisar Dockerfile e docker-compose | Média |
| **E4** | Kubernetes | Manifestos com Deployments, Services, ConfigMaps, Secrets e HPA | Alta |
| **E5** | Terraform (IaC) | Provisionamento de cluster e banco, local e AWS | Média |
| **E6** | CI/CD | Pipeline build → test → imagem → deploy | Alta |
| **E7** | Documentação e entrega | README, arquitetura, coleção, vídeo e PDF | Alta |

Detalhamento completo em [backlog.md](backlog.md).

---

## Achados da revisão de código (viram tarefas no E0)

Durante a revisão da base da Fase 1, foram identificados pontos que precisam de decisão/correção **antes** da evolução:

> **Fonte da verdade:** os PDFs da Fase 1 e Fase 2. Ambos listam **exatamente os mesmos 6 status** (Recebida, Em diagnóstico, Aguardando aprovação, Em execução, Finalizada, Entregue) e o **código está correto e fiel a eles**. O board do Miro (preenchido manualmente) adicionou `Aprovada` e `Cancelada`, que **não existem na especificação** — o Miro é que está divergente, não o código.

1. **Documentação de `/cancel` fora da especificação** — O `README.md` e o diagrama da máquina de estados documentam `POST /v1/work-orders/{id}/cancel` e o status `cancelada`, mas eles **não existem no código nem nos PDFs**. Não é dívida técnica de domínio — é a documentação que extrapolou (influência do Miro). **Correção:** remover `/cancel` e `cancelada` do README e do diagrama, alinhando a doc ao código e ao enunciado. Implementar cancelamento é um extra **opcional**, não exigido.

2. **Violação de dependência de camadas** — A camada de aplicação importa a de infraestrutura: `internal/application/usecase/ports.go` usa `repository.ListarOSParams`, `repository.RelatorioTempoMedioParams` e `repository.ItemTempoMedio`. Isso contraria o princípio de dependências da Clean Architecture (o domínio/aplicação não deve depender de infra). Como a Fase 2 é **avaliada** justamente em "separação adequada de camadas e dependências", esta correção é prioritária. **Esta é a única dívida técnica real da lista.**

3. **Miro divergente da especificação** — O board prevê os status `Aprovada` e `Cancelada`, ausentes dos PDFs e do código (corretamente). Como a Fase 2 exige o desenho de arquitetura consistente, **corrigir o Miro** para refletir os 6 status oficiais (remover `Aprovada`/`Cancelada`). O código **não** deve ser alterado.

---

## Critérios de aceite globais (Definition of Done da fase)

Uma entrega da Fase 2 só é considerada concluída quando:

- [ ] `go build ./...` e `go test ./...` passam sem erros.
- [ ] A aplicação sobe localmente com `docker compose up` e responde em `/health/ready`.
- [ ] A aplicação sobe em um cluster Kubernetes local e escala via HPA sob carga.
- [ ] O Terraform provisiona a infraestrutura descrita e está documentado.
- [ ] A pipeline de CI/CD executa build, testes, imagem Docker e deploy de ponta a ponta.
- [ ] O README contém o desenho da arquitetura, instruções de execução local, deploy K8s e Terraform.
- [ ] Existe coleção de APIs (Postman/Swagger) atualizada e linkada.
- [ ] Vídeo demonstrativo (≤15 min) publicado e linkado.
- [ ] Repositório compartilhado com o usuário `soat-architecture` e PDF de entrega submetido no portal.
