# Identidade AWS do pod do OS Service por EKS Pod Identity (ADR-016): o addon
# eks-pod-identity-agent já existe no cluster (oficina-infra-k8s).
#
# Menor privilégio: o orquestrador ENVIA comandos para as filas de entrada do
# Billing e do Execution e CONSOME só a própria fila de respostas. Sem acesso
# a DynamoDB nem ao RDS do Billing — o RDS do OS é acessado por rede + senha.

resource "aws_iam_role" "pod" {
  name = "${local.nome}-pod"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "pods.eks.amazonaws.com" }
      Action    = ["sts:AssumeRole", "sts:TagSession"]
      Condition = {
        StringEquals = { "aws:SourceAccount" = local.conta }
      }
    }]
  })
}

resource "aws_iam_role_policy" "pod" {
  name = "saga-orquestrador"
  role = aws_iam_role.pod.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "EnviarComandos"
        Effect   = "Allow"
        Action   = ["sqs:SendMessage", "sqs:GetQueueAttributes", "sqs:GetQueueUrl"]
        Resource = local.filas_comandos_arn
      },
      {
        Sid      = "ConsumirRespostas"
        Effect   = "Allow"
        Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:ChangeMessageVisibility", "sqs:GetQueueAttributes", "sqs:GetQueueUrl"]
        Resource = aws_sqs_queue.respostas.arn
      }
    ]
  })
}

resource "aws_eks_pod_identity_association" "os" {
  cluster_name    = data.aws_ssm_parameter.cluster_name.value
  namespace       = var.namespace
  service_account = var.service_account
  role_arn        = aws_iam_role.pod.arn
}
