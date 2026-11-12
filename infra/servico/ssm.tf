# Contrato com os outros repositórios: o CD do Billing e o do Execution leem
# a URL da fila de respostas daqui para gerar o ConfigMap deles.

resource "aws_ssm_parameter" "fila_respostas_url" {
  name  = "/oficina/${var.ambiente}/os/fila-respostas-url"
  type  = "String"
  value = aws_sqs_queue.respostas.url
}
