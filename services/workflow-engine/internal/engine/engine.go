package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/workflow-engine/internal/storage"
	pb "github.com/ChaitanyaJoshi1769/TitanOS/services/workflow/pb"
)

// WorkflowEngine orchestrates workflow execution
type WorkflowEngine struct {
	db            storage.Database
	executions    map[string]*ExecutionState
	executionsMu  sync.RWMutex
	workerPool    *WorkerPool
	eventStore    *EventStore
}

// ExecutionState tracks the state of a workflow execution
type ExecutionState struct {
	ExecutionID      string
	WorkflowID       string
	Status           string
	InputData        []byte
	OutputData       []byte
	ErrorMessage     string
	CurrentActivity  string
	CompletedActivities map[string]*ActivityExecutionState
	CreatedAt        time.Time
	StartedAt        time.Time
	CompletedAt      time.Time
}

// ActivityExecutionState tracks the state of an activity execution
type ActivityExecutionState struct {
	ActivityID    string
	Status        string
	InputData     []byte
	OutputData    []byte
	ErrorMessage  string
	RetryCount    int
	StartedAt     time.Time
	CompletedAt   time.Time
}

// EventStore implements event sourcing for workflow execution
type EventStore struct {
	events map[string][]*WorkflowEvent
	mu     sync.RWMutex
}

// WorkflowEvent represents an event in workflow execution
type WorkflowEvent struct {
	ExecutionID   string
	EventType     string // started, activity_started, activity_completed, activity_failed, completed, failed
	ActivityID    string
	Timestamp     time.Time
	Data          map[string]interface{}
	SequenceNum   int
}

// WorkerPool manages parallel activity execution
type WorkerPool struct {
	workers    int
	taskQueue  chan *ActivityTask
	stopChan   chan struct{}
}

// ActivityTask represents a task to execute an activity
type ActivityTask struct {
	ExecutionID string
	ActivityID  string
	InputData   []byte
}

// NewWorkflowEngine creates a new workflow engine instance
func NewWorkflowEngine(db storage.Database, numWorkers int) *WorkflowEngine {
	return &WorkflowEngine{
		db:          db,
		executions:  make(map[string]*ExecutionState),
		workerPool:  NewWorkerPool(numWorkers),
		eventStore:  &EventStore{events: make(map[string][]*WorkflowEvent)},
	}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(numWorkers int) *WorkerPool {
	return &WorkerPool{
		workers:   numWorkers,
		taskQueue: make(chan *ActivityTask, 100),
		stopChan:  make(chan struct{}),
	}
}

// Start starts the workflow engine
func (we *WorkflowEngine) Start(ctx context.Context) {
	for i := 0; i < we.workerPool.workers; i++ {
		go we.workerLoop(ctx)
	}
	log.Println("✓ Workflow engine started with", we.workerPool.workers, "workers")
}

// workerLoop processes activity tasks
func (we *WorkflowEngine) workerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-we.workerPool.stopChan:
			return
		case task := <-we.workerPool.taskQueue:
			we.executeActivity(ctx, task)
		}
	}
}

// ExecuteWorkflow starts a workflow execution
func (we *WorkflowEngine) ExecuteWorkflow(ctx context.Context, workflowID, projectID string, inputData []byte) (string, error) {
	// Load workflow definition
	workflow, err := we.db.GetWorkflow(ctx, workflowID)
	if err != nil {
		return "", fmt.Errorf("failed to load workflow: %w", err)
	}

	// Create execution
	executionID := generateExecutionID()
	now := time.Now()
	execState := &ExecutionState{
		ExecutionID:         executionID,
		WorkflowID:          workflowID,
		Status:              "running",
		InputData:           inputData,
		CreatedAt:           now,
		StartedAt:           now,
		CompletedActivities: make(map[string]*ActivityExecutionState),
	}

	// Store in memory and database
	we.executionsMu.Lock()
	we.executions[executionID] = execState
	we.executionsMu.Unlock()

	if err := we.db.CreateExecution(ctx, &storage.Execution{
		ID:        executionID,
		WorkflowID: workflowID,
		ProjectID: projectID,
		Status:    "running",
		InputData: inputData,
		CreatedAt: now,
		StartedAt: now,
	}); err != nil {
		return "", fmt.Errorf("failed to store execution: %w", err)
	}

	// Record workflow started event
	we.recordEvent(executionID, &WorkflowEvent{
		ExecutionID: executionID,
		EventType:   "started",
		Timestamp:   now,
	})

	// Start execution
	go we.executeWorkflowAsync(ctx, executionID, workflow, execState)

	log.Printf("Workflow execution started: %s (execution: %s)", workflowID, executionID)
	return executionID, nil
}

// executeWorkflowAsync executes the workflow asynchronously
func (we *WorkflowEngine) executeWorkflowAsync(ctx context.Context, executionID string, workflow *storage.Workflow, execState *ExecutionState) {
	// Find start activities (no incoming edges)
	startActivities := we.findStartActivities(workflow)

	// Execute start activities
	for _, activity := range startActivities {
		we.workerPool.taskQueue <- &ActivityTask{
			ExecutionID: executionID,
			ActivityID:  activity.ID,
			InputData:   execState.InputData,
		}
	}
}

// executeActivity executes a single activity
func (we *WorkflowEngine) executeActivity(ctx context.Context, task *ActivityTask) {
	we.executionsMu.RLock()
	execState, exists := we.executions[task.ExecutionID]
	we.executionsMu.RUnlock()

	if !exists {
		log.Printf("Execution not found: %s", task.ExecutionID)
		return
	}

	now := time.Now()
	activityExec := &ActivityExecutionState{
		ActivityID: task.ActivityID,
		Status:     "running",
		InputData:  task.InputData,
		StartedAt:  now,
	}

	// Record activity started
	we.recordEvent(task.ExecutionID, &WorkflowEvent{
		ExecutionID: task.ExecutionID,
		EventType:   "activity_started",
		ActivityID:  task.ActivityID,
		Timestamp:   now,
	})

	// Simulate activity execution (in production: actual task execution)
	time.Sleep(100 * time.Millisecond)

	// Record activity completion
	activityExec.Status = "completed"
	activityExec.OutputData = []byte("Activity output")
	activityExec.CompletedAt = time.Now()

	we.executionsMu.Lock()
	execState.CompletedActivities[task.ActivityID] = activityExec
	we.executionsMu.Unlock()

	we.recordEvent(task.ExecutionID, &WorkflowEvent{
		ExecutionID: task.ExecutionID,
		EventType:   "activity_completed",
		ActivityID:  task.ActivityID,
		Timestamp:   time.Now(),
	})

	// Check if workflow is complete
	if we.isWorkflowComplete(execState) {
		we.completeExecution(ctx, task.ExecutionID, execState)
	}
}

// GetExecution retrieves an execution
func (we *WorkflowEngine) GetExecution(ctx context.Context, executionID string) (*pb.WorkflowExecution, error) {
	execution, err := we.db.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	return executionToProto(execution), nil
}

// RecoverExecution replays an execution from its event history
func (we *WorkflowEngine) RecoverExecution(ctx context.Context, executionID string) error {
	we.executionsMu.Lock()
	if _, exists := we.executions[executionID]; exists {
		we.executionsMu.Unlock()
		return fmt.Errorf("execution already in progress")
	}
	we.executionsMu.Unlock()

	// Load execution and events
	execution, err := we.db.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}

	events, err := we.db.GetExecutionEvents(ctx, executionID)
	if err != nil {
		return err
	}

	// Rebuild state from events
	execState := &ExecutionState{
		ExecutionID:         executionID,
		WorkflowID:          execution.WorkflowID,
		Status:              execution.Status,
		InputData:           execution.InputData,
		OutputData:          execution.OutputData,
		CreatedAt:           execution.CreatedAt,
		StartedAt:           execution.StartedAt,
		CompletedActivities: make(map[string]*ActivityExecutionState),
	}

	for _, event := range events {
		if event.EventType == "activity_completed" {
			// Rebuild activity state
			execState.CompletedActivities[event.ActivityID] = &ActivityExecutionState{
				ActivityID: event.ActivityID,
				Status:     "completed",
			}
		}
	}

	we.executionsMu.Lock()
	we.executions[executionID] = execState
	we.executionsMu.Unlock()

	log.Printf("Execution recovered: %s (from %d events)", executionID, len(events))
	return nil
}

// Helper methods

func (we *WorkflowEngine) findStartActivities(workflow *storage.Workflow) []*storage.Activity {
	// Simplified: return all activities with no incoming edges
	return workflow.Activities
}

func (we *WorkflowEngine) isWorkflowComplete(execState *ExecutionState) bool {
	// Simplified: complete when all activities done (in production: check DAG)
	return len(execState.CompletedActivities) > 0
}

func (we *WorkflowEngine) completeExecution(ctx context.Context, executionID string, execState *ExecutionState) {
	now := time.Now()
	execState.Status = "completed"
	execState.CompletedAt = now

	we.recordEvent(executionID, &WorkflowEvent{
		ExecutionID: executionID,
		EventType:   "completed",
		Timestamp:   now,
	})

	// Update in database
	if err := we.db.UpdateExecution(ctx, &storage.Execution{
		ID:        executionID,
		Status:    "completed",
		OutputData: execState.OutputData,
		CompletedAt: now,
	}); err != nil {
		log.Printf("Failed to update execution: %v", err)
	}

	log.Printf("Workflow execution completed: %s", executionID)
}

func (we *WorkflowEngine) recordEvent(executionID string, event *WorkflowEvent) {
	we.eventStore.mu.Lock()
	defer we.eventStore.mu.Unlock()

	if _, exists := we.eventStore.events[executionID]; !exists {
		we.eventStore.events[executionID] = make([]*WorkflowEvent, 0)
	}

	event.SequenceNum = len(we.eventStore.events[executionID]) + 1
	we.eventStore.events[executionID] = append(we.eventStore.events[executionID], event)
}

func generateExecutionID() string {
	return fmt.Sprintf("exec-%d", time.Now().UnixNano())
}

func executionToProto(exec *storage.Execution) *pb.WorkflowExecution {
	pbExec := &pb.WorkflowExecution{
		Id:        exec.ID,
		WorkflowId: exec.WorkflowID,
		ProjectId: exec.ProjectID,
		Status:    exec.Status,
		CreatedAt: pbTimestamp(exec.CreatedAt),
		UpdatedAt: pbTimestamp(exec.UpdatedAt),
	}

	if exec.StartedAt.After(exec.CreatedAt) {
		pbExec.StartedAt = pbTimestamp(exec.StartedAt)
	}

	if !exec.CompletedAt.IsZero() {
		pbExec.CompletedAt = pbTimestamp(exec.CompletedAt)
	}

	return pbExec
}

func pbTimestamp(t time.Time) *pb.Timestamp {
	// This would use proper protobuf timestamp conversion
	// Simplified here
	return &pb.Timestamp{Seconds: int64(t.Unix())}
}
