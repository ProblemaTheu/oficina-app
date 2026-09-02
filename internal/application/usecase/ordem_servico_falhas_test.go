package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	"github.com/google/uuid"
)

// Este arquivo cobre os ramos de erro dos casos de uso de OS que dependem de
// falhas do repositório (banco indisponível, registros ausentes etc.),
// reutilizando os mocks definidos em os_usecase_test.go.

var errBanco = errors.New("falha de banco")

// ── stubs com falha para dependências de CriarOS ──────────────────────────────

type cliRepoFalha struct{ osCliRepo }

func (r *cliRepoFalha) BuscarPorID(_ string) (*entity.Cliente, error) { return nil, errBanco }

type veicRepoFalha struct{ osVeicRepo }

func (r *veicRepoFalha) BuscarPorID(_ string) (*entity.Veiculo, error) { return nil, errBanco }

type svcRepoFalha struct{ osSvcRepo }

func (r *svcRepoFalha) BuscarPorID(_ string) (*entity.Servico, error) { return nil, errBanco }

// ── AvancarStatus — ramos de erro ─────────────────────────────────────────────

func TestOSAvancarStatus_OSNaoEncontrada(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return nil, errBanco
		},
	}
	_, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID: uuid.New().String(), NovoStatus: entity.StatusEmDiagnostico,
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro do repositório, obteve: %v", err)
	}
}

func TestOSAvancarStatus_ErroBuscarStatusID(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusRecebida), nil
		},
		buscarStatusIDFn: func(_ context.Context, _ entity.Status) (uuid.UUID, error) {
			return uuid.Nil, errBanco
		},
	}
	_, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID: uuid.New().String(), NovoStatus: entity.StatusEmDiagnostico,
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro ao resolver status, obteve: %v", err)
	}
}

func TestOSAvancarStatus_ErroDeduzirEstoque(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
		deduzirEstoqueErr: errBanco,
	}
	_, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID: uuid.New().String(), NovoStatus: entity.StatusEmExecucao,
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro na dedução de estoque, obteve: %v", err)
	}
}

func TestOSAvancarStatus_ErroAtualizarStatus(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusRecebida), nil
		},
		atualizarStatusErr: errBanco,
	}
	_, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID: uuid.New().String(), NovoStatus: entity.StatusEmDiagnostico,
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro na atualização, obteve: %v", err)
	}
}

func TestOSAvancarStatus_EmExecucaoParaFinalizada(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusEmExecucao), nil
		},
	}
	if _, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID: uuid.New().String(), NovoStatus: entity.StatusFinalizada,
	}); err != nil {
		t.Errorf("esperava sucesso na finalização, obteve: %v", err)
	}
}

func TestOSAvancarStatus_AguardandoAprovacaoComDiagnostico(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusEmDiagnostico), nil
		},
	}
	diagnostico := "troca da correia dentada"
	if _, err := osUseCase(osR).AvancarStatus(context.Background(), usecase.AvancarStatusInput{
		OsID: uuid.New().String(), NovoStatus: entity.StatusAguardandoAprovacao, Diagnostico: &diagnostico,
	}); err != nil {
		t.Errorf("esperava sucesso com diagnóstico informado, obteve: %v", err)
	}
}

// ── AprovarOrcamento — ramos de erro ──────────────────────────────────────────

func TestOSAprovarOrcamento_OSNaoEncontrada(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return nil, errBanco
		},
	}
	if _, err := osUseCase(osR).AprovarOrcamento(context.Background(), uuid.New().String()); !errors.Is(err, errBanco) {
		t.Errorf("esperava erro do repositório, obteve: %v", err)
	}
}

func TestOSAprovarOrcamento_ErroBuscarStatusID(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
		buscarStatusIDFn: func(_ context.Context, _ entity.Status) (uuid.UUID, error) {
			return uuid.Nil, errBanco
		},
	}
	if _, err := osUseCase(osR).AprovarOrcamento(context.Background(), uuid.New().String()); !errors.Is(err, errBanco) {
		t.Errorf("esperava erro ao resolver status, obteve: %v", err)
	}
}

func TestOSAprovarOrcamento_ErroDeduzirEstoque(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
		deduzirEstoqueErr: errBanco,
	}
	if _, err := osUseCase(osR).AprovarOrcamento(context.Background(), uuid.New().String()); !errors.Is(err, errBanco) {
		t.Errorf("esperava erro na dedução de estoque, obteve: %v", err)
	}
}

func TestOSAprovarOrcamento_ErroAtualizarStatus(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
		atualizarStatusErr: errBanco,
	}
	if _, err := osUseCase(osR).AprovarOrcamento(context.Background(), uuid.New().String()); !errors.Is(err, errBanco) {
		t.Errorf("esperava erro na atualização, obteve: %v", err)
	}
}

// ── RejeitarOrcamento — ramos de erro ─────────────────────────────────────────

func TestOSRejeitarOrcamento_OSNaoEncontrada(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return nil, errBanco
		},
	}
	if _, err := osUseCase(osR).RejeitarOrcamento(context.Background(), uuid.New().String(), nil); !errors.Is(err, errBanco) {
		t.Errorf("esperava erro do repositório, obteve: %v", err)
	}
}

func TestOSRejeitarOrcamento_ErroBuscarStatusID(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
		buscarStatusIDFn: func(_ context.Context, _ entity.Status) (uuid.UUID, error) {
			return uuid.Nil, errBanco
		},
	}
	if _, err := osUseCase(osR).RejeitarOrcamento(context.Background(), uuid.New().String(), nil); !errors.Is(err, errBanco) {
		t.Errorf("esperava erro ao resolver status, obteve: %v", err)
	}
}

func TestOSRejeitarOrcamento_ErroAtualizarStatus(t *testing.T) {
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
		atualizarStatusErr: errBanco,
	}
	if _, err := osUseCase(osR).RejeitarOrcamento(context.Background(), uuid.New().String(), nil); !errors.Is(err, errBanco) {
		t.Errorf("esperava erro na atualização, obteve: %v", err)
	}
}

// ── CriarOS — ramos de erro nas dependências ──────────────────────────────────

func TestOSCriarOS_ErroBuscarCliente(t *testing.T) {
	uc := usecase.NewOrdemServicoUseCase(&mockOsRepo{}, &cliRepoFalha{}, &osVeicRepo{}, &osSvcRepo{}, &osPecaRepo{}, nil)
	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: uuid.New().String(),
		VeiculoID: uuid.New().String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro ao buscar cliente, obteve: %v", err)
	}
}

func TestOSCriarOS_ErroBuscarVeiculo(t *testing.T) {
	uc := usecase.NewOrdemServicoUseCase(&mockOsRepo{}, &osCliRepo{}, &veicRepoFalha{}, &osSvcRepo{}, &osPecaRepo{}, nil)
	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: uuid.New().String(),
		VeiculoID: uuid.New().String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro ao buscar veículo, obteve: %v", err)
	}
}

func TestOSCriarOS_ErroBuscarServico(t *testing.T) {
	clienteID := uuid.New()
	cli := &osCliRepo{cliente: &entity.Cliente{ID: clienteID}}
	veic := &osVeicRepo{veiculo: &entity.Veiculo{ID: uuid.New(), ClienteID: clienteID}}
	uc := usecase.NewOrdemServicoUseCase(&mockOsRepo{}, cli, veic, &svcRepoFalha{}, &osPecaRepo{}, nil)
	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(),
		VeiculoID: uuid.New().String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 1}},
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro ao buscar serviço, obteve: %v", err)
	}
}

func TestOSCriarOS_ErroBuscarPeca(t *testing.T) {
	clienteID := uuid.New()
	cli := &osCliRepo{cliente: &entity.Cliente{ID: clienteID}}
	veic := &osVeicRepo{veiculo: &entity.Veiculo{ID: uuid.New(), ClienteID: clienteID}}
	uc := usecase.NewOrdemServicoUseCase(&mockOsRepo{}, cli, veic, &osSvcRepo{}, &osPecaRepo{buscarErr: errBanco}, nil)
	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(),
		VeiculoID: uuid.New().String(),
		Pecas:     []usecase.ItemPecaOSInput{{PecaID: uuid.New().String(), Quantidade: 1}},
	})
	if !errors.Is(err, errBanco) {
		t.Errorf("esperava erro ao buscar peça, obteve: %v", err)
	}
}

func TestOSCriarOS_QuantidadeZeroViraUm(t *testing.T) {
	clienteID := uuid.New()
	cli := &osCliRepo{cliente: &entity.Cliente{ID: clienteID}}
	veic := &osVeicRepo{veiculo: &entity.Veiculo{ID: uuid.New(), ClienteID: clienteID}}
	svc := &osSvcRepo{servico: &entity.Servico{ID: uuid.New(), PrecoBase: 100.0}}
	peca := &osPecaRepo{peca: &entity.Peca{ID: uuid.New(), Preco: 50.0, EstoqueAtual: 10}}

	var valorPersistido float64
	osR := &mockOsRepo{
		criarFn: func(_ context.Context, os *entity.OrdemServico, _ []entity.ItemOsServico, _ []entity.ItemOsPeca) (*entity.OrdemServico, error) {
			os.ID = uuid.New()
			valorPersistido = os.ValorTotal
			return os, nil
		},
	}

	uc := usecase.NewOrdemServicoUseCase(osR, cli, veic, svc, peca, nil)
	_, err := uc.CriarOS(context.Background(), usecase.CriarOSInput{
		ClienteID: clienteID.String(),
		VeiculoID: uuid.New().String(),
		Servicos:  []usecase.ItemServicoOSInput{{ServicoID: uuid.New().String(), Quantidade: 0}},
		Pecas:     []usecase.ItemPecaOSInput{{PecaID: uuid.New().String(), Quantidade: -1}},
	})
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}
	// quantidade <= 0 vira 1: 1*100 (serviço) + 1*50 (peça) = 150
	if valorPersistido != 150.0 {
		t.Errorf("esperava valor total 150.0 com quantidades normalizadas, obteve %.2f", valorPersistido)
	}
}

// ── notificarMudancaStatus — falha ao buscar cliente ──────────────────────────

func TestOSNotificacao_ErroAoBuscarClienteNaoEnvia(t *testing.T) {
	notif := &mockNotifier{ch: make(chan usecase.NotificacaoStatus, 1)}
	osR := &mockOsRepo{
		buscarPorIDFn: func(_ context.Context, _ string) (*entity.OrdemServicoCompleta, error) {
			return osCompleta(entity.StatusAguardandoAprovacao), nil
		},
	}
	uc := usecase.NewOrdemServicoUseCase(osR, &cliRepoFalha{}, &osVeicRepo{}, &osSvcRepo{}, &osPecaRepo{}, notif)
	if _, err := uc.AprovarOrcamento(context.Background(), uuid.New().String()); err != nil {
		t.Fatalf("falha na notificação não deve falhar a transição; obteve: %v", err)
	}
	garantirSemNotificacao(t, notif.ch)
}
