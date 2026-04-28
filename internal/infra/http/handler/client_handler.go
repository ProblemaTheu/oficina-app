package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

// ClienteHandler gerencia as requisições HTTP relacionadas a clientes.
type ClienteHandler struct {
	criarCliente   *usecase.CriarClienteUseCase
	listarClientes *usecase.ListarClientesUseCase
}

// NovoClienteHandler cria uma nova instância de ClienteHandler com as dependências necessárias.
func NovoClienteHandler(db *sql.DB) *ClienteHandler {
	repo := repository.NovoClienteRepository(db)
	return &ClienteHandler{
		criarCliente:   usecase.NovoCriarClienteUseCase(repo),
		listarClientes: usecase.NovoListarClientesUseCase(repo),
	}
}

// CriarClienteRequest representa o corpo da requisição para criação de cliente.
type CriarClienteRequest struct {
	Nome      string  `json:"nome"`
	Documento string  `json:"documento"`
	Email     *string `json:"email"`
	Telefone  *string `json:"telefone"`
}

// CriarCliente processa a requisição de criação de um novo cliente.
func (h *ClienteHandler) CriarCliente(w http.ResponseWriter, r *http.Request) {
	slog.Info("requisição recebida: criar cliente")

	var input CriarClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.Warn("corpo da requisição inválido", "erro", err)
		http.Error(w, "corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	cliente, err := h.criarCliente.Executar(input.Nome, input.Documento, input.Email, input.Telefone)
	if err != nil {
		slog.Warn("erro ao criar cliente", "erro", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cliente)
	slog.Info("cliente criado com sucesso", "id", cliente.ID)
}

// ListarClientes processa a requisição de listagem de todos os clientes.
func (h *ClienteHandler) ListarClientes(w http.ResponseWriter, r *http.Request) {
	slog.Info("requisição recebida: listar clientes")

	clientes, err := h.listarClientes.Executar()
	if err != nil {
		slog.Error("erro ao listar clientes", "erro", err)
		http.Error(w, "erro ao listar clientes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clientes)
}
