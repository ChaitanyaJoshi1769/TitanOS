// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/internal/storage"
	pb "github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/pb"
)

// TestSchedulerIntegration tests the complete scheduler workflow
func TestSchedulerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test requires:
	// 1. PostgreSQL running at postgres://titan:titan_dev_password@localhost:5432/titan_test
	// 2. Scheduler service running
	// 3. Node agent running

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize database
	dbURL := "postgres://titan:titan_dev_password@postgres:5432/titan_test"
	db, err := storage.NewPostgresDB(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping: database not available: %v", err)
	}
	defer db.Close()

	// Test task creation
	task := &storage.Task{
		ID:             "test-task-1",
		ProjectID:      "test-project",
		Name:           "Test Task",
		Status:         "pending",
		InputData:      []byte("test input"),
		TimeoutSeconds: 60,
		Priority:       0,
		MaxRetries:     3,
		CreatedAt:      time.Now(),
	}

	if err := db.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Verify task was created
	retrievedTask, err := db.GetTask(ctx, "test-task-1")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrievedTask.Status != "pending" {
		t.Errorf("Expected task status 'pending', got '%s'", retrievedTask.Status)
	}

	// Test task status update
	update := &storage.TaskStatusUpdate{
		TaskID: "test-task-1",
		NodeID: "test-node-1",
		Status: "completed",
		OutputData: []byte("test output"),
	}

	if err := db.UpdateTaskStatus(ctx, update); err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Verify status was updated
	updatedTask, err := db.GetTask(ctx, "test-task-1")
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updatedTask.Status != "completed" {
		t.Errorf("Expected task status 'completed', got '%s'", updatedTask.Status)
	}

	t.Log("✓ Scheduler integration test passed")
}

// TestNodeRegistration tests node registration with the scheduler
func TestNodeRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize database
	dbURL := "postgres://titan:titan_dev_password@postgres:5432/titan_test"
	db, err := storage.NewPostgresDB(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping: database not available: %v", err)
	}
	defer db.Close()

	// Test node registration
	nodeState := &storage.NodeState{
		ID:              "test-node-1",
		Hostname:        "test-host",
		Status:          "healthy",
		CPUCores:        4,
		MemoryGB:        8,
		GPUCount:        0,
		DiskGB:          100,
		LastHeartbeat:   time.Now(),
		CreatedAt:       time.Now(),
	}

	if err := db.RegisterNode(ctx, nodeState); err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	// Verify node was registered
	node, err := db.GetNode(ctx, "test-node-1")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if node.Status != "healthy" {
		t.Errorf("Expected node status 'healthy', got '%s'", node.Status)
	}

	t.Log("✓ Node registration test passed")
}

// TestTaskQueuing tests multiple task submissions
func TestTaskQueuing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize database
	dbURL := "postgres://titan:titan_dev_password@postgres:5432/titan_test"
	db, err := storage.NewPostgresDB(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping: database not available: %v", err)
	}
	defer db.Close()

	// Create multiple tasks
	numTasks := 10
	for i := 0; i < numTasks; i++ {
		task := &storage.Task{
			ID:             "test-task-" + string(rune(i)),
			ProjectID:      "test-project",
			Name:           "Test Task",
			Status:         "pending",
			InputData:      []byte("test input"),
			TimeoutSeconds: 60,
			Priority:       i,
			MaxRetries:     3,
			CreatedAt:      time.Now(),
		}

		if err := db.CreateTask(ctx, task); err != nil {
			t.Fatalf("Failed to create task %d: %v", i, err)
		}
	}

	// List tasks
	tasks, total, err := db.ListTasks(ctx, &storage.ListTasksFilter{
		ProjectID: "test-project",
		Status:    "pending",
		Limit:     100,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if total < numTasks {
		t.Errorf("Expected at least %d tasks, got %d", numTasks, total)
	}

	if len(tasks) < numTasks {
		t.Errorf("Expected at least %d task objects, got %d", numTasks, len(tasks))
	}

	t.Logf("✓ Task queuing test passed (%d tasks)", len(tasks))
}

// BenchmarkTaskSubmission benchmarks task submission performance
func BenchmarkTaskSubmission(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Initialize database
	dbURL := "postgres://titan:titan_dev_password@postgres:5432/titan_test"
	db, err := storage.NewPostgresDB(ctx, dbURL)
	if err != nil {
		b.Skipf("Skipping: database not available: %v", err)
	}
	defer db.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		task := &storage.Task{
			ID:             "bench-task-" + string(rune(i)),
			ProjectID:      "bench-project",
			Name:           "Benchmark Task",
			Status:         "pending",
			InputData:      []byte("benchmark input"),
			TimeoutSeconds: 60,
			Priority:       0,
			MaxRetries:     3,
			CreatedAt:      time.Now(),
		}

		if err := db.CreateTask(ctx, task); err != nil {
			b.Fatalf("Failed to create task: %v", err)
		}
	}
}
