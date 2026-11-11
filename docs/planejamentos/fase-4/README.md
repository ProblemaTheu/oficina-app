# Planejamento — Tech Challenge Fase 4

> Quebrar a aplicação da oficina em **microsserviços independentes** — cada um com repositório, infraestrutura, banco e pipeline próprios — e coordenar o ciclo de vida da ordem de serviço com o **Saga Pattern**, com compensação segura em caso de falha, pagamento real pelo **Mercado Pago**, BDD, cobertura ≥ 80% por serviço e deploy automatizado no EKS.

Este diretório traduz o enunciado (`SOAT - Fase 4 - Tech challenge.pdf`) em épicos e tarefas executáveis, com critérios de aceite e **passo a passo técnico**, no mesmo formato das fases [2](../fase-2/) e [3](../fase-3/).

---

## Índice

| Documento | Conteúdo |
|-----------|----------|
| [README.md](README.md) | Este arquivo — resumo executivo, decisões, mapa de entregáveis, achados do estado atual, épicos e DoD |
| [arquitetura.md](arquitetura.md) | Arquitetura-alvo: componentes, posse de dados, catálogo de mensagens, máquina de estados da saga, sequências (caminho feliz e compensação) e topologia dos repositórios |
| [backlog.md](backlog.md) | Backlog detalhado — **é aqui que está o "como fazer"**, com código, comandos e critérios de aceite |
| **[plano-execucao.md](plano-execucao.md)** | ⏱️ **Documento de trabalho** — **replanejado em 10/11: 4 dias (ter 10/11 → sex 13/11)** em 3 trilhas paralelas, code freeze na qui 12/11 às 22 h, **submissão na sex 13/11 até 18 h** (prazo do portal 15/11). Cortes e ordem de sacrifício. **Comece por aqui.** |
| [roadmap.md](roadmap.md) | Sprints, grafo de dependências, caminho crítico, riscos, custo e divisão por integrante |
| [entrega.md](entrega.md) | Checklist de entrega, roteiro do vídeo e conteúdo do PDF do portal |
| [execucao.md](execucao.md) | 📋 Diário de execução — valores herdados da Fase 3 e espaço para registrar o que for feito |

---

## Resumo executivo

A Fase 3 deixou um **monólito modular em Go** (`oficina-app`) rodando no EKS, atrás de um API Gateway com autenticação por CPF, RDS PostgreSQL, New Relic e quatro repositórios (app, Lambda, infra-k8s, infra-db).

A Fase 4 muda o eixo de novo: sai de "um serviço bem operado" e entra em **"vários serviços que precisam concordar entre si"**. São seis frentes:

1. **Decomposição** — o `oficina-app` vira o **OS Service** e perde para dois serviços novos o orçamento/pagamento (**Billing Service**) e a fila de diagnóstico/reparo (**Execution Service**). Três repositórios de serviço, cada um com seu banco.
2. **Banco por serviço (SQL + NoSQL)** — OS e Billing em **PostgreSQL** (RDS, instâncias separadas); Execution em **DynamoDB**. Nenhum serviço lê o banco de outro — inclusive a Lambda de autenticação, que hoje lê `clientes` direto.
3. **Mensageria + Saga orquestrada** — **Amazon SQS FIFO** para comandos e respostas; o **OS Service orquestra** a saga da OS, com estado persistido, **outbox transacional**, consumidores idempotentes e compensações em ordem inversa.
4. **Pagamento real** — Billing integrado ao **Mercado Pago** (Checkout Pro em sandbox), webhook com validação de `x-signature`, confirmação ativa do pagamento e **estorno como compensação**.
5. **Qualidade** — testes unitários ≥ 80% **por serviço**, um fluxo completo em **BDD (godog)**, SonarCloud com quality gate nos três repositórios.
6. **CI/CD e observabilidade por serviço** — pipeline independente (build → testes → qualidade → deploy no EKS) em cada repo, `main` protegida, e **rastreamento distribuído atravessando as filas** no New Relic.

> **A regra de ouro desta fase:** a regra de negócio da OS já existe e está testada (98% de cobertura no critério do Sonar). Não reescreva — **mova**. O trabalho novo é a **costura**: contratos de mensagem, estado da saga, outbox/inbox, compensação, integração com o Mercado Pago e três pipelines. É aí que estão os bugs e a nota.

---

## Decisões técnicas recomendadas

Cada linha vira uma ADR ou RFC (ver [F4-8.2](backlog.md#f4-82--rfcs-e-adrs-da-fase-4)). As marcadas com 🔁 têm alternativa registrada nos [cortes](plano-execucao.md#os-cortes).

| Tema | Decisão | Racional |
|------|---------|----------|
| **Divisão** | **OS Service** (`oficina-app`), **Billing Service** (`oficina-billing`), **Execution Service** (`oficina-execution`) — exatamente os três sugeridos no enunciado | São os três momentos da OS com donos diferentes no negócio (atendimento, cobrança, oficina) e com ritmos de mudança distintos. Ir além de 3 aumenta o trabalho de costura sem aumentar a nota. |
| **O que fica no OS Service** | Clientes, veículos, usuários/login de funcionário, catálogo de serviços, **peças e estoque**, OS, histórico, relatórios e o **orquestrador da saga** | Estoque é consultado e reservado na abertura da OS; separá-lo criaria um quarto serviço e uma etapa remota a mais na saga. Fica registrado como evolução na RFC-005. |
| **Nome do repo do OS Service** | **Manter `oficina-app`** 🔁 | O rename da Fase 3 tocou 8 pontos e deixou a chave do Sonar pendente. O README do repo declara "OS Service" logo no título. |
| **Banco do OS Service** | PostgreSQL — o RDS que já existe (`oficina-infra-db`) | Zero migração de dados; o modelo é relacional e transacional. |
| **Banco do Billing** | PostgreSQL — **instância RDS própria**, provisionada pelo Terraform do próprio repo 🔁 | Pagamento exige ACID, unicidade (idempotência do webhook) e auditoria. Instância própria torna o "banco próprio" inquestionável na avaliação; `db.t4g.micro` custa ~US$ 0,40/dia. |
| **Banco do Execution** | **Amazon DynamoDB** (on-demand) — o requisito de NoSQL | A fila de execução é acesso por chave (`os_id`, `tarefa_id`) e por fila/estado — encaixe natural de chave-valor com GSI. Sem servidor, custo ≈ US$ 0, Terraform nativo, IAM em vez de senha. DocumentDB custaria ~US$ 200/mês; MongoDB in-cluster disputaria memória com o `nri-bundle` em nós `t3.small`. |
| **Mensageria** | **Amazon SQS FIFO** — uma fila de comandos por serviço e uma fila de respostas do orquestrador, cada uma com DLQ | Gerenciado, ~US$ 0 no volume da demo, IAM por Pod Identity. FIFO com `MessageGroupId = saga_id` garante **ordem por OS**, eliminando uma classe inteira de bugs de corrida. Kafka (MSK) custa ~US$ 5/dia parado; RabbitMQ no cluster é mais um stateful para operar. |
| **Estilo da saga** | **Orquestrada**, com o orquestrador **dentro do OS Service** | O fluxo é linear, tem uma espera humana (aprovação/pagamento) e compensações em cadeia. Com orquestração, o estado da saga está em **uma tabela**, a compensação é decidida em **um lugar** e a demonstração no vídeo é uma query. Coreografia espalharia "quem desfaz o quê" pelos três serviços. Justificativa completa na RFC-006. |
| **Atomicidade estado × mensagem** | **Transactional Outbox** em todos os serviços (tabela `outbox` no Postgres; item de outbox no mesmo `TransactWriteItems` no DynamoDB) + relay | Sem outbox, "gravei e não publiquei" ou "publiquei e não gravei" acontece no primeiro deploy durante uma saga. É o defeito que já existe hoje em `AprovarOrcamento` (ver achado 1). |
| **Entrega** | *At-least-once* + **consumidor idempotente** (inbox por `message_id`) | A deduplicação do FIFO vale por 5 minutos; reentrega após isso (visibility timeout, redrive) tem de ser inofensiva. |
| **Pagamento** | **Mercado Pago Checkout Pro via API de Orders** (`POST /v1/orders` → `checkout_url`), com credenciais de teste e usuários de teste. *Validado em 10/11: a API de Preferences cobra mas não estorna com conta de teste* | O cliente abre o link e paga com cartão de teste — o nome do titular decide o resultado (`APRO` aprova, `OTHE` recusa), o que **demonstra a falha sem truque**. Pix exigiria QR e app; Checkout Transparente exigiria front-end. |
| **Confirmação do pagamento** | `GET /v1/orders/{id}` antes de agir, disparado por **webhook** (validação de `x-signature`) **ou** por um **reconciliador** a cada 10 s | Nunca confiar no corpo da notificação. O webhook só diz "algo mudou"; a verdade vem da API. O reconciliador garante a confirmação mesmo sem entrega do webhook — no sandbox, orders de teste não notificam (validado em 10/11) |
| **Compensação de pagamento** | **Estorno** (`POST /v1/orders/{id}/refund`) com `X-Idempotency-Key` — idempotência e `409 order_already_refunded` validados em 10/11 | É a compensação mais "real" da saga e a melhor cena de falha do vídeo. |
| **Comunicação síncrona** | REST apenas onde a resposta é necessária na hora: **Lambda auth → OS Service** (cliente por CPF) e consultas de leitura entre serviços | Tudo o que muda estado entre serviços vai por mensagem. |
| **Autenticação entre serviços** | O mesmo JWT HS256 (segredo `oficina/prod/app`) validado por cada serviço; rotas internas (`/internal/*`) protegidas por header compartilhado e **fora** do API Gateway | Reaproveita o contrato da Fase 3 sem introduzir mTLS/service mesh. |
| **Exposição** | Um `Service type: LoadBalancer` (CLB) por serviço; o API Gateway roteia por prefixo (`/v1/*`, `/billing/*`, `/execution/*`) | A conta **não cria ELBv2** (achado da Fase 3); um CLB por serviço preserva a independência de deploy. Webhook do Mercado Pago é rota pública sem authorizer. |
| **Infra por serviço** | Cada repo de serviço tem `infra/` com o Terraform **dos próprios recursos** (filas, tabela/instância, role do Pod Identity). VPC, EKS e API Gateway continuam no `oficina-infra-k8s` | "Cada microsserviço com sua própria infraestrutura" — e o contrato via SSM da Fase 3 continua valendo. |
| **Acesso a AWS dos pods** | **EKS Pod Identity** (addon) — uma role por serviço | Mais simples que IRSA (sem anotação de ServiceAccount nem OIDC do cluster por role) e permite políticas mínimas por fila/tabela. |
| **BDD** | **godog** (Cucumber oficial para Go), no OS Service, contra o stack completo em `docker compose` com LocalStack e um stub do Mercado Pago | Mesmo ecossistema dos testes atuais; o `.feature` vira documentação viva do fluxo da saga. |
| **Ambiente local** | `docker compose` multi-serviço: 2 Postgres, **LocalStack** (SQS + DynamoDB), Mailpit, stub do Mercado Pago | Toda a saga roda sem nuvem e sem custo — inclusive no CI. |
| **Homologação** | Mantém a dispensa do fórum (corte 10 da Fase 3): só `prod` é implantado; `homolog` tem CI | Mesma evidência, agora citada nos 3 READMEs de serviço. |

---

## Mapa de entregáveis (enunciado → artefato)

| Requisito do enunciado | Artefato | Repositório | Tarefa |
|---|---|---|---|
| ≥ 3 microsserviços independentes | OS, Billing, Execution | 3 repos | E1, E2, E3 |
| Cada um com repositório, infraestrutura e banco próprios | `infra/` por repo + RDS (OS), RDS (Billing), DynamoDB (Execution) | 3 repos | F4-0.2, F4-6.2 |
| OS Service: abertura, atualização e consulta de status/histórico | `POST/GET /v1/work-orders`, histórico com etapas da saga | `oficina-app` | F4-1.1, F4-1.7 |
| Billing: gerar e enviar orçamento para aprovação | Comando `GerarOrcamento` + e-mail com link | `oficina-billing` | F4-2.2 |
| Billing: registrar e verificar pagamentos — **Mercado Pago** | Preference + webhook `x-signature` + `GET /payments/{id}` | `oficina-billing` | F4-2.3, F4-2.4 |
| Billing: atualizar status da OS após pagamento | Resposta `PagamentoConfirmado` → orquestrador | billing → app | F4-2.6 |
| Execution: fila de execução | Tabela DynamoDB + GSI por fila/estado | `oficina-execution` | F4-3.2 |
| Execution: status durante diagnóstico e reparo | API do mecânico (`iniciar`, `concluir-diagnostico`, `concluir`) | `oficina-execution` | F4-3.3 |
| Execution: comunicar finalização ao OS Service | Resposta `ExecucaoConcluida` | execution → app | F4-3.4 |
| ≥ 1 banco SQL e ≥ 1 NoSQL | PostgreSQL × 2 + DynamoDB | — | F4-0.1 |
| REST síncrono quando necessário | Lambda → `GET /internal/clientes/por-cpf/{cpf}` | app + lambda | F4-1.6 |
| Mensageria assíncrona | SQS FIFO + DLQ | 3 repos | F4-4.1 |
| Nenhum serviço acessa o banco de outro | Roles/usuários de banco por serviço; Lambda deixa de ler `clientes` | todos | F4-1.6, F4-6.2 |
| Saga Pattern com rollback/compensação | Orquestrador + compensações em ordem inversa | `oficina-app` | E4 |
| Documentar e justificar a escolha no README | Seção "Saga" no README + RFC-006 | `oficina-app` | F4-8.1, F4-8.2 |
| Testes unitários em todos os serviços | `go test` nos 3 | 3 repos | F4-5.1 |
| ≥ 1 fluxo completo em BDD | `features/saga_os.feature` (godog) | `oficina-app` | F4-5.2 |
| Cobertura ≥ 80% por serviço | Gate no CI + Sonar | 3 repos | F4-5.1, F4-5.4 |
| SonarQube ou similar no CI | SonarCloud com quality gate | 3 repos | F4-5.4 |
| Pipeline independente: build, testes, qualidade, deploy K8s | `ci.yml` + `security.yml` + `cd.yml` por repo | 3 repos | F4-6.1 |
| `main` protegida, PR obrigatório, checks automáticos | Rulesets | 3 repos (+ 3 de infra/lambda) | F4-0.2 |
| Dockerfile e manifestos Kubernetes | `Dockerfile` + `k8s/base` + `k8s/overlays/prod` | 3 repos | F4-6.3 |
| Monitoramento da Fase 3 | New Relic: APM por serviço, tracing via SQS, `SagaEvent`, dashboard e alertas | 3 repos | E7 |
| Evidência de cobertura no README | Badge do Sonar + print do relatório | 3 repos | F4-8.1 |
| Documentação da arquitetura do serviço | `docs/arquitetura.md` por repo | 3 repos | F4-8.1 |
| Swagger ou Postman atualizado | `docs/openapi.yaml` por serviço + coleção Postman única do fluxo | 3 repos | F4-8.3 |
| Vídeo ≤ 15 min (fluxo, saga + falha, deploy com testes, monitoramento) | Roteiro em [entrega.md](entrega.md) | — | F4-9.1 |
| PDF: participantes, links, vídeo, diagrama, estratégia da saga, justificativas | Modelo em [entrega.md](entrega.md#conteúdo-do-pdf) | — | F4-9.2 |

---

## Épicos

| ID | Épico | Objetivo | Prioridade |
|----|-------|----------|------------|
| **E0** | Fundação e contratos | Decisões fechadas, repos criados e protegidos, conta Mercado Pago, contrato de mensagens, ambiente local multi-serviço | Bloqueante |
| **E1** | OS Service (refatoração do `oficina-app`) | Remover o que migra, status `cancelada`, reserva de estoque, endpoint interno para a Lambda | Alta |
| **E2** | Billing Service | Orçamento, Mercado Pago, webhook, estorno, comandos/respostas | Alta |
| **E3** | Execution Service | Fila de execução em DynamoDB, API do mecânico, comandos/respostas, cancelamento | Alta |
| **E4** | Saga e mensageria | Orquestrador, outbox/inbox, filas, timeouts, compensações, injeção de falha | **Crítica** |
| **E5** | Testes e qualidade | ≥ 80% por serviço, BDD do fluxo, testes de contrato, Sonar | Alta |
| **E6** | CI/CD e infraestrutura | Pipelines independentes, Terraform por serviço, capacidade do cluster, rotas no Gateway | Alta |
| **E7** | Observabilidade distribuída | Tracing atravessando SQS, `saga_id` nos logs, dashboard da saga, alertas | Média |
| **E8** | Documentação | READMEs, RFC-005/006, ADRs 009–016, diagramas, OpenAPI e Postman | Alta |
| **E9** | Entrega | Vídeo, PDF, teardown | Alta |

Detalhamento completo em [backlog.md](backlog.md).

---

## Achados da revisão do estado atual

Levantamento feito sobre o código e a infraestrutura ao fim da Fase 3 (`main` em `e6904c7`). Cada item vira tarefa concreta:

1. **`AprovarOrcamento` já tem o bug que a saga existe para resolver.** Em `internal/application/usecase/ordem_servico_usecase.go:433`, o estoque é deduzido (`DeduzirEstoquePecas`, uma transação) e **depois** o status é atualizado (`AtualizarStatus`, outra transação). Se a segunda falhar, as peças saíram do estoque e a OS continua `aguardando_aprovacao` — sem compensação. O mesmo padrão está em `AvancarStatus` (linha 359), e o erro de `RegistrarHistorico` é descartado (`_ =`). Na Fase 4 isso vira **reserva → baixa** dentro da mesma transação local do passo da saga. → **F4-1.2**

2. **A Lambda `auth-token` lê a tabela `clientes` direto do RDS.** O enunciado é explícito (e grifado): *nenhum serviço pode acessar diretamente o banco de outro serviço*. A Lambda está em outro repositório, com outra role — é outro serviço. Passa a chamar `GET /internal/clientes/por-cpf/{cpf}` no OS Service. → **F4-1.6**

3. **Não há estado terminal de falha na OS.** A máquina de estados só conhece `finalizada` (usada tanto para "concluída" quanto para "rejeitada"). Uma saga compensada precisa de um destino próprio, senão o dashboard conta compensações como conclusões. → **F4-1.2** (status `cancelada`)

4. **Não existe reserva de estoque — só baixa.** `pecas` tem `estoque_atual`; não há como "segurar" peças entre o orçamento e o pagamento. Sem reserva, duas OS orçam a mesma última peça e a segunda falha só na execução, depois de o cliente ter pago — a compensação mais cara possível. → **F4-1.2**

5. **Aprovação e webhook de orçamento moram no serviço errado.** `POST /v1/work-orders/{id}/approve|reject` e `POST /v1/webhooks/budget-response` (HMAC) são responsabilidade do Billing pelo enunciado. Migram, com a mesma assinatura HMAC, e o contrato OpenAPI do OS Service perde essas rotas. → **F4-1.1, F4-2.2**

6. **`PATCH /v1/work-orders/{id}/status` permite avançar qualquer transição manualmente.** Com a saga, quem move a OS entre `em_diagnostico`, `aguardando_aprovacao` e `em_execucao` é o orquestrador. Se o PATCH continuar aberto para essas transições, um atendente consegue "pular" o pagamento. Restringir o PATCH manual a `finalizada → entregue`. → **F4-1.1**

7. **A cobertura de 80% "por serviço" ainda não é medida no CI.** O `ci.yml` gera `coverage.out`, mas nada falha abaixo de um limite, e o total bruto do `go test` é **61,7%** (inclui `api.gen.go`, `cmd/` e repositórios testados por integração). No critério do `sonar-project.properties` (com as exclusões) é **98,3%**. O avaliador pode olhar qualquer dos dois números: o gate precisa ser explícito e o README precisa mostrar o número do critério usado. → **F4-5.1**

8. **Nós `t3.small` comportam só 11 pods cada** (limite de ENI do VPC CNI). Com `aws-node`, `kube-proxy`, Pod Identity Agent e os DaemonSets do `nri-bundle` em cada nó, sobram ~4–5 vagas por nó. Três serviços com 2 réplicas cada **não cabem** em 2 nós quando o HPA escalar. → **F4-6.4**

9. **A branch protection já está ativa nos repositórios existentes**, mas os **2 repos novos** nascem sem ela — e na Fase 4 a proteção é requisito explícito. Replicar o mesmo ruleset, com os checks novos (`Cobertura ≥ 80%`, `SonarCloud`), e convidar o `soat-architecture`. A chave do Sonar (`tech-challenge-1`) continua pendente da Fase 3. → **F4-0.2**

10. **Os eventos de negócio (`OrdemServicoEvent`) ficaram parcialmente implementados na Fase 3** (registrado no diário de 06–07/09). A observabilidade da Fase 4 parte deles para o `SagaEvent`; conferir antes que o painel da Fase 3 tem dados. → **F4-7.2**

11. **O ambiente AWS provavelmente foi destruído após a entrega da Fase 3** (o `kubeconfig` local não tem contexto). O `apply` completo leva ~25 min e o pedido de quota de vCPU para 32 estava `CASE_OPENED`. Confirmar quota e estado no dia 1. → **F4-0.3**

12. **(10/11) O plano de 7 dias não começou.** Em 10/11 — data do D6 no plano original — não existiam os 2 repositórios novos, a migration `000007` nem código de saga, e o diário estava vazio. O cronograma foi refeito para 4 dias, com os cortes 7′ e 18–25 ([plano](plano-execucao.md)), e a revisão corrigiu defeitos de desenho no webhook do Mercado Pago, no relay do DynamoDB, na migration e na máquina de estados ([arquitetura](arquitetura.md), [backlog](backlog.md)).

---

## Critérios de aceite globais (Definition of Done da fase)

A Fase 4 só é considerada entregue quando:

- [ ] Existem **3 repositórios de serviço** (`oficina-app`, `oficina-billing`, `oficina-execution`), cada um com código, `Dockerfile`, manifestos K8s, `infra/` Terraform, pipelines, `docs/arquitetura.md`, `docs/openapi.yaml` e README com evidência de cobertura.
- [ ] Cada serviço tem **banco próprio**; há ao menos **um SQL** (PostgreSQL) e **um NoSQL** (DynamoDB); nenhum serviço — **nem a Lambda** — tem credencial para o banco de outro (comprovado pelas policies/usuários).
- [ ] Uma OS criada percorre **OS → Execution (diagnóstico) → Billing (orçamento + pagamento Mercado Pago) → Execution (reparo) → OS (finalizada)** sem intervenção além das ações humanas (mecânico e cliente).
- [ ] Os três cenários de falha compensam corretamente e deixam a OS `cancelada`, o estoque liberado e, quando houve pagamento, **o estorno registrado no Mercado Pago**: orçamento recusado · pagamento recusado · falha na execução. *(Aprovação expirada fica como evolução — corte 5 do [plano de 4 dias](plano-execucao.md#os-cortes).)* Pagamento aprovado **depois** da compensação é estornado automaticamente.
- [ ] Reprocessar qualquer mensagem (reentrega do SQS) não produz efeito duplicado — testado.
- [ ] Mensagem venenosa vai para a **DLQ** após N tentativas e gera alerta.
- [ ] Cobertura de testes unitários **≥ 80% em cada serviço**, com o CI **falhando** abaixo disso.
- [ ] O fluxo completo da saga (feliz + falha no reparo com estorno) roda em **BDD** — no CI, ou localmente com o relatório anexado se o CI ficar instável (sacrifício 5).
- [ ] SonarCloud com quality gate passando nos 3 repos de serviço.
- [ ] Cada serviço tem pipeline **independente** que faz build, testes, Sonar e deploy no EKS a cada merge na `main`, com rollback automático.
- [ ] `main` protegida com PR obrigatório e checks obrigatórios nos repos — comprovado por um push rejeitado.
- [ ] No New Relic: APM dos 3 serviços, **um trace distribuído atravessando as filas**, logs com `saga_id`/`correlation_id`, dashboard da saga (4+ painéis) e alerta de compensação; alarme da DLQ no CloudWatch.
- [ ] README do `oficina-app` documenta e **justifica** a escolha da saga orquestrada.
- [ ] Vídeo ≤ 15 min publicado com os 4 itens pedidos; PDF submetido com os 6 itens pedidos.
- [ ] Custo sob controle: `destroy` executado após a gravação (ver [F4-9.3](backlog.md#f4-93--teardown-e-custo)).
