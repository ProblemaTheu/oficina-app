package repository

import (
	"sync"

	"github.com/google/uuid"
	"github.com/problematheu/tech-challenge-1/internal/domain/entity"
)

type ClientRepository struct {
	mu      sync.Mutex
	storage map[string]*entity.Client
}

func NewClientRepository() *ClientRepository {
	return &ClientRepository{
		storage: make(map[string]*entity.Client),
	}
}

func (r *ClientRepository) Save(client *entity.Client) (*entity.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client.ID = uuid.New().String()
	r.storage[client.ID] = client

	return client, nil
}

func (r *ClientRepository) FindAll() ([]*entity.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	clients := []*entity.Client{}

	for _, c := range r.storage {
		clients = append(clients, c)
	}

	return clients, nil
}