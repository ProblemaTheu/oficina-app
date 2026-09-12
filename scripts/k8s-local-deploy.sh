#!/usr/bin/env bash
# Sobe a aplicação completa num cluster kind local:
#   1. cria o cluster (se não existir) e instala o metrics-server (p/ HPA)
#   2. builda a imagem e a carrega no kind
#   3. cria o Secret a partir do seu .env (copie de .env.example)
#   4. aplica o overlay local (API + Postgres + Mailpit + HPA)
#
# Pré-requisitos: docker, kind, kubectl e um arquivo .env na raiz.
# Uso: ./scripts/k8s-local-deploy.sh
set -euo pipefail

CLUSTER=oficina
IMAGE=oficina-api:local
REPO_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$REPO_DIR"

if [ ! -f .env ]; then
  echo "erro: .env não encontrado — copie de .env.example e ajuste os valores" >&2
  exit 1
fi
# shellcheck disable=SC1091
source .env

# ── 1. Cluster ────────────────────────────────────────────────────────────────
if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  echo "==> criando cluster kind '$CLUSTER'"
  kind create cluster --name "$CLUSTER" --wait 120s
else
  echo "==> cluster kind '$CLUSTER' já existe"
fi
kubectl config use-context "kind-$CLUSTER" >/dev/null

# metrics-server (necessário para o HPA) com flag para o TLS do kind
if ! kubectl -n kube-system get deploy metrics-server >/dev/null 2>&1; then
  echo "==> instalando metrics-server"
  kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
  kubectl -n kube-system patch deploy metrics-server --type=json \
    -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
fi

# ── 2. Imagem ─────────────────────────────────────────────────────────────────
echo "==> buildando e carregando a imagem $IMAGE"
docker build -t "$IMAGE" .
kind load docker-image "$IMAGE" --name "$CLUSTER"

# ── 3. Namespace + Secret (a partir do .env) ─────────────────────────────────
kubectl apply -f k8s/base/namespace.yaml
kubectl -n oficina create secret generic oficina-secrets \
  --from-literal=DB_USER="${DB_USER:?DB_USER ausente no .env}" \
  --from-literal=DB_PASSWORD="${DB_PASSWORD:?DB_PASSWORD ausente no .env}" \
  --from-literal=JWT_SECRET="${JWT_SECRET:?JWT_SECRET ausente no .env}" \
  --from-literal=WEBHOOK_SECRET="${WEBHOOK_SECRET:?WEBHOOK_SECRET ausente no .env}" \
  --dry-run=client -o yaml | kubectl apply -f -

# ── 4. Manifests ──────────────────────────────────────────────────────────────
echo "==> aplicando overlay local"
kubectl apply -k k8s/overlays/local

echo "==> aguardando rollout"
kubectl -n oficina rollout status statefulset/postgres --timeout=180s
kubectl -n oficina rollout status deployment/api --timeout=180s

cat <<'FIM'

✅ Ambiente local no ar!

Acessos (via port-forward):
  API:       kubectl -n oficina port-forward svc/api 8080:80
  Mailpit:   kubectl -n oficina port-forward svc/mailpit 8025:8025
  HPA:       kubectl -n oficina get hpa -w

Gerar carga para ver o HPA escalar (exemplo com hey):
  hey -z 60s -c 50 http://localhost:8080/health/ready
FIM
