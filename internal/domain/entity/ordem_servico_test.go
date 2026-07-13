package entity

import "testing"

// TestCanTransitionTo_MatrizCompleta valida a máquina de estados da OS de forma
// exaustiva: para cada par (origem, destino), confere se a transição é permitida
// exatamente quando prevista no fluxo de negócio.
func TestCanTransitionTo_MatrizCompleta(t *testing.T) {
	todos := []Status{
		StatusRecebida,
		StatusEmDiagnostico,
		StatusAguardandoAprovacao,
		StatusEmExecucao,
		StatusFinalizada,
		StatusEntregue,
	}

	permitidas := map[Status]map[Status]bool{
		StatusRecebida:            {StatusEmDiagnostico: true},
		StatusEmDiagnostico:       {StatusAguardandoAprovacao: true},
		StatusAguardandoAprovacao: {StatusEmExecucao: true, StatusFinalizada: true},
		StatusEmExecucao:          {StatusFinalizada: true},
		StatusFinalizada:          {StatusEntregue: true},
		StatusEntregue:            {},
	}

	for _, de := range todos {
		for _, para := range todos {
			esperado := permitidas[de][para]
			if obtido := de.CanTransitionTo(para); obtido != esperado {
				t.Errorf("transição '%s' → '%s': esperava %v, obteve %v", de, para, esperado, obtido)
			}
		}
	}
}

// TestCanTransitionTo_StatusDesconhecido garante que um status fora da máquina
// de estados não permite nenhuma transição.
func TestCanTransitionTo_StatusDesconhecido(t *testing.T) {
	desconhecido := Status("cancelada")
	if desconhecido.CanTransitionTo(StatusRecebida) {
		t.Error("status desconhecido não deve permitir transição")
	}
	if StatusRecebida.CanTransitionTo(desconhecido) {
		t.Error("transição para status desconhecido não deve ser permitida")
	}
}
