package saga

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
)

var agoraTeste = time.Date(2026, 11, 11, 16, 0, 0, 0, time.UTC)

// aplicar simula o caso de uso: persiste a saga quando a decisão manda.
func aplicar(t *testing.T, s Saga, tipo string, d Dados) (Saga, Decisao) {
	t.Helper()
	dec := Decidir(s, tipo, d, agoraTeste)
	if dec.MudouSaga {
		if dec.Saga.Versao != s.Versao+1 {
			t.Fatalf("%s: versão não incrementada (%d → %d)", tipo, s.Versao, dec.Saga.Versao)
		}
		return dec.Saga, dec
	}
	if dec.Saga.Versao != s.Versao || dec.Saga.Estado != s.Estado {
		t.Fatalf("%s: decisão sem mudança alterou a saga", tipo)
	}
	return s, dec
}

func tipos(cs []Comando) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Tipo)
	}
	return out
}

func TestCaminhoFeliz(t *testing.T) {
	s := Nova(uuid.New(), agoraTeste)
	orc, pag := uuid.New(), uuid.New()

	s, d := aplicar(t, s, TipoDiagnosticoIniciado, Dados{})
	if d.StatusOS != entity.StatusEmDiagnostico || d.MudouSaga {
		t.Fatalf("DiagnosticoIniciado é informativo: %+v", d)
	}

	passos := []struct {
		tipo     string
		dados    Dados
		estado   Estado
		statusOS entity.Status
		comandos []string
		etapa    Etapa
	}{
		{TipoDiagnosticoConcluido, Dados{Diagnostico: "troca do filtro"}, OrcamentoPendente, entity.StatusAguardandoAprovacao, []string{TipoGerarOrcamento}, EtapaDiagnostico},
		{TipoOrcamentoGerado, Dados{OrcamentoID: &orc}, AguardandoAprovacao, "", nil, EtapaOrcamento},
		{TipoOrcamentoAprovado, Dados{}, AguardandoPagamento, "", nil, EtapaAprovacao},
		{TipoPagamentoConfirmado, Dados{PagamentoID: &pag}, ExecucaoPendente, entity.StatusEmExecucao, []string{TipoExecutarReparo}, EtapaPagamento},
		{TipoExecucaoIniciada, Dados{}, EmExecucao, entity.StatusEmExecucao, nil, EtapaReparo},
		{TipoExecucaoConcluida, Dados{}, Concluida, entity.StatusFinalizada, nil, EtapaConclusao},
	}
	for _, p := range passos {
		var dec Decisao
		s, dec = aplicar(t, s, p.tipo, p.dados)
		if s.Estado != p.estado || dec.StatusOS != p.statusOS || !reflect.DeepEqual(tipos(dec.Comandos), p.comandos) {
			t.Fatalf("%s: estado=%s status=%q comandos=%v", p.tipo, s.Estado, dec.StatusOS, tipos(dec.Comandos))
		}
		if len(dec.Etapas) != 1 || dec.Etapas[0].Etapa != p.etapa || dec.Etapas[0].Resultado != ResultadoOK {
			t.Fatalf("%s: etapas = %+v", p.tipo, dec.Etapas)
		}
		switch p.tipo {
		case TipoDiagnosticoConcluido:
			if dec.Diagnostico != "troca do filtro" {
				t.Error("diagnóstico não repassado para gravar na OS")
			}
		case TipoExecucaoIniciada:
			if !dec.BaixarReserva || dec.Marco != MarcoIniciado {
				t.Error("início do reparo (pivô) deve baixar a reserva e marcar iniciado_em")
			}
		case TipoPagamentoConfirmado:
			if dec.Marco != MarcoAprovado || dec.Comandos[0].Destino != DestinoExecution {
				t.Errorf("pagamento: marco=%v destino=%s", dec.Marco, dec.Comandos[0].Destino)
			}
		}
	}
	if s.OrcamentoID == nil || *s.OrcamentoID != orc || s.PagamentoID == nil || *s.PagamentoID != pag {
		t.Errorf("ids de orçamento/pagamento não guardados: %+v", s)
	}
	if !s.Estado.Terminal() {
		t.Error("saga concluída é terminal")
	}
}

// avancarAte leva uma saga nova ao estado em que a falha acontece.
func avancarAte(t *testing.T, estado Estado) Saga {
	t.Helper()
	orc, pag := uuid.New(), uuid.New()
	s := Nova(uuid.New(), agoraTeste)
	for _, p := range []struct {
		tipo  string
		dados Dados
	}{
		{TipoDiagnosticoConcluido, Dados{Diagnostico: "x"}},
		{TipoOrcamentoGerado, Dados{OrcamentoID: &orc}},
		{TipoOrcamentoAprovado, Dados{}},
		{TipoPagamentoConfirmado, Dados{PagamentoID: &pag}},
	} {
		if s.Estado == estado {
			return s
		}
		s, _ = aplicar(t, s, p.tipo, p.dados)
	}
	if s.Estado != estado {
		t.Fatalf("não chegou em %s", estado)
	}
	return s
}

// Os 5 cenários da tabela de falhas (arquitetura §3.3), percorridos até COMPENSADA.
func TestCompensacoes_EmOrdemInversaAteCompensada(t *testing.T) {
	casos := []struct {
		nome       string
		estado     Estado
		falha      string
		etapaFalha Etapa
		sequencia  []string // comandos de compensação, em ordem
		destinos   []string
	}{
		{"T2 diagnóstico falhou", DiagnosticoPendente, TipoDiagnosticoFalhou, EtapaDiagnostico, nil, nil},
		{"T3 orçamento falhou", OrcamentoPendente, TipoOrcamentoFalhou, EtapaOrcamento, []string{TipoCancelarTarefas}, []string{DestinoExecution}},
		{"T4 cliente recusou", AguardandoAprovacao, TipoOrcamentoRecusado, EtapaAprovacao,
			[]string{TipoCancelarOrcamento, TipoCancelarTarefas}, []string{DestinoBilling, DestinoExecution}},
		{"T5 pagamento recusado", AguardandoPagamento, TipoPagamentoRecusado, EtapaPagamento,
			[]string{TipoCancelarOrcamento, TipoCancelarTarefas}, []string{DestinoBilling, DestinoExecution}},
		{"T6 falha no reparo (FAL*)", ExecucaoPendente, TipoExecucaoFalhou, EtapaReparo,
			[]string{TipoEstornarPagamento, TipoCancelarOrcamento, TipoCancelarTarefas}, []string{DestinoBilling, DestinoBilling, DestinoExecution}},
	}
	respostaDe := map[string]string{
		TipoEstornarPagamento: TipoPagamentoEstornado,
		TipoCancelarOrcamento: TipoOrcamentoCancelado,
		TipoCancelarTarefas:   TipoTarefasCanceladas,
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			s := avancarAte(t, c.estado)
			s, dec := aplicar(t, s, c.falha, Dados{Motivo: "detalhe"})

			if dec.StatusOS != entity.StatusCancelada || !dec.CancelarOS || dec.MotivoCancelamento == "" {
				t.Fatalf("OS deve ficar cancelada já no início da compensação: %+v", dec)
			}
			if s.EtapaFalha == nil || *s.EtapaFalha != c.etapaFalha || dec.Etapas[0].Resultado != ResultadoFalha {
				t.Fatalf("etapa da falha = %v, etapas = %+v", s.EtapaFalha, dec.Etapas)
			}

			var enviados, destinos []string
			for len(dec.Comandos) > 0 {
				if len(dec.Comandos) != 1 {
					t.Fatalf("uma compensação por vez: %v", tipos(dec.Comandos))
				}
				cmd := dec.Comandos[0]
				enviados, destinos = append(enviados, cmd.Tipo), append(destinos, cmd.Destino)
				if s.Estado != Compensando || dec.LiberarReserva {
					t.Fatalf("COMPENSADA antes da última confirmação (%s)", cmd.Tipo)
				}
				s, dec = aplicar(t, s, respostaDe[cmd.Tipo], Dados{})
				if dec.Etapas[0].Resultado != ResultadoCompensada {
					t.Fatalf("resposta de compensação não registrada como compensada: %+v", dec.Etapas)
				}
			}
			if !reflect.DeepEqual(enviados, c.sequencia) || !reflect.DeepEqual(destinos, c.destinos) {
				t.Fatalf("compensações = %v → %v; esperava %v → %v", enviados, destinos, c.sequencia, c.destinos)
			}
			if s.Estado != Compensada || !dec.LiberarReserva {
				t.Fatalf("fim: estado %s, liberar reserva %v", s.Estado, dec.LiberarReserva)
			}
			ultima := dec.Etapas[len(dec.Etapas)-1]
			if ultima.Etapa != EtapaAbertura || ultima.Resultado != ResultadoCompensada {
				t.Errorf("T1 (reserva) deve ser a última compensada: %+v", ultima)
			}
		})
	}
}

func TestCompensacao_LevaOrcamentoIDNoComando(t *testing.T) {
	s := avancarAte(t, ExecucaoPendente)
	_, dec := aplicar(t, s, TipoExecucaoFalhou, Dados{})
	p := dec.Comandos[0].Payload.(map[string]string)
	if p["orcamento_id"] != s.OrcamentoID.String() || p["motivo"] == "" {
		t.Errorf("EstornarPagamento precisa do orçamento para o Billing achar o pagamento: %+v", p)
	}
}

func TestCompensacao_RespostaForaDeOrdemEhIgnorada(t *testing.T) {
	s := avancarAte(t, ExecucaoPendente)
	s, _ = aplicar(t, s, TipoExecucaoFalhou, Dados{}) // em curso: estorno (T5)
	s2, dec := aplicar(t, s, TipoTarefasCanceladas, Dados{})
	if !dec.Ignorada || s2.EtapaAtual != EtapaPagamento || len(dec.Comandos) != 0 {
		t.Errorf("resposta que não é a da compensação em curso não pode avançar: %+v", dec)
	}
}

func TestPagamentoTardio_GeraEstornoSemMudarEstado(t *testing.T) {
	pag := uuid.New()
	// Cliente recusou (T4), saga compensou — e depois pagou no link.
	s := avancarAte(t, AguardandoAprovacao)
	s, _ = aplicar(t, s, TipoOrcamentoRecusado, Dados{})
	for _, r := range []string{TipoOrcamentoCancelado, TipoTarefasCanceladas} {
		s, _ = aplicar(t, s, r, Dados{})
	}
	if s.Estado != Compensada {
		t.Fatalf("pré-condição: %s", s.Estado)
	}

	s, dec := aplicar(t, s, TipoPagamentoConfirmado, Dados{PagamentoID: &pag})

	if s.Estado != Compensada || len(dec.Comandos) != 1 || dec.Comandos[0].Tipo != TipoEstornarPagamento {
		t.Fatalf("pagamento tardio deve gerar estorno e manter a saga: estado=%s comandos=%v", s.Estado, tipos(dec.Comandos))
	}
	if s.PagamentoID == nil || *s.PagamentoID != pag || dec.StatusOS != "" {
		t.Errorf("pagamento tardio: %+v / status %q", s, dec.StatusOS)
	}

	// Também durante a compensação.
	c := avancarAte(t, AguardandoAprovacao)
	c, _ = aplicar(t, c, TipoOrcamentoRecusado, Dados{})
	c, dec = aplicar(t, c, TipoPagamentoConfirmado, Dados{})
	if c.Estado != Compensando || tipos(dec.Comandos)[0] != TipoEstornarPagamento || c.EtapaAtual != EtapaOrcamento {
		t.Errorf("pagamento tardio no meio da compensação: %s %v %s", c.Estado, tipos(dec.Comandos), c.EtapaAtual)
	}
}

// Toda combinação estado × resposta: nada quebra, e o que não está nas tabelas
// é ignorado sem mudar a saga.
func TestTodasAsCombinacoes(t *testing.T) {
	estados := []Estado{DiagnosticoPendente, OrcamentoPendente, AguardandoAprovacao, AguardandoPagamento,
		ExecucaoPendente, EmExecucao, Concluida, Compensando, Compensada}
	respostas := []string{TipoDiagnosticoIniciado, TipoDiagnosticoConcluido, TipoDiagnosticoFalhou, TipoOrcamentoGerado,
		TipoOrcamentoFalhou, TipoOrcamentoAprovado, TipoOrcamentoRecusado, TipoPagamentoConfirmado, TipoPagamentoRecusado,
		TipoExecucaoIniciada, TipoExecucaoFalhou, TipoExecucaoConcluida, TipoPagamentoEstornado, TipoOrcamentoCancelado,
		TipoTarefasCanceladas, "TipoDesconhecido"}
	esperadas := 0
	for _, e := range estados {
		for _, r := range respostas {
			s := Nova(uuid.New(), agoraTeste)
			s.Estado = e
			if e == Compensando {
				f := EtapaReparo
				s.EtapaFalha, s.EtapaAtual = &f, EtapaPagamento
			}
			dec := Decidir(s, r, Dados{}, agoraTeste)

			_, noFluxo := fluxo[chave{e, r}]
			_, eFalha := falhas[chave{e, r}]
			informativo := e == DiagnosticoPendente && r == TipoDiagnosticoIniciado
			tardio := r == TipoPagamentoConfirmado && (e == Compensando || e == Compensada)
			compensacaoEsperada := e == Compensando && r == TipoPagamentoEstornado
			conhecida := noFluxo || eFalha || informativo || tardio || compensacaoEsperada
			if conhecida {
				esperadas++
			}
			if dec.Ignorada == conhecida {
				t.Errorf("%s × %s: ignorada=%v, esperado %v", e, r, dec.Ignorada, !conhecida)
			}
			if dec.Ignorada && (dec.MudouSaga || len(dec.Comandos) > 0 || dec.StatusOS != "") {
				t.Errorf("%s × %s: resposta ignorada não pode ter efeito: %+v", e, r, dec)
			}
		}
	}
	if esperadas != len(fluxo)+len(falhas)+1+2+1 {
		t.Errorf("combinações tratadas = %d; revise as tabelas", esperadas)
	}
}

func TestMotivoDaSaga_SemEtapaDeFalha(t *testing.T) {
	if motivoDaSaga(Saga{}) != "compensacao" {
		t.Error("sem etapa de falha o motivo é genérico")
	}
}
