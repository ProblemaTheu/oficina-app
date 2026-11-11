package saga

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNova_NasceEmDiagnosticoPendente(t *testing.T) {
	os := uuid.New()
	agora := time.Date(2026, 11, 11, 9, 0, 0, 0, time.FixedZone("BRT", -3*3600))

	s := Nova(os, agora)

	if s.ID == uuid.Nil || s.OsID != os {
		t.Errorf("identificadores errados: %+v", s)
	}
	if s.Estado != DiagnosticoPendente || s.EtapaAtual != EtapaDiagnostico || s.Versao != 1 {
		t.Errorf("estado inicial = %s/%s v%d; esperava DIAGNOSTICO_PENDENTE/T2 v1", s.Estado, s.EtapaAtual, s.Versao)
	}
	if !s.CriadoEm.Equal(agora) || s.CriadoEm.Location() != time.UTC {
		t.Errorf("criado_em deve ser o instante em UTC: %v", s.CriadoEm)
	}
}

func TestComandoIniciarDiagnostico(t *testing.T) {
	s := Nova(uuid.New(), time.Now())
	p := PayloadIniciarDiagnostico{NumeroOS: "OS-2026-00001", Veiculo: Veiculo{Placa: "ABC1D23", Modelo: "Gol"}}

	c := s.ComandoIniciarDiagnostico(p)

	if c.Destino != DestinoExecution || c.Tipo != TipoIniciarDiagnostico || c.SagaID != s.ID || c.OsID != s.OsID {
		t.Errorf("comando errado: %+v", c)
	}
	if c.Payload.(PayloadIniciarDiagnostico).Veiculo.Placa != "ABC1D23" {
		t.Error("payload não preservado")
	}
}

func TestEstado_Terminal(t *testing.T) {
	for e, terminal := range map[Estado]bool{
		Concluida: true, Compensada: true,
		DiagnosticoPendente: false, Compensando: false, EmExecucao: false,
	} {
		if e.Terminal() != terminal {
			t.Errorf("%s.Terminal() = %v", e, !terminal)
		}
	}
}
