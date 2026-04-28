package handler

import (
	"encoding/json"
	"net/http"

	"github.com/problematheu/tech-challenge-1/internal/application/usecase"
	"github.com/problematheu/tech-challenge-1/internal/infra/repository"
)

type ClientHandler struct {
	createClientUseCase *usecase.CreateClientUseCase
	listClientsUseCase  *usecase.ListClientsUseCase
	repo                *repository.ClientRepository
}

func NewClientHandler() *ClientHandler {
	repo := repository.NewClientRepository()

	return &ClientHandler{
		repo:                repo,
		createClientUseCase: usecase.NewCreateClientUseCase(repo),
		listClientsUseCase:  usecase.NewListClientsUseCase(repo),
	}
}

type ClientRequest struct {
	Name     string `json:"name"`
	Document string `json:"document"`
}

func (h *ClientHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var input ClientRequest

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	client, err := h.createClientUseCase.Execute(input.Name, input.Document)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(client)
}

func (h *ClientHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.listClientsUseCase.Execute()
	if err != nil {
		http.Error(w, "erro ao listar clientes", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(clients)
}
