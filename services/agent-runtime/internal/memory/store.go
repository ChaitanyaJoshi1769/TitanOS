package memory

import (
	"context"
	"fmt"
	"sync"
)

// Store interface for agent memory
type Store interface {
	Get(ctx context.Context, agentID, key string, dest interface{}) (interface{}, error)
	Set(ctx context.Context, agentID, key string, value interface{}) error
	Delete(ctx context.Context, agentID, key string) error
	GetAll(ctx context.Context, agentID string) (map[string]interface{}, error)
}

// InMemoryStore provides in-memory agent memory storage
type InMemoryStore struct {
	data map[string]map[string]interface{}
	mu   sync.RWMutex
}

// NewInMemoryStore creates a new in-memory store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string]map[string]interface{}),
	}
}

// Get retrieves a value from memory
func (s *InMemoryStore) Get(ctx context.Context, agentID, key string, dest interface{}) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentMem, exists := s.data[agentID]
	if !exists {
		return nil, fmt.Errorf("agent memory not found: %s", agentID)
	}

	value, exists := agentMem[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return value, nil
}

// Set stores a value in memory
func (s *InMemoryStore) Set(ctx context.Context, agentID, key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[agentID]; !exists {
		s.data[agentID] = make(map[string]interface{})
	}

	s.data[agentID][key] = value
	return nil
}

// Delete deletes a value from memory
func (s *InMemoryStore) Delete(ctx context.Context, agentID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if agentMem, exists := s.data[agentID]; exists {
		delete(agentMem, key)
	}

	return nil
}

// GetAll retrieves all memory for an agent
func (s *InMemoryStore) GetAll(ctx context.Context, agentID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentMem, exists := s.data[agentID]
	if !exists {
		return make(map[string]interface{}), nil
	}

	// Return a copy
	result := make(map[string]interface{})
	for k, v := range agentMem {
		result[k] = v
	}

	return result, nil
}
