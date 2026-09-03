package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ProblemaTheu/oficina-app/internal/application/usecase"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
	"github.com/ProblemaTheu/oficina-app/internal/domain/valueobject"
)

// escreverErro serializa o modelo Error padrão do contrato OpenAPI.
func escreverErro(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Error{Code: code, Message: message}); err != nil {
		slog.Error("falha ao escrever resposta de erro", "err", err)
	}
}

// TratarErroResposta mapeia erros devolvidos pelos handlers para o status HTTP
// correto, no formato do modelo Error do contrato. É registrado como
// ResponseErrorHandlerFunc do strict server, de modo que qualquer handler pode
// retornar o erro de domínio cru (`return nil, err`) e obter a resposta padronizada:
//
//   - ErrValidacao              → 400 VALIDATION_ERROR
//   - ErrNaoEncontrado          → 404 NOT_FOUND
//   - ErrProibido               → 403 (código da regra violada)
//   - ErrConflito               → 409 CONFLICT
//   - ErrNaoProcessavel         → 422 (código da regra violada)
//   - ErrEstoqueInsuficiente    → 422 INSUFFICIENT_STOCK
//   - ErrDocumentoInvalido      → 400 INVALID_DOCUMENT
//   - ErrPlacaInvalida          → 400 INVALID_PLATE
//   - demais                    → 500 INTERNAL_ERROR (com log de contexto)
func TratarErroResposta(w http.ResponseWriter, r *http.Request, err error) {
	var (
		errValidacao      *domainerros.ErrValidacao
		errNaoEncontrado  *domainerros.ErrNaoEncontrado
		errConflito       *domainerros.ErrConflito
		errNaoProcessavel *domainerros.ErrNaoProcessavel
		errProibido       *domainerros.ErrProibido
	)

	switch {
	case errors.As(err, &errValidacao):
		escreverErro(w, http.StatusBadRequest, "VALIDATION_ERROR", errValidacao.Mensagem)

	case errors.As(err, &errNaoEncontrado):
		escreverErro(w, http.StatusNotFound, "NOT_FOUND", err.Error())

	case errors.As(err, &errProibido):
		escreverErro(w, http.StatusForbidden, errProibido.Codigo, errProibido.Mensagem)

	case errors.As(err, &errConflito):
		escreverErro(w, http.StatusConflict, "CONFLICT", err.Error())

	case errors.As(err, &errNaoProcessavel):
		escreverErro(w, http.StatusUnprocessableEntity, errNaoProcessavel.Codigo, errNaoProcessavel.Mensagem)

	case errors.Is(err, usecase.ErrEstoqueInsuficiente):
		escreverErro(w, http.StatusUnprocessableEntity, "INSUFFICIENT_STOCK", err.Error())

	case errors.Is(err, valueobject.ErrDocumentoInvalido):
		escreverErro(w, http.StatusBadRequest, "INVALID_DOCUMENT", err.Error())

	case errors.Is(err, valueobject.ErrPlacaInvalida):
		escreverErro(w, http.StatusBadRequest, "INVALID_PLATE", err.Error())

	default:
		slog.Error("erro inesperado no handler",
			"method", r.Method, "path", r.URL.Path, "err", err)
		escreverErro(w, http.StatusInternalServerError, "INTERNAL_ERROR", "erro interno do servidor")
	}
}

// TratarErroRequisicao responde 400 no formato do modelo Error para falhas de
// parse/binding da requisição (registrado como RequestErrorHandlerFunc).
func TratarErroRequisicao(w http.ResponseWriter, _ *http.Request, err error) {
	escreverErro(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
}
