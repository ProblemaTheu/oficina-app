#!/usr/bin/env bash
# Sobe a Fase 4 inteira num Kubernetes local (kind sobre Colima/Docker):
# os 3 microsserviços com os manifests k8s/base de cada repositório, cada um
# com o próprio banco, conversando por SQS FIFO (LocalStack), com a saga
# orquestrada pelo OS Service e telemetria no New Relic.
#
#
#   cp .env.example .env                       # se ainda não existir
#   export NEW_RELIC_LICENSE_KEY=...           # opcional: APM, logs e SagaEvent
#   ./local/subir.sh                           # cria o cluster, se faltar, e sobe tudo
#   MP_REAL=1 ./local/subir.sh                 # Billing no sandbox REAL do Mercado Pago
#   SEM_BUILD=1 ./local/subir.sh               # reaplica sem reconstruir as imagens
#
# Billing e Execution são lidos das pastas irmãs (../oficina-billing e
# ../oficina-execution), como no docker-compose.saga.yml.
set -euo pipefail

RAIZ=$(cd "$(dirname "$0")/.." && pwd)
BILLING_DIR=${BILLING_DIR:-$RAIZ/../oficina-billing}
EXECUTION_DIR=${EXECUTION_DIR:-$RAIZ/../oficina-execution}
CLUSTER=oficina
CTX=kind-$CLUSTER
k() { kubectl --context "$CTX" "$@"; }

[[ -f "$RAIZ/.env" ]] || { echo "falta $RAIZ/.env — cp .env.example .env"; exit 1; }
set -a; # shellcheck disable=SC1091
. "$RAIZ/.env"; set +a
: "${JWT_SECRET:?}" "${WEBHOOK_SECRET:?}" "${DB_USER:?}" "${DB_PASSWORD:?}" "${DB_NAME:?}"
: "${BILLING_DB_USER:?}" "${BILLING_DB_PASSWORD:?}" "${BILLING_DB_NAME:?}" "${MP_WEBHOOK_SECRET:?}"
aleatorio() { head -c 48 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 40; }
# Sem valor fixo no código: o .env pode definir; senão, um por execução.
INTERNAL_TOKEN=${INTERNAL_TOKEN:-$(aleatorio)}

# ── 1. Cluster ──────────────────────────────────────────────────────────────
if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  echo "==> criando o cluster kind '$CLUSTER'"
  kind create cluster --config "$RAIZ/local/kind.yaml"
fi

# ── 2. Imagens (construídas aqui: o CD publica só linux/amd64) ──────────────
if [[ -z "${SEM_BUILD:-}" ]]; then
  echo "==> construindo as imagens"
  docker build -q -t oficina-api:local "$RAIZ" >/dev/null
  docker build -q -t oficina-billing:local "$BILLING_DIR" >/dev/null
  docker build -q -t oficina-execution:local "$EXECUTION_DIR" >/dev/null
  kind load docker-image --name "$CLUSTER" oficina-api:local oficina-billing:local oficina-execution:local >/dev/null
fi

# ── 3. Namespaces (em produção são do oficina-infra-k8s) ────────────────────
for ns in infra oficina-prod billing-prod execution-prod; do
  k create namespace "$ns" --dry-run=client -o yaml | k apply -f - >/dev/null
done

# ── 4. Segredos — do .env, por arquivo temporário (nada em argumento/ps) ────
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT; chmod 700 "$TMP"
segredo() { # segredo NAMESPACE NOME CHAVE=VALOR...
  local ns="$1" nome="$2"; shift 2
  printf '%s\n' "$@" > "$TMP/$nome.env"
  k -n "$ns" create secret generic "$nome" --from-env-file="$TMP/$nome.env" \
    --dry-run=client -o yaml | k apply -f - >/dev/null
}
NR=()
[[ -n "${NEW_RELIC_LICENSE_KEY:-}" ]] && NR=("NEW_RELIC_LICENSE_KEY=$NEW_RELIC_LICENSE_KEY")
# O LocalStack aceita qualquer credencial: geradas a cada execução.
AWS_FAKE=("AWS_ACCESS_KEY_ID=$(aleatorio)" "AWS_SECRET_ACCESS_KEY=$(aleatorio)")

segredo infra postgres-os POSTGRES_USER="$DB_USER" POSTGRES_PASSWORD="$DB_PASSWORD" POSTGRES_DB="$DB_NAME"
segredo infra postgres-billing POSTGRES_USER="$BILLING_DB_USER" POSTGRES_PASSWORD="$BILLING_DB_PASSWORD" POSTGRES_DB="$BILLING_DB_NAME"
segredo infra mp-stub MP_WEBHOOK_SECRET="$MP_WEBHOOK_SECRET"
k -n infra create configmap localstack-init --from-file=init.sh="$RAIZ/scripts/localstack-init.sh" \
  --dry-run=client -o yaml | k apply -f - >/dev/null

segredo oficina-prod oficina-secrets DB_USER="$DB_USER" DB_PASSWORD="$DB_PASSWORD" DB_NAME="$DB_NAME" \
  JWT_SECRET="$JWT_SECRET" WEBHOOK_SECRET="$WEBHOOK_SECRET" INTERNAL_TOKEN="$INTERNAL_TOKEN" \
  "${AWS_FAKE[@]}" ${NR[@]+"${NR[@]}"}
# Stub: aceita qualquer token. Sandbox real (MP_REAL=1): o TEST-... do .env.
MP_TOKEN=$(aleatorio)
[[ -n "${MP_REAL:-}" ]] && MP_TOKEN=${MP_ACCESS_TOKEN:?MP_REAL=1 exige MP_ACCESS_TOKEN no .env}
segredo billing-prod billing-secrets DB_USER="$BILLING_DB_USER" DB_PASSWORD="$BILLING_DB_PASSWORD" \
  DB_NAME="$BILLING_DB_NAME" JWT_SECRET="$JWT_SECRET" MP_ACCESS_TOKEN="$MP_TOKEN" \
  MP_WEBHOOK_SECRET="$MP_WEBHOOK_SECRET" "${AWS_FAKE[@]}" ${NR[@]+"${NR[@]}"}
segredo execution-prod execution-secrets JWT_SECRET="$JWT_SECRET" "${AWS_FAKE[@]}" ${NR[@]+"${NR[@]}"}

# ── 5. Dependências ─────────────────────────────────────────────────────────
echo "==> dependências (Postgres x2, LocalStack, stub do MP, Mailpit)"
k apply -k "$RAIZ/local/infra" >/dev/null
for d in postgres-os postgres-billing localstack mp-stub mailpit; do
  k -n infra rollout status "deployment/$d" --timeout=300s >/dev/null
done

# ── 6. Serviços (mesmo k8s/base de produção) ────────────────────────────────
echo "==> serviços"
k apply -k "$RAIZ/local/servicos/os" >/dev/null
k apply -k "$RAIZ/local/servicos/billing" >/dev/null
k apply -k "$RAIZ/local/servicos/execution" >/dev/null
if [[ -n "${MP_REAL:-}" ]]; then
  k -n billing-prod patch configmap billing-config --type merge \
    -p '{"data":{"MP_BASE_URL":"https://api.mercadopago.com","MP_RECONCILIAR_INTERVALO":"10s"}}' >/dev/null
fi
# Secret/ConfigMap mudam sem mudar o template do pod: reinicia para reler.
k -n oficina-prod rollout restart deployment/api >/dev/null
k -n billing-prod rollout restart deployment/billing >/dev/null
k -n execution-prod rollout restart deployment/execution >/dev/null
k -n oficina-prod rollout status deployment/api --timeout=300s
k -n billing-prod rollout status deployment/billing --timeout=300s
k -n execution-prod rollout status deployment/execution --timeout=300s

# Só libera a demo quando os pods ANTIGOS saíram. Um pod terminando no meio de
# um long-poll do SQS pode receber uma mensagem e sair sem processá-la: ela
# volta após o visibility timeout (60 s) — nada se perde, mas a saga atrasa.
echo "==> aguardando os pods antigos encerrarem"
for _ in $(seq 1 60); do
  k get pods -A --no-headers 2>/dev/null | grep -E '^(oficina|billing|execution)-prod ' | grep -q Terminating || break
  sleep 2
done

# E só quando os 3 respondem pela porta do HOST (o caminho da demo e do
# Postman): o kube-proxy leva alguns segundos para apontar o NodePort para o
# pod novo depois do rollout.
echo "==> aguardando os serviços responderem em localhost"
for porta in 8080 8081 8082; do
  for _ in $(seq 1 60); do
    curl -sf -m 2 "http://localhost:$porta/health/ready" >/dev/null && break
    sleep 1
  done
  curl -sf -m 2 "http://localhost:$porta/health/ready" >/dev/null || { echo "localhost:$porta não respondeu /health/ready"; exit 1; }
done

# Saga de aquecimento: a primeira resposta da saga após o (re)início dos
# consumidores pode levar o visibility timeout (60 s) para voltar. Depois
# dela, as sagas fluem em segundos — a demo começa já aquecida.
if [[ -z "${SEM_AQUECIMENTO:-}" ]]; then
  echo "==> saga de aquecimento (até ~1 min)"
  AMBIENTE=k8s "$RAIZ/scripts/demo-saga.sh" recusa >/dev/null 2>&1 \
    || echo "   aviso: a saga de aquecimento não concluiu — rode ./scripts/demo-saga.sh e veja o log"
fi

cat <<EOF

✅ Fase 4 no Kubernetes local (contexto $CTX)
   OS Service   http://localhost:8080   (Swagger em /swagger, se habilitado)
   Billing      http://localhost:8081
   Execution    http://localhost:8082
   Stub do MP   http://localhost:8090   $( [[ -n "${MP_REAL:-}" ]] && echo "(não usado: sandbox REAL)" )
   Mailpit      http://localhost:8025
   New Relic    $( [[ ${#NR[@]} -gt 0 ]] && echo "ligado (apps *-local)" || echo "desligado — exporte NEW_RELIC_LICENSE_KEY e rode de novo" )

   Saga ponta a ponta:  ./scripts/demo-saga.sh todos
   Pods:                kubectl --context $CTX get pods -A -l app.kubernetes.io/part-of
EOF
