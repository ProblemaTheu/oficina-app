package middleware

import (
	"context"

	"github.com/golang-jwt/jwt/v5"

	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
)

// Discriminadores do claim "tipo", definidos no contrato F3-0.2. São dois
// emissores assinando com o mesmo segredo — a Lambda, para clientes, e esta
// aplicação, para funcionários —, e é este claim que impede um token de
// cliente alcançar operação interna.
const (
	TipoCliente = "cliente"
	TipoUsuario = "usuario"
)

// ClaimsDoContexto devolve os claims que o middleware JWT guardou.
func ClaimsDoContexto(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(jwt.MapClaims)
	return claims, ok
}

// TipoDoContexto devolve o claim "tipo" do token.
//
// Token sem o claim é tratado como funcionário: são os emitidos antes deste
// contrato existir, que ainda estão válidos por até 8 horas. Depois desse
// prazo nenhum token sem "tipo" circula. A alternativa — rejeitar — deslogaria
// todo mundo no instante do deploy.
func TipoDoContexto(ctx context.Context) string {
	claims, ok := ClaimsDoContexto(ctx)
	if !ok {
		return ""
	}
	tipo, _ := claims["tipo"].(string)
	if tipo == "" {
		return TipoUsuario
	}
	return tipo
}

// SubDoContexto devolve o "sub" — o UUID do cliente ou do funcionário.
func SubDoContexto(ctx context.Context) string {
	claims, ok := ClaimsDoContexto(ctx)
	if !ok {
		return ""
	}
	sub, _ := claims["sub"].(string)
	return sub
}

// ExigirUsuario bloqueia tokens de cliente em operações internas.
//
// Autenticar é saber quem é; autorizar é saber o que pode. O authorizer do
// API Gateway faz o primeiro e para por aí — ele valida qualquer token bem
// assinado, inclusive o de um cliente pedindo para criar uma OS.
func ExigirUsuario(ctx context.Context) error {
	if TipoDoContexto(ctx) == TipoUsuario {
		return nil
	}
	return &domainerros.ErrProibido{
		Codigo:   "ACCESS_DENIED",
		Mensagem: "esta operação é restrita a usuários da oficina",
	}
}
