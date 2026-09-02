# Arquitetura-alvo — Fase 3

Documento de referência para a execução do [backlog](backlog.md). Os diagramas aqui são o **destino**; conforme os épicos forem concluídos eles migram para os READMEs dos respectivos repositórios (requisito do enunciado: "diagrama da arquitetura específica daquele repositório").

> ⏱️ A seção 1 mostra o **desenho completo**; a [seção 1.1](#11-variante-do-plano-de-10-dias-o-que-realmente-vai-ao-ar) mostra a **variante construída** — e é ela que vai para o README e para o PDF. Um diagrama com VPC Link onde existe um NLB público não é detalhe estético: é a primeira coisa que o avaliador confere contra o Terraform.

---

## 1. Diagrama de Componentes

Visão de nuvem, APIs, banco e monitoramento — como pedido no enunciado.

```mermaid
flowchart TB
    subgraph internet["Internet"]
        cliente["👤 Cliente final<br/>(CPF)"]
        func["👨‍🔧 Funcionário<br/>(e-mail + senha)"]
        ext["🔌 Sistema externo<br/>(webhook de orçamento)"]
    end

    subgraph aws["AWS — região us-east-1"]
        subgraph edge["Borda"]
            apigw["**API Gateway HTTP API**<br/>uma API por ambiente<br/>(homolog · prod)"]
            authz["**λ auth-authorizer**<br/>valida JWT HS256<br/>(REQUEST authorizer, cache 5 min)"]
            token["**λ auth-token**<br/>valida CPF · consulta cliente<br/>· emite JWT"]
        end

        subgraph vpc["VPC 10.0.0.0/16"]
            vpclink["VPC Link"]
            alb["ALB interno<br/>(criado pelo Terraform)"]

            subgraph eks["EKS — cluster oficina"]
                subgraph nsprod["ns: oficina-prod"]
                    tgb["TargetGroupBinding"]
                    svc["Service api"]
                    pods["Deployment api<br/>2–5 pods (HPA CPU 50%)"]
                end
                nshom["ns: oficina-homolog<br/>(mesma topologia)"]
                nrk8s["New Relic<br/>nri-bundle (DaemonSet)"]
                metrics["metrics-server"]
            end

            rds[("**RDS PostgreSQL 15**<br/>db oficina_prod / oficina_homolog<br/>subnets privadas")]
        end

        sm["Secrets Manager<br/>JWT_SECRET · DB creds · NR license"]
        ssm["SSM Parameter Store<br/>outputs entre repos de infra"]
        cw["CloudWatch Logs<br/>(Lambda)"]
    end

    nr["📊 **New Relic**<br/>APM · Infra · Logs · Dashboards · Alertas · Synthetics"]

    cliente -- "POST /auth/token {cpf}" --> apigw
    func -- "POST /v1/auth/login" --> apigw
    ext -- "POST /v1/webhooks/... (HMAC)" --> apigw
    apigw -- "rota /auth/token" --> token
    apigw -- "demais rotas: autoriza" --> authz
    apigw -- "encaminha" --> vpclink --> alb
    tgb -. registra pods no target group .-> alb
    alb --> svc --> pods
    pods --> rds
    token --> rds
    token -.lê segredo.-> sm
    authz -.lê segredo.-> sm
    pods -.lê segredo (External Secrets).-> sm
    pods -- "APM + logs JSON" --> nr
    token & authz -- "layer New Relic" --> nr
    nrk8s -- "CPU/mem/eventos" --> nr
    metrics -. HPA .-> pods
    token & authz --> cw
    apigw -.access logs.-> cw
```

### 1.1 Variante do plano de 10 dias (o que realmente vai ao ar)

Cinco simplificações em relação ao diagrama acima, cada uma registrada como ADR: sem NAT Gateway, sem VPC Link, sem ALB interno/`TargetGroupBinding`, sem Cluster Autoscaler e sem External Secrets Operator.

```mermaid
flowchart TB
    subgraph internet["Internet"]
        cliente["👤 Cliente (CPF)"]
        func["👨‍🔧 Funcionário"]
    end

    subgraph aws["AWS — us-east-1"]
        apigw["**API Gateway HTTP API**<br/>uma API por ambiente"]
        token["**λ auth-token**"]
        authz["**λ auth-authorizer**"]

        subgraph vpc["VPC — subnets públicas"]
            nlb["NLB<br/>(Service type: LoadBalancer)<br/>exige header compartilhado"]
            subgraph eks["EKS 1.36 — 2 × t3.small"]
                pods["Deployment api<br/>HPA 2–6 pods (CPU 50%)"]
                nr8s["New Relic nri-bundle"]
                ms["metrics-server"]
            end
            rds[("RDS PostgreSQL<br/>subnet privada")]
        end

        sm["Secrets Manager"]
        tfk8s["Secret do K8s<br/>criado pelo Terraform"]
    end

    nr["📊 New Relic"]

    cliente -- "POST /auth/token" --> apigw
    func -- "POST /v1/auth/login" --> apigw
    apigw --> token
    apigw -- "autoriza /v1/*" --> authz
    apigw -- "HTTP_PROXY + header" --> nlb --> pods
    pods --> rds
    token --> rds
    sm -.lido no apply.-> tfk8s -.envFrom.-> pods
    token & authz -.GetSecretValue.-> sm
    pods -- "APM + logs JSON" --> nr
    nr8s -- "CPU/memória" --> nr
    ms -. HPA .-> pods
```

**O que cada simplificação custa, para a ADR:**

| Simplificação | Risco assumido | Evolução |
|---|---|---|
| NLB público + header no lugar de VPC Link | o NLB é alcançável pela internet; a proteção é o header, não a rede | ALB interno + VPC Link |
| Nós em subnet pública, sem NAT | IP público nos nós; SG é a única barreira | subnets privadas + NAT |
| HPA sem Cluster Autoscaler | pods ficam `Pending` se os 2 nós lotarem | Cluster Autoscaler ou Karpenter |
| `Secret` do Terraform no lugar do ESO | rotação do segredo exige novo `apply` | External Secrets Operator |
| E-mail demonstrado local | SES não exercitado em produção | SES fora do sandbox |

---

### Responsabilidade de cada componente

| Componente | Responsabilidade | Onde é definido |
|---|---|---|
| API Gateway HTTP API | Única porta de entrada; roteamento, autorização na borda, rate limiting, access logs | repo `infra-k8s` |
| λ `auth-token` | Valida CPF (dígitos verificadores), consulta cliente no RDS, emite JWT HS256 | repo `lambda-auth` |
| λ `auth-authorizer` | Valida assinatura e expiração do JWT antes de qualquer chamada ao cluster | repo `lambda-auth` |
| VPC Link + ALB interno | Ponte privada entre o Gateway (fora da VPC) e o cluster (subnets privadas) | repo `infra-k8s` |
| EKS + HPA | Executa a aplicação com escala horizontal por CPU | repo `infra-k8s` (cluster) + repo `app` (manifestos) |
| RDS PostgreSQL | Persistência gerenciada, backup automático, subnets privadas | repo `infra-db` |
| Secrets Manager | `JWT_SECRET` (compartilhado app ↔ lambdas), credenciais do banco, license key do New Relic | `infra-db` (banco) e `infra-k8s` (demais) |
| SSM Parameter Store | Contrato entre os repos de infra (`/oficina/<env>/<chave>`) | todos os repos de infra |
| New Relic | APM, infraestrutura, logs, dashboards, alertas, synthetics | repo `app` (agente) + `infra-k8s` (Helm) |

---

## 2. Diagramas de Sequência

### 2.1 Autenticação do cliente por CPF

```mermaid
sequenceDiagram
    autonumber
    actor C as Cliente
    participant GW as API Gateway
    participant LT as λ auth-token
    participant SM as Secrets Manager
    participant DB as RDS PostgreSQL
    participant AZ as λ auth-authorizer
    participant API as Aplicação (EKS)

    Note over C,LT: Etapa 1 — obtenção do token
    C->>GW: POST /auth/token { "cpf": "529.982.247-25" }
    GW->>LT: invoke (rota pública, sem authorizer)
    LT->>LT: normaliza (só dígitos) e valida<br/>dígitos verificadores do CPF
    alt CPF sintaticamente inválido
        LT-->>C: 400 { code: "cpf_invalido" }
    end
    LT->>DB: SELECT id, nome, status FROM clientes<br/>WHERE cpf_digits = $1
    alt cliente não existe
        LT-->>C: 404 { code: "cliente_nao_encontrado" }
    else cliente com status <> 'ativo'
        LT-->>C: 403 { code: "cliente_inativo" }
    end
    LT->>SM: GetSecretValue(oficina/jwt-secret)
    Note right of LT: cacheado no /tmp da execução<br/>(reuso entre invocações quentes)
    LT->>LT: assina JWT HS256<br/>{sub, cpf, tipo:"cliente", iss, aud, exp:1h}
    LT-->>C: 200 { access_token, token_type, expires_in }

    Note over C,API: Etapa 2 — consumo de rota protegida
    C->>GW: GET /v1/work-orders/{id}<br/>Authorization: Bearer <token>
    GW->>AZ: invoke authorizer (cache 300 s por token)
    AZ->>SM: GetSecretValue (cache em memória)
    AZ->>AZ: verifica assinatura, exp, iss e aud
    alt token inválido ou expirado
        AZ-->>GW: { isAuthorized: false }
        GW-->>C: 401 (a requisição NÃO chega ao cluster)
    end
    AZ-->>GW: { isAuthorized: true, context: { cpf, tipo, sub } }
    GW->>API: encaminha via VPC Link → ALB → pod<br/>(headers de contexto propagados)
    API->>API: middleware JWT revalida o Bearer<br/>(defesa em profundidade)
    API->>DB: consulta a OS
    API-->>C: 200 { ordem de serviço }
```

### 2.2 Abertura de ordem de serviço

```mermaid
sequenceDiagram
    autonumber
    actor F as Atendente
    participant GW as API Gateway
    participant AZ as λ authorizer
    participant API as Aplicação (EKS)
    participant UC as CriarOSUseCase
    participant DB as RDS
    participant NR as New Relic
    participant MAIL as SMTP

    F->>GW: POST /v1/work-orders<br/>Bearer <jwt tipo=usuario><br/>X-Correlation-Id: <uuid>
    GW->>AZ: autoriza
    AZ-->>GW: isAuthorized: true
    GW->>API: encaminha (VPC Link → ALB → pod)
    activate API
    API->>API: middleware correlação:<br/>lê X-Correlation-Id, injeta no ctx e no logger
    API->>NR: inicia transação APM (trace.id, span.id)
    API->>UC: CriarOS(ctx, input)
    activate UC
    UC->>DB: BEGIN
    UC->>DB: valida cliente e veículo
    UC->>DB: GerarNumeroOS() → OS-2026-00042
    UC->>DB: INSERT ordens_servico (status = recebida)
    UC->>DB: INSERT itens_os_servicos / itens_os_pecas
    UC->>DB: INSERT historicos_status
    UC->>DB: COMMIT
    alt falha em qualquer passo
        UC->>DB: ROLLBACK
        UC->>NR: NoticeError + custom event<br/>OrdemServicoEvent{resultado:"falha"}
        Note right of NR: dispara o alerta<br/>"falha no processamento de OS"
        UC-->>F: 4xx/5xx (erro mapeado)
    end
    UC->>NR: RecordCustomEvent OrdemServicoEvent<br/>{os_id, status:"recebida", resultado:"sucesso"}
    deactivate UC
    API-)MAIL: notificação assíncrona (goroutine)
    API->>NR: log JSON { level, msg, correlation_id, trace.id, span.id }
    API-->>F: 201 Created { id, numero, status }
    deactivate API
```

---

## 3. Modelo Entidade-Relacionamento alvo

Em **negrito** as mudanças da Fase 3 (detalhadas em [F3-5.1](backlog.md#f3-51--revisão-do-modelo-relacional)).

```mermaid
erDiagram
    PAPEIS_USUARIO ||--o{ USUARIOS : "define papel de"
    CLIENTES       ||--o{ VEICULOS : possui
    CLIENTES       ||--o{ ORDENS_SERVICO : "é titular de"
    VEICULOS       ||--o{ ORDENS_SERVICO : "é objeto de"
    USUARIOS       ||--o{ ORDENS_SERVICO : "é responsável por"
    STATUS_ORDENS  ||--o{ ORDENS_SERVICO : classifica
    ORDENS_SERVICO ||--o{ ITENS_OS_SERVICOS : contém
    ORDENS_SERVICO ||--o{ ITENS_OS_PECAS : contém
    ORDENS_SERVICO ||--o{ HISTORICOS_STATUS : registra
    SERVICOS       ||--o{ ITENS_OS_SERVICOS : "é referenciado em"
    PECAS          ||--o{ ITENS_OS_PECAS : "é referenciada em"
    STATUS_ORDENS  ||--o{ HISTORICOS_STATUS : "origem/destino"

    CLIENTES {
        uuid id PK
        varchar nome
        varchar cpf_cnpj UK "com máscara, como digitado"
        varchar cpf_cnpj_digitos UK "NOVO — só dígitos, usado pela Lambda"
        varchar status "NOVO — ativo|inativo|bloqueado"
        varchar email UK
        varchar telefone
        timestamptz criado_em "ERA timestamp"
        timestamptz atualizado_em "ERA timestamp"
    }
    ORDENS_SERVICO {
        uuid id PK
        varchar numero UK "OS-YYYY-NNNNN"
        uuid cliente_id FK "NOVO índice"
        uuid veiculo_id FK "NOVO índice"
        uuid usuario_responsavel_id FK "NOVO índice"
        uuid status_id FK "NOVO índice composto (status_id, criado_em)"
        decimal valor_total
        text descricao
        text diagnostico
        timestamptz aprovado_em
        timestamptz reprovado_em
        timestamptz iniciado_em
        timestamptz finalizado_em
        timestamptz entregue_em
        timestamptz criado_em
    }
    VEICULOS {
        uuid id PK
        uuid cliente_id FK "NOVO índice"
        varchar placa UK
        varchar marca
        varchar modelo
        int ano
        varchar cor
    }
    HISTORICOS_STATUS {
        uuid id PK
        uuid os_id FK "NOVO índice (os_id, alterado_em)"
        uuid status_anterior_id FK
        uuid status_novo_id FK
        timestamptz alterado_em
        uuid alterado_por_usuario_id FK
        text observacao
    }
    ITENS_OS_PECAS {
        uuid id PK
        uuid os_id FK "NOVO índice"
        uuid peca_id FK "NOVO índice"
        int quantidade
        decimal preco_unitario "preço congelado no momento da OS"
    }
    ITENS_OS_SERVICOS {
        uuid id PK
        uuid os_id FK "NOVO índice"
        uuid servico_id FK "NOVO índice"
        decimal preco "preço congelado no momento da OS"
    }
    PECAS {
        uuid id PK
        varchar nome
        varchar codigo UK
        decimal preco
        int estoque_atual
        int estoque_minimo
    }
    SERVICOS {
        uuid id PK
        varchar nome UK
        text descricao
        decimal preco_base
        int tempo_minutos
    }
    USUARIOS {
        uuid id PK
        varchar nome
        varchar email UK
        varchar senha_hash
        uuid papel_id FK "NOVO índice"
    }
    PAPEIS_USUARIO {
        uuid id PK
        varchar nome_papel UK
    }
    STATUS_ORDENS {
        uuid id PK
        varchar nome_status UK
    }
```

### Explicação dos relacionamentos (para a RFC-002)

| Relacionamento | Cardinalidade | Por quê |
|---|---|---|
| `clientes` → `veiculos` | 1:N | Um cliente pode ter vários veículos; um veículo pertence a um único cliente (a placa é única no sistema). |
| `clientes` → `ordens_servico` | 1:N | O titular da OS é sempre o cliente. Redundante com `veiculos.cliente_id`? Não: o veículo pode ser transferido, e a OS precisa preservar **quem era o titular na abertura**. |
| `veiculos` → `ordens_servico` | 1:N | Uma OS é sempre sobre um veículo; o veículo acumula histórico de OS. |
| `usuarios` → `ordens_servico` | 1:N (opcional) | `usuario_responsavel_id` é nullable: a OS pode ser aberta antes de haver mecânico designado. |
| `status_ordens` → `ordens_servico` | 1:N | Status como **tabela de domínio** e não `ENUM`: permite adicionar status sem `ALTER TYPE`, e dá FK real ao histórico. |
| `ordens_servico` → `itens_os_servicos` / `itens_os_pecas` | 1:N | Tabelas associativas com atributo próprio (`quantidade`, `preco`) — por isso não são N:N puras. **O preço é copiado, não referenciado**: alterar o preço de uma peça no catálogo não pode alterar retroativamente o valor de OS já fechadas. |
| `ordens_servico` → `historicos_status` | 1:N | Trilha de auditoria imutável de cada transição — é a fonte do cálculo de "tempo médio por status" no dashboard. |
| `papeis_usuario` → `usuarios` | 1:N | Autorização por papel (administrador, mecânico, atendente). |

---

## 4. Topologia dos 4 repositórios e fluxo de dependências

```mermaid
flowchart LR
    subgraph repos["Repositórios GitHub"]
        direction TB
        R2["**2. infra-k8s**<br/>VPC · EKS · API Gateway<br/>VPC Link · ALB · Secrets<br/>New Relic (Helm)"]
        R3["**3. infra-db**<br/>RDS PostgreSQL<br/>Subnet group · SG<br/>credenciais no Secrets Manager"]
        R1["**1. lambda-auth**<br/>λ auth-token<br/>λ auth-authorizer<br/>rotas no Gateway"]
        R4["**4. app** (repo atual)<br/>API Go · manifestos k8s<br/>OpenAPI · Postman"]
    end

    ssm[("SSM Parameter Store<br/>/oficina/{env}/*")]

    R2 -- "publica: vpc_id, subnet_ids,<br/>apigw_id, sg_lambda, tg_arn" --> ssm
    R3 -- "publica: db_endpoint,<br/>db_secret_arn" --> ssm
    ssm -- "lê rede + apigw_id" --> R1
    ssm -- "lê vpc + subnets" --> R3
    ssm -- "lê db_endpoint,<br/>tg_arn, apigw_url" --> R4

    R2 ==>|1º| ordem
    R3 ==>|2º| ordem
    R1 ==>|3º| ordem
    R4 ==>|4º| ordem
    ordem["Ordem de<br/>provisionamento"]
```

> **Por que o `infra-k8s` vem primeiro e não o `infra-db`?** Porque a VPC vive nele. O RDS precisa das subnets privadas e do security group dos nós do EKS para restringir o acesso à porta 5432. Alternativa considerada e descartada: um quinto repo só de rede — o enunciado fixa quatro.

### Estratégia de branches (idêntica nos 4 repos)

```mermaid
gitGraph
    commit id: "main (produção)"
    branch homolog
    commit id: "homolog (staging)"
    branch feature/F3-2.1
    commit id: "trabalho"
    commit id: "trabalho"
    checkout homolog
    merge feature/F3-2.1 tag: "PR + CI verde → deploy homolog"
    checkout main
    merge homolog tag: "PR + CI verde → deploy prod"
```

| Branch | Protegida | Deploy automático | Ambiente |
|---|---|---|---|
| `feature/*` | não | não (só CI) | — |
| `homolog` | sim (PR obrigatório) | sim | namespace `oficina-homolog`, API Gateway `oficina-homolog` |
| `main` | sim (PR obrigatório, sem push direto) | sim, com *environment approval* | namespace `oficina-prod`, API Gateway `oficina-prod` |

> ⏱️ Com o ambiente de homologação dispensado, a tabela vira:
>
> | Branch | Protegida | Deploy automático | Ambiente |
> |---|---|---|---|
> | `feature/*` | não | não (só CI) | — |
> | `homolog` | sim (PR obrigatório) | **não** — apenas CI (build, testes, lint) | — |
> | `main` | sim (PR obrigatório, sem push direto) | sim, com *environment approval* | namespace `oficina-prod`, API Gateway `oficina-prod` |
>
> O fluxo `feature → homolog → main` com PR e CI bloqueante continua sendo demonstrado; o que deixa de existir é a infraestrutura de homologação, não o processo.
