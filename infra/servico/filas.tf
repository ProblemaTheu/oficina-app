# Fila de ENTRADA do OS Service (F4-4.1): as respostas que o orquestrador da
# saga consome. Cada serviço é dono da fila que consome; Billing e Execution
# recebem permissão de envio nas próprias roles (identity policy).
# Mesmos atributos do scripts/localstack-init.sh.

resource "aws_sqs_queue" "respostas_dlq" {
  name                      = "os-saga-replies-dlq.fifo"
  fifo_queue                = true
  message_retention_seconds = 1209600 # 14 dias para investigar
  sqs_managed_sse_enabled   = true
}

resource "aws_sqs_queue" "respostas" {
  name                       = "os-saga-replies.fifo"
  fifo_queue                 = true
  deduplication_scope        = "messageGroup"
  fifo_throughput_limit      = "perMessageGroupId"
  visibility_timeout_seconds = 60 # > tempo máximo de processamento
  receive_wait_time_seconds  = 20 # long polling
  message_retention_seconds  = 345600
  sqs_managed_sse_enabled    = true

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.respostas_dlq.arn
    maxReceiveCount     = 5
  })
}

# Só esta fila pode usar a DLQ como destino.
resource "aws_sqs_queue_redrive_allow_policy" "respostas_dlq" {
  queue_url = aws_sqs_queue.respostas_dlq.id
  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue"
    sourceQueueArns   = [aws_sqs_queue.respostas.arn]
  })
}
