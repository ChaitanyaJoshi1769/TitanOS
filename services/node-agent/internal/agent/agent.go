package agent

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	pb "github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds the configuration for a node agent
type Config struct {
	NodeID            string
	Hostname          string
	SchedulerAddress  string
	HeartbeatInterval time.Duration
	CPUCores          int
	MemoryGB          int
	GPUCount          int
	DiskGB            int
}

// NodeAgent represents a node in the cluster
type NodeAgent struct {
	config             *Config
	schedulerClient    pb.SchedulerServiceClient
	conn               *grpc.ClientConn
	runningTasks       map[string]*RunningTask
	tasksMu            sync.RWMutex
	registered         bool
	resourceMonitor    *ResourceMonitor
	taskExecutorChan   chan *pb.TaskAssignment
	stopChan           chan struct{}
}

// RunningTask tracks a task being executed on this node
type RunningTask struct {
	TaskID    string
	StartedAt time.Time
	Status    string
	Output    []byte
	Error     string
}

// ResourceMonitor tracks node resource usage
type ResourceMonitor struct {
	LastUpdate     time.Time
	AvailableCPU   int32
	AvailableMemory int32
	AvailableGPU   int32
	CPUUsage       float32
	MemoryUsage    float32
}

// NewNodeAgent creates a new node agent
func NewNodeAgent(config *Config) (*NodeAgent, error) {
	// Connect to scheduler
	conn, err := grpc.Dial(
		config.SchedulerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to scheduler: %w", err)
	}

	schedulerClient := pb.NewSchedulerServiceClient(conn)

	agent := &NodeAgent{
		config:           config,
		schedulerClient:  schedulerClient,
		conn:             conn,
		runningTasks:     make(map[string]*RunningTask),
		taskExecutorChan: make(chan *pb.TaskAssignment, 100),
		stopChan:         make(chan struct{}),
		resourceMonitor: &ResourceMonitor{
			AvailableCPU:    int32(config.CPUCores),
			AvailableMemory: int32(config.MemoryGB),
			AvailableGPU:    int32(config.GPUCount),
		},
	}

	return agent, nil
}

// Run starts the node agent main loop
func (n *NodeAgent) Run(ctx context.Context) error {
	defer n.conn.Close()

	// Register with scheduler
	if err := n.register(ctx); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	n.registered = true
	log.Printf("✓ Node %s registered with scheduler", n.config.NodeID)

	// Start heartbeat ticker
	ticker := time.NewTicker(n.config.HeartbeatInterval)
	defer ticker.Stop()

	// Start task executor
	go n.executeTasksLoop(ctx)

	// Main loop: send heartbeats and receive tasks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-n.stopChan:
			return nil

		case <-ticker.C:
			if err := n.sendHeartbeat(ctx); err != nil {
				log.Printf("Heartbeat failed: %v", err)
				// Could implement reconnection logic here
			}
		}
	}
}

// register registers the node with the scheduler
func (n *NodeAgent) register(ctx context.Context) error {
	req := &pb.RegisterNodeRequest{
		NodeId:   n.config.NodeID,
		Hostname: n.config.Hostname,
		CpuCores: int32(n.config.CPUCores),
		MemoryGb: int32(n.config.MemoryGB),
		GpuCount: int32(n.config.GPUCount),
		DiskGb:   int32(n.config.DiskGB),
		Labels: map[string]string{
			"arch":        runtime.GOARCH,
			"os":          runtime.GOOS,
			"agent_type":  "titan-node",
		},
	}

	resp, err := n.schedulerClient.RegisterNode(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("registration rejected: %s", resp.Message)
	}

	return nil
}

// sendHeartbeat sends a heartbeat to the scheduler and receives task assignments
func (n *NodeAgent) sendHeartbeat(ctx context.Context) error {
	n.tasksMu.RLock()
	runningTaskCount := len(n.runningTasks)
	n.tasksMu.RUnlock()

	// Update available resources (simplified - no actual monitoring yet)
	availableCPU := int32(n.config.CPUCores) - int32(runningTaskCount)
	if availableCPU < 0 {
		availableCPU = 0
	}

	req := &pb.HeartbeatRequest{
		NodeId:           n.config.NodeID,
		AvailableCpu:     availableCPU,
		AvailableMemory:  int32(n.config.MemoryGB),
		AvailableGpu:     int32(n.config.GPUCount),
		RunningTasks:     int32(runningTaskCount),
		CpuUsagePercent:  25.0,
		MemoryUsagePercent: 50.0,
		Status:           "healthy",
	}

	resp, err := n.schedulerClient.Heartbeat(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		log.Printf("Heartbeat rejected: %s", resp.Message)
		return nil
	}

	// Process task assignments
	for _, taskAssignment := range resp.TasksToRun {
		select {
		case n.taskExecutorChan <- taskAssignment:
			log.Printf("Task %s queued for execution", taskAssignment.TaskId)
		default:
			log.Printf("Task queue full, dropping task %s", taskAssignment.TaskId)
		}
	}

	return nil
}

// executeTasksLoop continuously executes tasks from the queue
func (n *NodeAgent) executeTasksLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case taskAssignment := <-n.taskExecutorChan:
			n.executeTask(ctx, taskAssignment)
		}
	}
}

// executeTask executes a single task
func (n *NodeAgent) executeTask(ctx context.Context, taskAssignment *pb.TaskAssignment) {
	taskID := taskAssignment.TaskId

	// Add to running tasks
	n.tasksMu.Lock()
	n.runningTasks[taskID] = &RunningTask{
		TaskID:    taskID,
		StartedAt: time.Now(),
		Status:    "running",
	}
	n.tasksMu.Unlock()

	log.Printf("Executing task: %s", taskID)

	// Simulate task execution
	// In production: actual task execution logic here
	go func() {
		time.Sleep(time.Second) // Simulate work

		// Mark as completed
		output := []byte("Task completed successfully")
		n.tasksMu.Lock()
		n.runningTasks[taskID].Status = "completed"
		n.runningTasks[taskID].Output = output
		n.tasksMu.Unlock()

		// Report completion
		if err := n.reportTaskCompletion(ctx, taskID, output, ""); err != nil {
			log.Printf("Failed to report task completion: %v", err)
		}

		// Remove from running tasks
		n.tasksMu.Lock()
		delete(n.runningTasks, taskID)
		n.tasksMu.Unlock()

		log.Printf("Task completed: %s", taskID)
	}()
}

// reportTaskCompletion reports a completed task back to the scheduler
func (n *NodeAgent) reportTaskCompletion(ctx context.Context, taskID string, output []byte, errorMsg string) error {
	req := &pb.UpdateTaskStatusRequest{
		TaskId:       taskID,
		NodeId:       n.config.NodeID,
		Status:       "completed",
		OutputData:   output,
		ErrorMessage: errorMsg,
	}

	resp, err := n.schedulerClient.UpdateTaskStatus(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("update rejected: %s", resp.Message)
	}

	return nil
}

// Stop gracefully stops the node agent
func (n *NodeAgent) Stop() {
	close(n.stopChan)
}

// GetRunningTasks returns the list of currently running tasks
func (n *NodeAgent) GetRunningTasks() []string {
	n.tasksMu.RLock()
	defer n.tasksMu.RUnlock()

	tasks := make([]string, 0, len(n.runningTasks))
	for taskID := range n.runningTasks {
		tasks = append(tasks, taskID)
	}
	return tasks
}
