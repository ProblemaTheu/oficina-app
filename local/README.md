# Fase 4 num Kubernetes local (kind)

Os 3 microsserviços rodam num cluster Kubernetes local, com os **mesmos manifests `k8s/base`** de produção e os **mesmos namespaces** (`oficina-prod`, `billing-prod`, `execution-prod`). Cada serviço tem o próprio banco, a saga trafega por SQS FIFO e a telemetria vai para o New Relic.

Serve para desenvolvimento, demonstração e para o deploy automatizado do CD. Os equivalentes gerenciados (EKS, RDS, SQS, DynamoDB) estão no Terraform de `infra/` de cada repositório.

| Componente | Em produção (AWS) | Aqui |
|---|---|---|
| Cluster | EKS | kind sobre Colima |
| Banco do OS / do Billing | RDS PostgreSQL ×2 | `postgres-os` / `postgres-billing` (namespace `infra`) |
| Filas FIFO + DLQ, tabela do Execution | SQS + DynamoDB | LocalStack, criado pelo mesmo `scripts/localstack-init.sh` |
| Mercado Pago | API de Orders | stub (`mp-stub`) ou sandbox real (`MP_REAL=1`) |
| Credenciais AWS dos pods | Pod Identity | chaves fictícias do LocalStack |
| Config de ambiente | CD: SSM → ConfigMap `*-ambiente` | `configMapGenerator` em `local/servicos/*` |
| Observabilidade | New Relic | New Relic (apps `oficina-*-local`) |

## Subir

```bash
cp .env.example .env                  # se ainda não existir
echo 'NEW_RELIC_LICENSE_KEY=...' >> .env   # opcional: APM, logs, traces e SagaEvent
./local/subir.sh                      # cria o cluster, constrói as imagens e sobe tudo
AMBIENTE=k8s ./scripts/demo-saga.sh todos   # os 4 cenários da saga
```

Billing e Execution são lidos das pastas irmãs (`../oficina-billing` e `../oficina-execution`). As portas no host são as do `docker-compose.saga.yml` (8080, 8081, 8082, stub do MP em 8090, Mailpit em 8025), então a coleção Postman com o ambiente `local` funciona igual.

| Variação | Comando |
|---|---|
| Mercado Pago sandbox real (o reconciliador confirma o pagamento sem webhook) | `MP_REAL=1 ./local/subir.sh` |
| Reaplicar sem reconstruir as imagens | `SEM_BUILD=1 ./local/subir.sh` |
| Apagar tudo | `kind delete cluster --name oficina` |

Pare o `docker-compose.saga.yml` antes (`docker compose -f docker-compose.saga.yml --profile saga stop`): os dois usam as mesmas portas.

## Deploy automatizado (CD)

Com `DEPLOY_ENABLED` diferente de `true` (o padrão), o `cd.yml` de **cada** serviço roda o job **Deploy no Kubernetes (kind)**, que roda depois de `Testes (gate de 80%)` e `Build e push`:

1. cria um cluster kind no próprio runner;
2. sobe só as dependências daquele serviço (`local/infra/*`);
3. aplica o overlay `local/servicos/<serviço>` com a **imagem que o pipeline acabou de publicar** (tag = SHA do commit);
4. espera o rollout e faz o smoke `/health/ready`, com diagnóstico (pods, describe, logs) se falhar.

Tudo isso é o [`local/deploy-ci.sh`](deploy-ci.sh). Para testar na máquina com uma imagem local: `CLUSTER=ci-teste PULL_POLICY=Never ./local/deploy-ci.sh execution oficina-execution:local`.
