# Alarme da DLQ (F4-7.4): mensagem que falhou 5× é falha de saga que ninguém
# vai ver no APM. Métrica da AWS, então fica no CloudWatch (+ SNS).

# Sem criptografia de propósito: o CloudWatch NÃO publica em tópico cifrado
# com a chave gerenciada aws/sns (a key policy dela não pode liberar
# cloudwatch.amazonaws.com) — o alarme falharia em silêncio. Uma CMK própria
# custaria US$ 1/mês para proteger uma mensagem que só diz "há itens na DLQ";
# o payload da mensagem continua na fila, que é cifrada.
resource "aws_sns_topic" "alertas" {
  name = "${local.nome}-alertas"
}

resource "aws_sns_topic_subscription" "email" {
  count     = var.alerta_email == "" ? 0 : 1
  topic_arn = aws_sns_topic.alertas.arn
  protocol  = "email"
  endpoint  = var.alerta_email
}

resource "aws_cloudwatch_metric_alarm" "dlq" {
  alarm_name          = "${local.nome}-dlq-com-mensagens"
  alarm_description   = "Mensagem na DLQ do OS: resposta da saga falhou 5 vezes no orquestrador (ver payload na fila ${aws_sqs_queue.respostas_dlq.name})"
  namespace           = "AWS/SQS"
  metric_name         = "ApproximateNumberOfMessagesVisible"
  dimensions          = { QueueName = aws_sqs_queue.respostas_dlq.name }
  statistic           = "Maximum"
  period              = 60
  evaluation_periods  = 1
  comparison_operator = "GreaterThanThreshold"
  threshold           = 0
  treat_missing_data  = "notBreaching"
  alarm_actions       = [aws_sns_topic.alertas.arn]
  ok_actions          = [aws_sns_topic.alertas.arn]
}
