# Infraestrutura como código — Terraform

Provisionamento da infraestrutura da Fase 2, separado por ambiente:

```
infra/
├── modules/
│   └── kind-cluster/        # módulo reutilizável: cluster kind local
└── environments/
    ├── local/               # kind + metrics-server (uso diário e demo)
    └── aws/                 # EKS + RDS (OPT-IN — gera custo!)
```

## Pré-requisitos

| Ferramenta | Uso |
|------------|-----|
| [Terraform](https://developer.hashicorp.com/terraform) ≥ 1.5 | ambos os ambientes |
| Docker | ambiente local (o kind roda em container) |
| Credenciais AWS (`aws configure` ou env vars) | somente ambiente aws |

## Ambiente local (kind)

Cria o cluster `oficina` (contexto `kind-oficina`, exportado automaticamente
para o seu kubeconfig) e instala o **metrics-server** via Helm com a flag
`--kubelet-insecure-tls` — pré-requisito do HPA no kind.

```bash
cd infra/environments/local
terraform init
terraform plan
terraform apply

# deploy da aplicação no cluster recém-criado:
cd ../../..
./scripts/k8s-local-deploy.sh

# desfazer tudo:
terraform -chdir=infra/environments/local destroy
```

### Recursos criados (local)

| Recurso | Descrição |
|---------|-----------|
| `kind_cluster.this` | Cluster Kubernetes local (1 control-plane) |
| `helm_release.metrics_server` | metrics-server 3.12.2 no namespace kube-system |

### Variáveis (local)

| Variável | Default | Descrição |
|----------|---------|-----------|
| `nome_cluster` | `oficina` | Nome do cluster kind |

## Ambiente AWS (EKS + RDS) — opt-in

> ⚠️ **Gera custo real**: EKS (~US$ 0,10/h) + nós EC2 + RDS + NAT Gateway.
> Nada é aplicado por padrão. Sempre rode `terraform destroy` ao terminar.

```bash
cd infra/environments/aws
terraform init
terraform plan     # revise antes!
terraform apply

# conectar o kubectl ao EKS:
aws eks update-kubeconfig --name oficina --region us-east-1

# obter o endpoint do RDS e a senha gerada:
terraform output rds_endpoint
terraform output -raw db_password

# destruir ao terminar (importante!):
terraform destroy
```

Depois do apply, ajustar `k8s/overlays/aws/kustomization.yaml` com o
`rds_endpoint` e criar o Secret `oficina-secrets` com a senha gerada.

### Recursos criados (aws)

| Recurso | Módulo | Descrição |
|---------|--------|-----------|
| VPC + subnets | `terraform-aws-modules/vpc` | 2 AZs, subnets públicas/privadas, NAT único |
| Cluster EKS | `terraform-aws-modules/eks` | K8s gerenciado + node group (t3.medium, 2–3 nós) |
| RDS PostgreSQL 15 | `terraform-aws-modules/rds` | db.t3.micro, privado, acesso só dos nós do EKS |
| Security Group | `terraform-aws-modules/security-group` | 5432 liberado apenas para o SG dos nós |
| Senha do banco | `random_password` | 32 caracteres, exposta como output sensível |

### Variáveis principais (aws)

| Variável | Default | Descrição |
|----------|---------|-----------|
| `regiao` | `us-east-1` | Região AWS |
| `eks_versao` | `1.31` | Versão do Kubernetes |
| `eks_instance_types` | `["t3.medium"]` | Instâncias do node group |
| `eks_min_nodes` / `eks_max_nodes` | `2` / `3` | Tamanho do node group |
| `db_instance_class` | `db.t3.micro` | Classe do RDS |

## Decisões

- **Registry**: Docker Hub (conta do time) — por isso **não há ECR** no módulo AWS.
- **State**: local em ambos os ambientes por enquanto; a migração para S3 +
  DynamoDB está esboçada (comentada) nos `versions.tf`.
- **Deploy da aplicação** não é papel do Terraform aqui: fica com o
  `scripts/k8s-local-deploy.sh` (local) e com a pipeline de CD (E6).
