package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/internal/storage"
	pb "github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Scheduler implements the core scheduling logic
type Scheduler struct {
	db           storage.Database
	nodeRegistry map[string]*NodeState
	taskQueue    []TaskQueueItem
	mu           sync.RWMutex
	metrics      *Metrics
}

// NodeState tracks the state of each node
type NodeState struct {
	ID              string
	Hostname        string
	Status          string
	CPUCores        int32
	MemoryGB        int32
	GPUCount        int32
	DiskGB          int32
	AvailableCPU    int32
	AvailableMemory int32
	AvailableGPU    int32
	RunningTasks    int32
	Labels          map[string]string
	LastHeartbeat   time.Time
	CreatedAt       time.Time
}

// TaskQueueItem represents a task in the queue
type TaskQueueItem struct {
	TaskID   string
	Priority int32
	QueuedAt time.Time
}

// Metrics tracks performance metrics
type Metrics struct {
	TasksSubmitted   int64
	TasksScheduled   int64
	TasksCompleted   int64
	TasksFailed      int64
	NodesRegistered  int64
	AvgScheduleTime  float64
	mu               sync.RWMutex
}

// NewScheduler creates a new scheduler instance
func NewScheduler(db storage.Database) *Scheduler {
	return &Scheduler{
		db:           db,
		nodeRegistry: make(map[string]*NodeState),
		taskQueue:    make([]TaskQueueItem, 0),
		metrics:      &Metrics{},
	}
}

// RegisterNode registers a new node with the scheduler
func (s *Scheduler) RegisterNode(ctx context.Context, req *pb.RegisterNodeRequest) (*pb.RegisterNodeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	nodeState := &NodeState{
		ID:              req.NodeId,
		Hostname:        req.Hostname,
		Status:          "healthy",
		CPUCores:        req.CpuCores,
		MemoryGB:        req.MemoryGb,
		GPUCount:        req.GpuCount,
		DiskGB:          req.DiskGb,
		AvailableCPU:    req.CpuCores,
		AvailableMemory: req.MemoryGb,
		AvailableGPU:    req.GpuCount,
		RunningTasks:    0,
		Labels:          req.Labels,
		LastHeartbeat:   now,
		CreatedAt:       now,
	}

	// Register in memory
	s.nodeRegistry[req.NodeId] = nodeState

	// Persist to database
	if err := s.db.RegisterNode(ctx, nodeState); err != nil {
		return &pb.RegisterNodeResponse{
			NodeId:  req.NodeId,
			Success: false,
			Message: fmt.Sprintf("failed to persist: %v", err),
		}, nil
	}

	s.metrics.mu.Lock()
	s.metrics.NodesRegistered++
	s.metrics.mu.Unlock()

	log.Printf("Node registered: %s (%s) with %d cores, %d GB RAM", req.NodeId, req.Hostname, req.CpuCores, req.MemoryGb)

	return &pb.RegisterNodeResponse{
		NodeId:  req.NodeId,
		Success: true,
		Message: "Node registered successfully",
	}, nil
}

// Heartbeat processes node heartbeats and assigns tasks
func (s *Scheduler) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodeState, exists := s.nodeRegistry[req.NodeId]
	if !exists {
		return &pb.HeartbeatResponse{
			Success: false,
			Message: "Node not registered",
		}, nil
	}

	// Update node state
	nodeState.LastHeartbeat = time.Now()
	nodeState.AvailableCPU = req.AvailableCpu
	nodeState.AvailableMemory = req.AvailableMemory
	nodeState.AvailableGPU = req.AvailableGpu
	nodeState.RunningTasks = req.RunningTasks
	nodeState.Status = req.Status

	// Persist heartbeat
	if err := s.db.UpdateNodeHeartbeat(ctx, req.NodeId); err != nil {
		log.Printf("Failed to update heartbeat for %s: %v", req.NodeId, err)
	}

	// Assign tasks from queue
	tasksToAssign := s.selectTasksForNode(nodeState)

	// Convert to protobuf
	taskAssignments := make([]*pb.TaskAssignment, 0, len(tasksToAssign))
	for _, task := range tasksToAssign {
		taskAssignments = append(taskAssignments, &pb.TaskAssignment{
			TaskId:          task.ID,
			InputData:       task.InputData,
			TimeoutSeconds:  int32(task.TimeoutSeconds),
		})

		// Update task status to assigned
		if err := s.db.UpdateTaskStatus(ctx, &storage.TaskStatusUpdate{
			TaskID:   task.ID,
			NodeID:   req.NodeId,
			Status:   "assigned",
			Error:    "",
		}); err != nil {
			log.Printf("Failed to update task %s status: %v", task.ID, err)
		}
	}

	return &pb.HeartbeatResponse{
		Success:     true,
		Message:     "Heartbeat processed",
		TasksToRun:  taskAssignments,
	}, nil
}

// SubmitTask submits a task to the scheduler
func (s *Scheduler) SubmitTask(ctx context.Context, req *pb.SubmitTaskRequest) (*pb.SubmitTaskResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Persist to database
	task := &storage.Task{
		ID:           req.TaskId,
		ProjectID:    req.ProjectId,
		Name:         req.Name,
		Status:       "pending",
		InputData:    req.InputData,
		TimeoutSeconds: int(req.TimeoutSeconds),
		Priority:     int(req.Priority),
		MaxRetries:   int(req.MaxRetries),
		Labels:       req.Labels,
		CreatedAt:    time.Now(),
	}

	if err := s.db.CreateTask(ctx, task); err != nil {
		return &pb.SubmitTaskResponse{
			TaskId:  req.TaskId,
			Success: false,
			Message: fmt.Sprintf("failed to create task: %v", err),
		}, nil
	}

	// Add to queue
	s.taskQueue = append(s.taskQueue, TaskQueueItem{
		TaskID:   req.TaskId,
		Priority: req.Priority,
		QueuedAt: time.Now(),
	})

	s.metrics.mu.Lock()
	s.metrics.TasksSubmitted++
	s.metrics.mu.Unlock()

	log.Printf("Task submitted: %s (priority: %d)", req.TaskId, req.Priority)

	return &pb.SubmitTaskResponse{
		TaskId:  req.TaskId,
		Success: true,
		Message: "Task submitted successfully",
	}, nil
}

// UpdateTaskStatus updates the status of a task
func (s *Scheduler) UpdateTaskStatus(ctx context.Context, req *pb.UpdateTaskStatusRequest) (*pb.UpdateTaskStatusResponse, error) {
	statusUpdate := &storage.TaskStatusUpdate{
		TaskID:      req.TaskId,
		NodeID:      req.NodeId,
		Status:      req.Status,
		OutputData:  req.OutputData,
		Error:       req.ErrorMessage,
	}

	if err := s.db.UpdateTaskStatus(ctx, statusUpdate); err != nil {
		return &pb.UpdateTaskStatusResponse{
			Success: false,
			Message: fmt.Sprintf("failed to update: %v", err),
		}, nil
	}

	if req.Status == "completed" {
		s.metrics.mu.Lock()
		s.metrics.TasksCompleted++
		s.metrics.mu.Unlock()
		log.Printf("Task completed: %s", req.TaskId)
	} else if req.Status == "failed" {
		s.metrics.mu.Lock()
		s.metrics.TasksFailed++
		s.metrics.mu.Unlock()
		log.Printf("Task failed: %s (%s)", req.TaskId, req.ErrorMessage)
	}

	return &pb.UpdateTaskStatusResponse{
		Success: true,
		Message: "Status updated successfully",
	}, nil
}

// GetTask retrieves a task by ID
func (s *Scheduler) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	task, err := s.db.GetTask(ctx, req.TaskId)
	if err != nil {
		return &pb.GetTaskResponse{
			Task: nil,
		}, nil
	}

	return &pb.GetTaskResponse{
		Task: taskToProto(task),
	}, nil
}

// ListTasks lists tasks with filtering and pagination
func (s *Scheduler) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	tasks, total, err := s.db.ListTasks(ctx, &storage.ListTasksFilter{
		ProjectID: req.ProjectId,
		Status:    req.Status,
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	})
	if err != nil {
		return &pb.ListTasksResponse{
			Tasks: make([]*pb.Task, 0),
			Total: 0,
		}, nil
	}

	protoTasks := make([]*pb.Task, 0, len(tasks))
	for _, task := range tasks {
		protoTasks = append(protoTasks, taskToProto(task))
	}

	return &pb.ListTasksResponse{
		Tasks: protoTasks,
		Total: int32(total),
	}, nil
}

// GetNode retrieves a node by ID
func (s *Scheduler) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	s.mu.RLock()
	nodeState, exists := s.nodeRegistry[req.NodeId]
	s.mu.RUnlock()

	if !exists {
		return &pb.GetNodeResponse{
			Node: nil,
		}, nil
	}

	return &pb.GetNodeResponse{
		Node: nodeStateToProto(nodeState),
	}, nil
}

// ListNodes lists all nodes with pagination
func (s *Scheduler) ListNodes(ctx context.Context, req *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*pb.Node, 0, len(s.nodeRegistry))
	for _, nodeState := range s.nodeRegistry {
		nodes = append(nodes, nodeStateToProto(nodeState))
	}

	// Simple in-memory pagination
	total := len(nodes)
	start := int(req.Offset)
	end := start + int(req.Limit)
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	return &pb.ListNodesResponse{
		Nodes: nodes[start:end],
		Total: int32(total),
	}, nil
}

// selectTasksForNode selects tasks to assign to a node
func (s *Scheduler) selectTasksForNode(nodeState *NodeState) []*storage.Task {
	// Simple FIFO scheduling - can be enhanced with:
	// - Priority-based scheduling
	// - Resource-aware scheduling
	// - Affinity-based scheduling
	// - Load balancing

	selectedTasks := make([]*storage.Task, 0)

	// Can assign up to the number of available resources
	maxTasks := int(nodeState.AvailableCPU) // Simplified: 1 task per core

	for i := 0; i < len(s.taskQueue) && i < maxTasks; i++ {
		taskItem := s.taskQueue[i]

		// Fetch task details from database (simplified here)
		// In production: batch fetch with proper error handling
		task := &storage.Task{
			ID: taskItem.TaskID,
		}
		selectedTasks = append(selectedTasks, task)
	}

	// Remove assigned tasks from queue
	if len(selectedTasks) > 0 {
		s.taskQueue = s.taskQueue[len(selectedTasks):]
	}

	return selectedTasks
}

// Helper functions to convert between internal and protobuf types

func taskToProto(task *storage.Task) *pb.Task {
	return &pb.Task{
		Id:               task.ID,
		ProjectId:        task.ProjectID,
		NodeId:           task.NodeID,
		Name:             task.Name,
		Status:           task.Status,
		InputData:        task.InputData,
		OutputData:       task.OutputData,
		ErrorMessage:     task.ErrorMessage,
		RetryCount:       int32(task.RetryCount),
		MaxRetries:       int32(task.MaxRetries),
		TimeoutSeconds:   int32(task.TimeoutSeconds),
		Priority:         int32(task.Priority),
		Labels:           task.Labels,
		CreatedAt:        timestamppb.New(task.CreatedAt),
		StartedAt:        timestamppb.New(task.StartedAt),
		CompletedAt:      timestamppb.New(task.CompletedAt),
		UpdatedAt:        timestamppb.New(task.UpdatedAt),
	}
}

func nodeStateToProto(nodeState *NodeState) *pb.Node {
	return &pb.Node{
		Id:            nodeState.ID,
		Name:          nodeState.Hostname,
		Hostname:      nodeState.Hostname,
		Status:        nodeState.Status,
		CpuCores:      nodeState.CPUCores,
		MemoryGb:      nodeState.MemoryGB,
		GpuCount:      nodeState.GPUCount,
		DiskGb:        nodeState.DiskGB,
		Labels:        nodeState.Labels,
		LastHeartbeat: timestamppb.New(nodeState.LastHeartbeat),
		CreatedAt:     timestamppb.New(nodeState.CreatedAt),
		UpdatedAt:     timestamppb.New(time.Now()),
	}
}
