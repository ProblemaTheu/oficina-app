//go:build bdd

// BDD da saga da OS (F4-5.2) contra os três serviços reais do
// docker-compose.saga.yml:
//
//	docker compose -f docker-compose.saga.yml --profile saga up -d --build --wait
//	go test -tags bdd ./features/ -v
//
// Os passos só falam HTTP (OS, Billing, Execution e o stub do Mercado Pago).
package features

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
)

func env(nome, padrao string) string {
	if v := os.Getenv(nome); v != "" {
		return v
	}
	return padrao
}

var (
	urlOS        = env("BDD_OS_URL", "http://localhost:8080")
	urlBilling   = env("BDD_BILLING_URL", "http://localhost:8081")
	urlExecution = env("BDD_EXECUTION_URL", "http://localhost:8082")
	urlMP        = env("BDD_MP_URL", "http://localhost:8090")
	prazo        = 20 * time.Second
	cliente      = &http.Client{Timeout: 10 * time.Second}
)

// mundo é o estado de um cenário.
type mundo struct {
	tokenAdmin, tokenMecanico string
	pecaID, clienteID         string
	veiculoID, osID           string
	orcamentoID, orderID      string
}

func chamar(metodo, url, token string, corpo any) (int, map[string]any, error) {
	var leitor io.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequest(metodo, url, leitor)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-Id", "bdd-"+fmt.Sprint(time.Now().UnixNano()))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := cliente.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	dados, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if len(dados) > 0 && dados[0] == '{' {
		_ = json.Unmarshal(dados, &m)
	} else if len(dados) > 0 && dados[0] == '[' {
		var lista []any
		_ = json.Unmarshal(dados, &lista)
		m = map[string]any{"itens": lista}
	}
	return resp.StatusCode, m, nil
}

// esperar repete cond até ela devolver nil ou o prazo acabar ("eventualmente").
func esperar(cond func() error) error {
	limite := time.Now().Add(prazo)
	for {
		err := cond()
		if err == nil || time.Now().After(limite) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func login(email, senha string) (string, error) {
	st, m, err := chamar("POST", urlOS+"/v1/auth/login", "", map[string]string{"email": email, "senha": senha})
	if err != nil || st != 200 {
		return "", fmt.Errorf("login %s: %d %v", email, st, err)
	}
	return m["access_token"].(string), nil
}

// cpfValido gera um CPF com dígitos verificadores corretos.
func cpfValido() string {
	d := make([]int, 11)
	for i := 0; i < 9; i++ {
		d[i] = rand.IntN(10)
	}
	for _, n := range []int{9, 10} {
		soma := 0
		for i := 0; i < n; i++ {
			soma += d[i] * (n + 1 - i)
		}
		if r := soma * 10 % 11; r < 10 {
			d[n] = r
		}
	}
	var b strings.Builder
	for _, x := range d {
		fmt.Fprint(&b, x)
	}
	return b.String()
}

// placa Mercosul aleatória (ABC1D23), com prefixo opcional.
func placa(prefixo string) string {
	letras := "ABCDEGHJKLMNPRSTUVWXYZ"
	p := prefixo
	for len(p) < 3 {
		p += string(letras[rand.IntN(len(letras))])
	}
	return fmt.Sprintf("%s%d%c%02d", p, rand.IntN(10), letras[rand.IntN(len(letras))], rand.IntN(100))
}

func (w *mundo) umaPeca(qtd int) error {
	codigo := fmt.Sprintf("BDD-%06d", rand.IntN(1000000))
	st, m, err := chamar("POST", urlOS+"/v1/parts", w.tokenAdmin, map[string]any{
		"nome": "Filtro BDD " + codigo, "codigo": codigo, "preco": 28.90, "estoque_minimo": 1, "estoque_atual": qtd,
	})
	if err != nil || st != 201 {
		return fmt.Errorf("criar peça: %d %v %v", st, m, err)
	}
	w.pecaID = m["id"].(string)
	return nil
}

func (w *mundo) clienteComVeiculo(prefixo string) error {
	st, m, err := chamar("POST", urlOS+"/v1/clients", w.tokenAdmin, map[string]any{
		"nome": "Cliente BDD", "documento": cpfValido(), "email": fmt.Sprintf("bdd-%d@exemplo.com", rand.IntN(1e9)),
	})
	if err != nil || st != 201 {
		return fmt.Errorf("criar cliente: %d %v %v", st, m, err)
	}
	w.clienteID = m["id"].(string)
	st, m, err = chamar("POST", urlOS+"/v1/vehicles", w.tokenAdmin, map[string]any{
		"cliente_id": w.clienteID, "placa": placa(prefixo), "marca": "VW", "modelo": "Gol", "ano": 2020,
	})
	if err != nil || st != 201 {
		return fmt.Errorf("criar veículo: %d %v %v", st, m, err)
	}
	w.veiculoID = m["id"].(string)
	return nil
}

func (w *mundo) abreOS(qtd int) error {
	st, m, err := chamar("POST", urlOS+"/v1/work-orders", w.tokenAdmin, map[string]any{
		"cliente_id": w.clienteID, "veiculo_id": w.veiculoID, "descricao": "barulho na suspensão",
		"pecas": []map[string]any{{"peca_id": w.pecaID, "quantidade": qtd}},
	})
	if err != nil || st != 201 {
		return fmt.Errorf("abrir OS: %d %v %v", st, m, err)
	}
	w.osID = m["id"].(string)
	return nil
}

func (w *mundo) statusOS() (string, error) {
	st, m, err := chamar("GET", urlOS+"/v1/work-orders/"+w.osID, w.tokenAdmin, nil)
	if err != nil || st != 200 {
		return "", fmt.Errorf("consultar OS: %d %v", st, err)
	}
	s, _ := m["status"].(string)
	return s, nil
}

func (w *mundo) osFica(esperado string) error {
	return esperar(func() error {
		s, err := w.statusOS()
		if err != nil {
			return err
		}
		if s != esperado {
			return fmt.Errorf("OS está %q, esperava %q", s, esperado)
		}
		return nil
	})
}

func (w *mundo) estoque() (atual, reservado int, err error) {
	st, m, err := chamar("GET", urlOS+"/v1/parts/"+w.pecaID, w.tokenAdmin, nil)
	if err != nil || st != 200 {
		return 0, 0, fmt.Errorf("consultar peça: %d %v", st, err)
	}
	return int(m["estoque_atual"].(float64)), int(m["estoque_reservado"].(float64)), nil
}

func (w *mundo) pecaTem(atual, reservado int) error {
	return esperar(func() error {
		a, r, err := w.estoque()
		if err != nil {
			return err
		}
		if a != atual || r != reservado {
			return fmt.Errorf("peça com atual=%d reservado=%d, esperava %d e %d", a, r, atual, reservado)
		}
		return nil
	})
}

func (w *mundo) osRecebidaComReserva(status string, reservado int) error {
	if s, err := w.statusOS(); err != nil || s != status {
		return fmt.Errorf("OS %q (%v), esperava %q", s, err, status)
	}
	_, r, err := w.estoque()
	if err != nil || r != reservado {
		return fmt.Errorf("reservado=%d (%v), esperava %d", r, err, reservado)
	}
	return nil
}

func (w *mundo) tarefaDaOS(tipo string) (string, error) {
	var id string
	err := esperar(func() error {
		_, m, err := chamar("GET", urlExecution+"/execution/v1/tarefas?os_id="+w.osID, w.tokenMecanico, nil)
		if err != nil {
			return err
		}
		for _, t := range m["itens"].([]any) {
			x := t.(map[string]any)
			if x["tipo"] == tipo && x["estado"] != "CANCELADA" {
				id = x["id"].(string)
				return nil
			}
		}
		return fmt.Errorf("nenhuma tarefa %s para a OS no Execution", tipo)
	})
	return id, err
}

func (w *mundo) concluiDiagnostico(diagnostico string) error {
	t, err := w.tarefaDaOS("DIAGNOSTICO")
	if err != nil {
		return err
	}
	st, m, err := chamar("POST", urlExecution+"/execution/v1/tarefas/"+t+"/concluir-diagnostico", w.tokenMecanico, map[string]string{"diagnostico": diagnostico})
	if err != nil || st != 200 {
		return fmt.Errorf("concluir diagnóstico: %d %v %v", st, m, err)
	}
	return nil
}

func (w *mundo) orcamentoGerado(status string) error {
	if err := w.osFica(status); err != nil {
		return err
	}
	return esperar(func() error {
		st, m, err := chamar("GET", urlBilling+"/billing/v1/orcamentos?os_id="+w.osID, w.tokenAdmin, nil)
		if err != nil || st != 200 {
			return fmt.Errorf("orçamento ainda não existe no Billing: %d %v", st, err)
		}
		w.orcamentoID = m["id"].(string)
		return nil
	})
}

func (w *mundo) aprova() error {
	st, m, err := chamar("POST", urlBilling+"/billing/v1/orcamentos/"+w.orcamentoID+"/aprovar", w.tokenAdmin, nil)
	if err != nil || st != 200 {
		return fmt.Errorf("aprovar: %d %v %v", st, m, err)
	}
	p := m["pagamento"].(map[string]any)
	if p["link_pagamento"] == "" {
		return fmt.Errorf("aprovação sem link de pagamento")
	}
	w.orderID = p["mp_order_id"].(string)
	return nil
}

func (w *mundo) recusa() error {
	st, m, err := chamar("POST", urlBilling+"/billing/v1/orcamentos/"+w.orcamentoID+"/recusar", w.tokenAdmin, map[string]string{"motivo": "caro demais"})
	if err != nil || st != 200 {
		return fmt.Errorf("recusar: %d %v %v", st, m, err)
	}
	return nil
}

func (w *mundo) mercadoPago(resultado string) error {
	st, m, err := chamar("POST", urlMP+"/_stub/pagar", "", map[string]string{"order_id": w.orderID, "resultado": resultado})
	if err != nil || st != 200 {
		return fmt.Errorf("pagar no stub (%s): %d %v %v", resultado, st, m, err)
	}
	return nil
}

func (w *mundo) mpAprova() error { return w.mercadoPago("processed") }
func (w *mundo) mpRecusa() error { return w.mercadoPago("failed") }

func (w *mundo) novaTentativaRecusada() error {
	st, _, err := chamar("POST", urlMP+"/_stub/pagar", "", map[string]string{"order_id": w.orderID, "resultado": "processed"})
	if err != nil {
		return err
	}
	if st != http.StatusConflict {
		return fmt.Errorf("nova tentativa no link aceita (%d): o cliente pagaria depois da compensação", st)
	}
	return nil
}

func (w *mundo) iniciaEConcluiReparo() error {
	t, err := w.tarefaDaOS("REPARO")
	if err != nil {
		return err
	}
	if st, m, err := chamar("POST", urlExecution+"/execution/v1/tarefas/"+t+"/iniciar", w.tokenMecanico, nil); err != nil || st != 200 {
		return fmt.Errorf("iniciar reparo: %d %v %v", st, m, err)
	}
	// O OS precisa receber ExecucaoIniciada (pivô) antes da conclusão.
	if err := w.pecaBaixada(); err != nil {
		return err
	}
	if st, m, err := chamar("POST", urlExecution+"/execution/v1/tarefas/"+t+"/concluir", w.tokenMecanico, nil); err != nil || st != 200 {
		return fmt.Errorf("concluir reparo: %d %v %v", st, m, err)
	}
	return nil
}

func (w *mundo) pecaBaixada() error {
	return esperar(func() error {
		_, r, err := w.estoque()
		if err != nil || r != 0 {
			return fmt.Errorf("reserva ainda não baixada (reservado=%d) %v", r, err)
		}
		return nil
	})
}

func (w *mundo) pagamento(esperado string) error {
	return esperar(func() error {
		st, m, err := chamar("GET", urlBilling+"/billing/v1/orcamentos/"+w.orcamentoID, w.tokenAdmin, nil)
		if err != nil || st != 200 {
			return fmt.Errorf("consultar orçamento: %d %v", st, err)
		}
		p, _ := m["pagamento"].(map[string]any)
		if p == nil || p["status"] != esperado {
			return fmt.Errorf("pagamento %v, esperava %q", p, esperado)
		}
		return nil
	})
}

func (w *mundo) estornadoNoMP() error {
	st, m, err := chamar("GET", urlMP+"/v1/orders/"+w.orderID, "", nil)
	if err != nil || st != 200 || m["status"] != "refunded" {
		return fmt.Errorf("order no Mercado Pago: %d %v %v", st, m["status"], err)
	}
	return nil
}

func iniciarCenario(sc *godog.ScenarioContext) {
	w := &mundo{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		var err error
		if w.tokenAdmin, err = login("admin@oficina.com", "Admin@123"); err != nil {
			return ctx, err
		}
		w.tokenMecanico, err = login("joao@oficina.com", "Mecanico@123")
		return ctx, err
	})
	sc.Step(`^uma peça com (\d+) unidades em estoque$`, w.umaPeca)
	sc.Step(`^um cliente ativo com um veículo$`, func() error { return w.clienteComVeiculo("") })
	sc.Step(`^um cliente ativo com um veículo de placa de falha simulada$`, func() error { return w.clienteComVeiculo("FAL") })
	sc.Step(`^o atendente abre uma OS com (\d+) unidades da peça$`, w.abreOS)
	sc.Step(`^a OS fica "([^"]*)" e (\d+) unidades da peça ficam reservadas$`, w.osRecebidaComReserva)
	sc.Step(`^o mecânico conclui o diagnóstico "([^"]*)"$`, w.concluiDiagnostico)
	sc.Step(`^eventualmente a OS fica "([^"]*)" com um orçamento gerado$`, w.orcamentoGerado)
	sc.Step(`^o cliente aprova o orçamento$`, w.aprova)
	sc.Step(`^o cliente recusa o orçamento$`, w.recusa)
	sc.Step(`^o Mercado Pago aprova o pagamento$`, w.mpAprova)
	sc.Step(`^o Mercado Pago recusa o pagamento$`, w.mpRecusa)
	sc.Step(`^o mecânico inicia e conclui o reparo$`, w.iniciaEConcluiReparo)
	sc.Step(`^eventualmente a OS fica "([^"]*)"$`, w.osFica)
	sc.Step(`^(?:eventualmente )?a peça tem (\d+) unidades e nenhuma reservada$`, func(n int) error { return w.pecaTem(n, 0) })
	sc.Step(`^(?:eventualmente )?o pagamento do orçamento está "([^"]*)"$`, w.pagamento)
	sc.Step(`^a cobrança consta como estornada no Mercado Pago$`, w.estornadoNoMP)
	sc.Step(`^uma nova tentativa de pagamento no mesmo link é recusada$`, w.novaTentativaRecusada)
}

func TestFeatures(t *testing.T) {
	formato := env("BDD_FORMATO", "pretty")
	suite := godog.TestSuite{
		ScenarioInitializer: iniciarCenario,
		Options: &godog.Options{Format: formato, Paths: []string{"."}, TestingT: t, Strict: true,
			Output: os.Stdout},
	}
	if suite.Run() != 0 {
		t.Fatal("cenários BDD falharam")
	}
}
