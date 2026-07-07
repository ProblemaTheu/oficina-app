package notification

import (
	"context"
	"strings"
	"testing"

	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
)

func notificacaoExemplo(motivo *string) usecase.NotificacaoStatus {
	return usecase.NotificacaoStatus{
		DestinatarioEmail: "cliente@teste.com",
		DestinatarioNome:  "Cliente Teste",
		NumeroOS:          "OS-2026-00001",
		NovoStatus:        entity.StatusEmExecucao,
		Motivo:            motivo,
	}
}

func TestLogNotifier_NaoRetornaErro(t *testing.T) {
	if err := (LogNotifier{}).NotificarMudancaStatus(context.Background(), notificacaoExemplo(nil)); err != nil {
		t.Errorf("LogNotifier nunca deve falhar; obteve: %v", err)
	}
}

func TestNovoNotifierDoAmbiente_PadraoLog(t *testing.T) {
	t.Setenv("NOTIFIER", "")
	if _, ok := NovoNotifierDoAmbiente().(LogNotifier); !ok {
		t.Error("sem NOTIFIER definido, esperava LogNotifier")
	}
}

func TestNovoNotifierDoAmbiente_SMTP(t *testing.T) {
	t.Setenv("NOTIFIER", "smtp")
	t.Setenv("SMTP_HOST", "mail.exemplo.com")
	t.Setenv("SMTP_PORT", "587")
	n, ok := NovoNotifierDoAmbiente().(*SMTPNotifier)
	if !ok {
		t.Fatal("com NOTIFIER=smtp, esperava *SMTPNotifier")
	}
	if n.host != "mail.exemplo.com" || n.port != "587" {
		t.Errorf("configuração SMTP não veio do ambiente: %s:%s", n.host, n.port)
	}
}

func TestMontarMensagem_ConteudoBasico(t *testing.T) {
	msg := string(montarMensagem("oficina@teste.com", notificacaoExemplo(nil)))

	for _, trecho := range []string{
		"From: oficina@teste.com",
		"To: cliente@teste.com",
		"Subject: OS OS-2026-00001: status atualizado para em_execucao",
		"Olá, Cliente Teste!",
		"charset=UTF-8",
	} {
		if !strings.Contains(msg, trecho) {
			t.Errorf("mensagem deveria conter %q", trecho)
		}
	}
	if strings.Contains(msg, "Observação:") {
		t.Error("sem motivo, a mensagem não deve ter linha de observação")
	}
}

func TestMontarMensagem_ComMotivo(t *testing.T) {
	motivo := "peça em falta no fornecedor"
	msg := string(montarMensagem("oficina@teste.com", notificacaoExemplo(&motivo)))
	if !strings.Contains(msg, "Observação: "+motivo) {
		t.Error("mensagem deveria conter a observação com o motivo")
	}
}
