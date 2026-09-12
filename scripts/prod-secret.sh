#!/usr/bin/env bash
# Cria o Secret oficina-secrets no cluster a partir do AWS Secrets Manager.
#
# Nada aqui é versionado e nada passa por argumento de linha de comando
# (argumento aparece em `ps`): os valores vão para o kubectl por stdin.
#
#   ./scripts/prod-secret.sh
set -euo pipefail

REGIAO=${REGIAO:-us-east-1}
NS=${NS:-oficina-prod}
SEGREDO_DB=${SEGREDO_DB:-oficina/prod/db}
SEGREDO_APP=${SEGREDO_APP:-oficina/prod/app}

ler() { aws secretsmanager get-secret-value --secret-id "$1" --region "$REGIAO" --query SecretString --output text 2>/dev/null; }

# ── Credenciais do banco: criadas pelo oficina-infra-db ─────────────────────
DB_JSON=$(ler "$SEGREDO_DB") || { echo "segredo $SEGREDO_DB não encontrado — aplique o oficina-infra-db antes"; exit 1; }
DB_USER=$(printf '%s' "$DB_JSON" | python3 -c 'import sys,json;print(json.load(sys.stdin)["username"])')
DB_PASSWORD=$(printf '%s' "$DB_JSON" | python3 -c 'import sys,json;print(json.load(sys.stdin)["password"])')

# ── Segredos da aplicação: criados aqui na primeira execução ────────────────
# O JWT_SECRET vira segredo COMPARTILHADO com a Lambda no dia 5 (F3-2.5).
# Guardá-lo no Secrets Manager desde já evita ter que rotacionar depois.
if ! APP_JSON=$(ler "$SEGREDO_APP"); then
  echo "==> criando $SEGREDO_APP com segredos aleatórios"
  APP_JSON=$(python3 -c '
import json, secrets
print(json.dumps({"jwt_secret": secrets.token_urlsafe(48), "webhook_secret": secrets.token_urlsafe(32)}))')
  aws secretsmanager create-secret --name "$SEGREDO_APP" --region "$REGIAO" \
    --secret-string "$APP_JSON" >/dev/null
fi
JWT_SECRET=$(printf '%s' "$APP_JSON" | python3 -c 'import sys,json;print(json.load(sys.stdin)["jwt_secret"])')
WEBHOOK_SECRET=$(printf '%s' "$APP_JSON" | python3 -c 'import sys,json;print(json.load(sys.stdin)["webhook_secret"])')

# `apply -f -` em vez de `create`: idempotente, e os valores não aparecem em ps.
kubectl apply -n "$NS" -f - <<YAML
apiVersion: v1
kind: Secret
metadata:
  name: oficina-secrets
  labels:
    app.kubernetes.io/part-of: tech-challenge
type: Opaque
stringData:
  DB_USER: "${DB_USER}"
  DB_PASSWORD: "${DB_PASSWORD}"
  JWT_SECRET: "${JWT_SECRET}"
  WEBHOOK_SECRET: "${WEBHOOK_SECRET}"
YAML
