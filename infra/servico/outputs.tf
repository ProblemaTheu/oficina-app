output "fila_respostas_url" { value = aws_sqs_queue.respostas.url }
output "fila_respostas_dlq_url" { value = aws_sqs_queue.respostas_dlq.url }
output "pod_role_arn" { value = aws_iam_role.pod.arn }
output "alertas_topic_arn" { value = aws_sns_topic.alertas.arn }
