package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Database interface for agent persistence
type Database interface {
	CreateAgent(ctx context.Context, agent *Agent) error
	GetAgent(ctx context.Context, agentID string) (*Agent, error)
	UpdateAgent(ctx context.Context, agent *Agent) error
	DeleteAgent(ctx context.Context, agentID string) error
	ListAgents(ctx context.Context, projectID string, limit, offset int) ([]*Agent, int, error)
}

// Agent represents an agent
type Agent struct {
	ID           string
	ProjectID    string
	Name         string
	Status       string
	AgentType    string
	Memory       map[string]interface{}
	Config       map[string]interface{}
	Tools        []string
	Budget       float64
	RateLimit    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastHeartbeat time.Time
}

// InMemoryDatabase provides in-memory agent storage for development
type InMemoryDatabase struct {
	agents map[string]*Agent
	mu     sync.RWMutex
}

// NewInMemoryDatabase creates a new in-memory database
func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{
		agents: make(map[string]*Agent),
	}
}

// CreateAgent creates a new agent
func (db *InMemoryDatabase) CreateAgent(ctx context.Context, agent *Agent) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.agents[agent.ID]; exists {
		return fmt.Errorf("agent already exists: %s", agent.ID)
	}

	db.agents[agent.ID] = agent
	return nil
}

// GetAgent retrieves an agent
func (db *InMemoryDatabase) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	agent, exists := db.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// UpdateAgent updates an agent
func (db *InMemoryDatabase) UpdateAgent(ctx context.Context, agent *Agent) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.agents[agent.ID]; !exists {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	db.agents[agent.ID] = agent
	return nil
}

// DeleteAgent deletes an agent
func (db *InMemoryDatabase) DeleteAgent(ctx context.Context, agentID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.agents, agentID)
	return nil
}

// ListAgents lists agents
func (db *InMemoryDatabase) ListAgents(ctx context.Context, projectID string, limit, offset int) ([]*Agent, int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var results []*Agent
	for _, agent := range db.agents {
		if agent.ProjectID == projectID {
			results = append(results, agent)
		}
	}

	total := len(results)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}

	return results[offset:end], total, nil
}
