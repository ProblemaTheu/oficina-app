package notification

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
)

// SMTPNotifier envia e-mails de notificação via SMTP. Compatível com
// provedores reais (autenticação PLAIN quando SMTP_USER é definido) e com
// MailHog/Mailpit locais (sem autenticação).
type SMTPNotifier struct {
	host string
	port string
	user string
	pass string
	from string
}

// NovoSMTPNotifier monta o notifier a partir das variáveis de ambiente
// SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD e EMAIL_FROM.
func NovoSMTPNotifier() *SMTPNotifier {
	return &SMTPNotifier{
		host: envOu("SMTP_HOST", "localhost"),
		port: envOu("SMTP_PORT", "1025"),
		user: os.Getenv("SMTP_USER"),
		pass: os.Getenv("SMTP_PASSWORD"),
		from: envOu("EMAIL_FROM", "oficina@example.com"),
	}
}

// NotificarMudancaStatus envia o e-mail ao cliente. Erros são devolvidos ao
// chamador (a aplicação decide logar sem falhar a transição).
func (s *SMTPNotifier) NotificarMudancaStatus(_ context.Context, n usecase.NotificacaoStatus) error {
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}

	msg := montarMensagem(s.from, n)
	addr := s.host + ":" + s.port

	if err := smtp.SendMail(addr, auth, s.from, []string{n.DestinatarioEmail}, msg); err != nil {
		return fmt.Errorf("SMTPNotifier: %w", err)
	}
	return nil
}

// montarMensagem constrói o e-mail (headers + corpo) em texto simples.
func montarMensagem(from string, n usecase.NotificacaoStatus) []byte {
	assunto := fmt.Sprintf("OS %s: status atualizado para %s", n.NumeroOS, n.NovoStatus)

	var corpo strings.Builder
	fmt.Fprintf(&corpo, "Olá, %s!\r\n\r\n", n.DestinatarioNome)
	fmt.Fprintf(&corpo, "A sua Ordem de Serviço %s mudou de status: %s.\r\n", n.NumeroOS, n.NovoStatus)
	if n.Motivo != nil && *n.Motivo != "" {
		fmt.Fprintf(&corpo, "Observação: %s\r\n", *n.Motivo)
	}
	corpo.WriteString("\r\nQualquer dúvida, entre em contato com a oficina.\r\n")

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n",
		from, n.DestinatarioEmail, assunto,
	)
	return []byte(headers + corpo.String())
}

func envOu(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
