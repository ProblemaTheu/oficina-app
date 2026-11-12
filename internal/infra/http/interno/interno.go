// Package interno expõe as rotas de serviço para serviço do OS Service, fora
// do OpenAPI público e fora do API Gateway (F4-1.5, ADR-015).
//
// Primeiro consumidor: a Lambda auth-token, que deixou de ler o banco do OS
// ("nenhum serviço acessa o banco de outro") e passou a consultar o cliente
// por aqui.
//
// "Fora do API Gateway" não é "privada": a Lambda roda fora da VPC e chama o
// balanceador público do OS, então /internal/* é alcançável pela internet e o
// token de serviço é a única proteção. Por isso:
//   - sem token configurado (INTERNAL_TOKEN ausente ou curto), TODA rota
//     responde 404 — a rota não existe até alguém ligá-la de propósito;
//   - header ausente ou errado também é 404, não 401: não revela que a rota
//     existe;
//   - a comparação é em tempo constante;
//   - a resposta traz só o que a autenticação precisa (id, nome, status) —
//     nada de e-mail, telefone ou documento.
package interno

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"github.com/ProblemaTheu/oficina-app/internal/domain/entity"
	domainerros "github.com/ProblemaTheu/oficina-app/internal/domain/erros"
)

// HeaderToken é o header que carrega o token de serviço.
const HeaderToken = "X-Internal-Token"

// tamanhoMinimoToken: o mesmo piso das chaves HMAC da aplicação.
const tamanhoMinimoToken = 32

var apenasOnzeDigitos = regexp.MustCompile(`^[0-9]{11}$`)

// BuscadorCliente é o recorte do caso de uso de clientes que as rotas usam.
type BuscadorCliente interface {
	FindByDocumento(documento string) (*entity.Cliente, error)
}

// ClienteResumo é a resposta de GET /internal/clientes/por-cpf/{cpf}.
type ClienteResumo struct {
	ID     string `json:"id"`
	Nome   string `json:"nome"`
	Status string `json:"status"`
}

// Rotas devolve o handler de /internal/*, para montar com r.Mount("/internal", ...).
func Rotas(token string, clientes BuscadorCliente) http.Handler {
	r := chi.NewRouter()
	r.NotFound(naoEncontrado)
	r.MethodNotAllowed(naoEncontrado)
	if len(token) < tamanhoMinimoToken {
		if token != "" {
			slog.Warn("rotas internas desligadas: INTERNAL_TOKEN com menos de 32 caracteres")
		}
		return r // sem rotas: tudo cai no NotFound
	}
	r.Use(exigirToken([]byte(token)))
	r.Get("/clientes/por-cpf/{cpf}", porCPF(clientes))
	return r
}

func exigirToken(esperado []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recebido := []byte(r.Header.Get(HeaderToken))
			if subtle.ConstantTimeCompare(recebido, esperado) != 1 {
				naoEncontrado(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func porCPF(clientes BuscadorCliente) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cpf := chi.URLParam(r, "cpf")
		// O chamador (a Lambda) já validou os dígitos verificadores; aqui só
		// se recusa o que nem tem formato de CPF, antes de tocar no banco.
		if !apenasOnzeDigitos.MatchString(cpf) {
			naoEncontrado(w, r)
			return
		}
		cliente, err := clientes.FindByDocumento(cpf)
		var nf *domainerros.ErrNaoEncontrado
		switch {
		case errors.As(err, &nf):
			naoEncontrado(w, r)
			return
		case err != nil:
			// O CPF nunca vai para o log (LGPD).
			slog.ErrorContext(r.Context(), "interno: falha ao buscar cliente por CPF", "error", err)
			escrever(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL_ERROR", "message": "erro ao consultar cliente"})
			return
		}
		escrever(w, http.StatusOK, ClienteResumo{ID: cliente.ID.String(), Nome: cliente.Nome, Status: cliente.Status})
	}
}

func naoEncontrado(w http.ResponseWriter, _ *http.Request) {
	escrever(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "recurso não encontrado"})
}

func escrever(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(corpo); err != nil {
		slog.Error("interno: falha ao escrever resposta", "error", err)
	}
}
