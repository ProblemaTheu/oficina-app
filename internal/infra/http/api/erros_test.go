package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/ProblemaTheu/oficina-app/internal/domain/valueobject"
)

func TestTratarErroResposta_MapeamentoCompleto(t *testing.T) {
	casos := []struct {
		nome        string
		err         error
		statusEsper int
		codigoEsper string
	}{
		{"validação", &domainerros.ErrValidacao{Mensagem: "nome é obrigatório"}, http.StatusBadRequest, "VALIDATION_ERROR"},
		{"não encontrado", &domainerros.ErrNaoEncontrado{Recurso: "cliente"}, http.StatusNotFound, "NOT_FOUND"},
		{"conflito", &domainerros.ErrConflito{Campo: "email"}, http.StatusConflict, "CONFLICT"},
		{"regra de negócio", &domainerros.ErrNaoProcessavel{Codigo: "INVALID_STATUS_TRANSITION", Mensagem: "transição inválida"}, http.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION"},
		{"estoque insuficiente", usecase.ErrEstoqueInsuficiente, http.StatusUnprocessableEntity, "INSUFFICIENT_STOCK"},
		{"documento inválido", valueobject.ErrDocumentoInvalido, http.StatusBadRequest, "INVALID_DOCUMENT"},
		{"placa inválida", valueobject.ErrPlacaInvalida, http.StatusBadRequest, "INVALID_PLATE"},
		{"inesperado", errors.New("falha de banco"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/teste", nil)

			TratarErroResposta(rec, req, c.err)

			if rec.Code != c.statusEsper {
				t.Errorf("esperava HTTP %d, obteve %d", c.statusEsper, rec.Code)
			}
			var body Error
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("resposta não é JSON válido: %v", err)
			}
			if body.Code != c.codigoEsper {
				t.Errorf("esperava code '%s', obteve '%s'", c.codigoEsper, body.Code)
			}
			if body.Message == "" {
				t.Error("message não deve ser vazia")
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("esperava Content-Type application/json, obteve %q", ct)
			}
		})
	}
}

func TestTratarErroResposta_ErroEmbrulhadoEhDesembrulhado(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/teste", nil)

	embrulhado := errors.Join(errors.New("contexto"), &domainerros.ErrNaoEncontrado{Recurso: "peça"})
	TratarErroResposta(rec, req, embrulhado)

	if rec.Code != http.StatusNotFound {
		t.Errorf("errors.As deve desembrulhar o erro de domínio; esperava 404, obteve %d", rec.Code)
	}
}

func TestTratarErroResposta_500NaoVazaDetalhes(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/teste", nil)

	TratarErroResposta(rec, req, errors.New("pq: connection refused em 10.0.0.5"))

	var body Error
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Message != "erro interno do servidor" {
		t.Errorf("500 não deve vazar detalhes internos; obteve %q", body.Message)
	}
}

func TestTratarErroRequisicao_Retorna400JSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/teste", nil)

	TratarErroRequisicao(rec, req, errors.New("json malformado"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", rec.Code)
	}
	var body Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta não é JSON válido: %v", err)
	}
	if body.Code != "BAD_REQUEST" {
		t.Errorf("esperava code BAD_REQUEST, obteve %q", body.Code)
	}
}
