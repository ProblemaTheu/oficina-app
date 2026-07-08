package notification

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
)

// fakeSMTPServer sobe um servidor SMTP mínimo em porta efêmera que aceita uma
// única sessão de envio e captura a mensagem recebida.
func fakeSMTPServer(t *testing.T) (addr string, recebido chan string) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao abrir listener: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	recebido = make(chan string, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }

		write("220 fake.local ESMTP")
		var corpo strings.Builder
		emData := false

		for {
			linha, err := r.ReadString('\n')
			if err != nil {
				return
			}
			linha = strings.TrimRight(linha, "\r\n")

			if emData {
				if linha == "." {
					emData = false
					recebido <- corpo.String()
					write("250 OK")
					continue
				}
				corpo.WriteString(linha + "\n")
				continue
			}

			cmd := strings.ToUpper(linha)
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				write("250-fake.local")
				write("250 8BITMIME")
			case strings.HasPrefix(cmd, "MAIL FROM"), strings.HasPrefix(cmd, "RCPT TO"):
				write("250 OK")
			case strings.HasPrefix(cmd, "DATA"):
				emData = true
				write("354 fim com <CRLF>.<CRLF>")
			case strings.HasPrefix(cmd, "QUIT"):
				write("221 tchau")
				return
			default:
				write("250 OK")
			}
		}
	}()

	return ln.Addr().String(), recebido
}

func TestSMTPNotifier_EnviaMensagem(t *testing.T) {
	addr, recebido := fakeSMTPServer(t)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("endereço inválido: %v", err)
	}

	notifier := &SMTPNotifier{host: host, port: port, from: "oficina@teste.com"}
	if err := notifier.NotificarMudancaStatus(context.Background(), notificacaoExemplo(nil)); err != nil {
		t.Fatalf("esperava envio bem-sucedido, obteve: %v", err)
	}

	msg := <-recebido
	for _, trecho := range []string{
		"To: cliente@teste.com",
		"Subject: OS OS-2026-00001",
		"Cliente Teste",
	} {
		if !strings.Contains(msg, trecho) {
			t.Errorf("mensagem entregue ao servidor deveria conter %q; obteve:\n%s", trecho, msg)
		}
	}
}

func TestSMTPNotifier_ErroDeConexao(t *testing.T) {
	// Porta reservada e fechada: conexão deve falhar imediatamente.
	notifier := &SMTPNotifier{host: "127.0.0.1", port: "1", from: "oficina@teste.com"}
	err := notifier.NotificarMudancaStatus(context.Background(), notificacaoExemplo(nil))
	if err == nil {
		t.Fatal("esperava erro de conexão com servidor inexistente")
	}
	if !strings.Contains(err.Error(), "SMTPNotifier:") {
		t.Errorf("erro deve ser embrulhado com contexto do notifier: %v", err)
	}
}

func TestNovoSMTPNotifier_DefaultsSemAmbiente(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("EMAIL_FROM", "")

	n := NovoSMTPNotifier()
	if n.host != "localhost" || n.port != "1025" || n.from != "oficina@example.com" {
		t.Errorf("defaults inesperados: host=%s port=%s from=%s", n.host, n.port, n.from)
	}
}

func TestValorOuVazio(t *testing.T) {
	if got := valorOuVazio(nil); got != "" {
		t.Errorf("nil deve virar string vazia, obteve %q", got)
	}
	s := "motivo"
	if got := valorOuVazio(&s); got != "motivo" {
		t.Errorf("esperava %q, obteve %q", s, got)
	}
}
