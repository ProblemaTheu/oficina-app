# Ambiente AWS — EKS + RDS (OPT-IN, preparado para o futuro).
#
# ⚠️ ATENÇÃO: aplicar este ambiente GERA CUSTO na AWS (EKS ~US$ 0,10/h +
# nós EC2 + RDS + NAT Gateway). Nada aqui é aplicado por padrão; requer
# credenciais AWS configuradas e um `terraform apply` consciente.
#
# Registry de imagens: Docker Hub (decisão do projeto) — por isso NÃO há
# ECR aqui. O deploy usa o overlay k8s/overlays/aws, ajustando DB_HOST
# para o endpoint do RDS (output `rds_endpoint`).

data "aws_availability_zones" "disponiveis" {
  state = "available"
}

locals {
  azs             = slice(data.aws_availability_zones.disponiveis.names, 0, 2)
  subnets_privadas = [for i, az in local.azs : cidrsubnet(var.vpc_cidr, 8, i)]
  subnets_publicas = [for i, az in local.azs : cidrsubnet(var.vpc_cidr, 8, i + 10)]
}

# ── Rede ──────────────────────────────────────────────────────────────────────
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.8"

  name = "${var.projeto}-vpc"
  cidr = var.vpc_cidr
  azs  = local.azs

  private_subnets = local.subnets_privadas
  public_subnets  = local.subnets_publicas

  # NAT único para economizar — suficiente para o desafio
  enable_nat_gateway = true
  single_nat_gateway = true

  enable_dns_hostnames = true

  # Tags exigidas pelo EKS para descoberta de subnets
  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }
  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }
}

# ── Cluster EKS ───────────────────────────────────────────────────────────────
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.24"

  cluster_name    = "${var.projeto}-eks"
  cluster_version = var.eks_versao

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # Acesso público à API do cluster (kubectl da máquina do dev/CI)
  cluster_endpoint_public_access           = true
  enable_cluster_creator_admin_permissions = true

  eks_managed_node_groups = {
    default = {
      instance_types = var.eks_instance_types
      min_size       = var.eks_min_nodes
      max_size       = var.eks_max_nodes
      desired_size   = var.eks_min_nodes
    }
  }
}

# ── Banco de dados (RDS PostgreSQL) ──────────────────────────────────────────
resource "random_password" "db" {
  length  = 32
  special = false
}

module "rds_sg" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "~> 5.1"

  name   = "${var.projeto}-rds"
  vpc_id = module.vpc.vpc_id

  # Postgres acessível apenas a partir dos nós do EKS
  ingress_with_source_security_group_id = [
    {
      from_port                = 5432
      to_port                  = 5432
      protocol                 = "tcp"
      source_security_group_id = module.eks.node_security_group_id
    }
  ]
}

module "rds" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 6.7"

  identifier = "${var.projeto}-db"

  engine               = "postgres"
  engine_version       = "15"
  family               = "postgres15"
  major_engine_version = "15"
  instance_class       = var.db_instance_class

  allocated_storage = 20
  db_name           = var.db_nome
  username          = var.db_usuario
  password          = random_password.db.result

  manage_master_user_password = false

  multi_az               = false
  publicly_accessible    = false
  create_db_subnet_group = true
  subnet_ids             = module.vpc.private_subnets
  vpc_security_group_ids = [module.rds_sg.security_group_id]

  # Facilita o destroy no contexto do desafio (não usar assim em produção)
  skip_final_snapshot = true
  deletion_protection = false
}
