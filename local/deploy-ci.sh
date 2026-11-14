#!/usr/bin/env bash
# Deploy de UM microsserviço num Kubernetes efêmero (kind) dentro do CD:
# deploy automatizado em ambiente Kubernetes, depois dos testes (o deploy no
# EKS é o outro destino, com DEPLOY_ENABLED=true).
#
#   local/deploy-ci.sh <os|billing|execution> <imagem:tag>
#
# Usa a imagem que o próprio pipeline acabou de publicar (tag = SHA), os
# manifests k8s/base de produção (via local/servicos/<serviço>) e só as
# dependências daquele serviço. Termina com rollout + smoke /health/ready;
# qualquer falha devolve exit != 0 com diagnóstico no log.
#
# Espera os repositórios como pastas irmãs (oficina-app, oficina-billing,
# oficina-execution), como no docker-compose.saga.yml.
#
# PULL_POLICY=Never + kind load: para testar localmente com imagem local.
set -euo pipefail

SERVICO=${1:?uso: deploy-ci.sh <os|billing|execution> <imagem:tag>}
IMAGEM=${2:?uso: deploy-ci.sh <os|billing|execution> <imagem:tag>}
RAIZ=$(cd "$(dirname "$0")/.." && pwd)
CLUSTER=${CLUSTER:-ci}
PULL_POLICY=${PULL_POLICY:-IfNotPresent}

case "$SERVICO" in
  os)        NS=oficina-prod;   DEPLOY=api;       LOCAL_IMG=oficina-api:local;       DEPS="postgres-os localstack mailpit" ;;
  billing)   NS=billing-prod;   DEPLOY=billing;   LOCAL_IMG=oficina-billing:local;   DEPS="postgres-billing localstack mailpit" ;;
  execution) NS=execution-prod; DEPLOY=execution; LOCAL_IMG=oficina-execution:local; DEPS="localstack" ;;
  *) echo "serviço desconhecido: $SERVICO"; exit 2 ;;
esac
k() { kubectl --context "kind-$CLUSTER" "$@"; }
grupo() { [[ -n "${GITHUB_ACTIONS:-}" ]] && echo "::group::$*" || echo "==> $*"; }
fim() { [[ -n "${GITHUB_ACTIONS:-}" ]] && echo "::endgroup::" || true; }

diagnostico() {
  echo "::error::deploy de $SERVICO falhou — diagnóstico abaixo"
  k get pods -A -o wide || true
  k -n "$NS" describe "deployment/$DEPLOY" | tail -30 || true
  k -n "$NS" logs "deployment/$DEPLOY" --tail=80 || true
}
trap 'diagnostico' ERR

grupo "cluster kind '$CLUSTER'"
if ! command -v kind >/dev/null; then
  curl -sSLo /tmp/kind --proto '=https' --tlsv1.2 https://kind.sigs.k8s.io/dl/v0.31.0/kind-linux-amd64
  chmod +x /tmp/kind && sudo mv /tmp/kind /usr/local/bin/kind
fi
kind get clusters 2>/dev/null | grep -qx "$CLUSTER" || kind create cluster --name "$CLUSTER" --wait 180s
[[ "$PULL_POLICY" = Never ]] && kind load docker-image --name "$CLUSTER" "$IMAGEM"
for ns in infra "$NS"; do k create namespace "$ns" --dry-run=client -o yaml | k apply -f - >/dev/null; done
fim

grupo "segredos efêmeros (cluster descartável: valores aleatórios)"
aleatorio() { head -c 48 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 40; }
TMP=$(mktemp -d); chmod 700 "$TMP"
segredo() { local ns="$1" nome="$2"; shift 2; printf '%s\n' "$@" > "$TMP/$nome"
  k -n "$ns" create secret generic "$nome" --from-env-file="$TMP/$nome" --dry-run=client -o yaml | k apply -f - >/dev/null; }
JWT=$(aleatorio); PG=$(aleatorio)
# O LocalStack aceita qualquer credencial: nada fixo no código.
LS_ID=$(aleatorio); LS_CHAVE=$(aleatorio)
case "$SERVICO" in
  os)
    segredo infra postgres-os POSTGRES_USER=oficina POSTGRES_PASSWORD="$PG" POSTGRES_DB=tech_challenge_db
    segredo "$NS" oficina-secrets DB_USER=oficina DB_PASSWORD="$PG" DB_NAME=tech_challenge_db JWT_SECRET="$JWT" \
      WEBHOOK_SECRET="$(aleatorio)" INTERNAL_TOKEN="$(aleatorio)" AWS_ACCESS_KEY_ID="$LS_ID" AWS_SECRET_ACCESS_KEY="$LS_CHAVE" ;;
  billing)
    segredo infra postgres-billing POSTGRES_USER=billing POSTGRES_PASSWORD="$PG" POSTGRES_DB=billing
    segredo "$NS" billing-secrets DB_USER=billing DB_PASSWORD="$PG" DB_NAME=billing JWT_SECRET="$JWT" \
      MP_ACCESS_TOKEN="$(aleatorio)" MP_WEBHOOK_SECRET="$(aleatorio)" AWS_ACCESS_KEY_ID="$LS_ID" AWS_SECRET_ACCESS_KEY="$LS_CHAVE" ;;
  execution)
    segredo "$NS" execution-secrets JWT_SECRET="$JWT" AWS_ACCESS_KEY_ID="$LS_ID" AWS_SECRET_ACCESS_KEY="$LS_CHAVE" ;;
  *) echo "serviço desconhecido: $SERVICO"; exit 2 ;;
esac
rm -rf "$TMP"
fim

grupo "dependências: $DEPS"
k -n infra create configmap localstack-init --from-file=init.sh="$RAIZ/scripts/localstack-init.sh" \
  --dry-run=client -o yaml | k apply -f - >/dev/null
for d in $DEPS; do k -n infra apply -f "$RAIZ/local/infra/$d.yaml" >/dev/null; done
for d in $DEPS; do k -n infra rollout status "deployment/$d" --timeout=420s; done
fim

grupo "deploy de $SERVICO com $IMAGEM (manifests k8s/base de produção)"
# O overlay local aponta para a imagem construída na máquina; aqui entra a
# imagem publicada pelo pipeline.
kubectl kustomize "$RAIZ/local/servicos/$SERVICO" \
  | sed -e "s#image: $LOCAL_IMG#image: $IMAGEM#" -e "s#imagePullPolicy: Never#imagePullPolicy: $PULL_POLICY#" \
  | tee /tmp/manifesto-$SERVICO.yaml | grep -E '^\s+image:'
k apply -f /tmp/manifesto-$SERVICO.yaml
k -n "$NS" rollout status "deployment/$DEPLOY" --timeout=300s
fim

grupo "smoke test /health/ready"
PRONTO=$(k -n "$NS" exec "deployment/$DEPLOY" -- wget -qO- http://localhost:8080/health/ready)
echo "$PRONTO"
echo "$PRONTO" | grep -q '"status":"UP"'
fim

trap - ERR
k get pods -n "$NS" -o wide
echo "✅ $SERVICO implantado em Kubernetes ($NS/$DEPLOY) com $IMAGEM e saudável"
