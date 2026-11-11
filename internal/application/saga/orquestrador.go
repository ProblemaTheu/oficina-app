package saga

import (
	"time"

	"github.com/google/uuid"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
)

// Dados são os campos das respostas que interessam ao orquestrador.
type Dados struct {
	OrcamentoID *uuid.UUID
	PagamentoID *uuid.UUID
	Diagnostico string
	Motivo      string
}

// Marco é o carimbo de tempo da OS que acompanha uma mudança de status.
type Marco int

const (
	SemMarco Marco = iota
	MarcoAprovado
	MarcoIniciado
	MarcoFinalizado
)

// RegistroEtapa é uma linha a gravar em sagas_os_etapas.
type RegistroEtapa struct {
	Etapa     Etapa
	Resultado Resultado
	Detalhe   map[string]string
}

// Decisao é o que fazer com uma resposta — o caso de uso aplica tudo numa
// transação só. Comandos sem Payload (GerarOrcamento, ExecutarReparo) são
// completados pelo caso de uso com os dados da OS.
type Decisao struct {
	Saga               Saga // estado novo; só persistir se MudouSaga
	MudouSaga          bool
	StatusOS           entity.Status // "" = não muda
	Marco              Marco
	Diagnostico        string
	CancelarOS         bool
	MotivoCancelamento string
	Etapas             []RegistroEtapa
	Comandos           []Comando
	BaixarReserva      bool
	LiberarReserva     bool
	Ignorada           bool
}

// passo é uma transição do caminho feliz.
type passo struct {
	para     Estado
	etapaOK  Etapa
	proxima  Etapa
	statusOS entity.Status
	marco    Marco
	comando  string // tipo do próximo comando, "" = nenhum (espera humana)
	baixar   bool
}

type chave struct {
	estado Estado
	tipo   string
}

// fluxo é o caminho feliz inteiro. Ler esta tabela é ler o processo
// (arquitetura §3.2–3.3).
var fluxo = map[chave]passo{
	{DiagnosticoPendente, TipoDiagnosticoConcluido}: {para: OrcamentoPendente, etapaOK: EtapaDiagnostico, proxima: EtapaOrcamento,
		statusOS: entity.StatusAguardandoAprovacao, comando: TipoGerarOrcamento},
	{OrcamentoPendente, TipoOrcamentoGerado}:     {para: AguardandoAprovacao, etapaOK: EtapaOrcamento, proxima: EtapaAprovacao},
	{AguardandoAprovacao, TipoOrcamentoAprovado}: {para: AguardandoPagamento, etapaOK: EtapaAprovacao, proxima: EtapaPagamento},
	{AguardandoPagamento, TipoPagamentoConfirmado}: {para: ExecucaoPendente, etapaOK: EtapaPagamento, proxima: EtapaReparo,
		statusOS: entity.StatusEmExecucao, marco: MarcoAprovado, comando: TipoExecutarReparo},
	// Pivô (T6): a reserva vira saída. Daqui em diante não se compensa.
	{ExecucaoPendente, TipoExecucaoIniciada}: {para: EmExecucao, etapaOK: EtapaReparo, proxima: EtapaConclusao,
		statusOS: entity.StatusEmExecucao, marco: MarcoIniciado, baixar: true},
	{EmExecucao, TipoExecucaoConcluida}: {para: Concluida, etapaOK: EtapaConclusao, proxima: EtapaConclusao,
		statusOS: entity.StatusFinalizada, marco: MarcoFinalizado},
}

// falhas leva cada resposta de falha à etapa que falhou e ao motivo gravado na OS.
var falhas = map[chave]struct {
	etapa  Etapa
	motivo string
}{
	{DiagnosticoPendente, TipoDiagnosticoFalhou}: {EtapaDiagnostico, "diagnostico_falhou"},
	{OrcamentoPendente, TipoOrcamentoFalhou}:     {EtapaOrcamento, "orcamento_falhou"},
	{AguardandoAprovacao, TipoOrcamentoRecusado}: {EtapaAprovacao, "orcamento_recusado"},
	{AguardandoPagamento, TipoPagamentoRecusado}: {EtapaPagamento, "pagamento_recusado"},
	{ExecucaoPendente, TipoExecucaoFalhou}:       {EtapaReparo, "execucao_falhou"},
}

// compensacoes é a pilha, em ordem inversa, das etapas já concluídas quando a
// etapa `falha` falhou. A liberação da reserva (T1) é local e fecha a pilha.
func compensacoes(falha Etapa) []Etapa {
	switch falha {
	case EtapaOrcamento:
		return []Etapa{EtapaDiagnostico}
	case EtapaAprovacao, EtapaPagamento:
		return []Etapa{EtapaOrcamento, EtapaDiagnostico}
	case EtapaReparo:
		return []Etapa{EtapaPagamento, EtapaOrcamento, EtapaDiagnostico}
	default: // T2: nada remoto a desfazer
		return nil
	}
}

// compensacao: etapa → comando que a desfaz, destino e resposta esperada.
var compensacao = map[Etapa]struct {
	comando, destino, resposta string
}{
	EtapaPagamento:   {TipoEstornarPagamento, DestinoBilling, TipoPagamentoEstornado},
	EtapaOrcamento:   {TipoCancelarOrcamento, DestinoBilling, TipoOrcamentoCancelado},
	EtapaDiagnostico: {TipoCancelarTarefas, DestinoExecution, TipoTarefasCanceladas},
}

// Decidir aplica uma resposta à saga. Função pura: não faz IO.
//
// A versão (lock otimista) só avança quando a saga muda de fato — e é
// incrementada aqui, num lugar só.
func Decidir(s Saga, tipo string, d Dados, agora time.Time) Decisao {
	dec := decidir(s, tipo, d)
	if !dec.MudouSaga {
		dec.Saga = s
		return dec
	}
	dec.Saga.Versao = s.Versao + 1
	dec.Saga.AtualizadoEm = agora.UTC()
	return dec
}

func decidir(s Saga, tipo string, d Dados) Decisao {
	dec := Decisao{Saga: s}

	switch {
	case tipo == TipoPagamentoConfirmado && (s.Estado == Compensando || s.Estado == Compensada):
		return estornoTardio(dec, s, d)
	case s.Estado == Compensando:
		return continuarCompensacao(dec, s, tipo)
	case s.Estado.Terminal():
		return ignorar(dec, s, tipo)
	case tipo == TipoDiagnosticoIniciado && s.Estado == DiagnosticoPendente:
		// Informativo: só a OS muda (em_diagnostico); a saga não.
		dec.StatusOS = entity.StatusEmDiagnostico
		return dec
	}

	if p, ok := fluxo[chave{s.Estado, tipo}]; ok {
		dec.MudouSaga = true
		dec.Saga.Estado, dec.Saga.EtapaAtual = p.para, p.proxima
		dec.StatusOS, dec.Marco, dec.BaixarReserva = p.statusOS, p.marco, p.baixar
		dec.Etapas = []RegistroEtapa{{Etapa: p.etapaOK, Resultado: ResultadoOK}}
		if tipo == TipoDiagnosticoConcluido {
			dec.Diagnostico = d.Diagnostico
		}
		if d.OrcamentoID != nil && tipo == TipoOrcamentoGerado {
			dec.Saga.OrcamentoID = d.OrcamentoID
		}
		if d.PagamentoID != nil && tipo == TipoPagamentoConfirmado {
			dec.Saga.PagamentoID = d.PagamentoID
		}
		if p.comando != "" {
			dec.Comandos = []Comando{comandoDoFluxo(p.comando, s)}
		}
		return dec
	}

	if f, ok := falhas[chave{s.Estado, tipo}]; ok {
		return iniciarCompensacao(dec, s, f.etapa, motivo(f.motivo, d.Motivo))
	}
	return ignorar(dec, s, tipo)
}

func motivo(base, detalhe string) string {
	if detalhe == "" {
		return base
	}
	return base + ": " + detalhe
}

func comandoDoFluxo(tipo string, s Saga) Comando {
	destino := DestinoBilling
	if tipo == TipoExecutarReparo {
		destino = DestinoExecution
	}
	return Comando{Destino: destino, Tipo: tipo, SagaID: s.ID, OsID: s.OsID}
}

func comandoDeCompensacao(e Etapa, s Saga, motivo string) Comando {
	c := compensacao[e]
	payload := map[string]string{"motivo": motivo}
	if s.OrcamentoID != nil {
		payload["orcamento_id"] = s.OrcamentoID.String()
	}
	return Comando{Destino: c.destino, Tipo: c.comando, SagaID: s.ID, OsID: s.OsID, Payload: payload}
}

func motivoDaSaga(s Saga) string {
	if s.EtapaFalha == nil {
		return "compensacao"
	}
	return "compensacao_" + string(*s.EtapaFalha)
}

// iniciarCompensacao: OS cancelada já no início (o cliente não deve ver
// "aguardando pagamento" enquanto o estorno corre) e a 1ª compensação sai.
func iniciarCompensacao(dec Decisao, s Saga, falha Etapa, motivo string) Decisao {
	dec.MudouSaga = true
	dec.Saga.Estado = Compensando
	dec.Saga.EtapaFalha = &falha
	dec.StatusOS, dec.CancelarOS, dec.MotivoCancelamento = entity.StatusCancelada, true, motivo
	dec.Etapas = []RegistroEtapa{{Etapa: falha, Resultado: ResultadoFalha, Detalhe: map[string]string{"motivo": motivo}}}
	pilha := compensacoes(falha)
	if len(pilha) == 0 {
		return concluirCompensacao(dec)
	}
	dec.Saga.EtapaAtual = pilha[0]
	dec.Comandos = []Comando{comandoDeCompensacao(pilha[0], dec.Saga, motivoDaSaga(dec.Saga))}
	return dec
}

// continuarCompensacao: uma compensação por vez. etapa_atual guarda a que
// está em curso; a resposta esperada a confirma e libera a próxima.
func continuarCompensacao(dec Decisao, s Saga, tipo string) Decisao {
	emCurso, ok := compensacao[s.EtapaAtual]
	if !ok || tipo != emCurso.resposta || s.EtapaFalha == nil {
		return ignorar(dec, s, tipo)
	}
	dec.MudouSaga = true
	dec.Etapas = []RegistroEtapa{{Etapa: s.EtapaAtual, Resultado: ResultadoCompensada}}
	pilha := compensacoes(*s.EtapaFalha)
	for i, e := range pilha {
		if e == s.EtapaAtual && i+1 < len(pilha) {
			dec.Saga.EtapaAtual = pilha[i+1]
			dec.Comandos = []Comando{comandoDeCompensacao(pilha[i+1], dec.Saga, motivoDaSaga(dec.Saga))}
			return dec
		}
	}
	return concluirCompensacao(dec)
}

// concluirCompensacao: a última compensação é local — liberar a reserva (T1).
// Só aqui a saga vira COMPENSADA (nunca antes de todas as confirmações).
func concluirCompensacao(dec Decisao) Decisao {
	dec.LiberarReserva = true
	dec.Saga.Estado, dec.Saga.EtapaAtual = Compensada, EtapaAbertura
	dec.Etapas = append(dec.Etapas, RegistroEtapa{Etapa: EtapaAbertura, Resultado: ResultadoCompensada})
	return dec
}

// estornoTardio: pagamento aprovado depois de a saga já compensar (o cliente
// pagou de novo no link). Defesa em profundidade com o Billing (F4-2.5): o
// dinheiro volta, a saga continua onde estava.
func estornoTardio(dec Decisao, s Saga, d Dados) Decisao {
	dec.MudouSaga = true
	if d.PagamentoID != nil {
		dec.Saga.PagamentoID = d.PagamentoID
	}
	dec.Etapas = []RegistroEtapa{{Etapa: EtapaPagamento, Resultado: ResultadoIgnorada,
		Detalhe: map[string]string{"tipo": TipoPagamentoConfirmado, "acao": "estorno_tardio"}}}
	dec.Comandos = []Comando{comandoDeCompensacao(EtapaPagamento, dec.Saga, "pagamento_apos_compensacao")}
	return dec
}

// ignorar: resposta inesperada para o estado atual (atrasada, fora de ordem ou
// de outra versão do fluxo). Registra e segue — nunca muda o estado.
func ignorar(dec Decisao, s Saga, tipo string) Decisao {
	dec.Ignorada = true
	dec.Etapas = []RegistroEtapa{{Etapa: s.EtapaAtual, Resultado: ResultadoIgnorada,
		Detalhe: map[string]string{"tipo": tipo, "estado": string(s.Estado)}}}
	return dec
}
