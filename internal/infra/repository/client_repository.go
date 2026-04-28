package repository

import (
	"sync"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
)

type ClientRepository struct {
	mu      sync.Mutex
	storage map[uuid.UUID]*entity.Cliente
}

func NewClientRepository() *ClientRepository {
	return &ClientRepository{
		storage: make(map[uuid.UUID]*entity.Cliente),
	}
}

func (r *ClientRepository) Save(client *entity.Cliente) (*entity.Cliente, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client.ID = uuid.New()
	r.storage[client.ID] = client

	return client, nil
}

func (r *ClientRepository) FindAll() ([]*entity.Cliente, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	clients := []*entity.Cliente{}

	for _, c := range r.storage {
		clients = append(clients, c)
	}

	return clients, nil
}