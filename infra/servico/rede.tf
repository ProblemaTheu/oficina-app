# O cluster vem do oficina-infra-k8s pela SSM — nenhum state lê o state de
# outro repositório. O RDS do OS continua no oficina-infra-db.

data "aws_caller_identity" "atual" {}

data "aws_ssm_parameter" "cluster_name" {
  name = "/oficina/shared/eks/cluster_name"
}

locals {
  nome  = "oficina-os"
  conta = data.aws_caller_identity.atual.account_id

  # Filas de ENTRADA dos outros serviços: cada uma é do repositório que a
  # consome. ARN por convenção de nome, sem ler o state deles — assim os três
  # infra/ aplicam em qualquer ordem (F4-4.1, arquitetura §7).
  filas_comandos_arn = [
    for f in ["billing-commands", "execution-commands"] :
    "arn:aws:sqs:${var.regiao}:${local.conta}:${f}.fifo"
  ]
}
