# Roteiro do vídeo demonstrativo (≤ 15 min)

O PDF da Fase 2 exige que o vídeo demonstre **4 coisas**: deploy da aplicação,
execução do CI/CD, consumo das APIs e escalabilidade automática. Roteiro
sugerido com tempos aproximados (total ~13 min, com folga para imprevistos):

## Preparação (antes de gravar)

- [ ] `kind delete cluster --name oficina` (para demonstrar o provisionamento do zero)
- [ ] `.env` presente na raiz (copiado de `.env.example`)
- [ ] Uma PR aberta (ou pronta para abrir) para demonstrar o CI
- [ ] Postman com a coleção `docs/postman_collection.json` importada
- [ ] Terminal com fonte grande e 2 abas (comandos + observação)
- [ ] `hey` instalado para o teste de carga (`brew install hey`)

## 1. Introdução (1 min)

- Quem é o grupo, o problema (oficina em expansão) e o que a Fase 2 entrega:
  K8s + HPA, Terraform, CI/CD, webhook de orçamento e e-mail de status.
- Mostrar rapidamente o diagrama de arquitetura do README.

## 2. Provisionamento com Terraform (2 min)

```bash
terraform -chdir=infra/environments/local apply -auto-approve
```

- Narrar enquanto aplica: cria o cluster kind + metrics-server (pré-requisito do HPA).
- Mostrar os outputs (`contexto_kubectl`) e `kubectl get nodes`.

## 3. Deploy da aplicação no Kubernetes (2 min)

```bash
./scripts/k8s-local-deploy.sh
kubectl -n oficina get pods
```

- Narrar o que o script faz: imagem, Secret a partir do `.env`, `kubectl apply -k`.
- Mostrar os pods: 2 réplicas da API, Postgres (StatefulSet), Mailpit.

## 4. Execução do CI/CD (2 min)

- Abrir a PR no GitHub e mostrar os workflows rodando/verdes:
  - `ci.yml` (build + testes + lint) com o Job Summary (placar, cobertura);
  - `integration.yml` (Postgres real) com o relatório;
  - `security.yml` (scanners).
- Comentar: pipeline roda em PR para a main; merge bloqueável por branch protection.

## 5. Consumo das APIs (3 min)

Com `kubectl -n oficina port-forward svc/api 8080:80` ativo, no Postman:

1. `POST /v1/auth/register` + `login` → token JWT;
2. Criar cliente (com e-mail!), veículo e uma OS;
3. Avançar status: `em_diagnostico` → `aguardando_aprovacao` (com diagnóstico);
4. **Mailpit** (`port-forward svc/mailpit 8025:8025`): mostrar o e-mail de
   notificação que chegou;
5. **Webhook**: request "Resposta de orçamento" da coleção (a assinatura HMAC
   é calculada pelo pre-request script) → OS vai para `em_execucao`;
6. `GET /v1/work-orders`: mostrar a ordenação por prioridade de status e que
   finalizadas/entregues não aparecem (criar/finalizar uma antes, se der tempo).

## 6. Escalabilidade automática (2,5 min)

Em um terminal:

```bash
kubectl -n oficina get hpa -w
```

Em outro:

```bash
hey -z 90s -c 50 http://localhost:8080/health/ready
```

- Narrar: réplicas sobem de 2 para 5 com CPU acima de 50% do request;
- Ao final da carga, comentar a janela de estabilização do scale-down.

## 7. Encerramento (0,5 min)

- Recapitular os entregáveis (`/k8s`, `/infra`, workflows, README com arquitetura);
- Onde encontrar instruções (README) e coleção de APIs.

## Publicação

- YouTube ou Vimeo, **público ou não listado**, até 15 minutos;
- Adicionar o link no README (seção "Onde está cada coisa") e no PDF da entrega.
