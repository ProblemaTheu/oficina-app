// Package notification implementa a porta de notificação da aplicação
// (usecase.NotificacaoStatus) para comunicar o cliente sobre mudanças de
// status da sua Ordem de Serviço.
package notification

import (
	"context"
	"log/slog"
	"os"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
)

// Notifier é o contrato satisfeito pelas implementações deste pacote,
// compatível com a porta de notificação da camada de aplicação.
type Notifier interface {
	NotificarMudancaStatus(ctx context.Context, n usecase.NotificacaoStatus) error
}

// NovoNotifierDoAmbiente seleciona a implementação via variável NOTIFIER:
//   - "smtp": envia e-mail real via SMTP (ver SMTPNotifier); serve tanto para
//     provedores de produção quanto para MailHog/Mailpit em desenvolvimento.
//   - "log" (padrão): apenas registra a notificação no log da aplicação,
//     permitindo demonstrar o fluxo sem provedor externo.
func NovoNotifierDoAmbiente() Notifier {
	switch os.Getenv("NOTIFIER") {
	case "smtp":
		return NovoSMTPNotifier()
	default:
		return LogNotifier{}
	}
}

// LogNotifier registra a notificação no log — implementação para ambiente
// local e testes, sem dependência de provedor de e-mail.
type LogNotifier struct{}

// NotificarMudancaStatus loga a notificação que seria enviada ao cliente.
func (LogNotifier) NotificarMudancaStatus(_ context.Context, n usecase.NotificacaoStatus) error {
	slog.Info("notificação de mudança de status (modo log)",
		"destinatario", n.DestinatarioEmail,
		"cliente", n.DestinatarioNome,
		"os", n.NumeroOS,
		"novo_status", n.NovoStatus,
		"motivo", valorOuVazio(n.Motivo),
	)
	return nil
}

func valorOuVazio(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
