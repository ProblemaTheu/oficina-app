package interno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
)

const tokenValido = "token-de-servico-com-mais-de-32-caracteres"

type buscadorFalso struct {
	cliente  *entity.Cliente
	err      error
	recebido string
	chamadas int
}

func (b *buscadorFalso) FindByDocumento(documento string) (*entity.Cliente, error) {
	b.chamadas++
	b.recebido = documento
	return b.cliente, b.err
}

func clienteAtivo() *entity.Cliente {
	email := "nao-deve-vazar@exemplo.com"
	return &entity.Cliente{ID: uuid.MustParse("11111111-2222-3333-4444-555555555555"), Nome: "Maria", CpfCnpj: "529.982.247-25", Status: "ativo", Email: &email}
}

func chamar(t *testing.T, h http.Handler, caminho, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, caminho, nil)
	if token != "" {
		req.Header.Set(HeaderToken, token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPorCPF_TokenCorreto_DevolveSoOResumo(t *testing.T) {
	b := &buscadorFalso{cliente: clienteAtivo()}
	rec := chamar(t, Rotas(tokenValido, b), "/clientes/por-cpf/52998224725", tokenValido)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quer 200", rec.Code)
	}
	if b.recebido != "52998224725" {
		t.Errorf("documento repassado = %q", b.recebido)
	}
	var corpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
		t.Fatal(err)
	}
	if corpo["id"] != "11111111-2222-3333-4444-555555555555" || corpo["nome"] != "Maria" || corpo["status"] != "ativo" {
		t.Errorf("corpo = %v", corpo)
	}
	if len(corpo) != 3 || strings.Contains(rec.Body.String(), "nao-deve-vazar") {
		t.Errorf("resposta expõe mais que id/nome/status: %s", rec.Body.String())
	}
}

func TestPorCPF_ClienteInativoPassaOStatus(t *testing.T) {
	c := clienteAtivo()
	c.Status = "bloqueado"
	rec := chamar(t, Rotas(tokenValido, &buscadorFalso{cliente: c}), "/clientes/por-cpf/52998224725", tokenValido)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"bloqueado"`) {
		t.Fatalf("status = %d corpo = %s", rec.Code, rec.Body.String())
	}
}

// Sem header, com token errado, sem token configurado ou com token curto: 404
// e o banco nem é consultado.
func TestPorCPF_SemAutorizacaoResponde404SemConsultar(t *testing.T) {
	casos := []struct {
		nome, configurado, enviado string
	}{
		{"sem header", tokenValido, ""},
		{"token errado", tokenValido, tokenValido + "x"},
		{"prefixo do token", tokenValido, tokenValido[:20]},
		{"rotas desligadas sem token", "", tokenValido},
		{"token configurado curto demais", "curto", "curto"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			b := &buscadorFalso{cliente: clienteAtivo()}
			rec := chamar(t, Rotas(c.configurado, b), "/clientes/por-cpf/52998224725", c.enviado)
			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, quer 404", rec.Code)
			}
			if b.chamadas != 0 {
				t.Errorf("consultou o banco %d vez(es) sem autorização", b.chamadas)
			}
		})
	}
}

func TestPorCPF_ClienteNaoEncontrado404(t *testing.T) {
	b := &buscadorFalso{err: &domainerros.ErrNaoEncontrado{Recurso: "cliente"}}
	rec := chamar(t, Rotas(tokenValido, b), "/clientes/por-cpf/52998224725", tokenValido)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404", rec.Code)
	}
}

func TestPorCPF_ErroDoBanco500(t *testing.T) {
	b := &buscadorFalso{err: errors.New("conexão recusada")}
	rec := chamar(t, Rotas(tokenValido, b), "/clientes/por-cpf/52998224725", tokenValido)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, quer 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "conexão recusada") {
		t.Error("detalhe interno vazou na resposta")
	}
}

func TestPorCPF_FormatoInvalidoNaoConsulta(t *testing.T) {
	for _, cpf := range []string{"529.982.247-25", "123", "5299822472a", "529982247250"} {
		b := &buscadorFalso{cliente: clienteAtivo()}
		rec := chamar(t, Rotas(tokenValido, b), "/clientes/por-cpf/"+cpf, tokenValido)
		if rec.Code != http.StatusNotFound || b.chamadas != 0 {
			t.Errorf("cpf %q: status = %d, chamadas = %d", cpf, rec.Code, b.chamadas)
		}
	}
}

func TestRotaInexistenteEMetodoErrado404(t *testing.T) {
	h := Rotas(tokenValido, &buscadorFalso{cliente: clienteAtivo()})
	if rec := chamar(t, h, "/outra", tokenValido); rec.Code != http.StatusNotFound {
		t.Errorf("rota inexistente: %d", rec.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/clientes/por-cpf/52998224725", nil)
	req.Header.Set(HeaderToken, tokenValido)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("POST: %d, quer 404", rec.Code)
	}
}
