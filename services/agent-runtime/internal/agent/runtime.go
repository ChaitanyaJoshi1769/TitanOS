package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/agent-runtime/internal/memory"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/agent-runtime/internal/storage"
)

// AgentRuntime manages agent lifecycle and execution
type AgentRuntime struct {
	agents      map[string]*AgentState
	agentsMu    sync.RWMutex
	toolRegistry *ToolRegistry
	memoryStore  memory.Store
	stateStore   storage.Database
}

// AgentState represents the state of an agent
type AgentState struct {
	ID           string
	ProjectID    string
	Name         string
	Status       string // offline, online, sleeping, executing
	AgentType    string
	Memory       map[string]interface{}
	Config       map[string]interface{}
	Tools        []string
	Budget       float64 // Cost budget
	RateLimit    int     // Requests per minute
	ExecutionLog []ExecutionEntry
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastHeartbeat time.Time
}

// ExecutionEntry records an agent execution
type ExecutionEntry struct {
	Timestamp time.Time
	ToolName  string
	Input     map[string]interface{}
	Output    map[string]interface{}
	Error     string
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]*ToolDefinition
	mu    sync.RWMutex
}

// ToolDefinition defines a tool
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	OutputSchema map[string]interface{}
	Timeout     time.Duration
	Handler     func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// NewAgentRuntime creates a new agent runtime
func NewAgentRuntime(stateStore storage.Database, memoryStore memory.Store) *AgentRuntime {
	return &AgentRuntime{
		agents:        make(map[string]*AgentState),
		toolRegistry:  &ToolRegistry{tools: make(map[string]*ToolDefinition)},
		memoryStore:   memoryStore,
		stateStore:    stateStore,
	}
}

// CreateAgent creates a new agent
func (ar *AgentRuntime) CreateAgent(ctx context.Context, projectID, name, agentType string, config map[string]interface{}) (string, error) {
	agentID := generateAgentID()
	now := time.Now()

	agentState := &AgentState{
		ID:        agentID,
		ProjectID: projectID,
		Name:      name,
		AgentType: agentType,
		Status:    "online",
		Memory:    make(map[string]interface{}),
		Config:    config,
		Tools:     make([]string, 0),
		Budget:    1000.0, // Default budget
		RateLimit: 60,     // Default: 60 requests/minute
		CreatedAt: now,
		UpdatedAt: now,
		LastHeartbeat: now,
	}

	ar.agentsMu.Lock()
	ar.agents[agentID] = agentState
	ar.agentsMu.Unlock()

	// Persist to storage
	if err := ar.stateStore.CreateAgent(ctx, agentState); err != nil {
		return "", fmt.Errorf("failed to persist agent: %w", err)
	}

	// Initialize memory
	if err := ar.memoryStore.Set(ctx, agentID, "state", agentState); err != nil {
		return "", fmt.Errorf("failed to initialize memory: %w", err)
	}

	log.Printf("Agent created: %s (%s)", agentID, name)
	return agentID, nil
}

// GetAgent retrieves an agent
func (ar *AgentRuntime) GetAgent(ctx context.Context, agentID string) (*AgentState, error) {
	ar.agentsMu.RLock()
	agentState, exists := ar.agents[agentID]
	ar.agentsMu.RUnlock()

	if !exists {
		// Try to load from storage
		state, err := ar.stateStore.GetAgent(ctx, agentID)
		if err != nil {
			return nil, fmt.Errorf("agent not found: %s", agentID)
		}
		ar.agentsMu.Lock()
		ar.agents[agentID] = state
		ar.agentsMu.Unlock()
		return state, nil
	}

	return agentState, nil
}

// ExecuteTool executes a tool for an agent
func (ar *AgentRuntime) ExecuteTool(ctx context.Context, agentID, toolName string, input map[string]interface{}) (map[string]interface{}, error) {
	agent, err := ar.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	if agent.Status != "online" && agent.Status != "executing" {
		return nil, fmt.Errorf("agent not available (status: %s)", agent.Status)
	}

	// Get tool definition
	ar.toolRegistry.mu.RLock()
	toolDef, exists := ar.toolRegistry.tools[toolName]
	ar.toolRegistry.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, toolDef.Timeout)
	defer cancel()

	start := time.Now()
	output, err := toolDef.Handler(ctx, input)
	elapsed := time.Since(start)

	// Record execution
	entry := ExecutionEntry{
		Timestamp: start,
		ToolName:  toolName,
		Input:     input,
		Output:    output,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	ar.agentsMu.Lock()
	agent.ExecutionLog = append(agent.ExecutionLog, entry)
	ar.agentsMu.Unlock()

	// Update budget
	cost := elapsed.Seconds() * 0.1 // Example: $0.1 per second
	agent.Budget -= cost

	if agent.Budget < 0 {
		return nil, fmt.Errorf("agent budget exceeded")
	}

	log.Printf("Agent %s executed tool %s in %v", agentID, toolName, elapsed)
	return output, err
}

// SleepAgent puts an agent to sleep (conserve resources)
func (ar *AgentRuntime) SleepAgent(ctx context.Context, agentID string) error {
	agent, err := ar.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}

	ar.agentsMu.Lock()
	agent.Status = "sleeping"
	agent.UpdatedAt = time.Now()
	ar.agentsMu.Unlock()

	// Persist state
	return ar.stateStore.UpdateAgent(ctx, agent)
}

// WakeAgent wakes up an agent
func (ar *AgentRuntime) WakeAgent(ctx context.Context, agentID string) error {
	agent, err := ar.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}

	ar.agentsMu.Lock()
	agent.Status = "online"
	agent.UpdatedAt = time.Now()
	agent.LastHeartbeat = time.Now()
	ar.agentsMu.Unlock()

	// Restore memory
	if err := ar.memoryStore.Get(ctx, agentID, "state", agent); err != nil {
		log.Printf("Warning: failed to restore memory for agent %s: %v", agentID, err)
	}

	// Persist state
	return ar.stateStore.UpdateAgent(ctx, agent)
}

// GetAgentMemory retrieves agent memory
func (ar *AgentRuntime) GetAgentMemory(ctx context.Context, agentID, key string) (interface{}, error) {
	return ar.memoryStore.Get(ctx, agentID, key, nil)
}

// SetAgentMemory sets agent memory
func (ar *AgentRuntime) SetAgentMemory(ctx context.Context, agentID, key string, value interface{}) error {
	return ar.memoryStore.Set(ctx, agentID, key, value)
}

// RegisterTool registers a tool
func (ar *AgentRuntime) RegisterTool(toolDef *ToolDefinition) error {
	if toolDef.Timeout == 0 {
		toolDef.Timeout = 30 * time.Second // Default timeout
	}

	ar.toolRegistry.mu.Lock()
	ar.toolRegistry.tools[toolDef.Name] = toolDef
	ar.toolRegistry.mu.Unlock()

	log.Printf("Tool registered: %s", toolDef.Name)
	return nil
}

// ListTools lists available tools
func (ar *AgentRuntime) ListTools() []string {
	ar.toolRegistry.mu.RLock()
	defer ar.toolRegistry.mu.RUnlock()

	tools := make([]string, 0, len(ar.toolRegistry.tools))
	for name := range ar.toolRegistry.tools {
		tools = append(tools, name)
	}
	return tools
}

// Heartbeat processes agent heartbeat
func (ar *AgentRuntime) Heartbeat(ctx context.Context, agentID string) error {
	agent, err := ar.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}

	ar.agentsMu.Lock()
	agent.LastHeartbeat = time.Now()
	ar.agentsMu.Unlock()

	return nil
}

// TerminateAgent terminates an agent
func (ar *AgentRuntime) TerminateAgent(ctx context.Context, agentID string) error {
	ar.agentsMu.Lock()
	delete(ar.agents, agentID)
	ar.agentsMu.Unlock()

	// Persist termination
	return ar.stateStore.DeleteAgent(ctx, agentID)
}

// Helper function
func generateAgentID() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}
