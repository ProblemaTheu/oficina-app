# Kubernetes — Tech Challenge (Fase 2)

Manifestos organizados com **Kustomize**: uma `base` portável e overlays por
ambiente.

```
k8s/
├── base/                    # recursos comuns a todos os ambientes
│   ├── namespace.yaml       # namespace `oficina`
│   ├── configmap.yaml       # config não sensível (DB_HOST, NOTIFIER, ...)
│   ├── secret.example.yaml  # TEMPLATE do Secret (valores reais fora do git)
│   ├── deployment.yaml      # API: 2 réplicas, probes, resources, non-root
│   ├── service.yaml         # ClusterIP 80 → 8080
│   └── hpa.yaml             # autoscaling/v2: 2–5 réplicas, CPU 50%
└── overlays/
    ├── local/               # kind/minikube: + Postgres in-cluster + Mailpit
    └── aws/                 # EKS + RDS (preparado; concluído no épico E5)
```

## Subir o ambiente local (kind)

O caminho rápido é o script que automatiza cluster, imagem, secret e deploy:

```bash
cp .env.example .env    # se ainda não existir
./scripts/k8s-local-deploy.sh
```

Passo a passo equivalente:

```bash
kind create cluster --name oficina
# metrics-server (HPA) — precisa da flag de TLS no kind:
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl -n kube-system patch deploy metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'

docker build -t oficina-api:local .
kind load docker-image oficina-api:local --name oficina

kubectl apply -f k8s/base/namespace.yaml
kubectl -n oficina create secret generic oficina-secrets \
  --from-literal=DB_USER=postgres \
  --from-literal=DB_PASSWORD=... \
  --from-literal=JWT_SECRET=... \
  --from-literal=WEBHOOK_SECRET=...

kubectl apply -k k8s/overlays/local
```

## Acessos

| Serviço | Comando |
|---------|---------|
| API | `kubectl -n oficina port-forward svc/api 8080:80` → http://localhost:8080 |
| Mailpit (e-mails) | `kubectl -n oficina port-forward svc/mailpit 8025:8025` → http://localhost:8025 |

## Demonstrar a escalabilidade (HPA)

```bash
kubectl -n oficina get hpa -w           # acompanhar em um terminal
hey -z 60s -c 50 http://localhost:8080/health/ready   # carga em outro
```

Com `requests.cpu: 100m` e alvo de 50%, a carga acima leva o deployment de
2 para até 5 réplicas; ao cessar, o HPA reduz após a janela de estabilização.

## Segredos

`base/secret.example.yaml` é só um template. Os valores reais **nunca** são
versionados: o script cria o Secret a partir do seu `.env`. Para produção,
considerar Sealed Secrets ou SOPS (evolução documentada, não implementada).

## Decisão — migrations (F2-4.7)

As migrations rodam no boot da aplicação. Mesmo com 2+ réplicas não há
corrida: o `golang-migrate` usa advisory lock do Postgres, então execuções
concorrentes são serializadas e idempotentes. Por isso o Job dedicado de
migrations (opcional no backlog) foi dispensado.
