// Package config resolve os segredos de assinatura da aplicação.
//
// Existia uma cópia de cada resolução em dois lugares, e ambas caíam num
// literal quando a variável estava ausente. Um segredo padrão versionado
// significa que qualquer pessoa com acesso ao repositório forja tokens
// válidos — e o padrão é justamente o que roda quando alguém erra a
// configuração do ambiente.
package config

import (
	"fmt"
	"os"
)

// tamanhoMinimo é o piso para uma chave HMAC-SHA256 não ser adivinhável.
const tamanhoMinimo = 32

// JWTSecret devolve o segredo de assinatura do JWT.
//
// Entra em pânico quando ausente ou curto demais, e isso é deliberado: o pod
// entra em CrashLoopBackOff, a readiness nunca passa, o rollout falha e a
// versão anterior continua servindo. Falhar alto na inicialização é melhor
// do que subir inseguro em silêncio.
//
// O mesmo valor é lido pelas Lambdas de autenticação a partir do campo
// jwt_secret do segredo oficina/{ambiente}/app no Secrets Manager. É a
// igualdade desses dois valores que faz o HS256 funcionar entre os dois
// emissores.
func JWTSecret() []byte {
	return obrigatorio("JWT_SECRET")
}

// WebhookSecret devolve o segredo compartilhado do HMAC dos webhooks.
func WebhookSecret() []byte {
	return obrigatorio("WEBHOOK_SECRET")
}

func obrigatorio(nome string) []byte {
	v := os.Getenv(nome)
	if len(v) < tamanhoMinimo {
		panic(fmt.Sprintf(
			"%s ausente ou com menos de %d caracteres: a aplicação não sobe sem segredo de assinatura",
			nome, tamanhoMinimo))
	}
	return []byte(v)
}
