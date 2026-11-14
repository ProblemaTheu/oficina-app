#!/usr/bin/env python3
"""Cria no New Relic o dashboard da saga e o alerta de taxa de compensação.

Idempotente: o que já existe (pelo nome) é reaproveitado, não duplicado.

    export NEW_RELIC_API_KEY=NRAK-...        # User key (não a license key)
    ./observability/aplicar-newrelic.py [--conta 8465573] [--email equipe@exemplo.com]

Sem --email, cria só o dashboard, a policy e a condição (sem notificação).
"""
import argparse
import json
import os
import pathlib
import sys
import urllib.request

AQUI = pathlib.Path(__file__).parent
POLICY = "Oficina — Saga (Fase 4)"
CONDICAO = {
    "name": "Taxa de compensação da saga > 30%",
    "description": "Mais de 30% das sagas que terminaram nos últimos 15 min foram COMPENSADAS "
                   "(SagaEvent.terminou). Ver o dashboard 'Oficina — Saga da OS (Fase 4)' > Onde as sagas falham.",
    "enabled": True,
    "nrql": {"query": "SELECT percentage(count(*), WHERE estado = 'COMPENSADA') FROM SagaEvent WHERE terminou IS true"},
    # Janela deslizante de 15 min avaliada a cada 5; sem saga terminada não há sinal.
    "signal": {"aggregationWindow": 900, "slideBy": 300, "aggregationMethod": "EVENT_TIMER",
               "aggregationTimer": 60, "fillOption": "NONE"},
    "terms": [{"threshold": 30, "thresholdOccurrences": "ALL", "thresholdDuration": 300,
               "operator": "ABOVE", "priority": "CRITICAL"}],
    "violationTimeLimitSeconds": 86400,
}


def ng(query, variables=None):
    req = urllib.request.Request(
        "https://api.newrelic.com/graphql",
        data=json.dumps({"query": query, "variables": variables or {}}).encode(),
        headers={"API-Key": os.environ["NEW_RELIC_API_KEY"], "Content-Type": "application/json"})
    resposta = json.load(urllib.request.urlopen(req))
    if resposta.get("errors"):
        sys.exit(json.dumps(resposta["errors"], ensure_ascii=False))
    return resposta["data"]


def dashboard(conta):
    d = json.loads((AQUI / "dashboard-saga.json").read_text())
    achados = ng("query($q:String!){ actor { entitySearch(query:$q) { results { entities { guid } } } } }",
                 {"q": f"type = 'DASHBOARD' AND name = '{d['name']}'"})["actor"]["entitySearch"]["results"]["entities"]
    if achados:
        return achados[0]["guid"], "já existia"
    entrada = {k: d[k] for k in ("name", "description", "permissions")}
    entrada["pages"] = [{"name": p["name"], "widgets": [
        {k: w[k] for k in ("title", "layout", "visualization", "rawConfiguration")} for w in p["widgets"]]}
        for p in d["pages"]]
    r = ng("mutation($a:Int!,$d:DashboardInput!){ dashboardCreate(accountId:$a, dashboard:$d){ entityResult { guid } errors { description } } }",
           {"a": conta, "d": entrada})["dashboardCreate"]
    if r["errors"]:
        sys.exit(r["errors"])
    return r["entityResult"]["guid"], "criado"


def alerta(conta):
    achadas = ng("query($a:Int!,$n:String!){ actor { account(id:$a) { alerts { policiesSearch(searchCriteria:{name:$n}) { policies { id } } } } } }",
                 {"a": conta, "n": POLICY})["actor"]["account"]["alerts"]["policiesSearch"]["policies"]
    if achadas:
        pid = achadas[0]["id"]
    else:
        pid = ng("mutation($a:Int!,$n:String!){ alertsPolicyCreate(accountId:$a, policy:{name:$n, incidentPreference:PER_CONDITION}){ id } }",
                 {"a": conta, "n": POLICY})["alertsPolicyCreate"]["id"]
    condicoes = ng("query($a:Int!,$p:ID!){ actor { account(id:$a) { alerts { nrqlConditionsSearch(searchCriteria:{policyId:$p}) { nrqlConditions { id name } } } } } }",
                   {"a": conta, "p": pid})["actor"]["account"]["alerts"]["nrqlConditionsSearch"]["nrqlConditions"]
    if not any(c["name"] == CONDICAO["name"] for c in condicoes):
        ng("mutation($a:Int!,$p:ID!,$c:AlertsNrqlConditionStaticInput!){ alertsNrqlConditionStaticCreate(accountId:$a, policyId:$p, condition:$c){ id } }",
           {"a": conta, "p": pid, "c": CONDICAO})
    return pid


def notificacao(conta, pid, email):
    destino = ng("mutation($a:Int!,$e:String!){ aiNotificationsCreateDestination(accountId:$a, destination:{type:EMAIL, name:\"Equipe Tech Challenge (e-mail)\", properties:[{key:\"email\", value:$e}]}){ destination { id } } }",
                 {"a": conta, "e": email})["aiNotificationsCreateDestination"]["destination"]["id"]
    canal = ng("mutation($a:Int!,$d:ID!){ aiNotificationsCreateChannel(accountId:$a, channel:{type:EMAIL, name:\"Saga — e-mail\", destinationId:$d, product:IINT, properties:[{key:\"subject\", value:\"[Oficina] {{ issueTitle }}\"}]}){ channel { id } } }",
               {"a": conta, "d": destino})["aiNotificationsCreateChannel"]["channel"]["id"]
    ng("""mutation($a:Int!,$c:ID!,$p:String!){ aiWorkflowsCreateWorkflow(accountId:$a, createWorkflowData:{
          name:"Oficina — Saga (Fase 4)", enrichmentsEnabled:true, workflowEnabled:true, destinationsEnabled:true,
          mutingRulesHandling:NOTIFY_ALL_ISSUES,
          issuesFilter:{type:FILTER, name:"policy da saga", predicates:[{attribute:"labels.policyIds", operator:EXACTLY_MATCHES, values:[$p]}]},
          destinationConfigurations:[{channelId:$c}] }){ errors { description } } }""",
       {"a": conta, "c": canal, "p": str(pid)})


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--conta", type=int, default=8465573)
    p.add_argument("--email")
    a = p.parse_args()
    if "NEW_RELIC_API_KEY" not in os.environ:
        sys.exit("exporte NEW_RELIC_API_KEY (User key NRAK-...)")
    guid, como = dashboard(a.conta)
    print(f"dashboard {como}: https://one.newrelic.com/redirect/entity/{guid}")
    pid = alerta(a.conta)
    print(f"policy '{POLICY}' (id {pid}) com a condição '{CONDICAO['name']}'")
    if a.email:
        notificacao(a.conta, pid, a.email)
        print(f"notificação por e-mail para {a.email}")


if __name__ == "__main__":
    main()
