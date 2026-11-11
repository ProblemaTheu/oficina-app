#!/bin/bash
# Recursos AWS da saga no LocalStack (F4-0.4). Roda como hook "ready" do
# container — ver docker-compose.saga.yml.
#
# Mesmos atributos que o Terraform de cada serviço vai criar (F4-4.1, F4-6.2):
# FIFO, deduplicação por grupo, visibility 60 s, redrive para a DLQ após 5
# recebimentos; tabela DynamoDB single-table com GSI e TTL.
set -euo pipefail

REGIAO="${AWS_DEFAULT_REGION:-us-east-1}"
CONTA="000000000000"

criar_fila() {
  local nome="$1"
  awslocal sqs create-queue --queue-name "${nome}-dlq.fifo" \
    --attributes '{"FifoQueue":"true","MessageRetentionPeriod":"1209600"}' >/dev/null

  local dlq_arn="arn:aws:sqs:${REGIAO}:${CONTA}:${nome}-dlq.fifo"
  local redrive
  redrive=$(printf '{"deadLetterTargetArn":"%s","maxReceiveCount":"5"}' "$dlq_arn")

  # O valor de RedrivePolicy é uma string JSON dentro do JSON de atributos.
  awslocal sqs create-queue --queue-name "${nome}.fifo" --attributes "$(cat <<EOF
{
  "FifoQueue": "true",
  "DeduplicationScope": "messageGroup",
  "FifoThroughputLimit": "perMessageGroupId",
  "VisibilityTimeout": "60",
  "ReceiveMessageWaitTimeSeconds": "20",
  "RedrivePolicy": $(printf '%s' "$redrive" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')
}
EOF
)" >/dev/null
  echo "fila ${nome}.fifo (+ DLQ) criada"
}

criar_fila os-saga-replies
criar_fila billing-commands
criar_fila execution-commands

awslocal dynamodb create-table --table-name execution-tarefas \
  --billing-mode PAY_PER_REQUEST \
  --attribute-definitions \
    AttributeName=pk,AttributeType=S AttributeName=sk,AttributeType=S \
    AttributeName=gsi1pk,AttributeType=S AttributeName=gsi1sk,AttributeType=S \
  --key-schema AttributeName=pk,KeyType=HASH AttributeName=sk,KeyType=RANGE \
  --global-secondary-indexes '[{"IndexName":"gsi1","KeySchema":[{"AttributeName":"gsi1pk","KeyType":"HASH"},{"AttributeName":"gsi1sk","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}}]' \
  >/dev/null
awslocal dynamodb update-time-to-live --table-name execution-tarefas \
  --time-to-live-specification Enabled=true,AttributeName=ttl >/dev/null
echo "tabela execution-tarefas (+ gsi1, ttl) criada"

echo "recursos da saga prontos"
