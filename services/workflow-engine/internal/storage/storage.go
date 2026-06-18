package storage

import (
	"context"
	"time"
)

// Database interface for workflow persistence
type Database interface {
	CreateWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
	CreateExecution(ctx context.Context, execution *Execution) error
	GetExecution(ctx context.Context, executionID string) (*Execution, error)
	UpdateExecution(ctx context.Context, execution *Execution) error
	GetExecutionEvents(ctx context.Context, executionID string) ([]*ExecutionEvent, error)
	StoreEvent(ctx context.Context, event *ExecutionEvent) error
	ListExecutions(ctx context.Context, workflowID string, limit, offset int) ([]*Execution, int, error)
}

// Workflow represents a workflow definition
type Workflow struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Version     int
	Activities  []*Activity
	Edges       []*Edge
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Activity represents a workflow activity/step
type Activity struct {
	ID            string
	Type          string
	Name          string
	Config        map[string]interface{}
	RetryCount    int
	RetryPolicy   string
	TimeoutSeconds int
}

// Edge represents a connection between activities
type Edge struct {
	FromActivity string
	ToActivity   string
	Condition    string
}

// Execution represents a workflow execution instance
type Execution struct {
	ID          string
	WorkflowID  string
	ProjectID   string
	Status      string
	InputData   []byte
	OutputData  []byte
	ErrorMessage string
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	UpdatedAt   time.Time
}

// ExecutionEvent represents an event during workflow execution
type ExecutionEvent struct {
	ExecutionID string
	EventType   string
	ActivityID  string
	Timestamp   time.Time
	Data        map[string]interface{}
	SequenceNum int
}

// InMemoryDB provides a simple in-memory implementation for development
type InMemoryDB struct {
	workflows map[string]*Workflow
	executions map[string]*Execution
	events     map[string][]*ExecutionEvent
}

// NewInMemoryDB creates a new in-memory database
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		workflows:  make(map[string]*Workflow),
		executions: make(map[string]*Execution),
		events:     make(map[string][]*ExecutionEvent),
	}
}

// CreateWorkflow creates a new workflow
func (db *InMemoryDB) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	db.workflows[workflow.ID] = workflow
	return nil
}

// GetWorkflow retrieves a workflow
func (db *InMemoryDB) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error) {
	workflow, exists := db.workflows[workflowID]
	if !exists {
		return nil, ErrNotFound
	}
	return workflow, nil
}

// CreateExecution creates a new execution
func (db *InMemoryDB) CreateExecution(ctx context.Context, execution *Execution) error {
	db.executions[execution.ID] = execution
	return nil
}

// GetExecution retrieves an execution
func (db *InMemoryDB) GetExecution(ctx context.Context, executionID string) (*Execution, error) {
	execution, exists := db.executions[executionID]
	if !exists {
		return nil, ErrNotFound
	}
	return execution, nil
}

// UpdateExecution updates an execution
func (db *InMemoryDB) UpdateExecution(ctx context.Context, execution *Execution) error {
	if _, exists := db.executions[execution.ID]; !exists {
		return ErrNotFound
	}
	db.executions[execution.ID] = execution
	return nil
}

// GetExecutionEvents retrieves events for an execution
func (db *InMemoryDB) GetExecutionEvents(ctx context.Context, executionID string) ([]*ExecutionEvent, error) {
	events, exists := db.events[executionID]
	if !exists {
		return make([]*ExecutionEvent, 0), nil
	}
	return events, nil
}

// StoreEvent stores an execution event
func (db *InMemoryDB) StoreEvent(ctx context.Context, event *ExecutionEvent) error {
	if _, exists := db.events[event.ExecutionID]; !exists {
		db.events[event.ExecutionID] = make([]*ExecutionEvent, 0)
	}
	db.events[event.ExecutionID] = append(db.events[event.ExecutionID], event)
	return nil
}

// ListExecutions lists executions
func (db *InMemoryDB) ListExecutions(ctx context.Context, workflowID string, limit, offset int) ([]*Execution, int, error) {
	var results []*Execution
	for _, exec := range db.executions {
		if exec.WorkflowID == workflowID {
			results = append(results, exec)
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

// Error definitions
var (
	ErrNotFound = &Error{Code: "NOT_FOUND", Message: "Resource not found"}
)

// Error represents an error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
