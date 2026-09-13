package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	"github.com/google/uuid"
)

// fakeEventRecorder captura os eventos de negócio emitidos pelo use case,
// satisfazendo a porta usecase.eventRecorder por composição estrutural.
type fakeEventRecorder struct {
	eventos []usecase.EventoOS
}

func (f *fakeEventRecorder) RegistrarEventoOS(_ context.Context, e usecase.EventoOS) {
	f.eventos = append(f.eventos, e)
}

func TestCriarOS_EmiteEventoDeSucesso(t *testing.T) {
	rec := &fakeEventRecorder{}
	clienteID := uuid.New()
	cli := &osCliRepo{cliente: &entity.Cliente{ID: clienteID}}
	veic := &osVeicRepo{veiculo: &entity.Veiculo{ID: uuid.New(), ClienteID: clienteID}}
	uc := usecase.NewOrdemServicoUseCase(
		&mockOsRepo{}, cli, veic, &osSvcRepo{}, &osPecaRepo{}, nil,
		usecase.ComEventRecorder(rec),
	)

	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(),
		VeiculoID: uuid.New().String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
	})
	if err != nil {
		t.Fatalf("CriarOS retornou erro inesperado: %v", err)
	}

	if len(rec.eventos) != 1 {
		t.Fatalf("esperava 1 evento, obteve %d", len(rec.eventos))
	}
	ev := rec.eventos[0]
	if ev.Resultado != usecase.ResultadoSucesso {
		t.Errorf("resultado = %q, esperava %q", ev.Resultado, usecase.ResultadoSucesso)
	}
	if ev.Status != string(entity.StatusRecebida) {
		t.Errorf("status = %q, esperava %q", ev.Status, entity.StatusRecebida)
	}
	if ev.OsID == "" {
		t.Error("os_id não deveria ser vazio no sucesso")
	}
}

func TestCriarOS_EmiteEventoDeFalhaNaPersistencia(t *testing.T) {
	rec := &fakeEventRecorder{}
	clienteID := uuid.New()
	cli := &osCliRepo{cliente: &entity.Cliente{ID: clienteID}}
	veic := &osVeicRepo{veiculo: &entity.Veiculo{ID: uuid.New(), ClienteID: clienteID}}
	osR := &mockOsRepo{
		criarFn: func(_ context.Context, _ *entity.OrdemServico, _ []entity.ItemOsServico, _ []entity.ItemOsPeca) (*entity.OrdemServico, error) {
			return nil, errors.New("falha no banco")
		},
	}
	uc := usecase.NewOrdemServicoUseCase(
		osR, cli, veic, &osSvcRepo{}, &osPecaRepo{}, nil,
		usecase.ComEventRecorder(rec),
	)

	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(),
		VeiculoID: uuid.New().String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
	})
	if err == nil {
		t.Fatal("esperava erro de persistência, obteve nil")
	}

	if len(rec.eventos) != 1 {
		t.Fatalf("esperava 1 evento, obteve %d", len(rec.eventos))
	}
	if rec.eventos[0].Resultado != usecase.ResultadoFalha {
		t.Errorf("resultado = %q, esperava %q", rec.eventos[0].Resultado, usecase.ResultadoFalha)
	}
	if rec.eventos[0].Motivo != "persistencia_os" {
		t.Errorf("motivo = %q, esperava %q", rec.eventos[0].Motivo, "persistencia_os")
	}
}

func TestCriarOS_ValidacaoNaoEmiteEvento(t *testing.T) {
	rec := &fakeEventRecorder{}
	uc := usecase.NewOrdemServicoUseCase(
		&mockOsRepo{}, &osCliRepo{}, &osVeicRepo{}, &osSvcRepo{}, &osPecaRepo{}, nil,
		usecase.ComEventRecorder(rec),
	)

	// OS sem itens: erro de validação, não é falha de processamento — não deve
	// emitir evento (senão o alerta dispararia para erro do cliente).
	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: uuid.New().String(),
		VeiculoID: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("esperava erro de validação, obteve nil")
	}
	if len(rec.eventos) != 0 {
		t.Fatalf("erro de validação não deveria emitir evento, emitiu %d", len(rec.eventos))
	}
}

func TestCriarOS_SemRecorderNaoPanica(t *testing.T) {
	// nil recorder é o padrão em produção local/testes: o use case deve operar
	// normalmente sem emitir eventos.
	clienteID := uuid.New()
	cli := &osCliRepo{cliente: &entity.Cliente{ID: clienteID}}
	veic := &osVeicRepo{veiculo: &entity.Veiculo{ID: uuid.New(), ClienteID: clienteID}}
	uc := usecase.NewOrdemServicoUseCase(&mockOsRepo{}, cli, veic, &osSvcRepo{}, &osPecaRepo{}, nil)

	if _, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(),
		VeiculoID: uuid.New().String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
	}); err != nil {
		t.Fatalf("CriarOS sem recorder retornou erro: %v", err)
	}
}
