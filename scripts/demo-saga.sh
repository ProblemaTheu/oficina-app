#!/usr/bin/env bash
# Conduz a saga da Fase 4 no ambiente local (docker-compose.saga.yml), passo a
# passo, como fariam o atendente, o mecânico, o cliente e o Mercado Pago.
#
# Pré-requisito: docker compose -f docker-compose.saga.yml --profile saga up -d --build
#
# Uso: scripts/demo-saga.sh [feliz|falha-reparo|recusa|cartao-recusado|todos]
set -euo pipefail
cd "$(dirname "$0")/.."

OS=${OS_URL:-http://localhost:8080}
BILLING=${BILLING_URL:-http://localhost:8081}
EXEC=${EXECUTION_URL:-http://localhost:8082}
MP=${MP_URL:-http://localhost:8090}

dc() { docker compose -f docker-compose.saga.yml --profile saga "$@"; }
j() { python3 -c "import json,sys;d=json.load(sys.stdin);print($1)"; }
passo() { printf '\n\033[1;36m▶ %s\033[0m\n' "$*"; }
psqlos() { dc exec -T os-db psql -U postgres -d tech_challenge_db -tA -c "$1"; }

login() {
  curl -sf -X POST "$OS/v1/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$1\",\"senha\":\"$2\"}" | j "d['access_token']"
}

post() { # post URL TOKEN [JSON]
  curl -sf -X POST "$1" -H "Authorization: Bearer $2" -H 'Content-Type: application/json' -d "${3:-{\}}"
}

estado() {
  psqlos "select s.estado||' · OS '||st.nome_status from sagas_os s
          join ordens_servico o on o.id=s.os_id join status_ordens st on st.id=o.status_id
          where s.os_id='$1'"
}

esperar_estado() { # esperar_estado OS_ID PREFIXO
  local e
  for _ in $(seq 1 60); do
    e=$(estado "$1")
    if [[ "$e" == "$2"* ]]; then echo "  saga: $e"; return 0; fi
    sleep 0.5
  done
  echo "  TIMEOUT — saga em: $(estado "$1") (esperava $2)" >&2
  return 1
}

etapas() {
  psqlos "select string_agg(e.etapa||'='||e.resultado, '  ' order by e.ocorrido_em, e.id)
          from sagas_os_etapas e join sagas_os s on s.id=e.saga_id
          where s.os_id='$1' and e.resultado<>'ignorada'"
}

estoque() { psqlos "select 'estoque atual='||estoque_atual||' reservado='||estoque_reservado from pecas where id='$1'"; }

cpf() {
  python3 - <<'PY'
import random
d=[random.randint(0,9) for _ in range(9)]
for n in (10,11):
    s=sum(v*(n-i) for i,v in enumerate(d)); r=(s*10)%11; d.append(0 if r==10 else r)
print(''.join(map(str,d)))
PY
}

placa() { printf '%s%d%s%02d' "$1" $((RANDOM % 10)) "$(printf '\\%03o' $((65 + RANDOM % 26)))" $((RANDOM % 100)); }

preparar() { # preparar PREFIXO_PLACA → define CLIENTE VEICULO PECA
  local p; p=$(printf '%b' "$(placa "$1")")
  CLIENTE=$(post "$OS/v1/clients" "$ADM" "{\"nome\":\"Cliente Demo\",\"documento\":\"$(cpf)\",\"email\":\"demo-$RANDOM@exemplo.com\"}" | j "d['id']")
  VEICULO=$(post "$OS/v1/vehicles" "$ADM" "{\"cliente_id\":\"$CLIENTE\",\"placa\":\"$p\",\"marca\":\"VW\",\"modelo\":\"Gol\",\"ano\":2020}" | j "d['id']")
  PECA=$(post "$OS/v1/parts" "$ADM" "{\"nome\":\"Amortecedor demo $RANDOM\",\"codigo\":\"DEMO-$RANDOM$RANDOM\",\"preco\":189.90,\"estoque_minimo\":1,\"estoque_atual\":10}" | j "d['id']")
  echo "  cliente $CLIENTE · veículo $p · peça $PECA"
}

abrir_os() {
  OS_ID=$(post "$OS/v1/work-orders" "$ADM" \
    "{\"cliente_id\":\"$CLIENTE\",\"veiculo_id\":\"$VEICULO\",\"descricao\":\"barulho na suspensão\",\"pecas\":[{\"peca_id\":\"$PECA\",\"quantidade\":2}]}" | j "d['id']")
  echo "  OS $OS_ID"
  echo "  $(estoque "$PECA")"
}

tarefa() { # tarefa OS_ID TIPO
  local t
  for _ in $(seq 1 60); do
    t=$(curl -sf "$EXEC/execution/v1/tarefas?os_id=$1" -H "Authorization: Bearer $MEC" |
      j "next((x['id'] for x in d if x['tipo']=='$2' and x['estado']!='CANCELADA'), '')" 2>/dev/null || true)
    if [ -n "$t" ]; then echo "$t"; return 0; fi
    sleep 0.5
  done
  echo "tarefa $2 não apareceu na fila" >&2; return 1
}

orcamento() {
  local o
  for _ in $(seq 1 60); do
    o=$(curl -sf "$BILLING/billing/v1/orcamentos?os_id=$1" -H "Authorization: Bearer $ADM" | j "d.get('id','')" 2>/dev/null || true)
    if [ -n "$o" ]; then echo "$o"; return 0; fi
    sleep 0.5
  done
  echo "orçamento não foi gerado" >&2; return 1
}

diagnosticar() {
  passo "Mecânico conclui o diagnóstico (Execution)"
  local t; t=$(tarefa "$OS_ID" DIAGNOSTICO)
  post "$EXEC/execution/v1/tarefas/$t/iniciar" "$MEC" >/dev/null
  post "$EXEC/execution/v1/tarefas/$t/concluir-diagnostico" "$MEC" '{"diagnostico":"amortecedor dianteiro vazando"}' >/dev/null
  esperar_estado "$OS_ID" AGUARDANDO_APROVACAO
}

aprovar() {
  passo "Cliente aprova o orçamento (Billing → cria a order no Mercado Pago)"
  ORDER=$(post "$BILLING/billing/v1/orcamentos/$(orcamento "$OS_ID")/aprovar" "$ADM" | j "d['pagamento']['mp_order_id']")
  echo "  order $ORDER · checkout $MP/checkout/$ORDER"
  esperar_estado "$OS_ID" AGUARDANDO_PAGAMENTO
}

pagar() { # pagar processed|failed
  passo "Mercado Pago: pagamento $1"
  curl -sf -X POST "$MP/_stub/pagar" -H 'Content-Type: application/json' -d "{\"order_id\":\"$ORDER\",\"resultado\":\"$1\"}" >/dev/null
}

resumo() {
  passo "Resultado"
  echo "  saga:    $(estado "$OS_ID")"
  echo "  etapas:  $(etapas "$OS_ID")"
  echo "  $(estoque "$PECA")"
  if [ -n "${ORDER:-}" ]; then
    echo "  order:   $(curl -sf "$MP/v1/orders/$ORDER" | j "d['status']+(' · estorno '+d['transactions']['refunds'][0]['id'] if d['transactions'].get('refunds') else '')")"
  fi
}

feliz() {
  passo "Caminho feliz — atendente abre a OS (T1: reserva 2 peças)"
  ORDER=""; preparar ABC; abrir_os
  diagnosticar; aprovar; pagar processed
  esperar_estado "$OS_ID" EXECUCAO_PENDENTE
  passo "Mecânico inicia (pivô T6: baixa do estoque) e conclui o reparo"
  local t; t=$(tarefa "$OS_ID" REPARO)
  post "$EXEC/execution/v1/tarefas/$t/iniciar" "$MEC" >/dev/null
  esperar_estado "$OS_ID" EM_EXECUCAO
  post "$EXEC/execution/v1/tarefas/$t/concluir" "$MEC" >/dev/null
  esperar_estado "$OS_ID" CONCLUIDA
  resumo
}

falha_reparo() {
  passo "Falha no reparo — veículo FAL* (Execution com PERMITIR_FALHA_SIMULADA=true)"
  ORDER=""; preparar FAL; abrir_os
  diagnosticar; aprovar; pagar processed
  passo "Execution responde ExecucaoFalhou → estorno, cancelamentos e liberação da reserva"
  esperar_estado "$OS_ID" COMPENSADA
  resumo
}

recusa() {
  passo "Orçamento recusado pelo cliente"
  ORDER=""; preparar ABC; abrir_os
  diagnosticar
  post "$BILLING/billing/v1/orcamentos/$(orcamento "$OS_ID")/recusar" "$ADM" '{"motivo":"caro demais"}' >/dev/null
  esperar_estado "$OS_ID" COMPENSADA
  resumo
}

cartao_recusado() {
  passo "Cartão recusado"
  ORDER=""; preparar ABC; abrir_os
  diagnosticar; aprovar; pagar failed
  esperar_estado "$OS_ID" COMPENSADA
  passo "Cliente tenta pagar de novo no mesmo link — a order foi cancelada"
  if curl -sf -o /dev/null -X POST "$MP/_stub/pagar" -H 'Content-Type: application/json' -d "{\"order_id\":\"$ORDER\",\"resultado\":\"processed\"}"; then
    echo "  ⚠ o stub aceitou o pagamento (a saga estorna como pagamento tardio)"
  else
    echo "  pagamento bloqueado ✔"
  fi
  resumo
}

ADM=$(login admin@oficina.com Admin@123)
MEC=$(login joao@oficina.com Mecanico@123)

case "${1:-feliz}" in
  feliz) feliz ;;
  falha-reparo) falha_reparo ;;
  recusa) recusa ;;
  cartao-recusado) cartao_recusado ;;
  todos) feliz; falha_reparo; recusa; cartao_recusado ;;
  *) echo "uso: $0 [feliz|falha-reparo|recusa|cartao-recusado|todos]" >&2; exit 2 ;;
esac
