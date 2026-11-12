#!/usr/bin/env bash
# Gate de cobertura unitária (F4-5.1), o mesmo script dos outros dois
# serviços. Mesmas exclusões do sonar.coverage.exclusions em
# sonar-project.properties — mantenha as duas listas iguais:
#   - cmd/ e scripts/: bootstrap e utilitários;
#   - internal/infra/repository/ e database/: testados contra PostgreSQL real
#     (workflow "Testes de integração");
#   - api.gen.go: código gerado pelo oapi-codegen.
set -euo pipefail
MINIMO="${MINIMO:-80}"
EXCLUSOES='(/cmd/|/scripts/|/internal/infra/repository/|/internal/infra/database/|api\.gen\.go)'

go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
{ head -1 coverage.out; tail -n +2 coverage.out | grep -vE "$EXCLUSOES"; } > coverage.unit.out
total=$(go tool cover -func=coverage.unit.out | awk '/^total:/ { sub("%", "", $3); print $3 }')
echo "Cobertura unitária (critério do Sonar): ${total}%"
echo "cobertura=${total}" >> "${GITHUB_OUTPUT:-/dev/null}"
awk -v t="$total" -v m="$MINIMO" 'BEGIN { exit !(t >= m) }' \
  || { echo "::error::Cobertura ${total}% abaixo do mínimo de ${MINIMO}%"; exit 1; }
