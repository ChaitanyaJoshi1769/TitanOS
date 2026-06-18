package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Database interface defines all database operations
type Database interface {
	RegisterNode(ctx context.Context, node *NodeState) error
	UpdateNodeHeartbeat(ctx context.Context, nodeID string) error
	CreateTask(ctx context.Context, task *Task) error
	UpdateTaskStatus(ctx context.Context, update *TaskStatusUpdate) error
	GetTask(ctx context.Context, taskID string) (*Task, error)
	ListTasks(ctx context.Context, filter *ListTasksFilter) ([]*Task, int, error)
	GetNode(ctx context.Context, nodeID string) (*Node, error)
	ListNodes(ctx context.Context, limit, offset int) ([]*Node, int, error)
	Close() error
}

// PostgreSQL implementation
type PostgresDB struct {
	conn *sql.DB
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(ctx context.Context, dsn string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresDB{conn: db}, nil
}

// Task represents a task in the database
type Task struct {
	ID             string
	ProjectID      string
	NodeID         string
	Name           string
	Status         string
	InputData      []byte
	OutputData     []byte
	ErrorMessage   string
	RetryCount     int
	MaxRetries     int
	TimeoutSeconds int
	Priority       int
	Labels         map[string]string
	CreatedAt      time.Time
	StartedAt      time.Time
	CompletedAt    time.Time
	UpdatedAt      time.Time
}

// Node represents a node in the database
type Node struct {
	ID            string
	ProjectID     string
	Name          string
	Hostname      string
	Status        string
	CPUCores      int32
	MemoryGB      int32
	GPUCount      int32
	DiskGB        int32
	Labels        map[string]string
	LastHeartbeat time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NodeState mirrors the server-side NodeState
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

// TaskStatusUpdate represents an update to task status
type TaskStatusUpdate struct {
	TaskID   string
	NodeID   string
	Status   string
	OutputData []byte
	Error    string
}

// ListTasksFilter represents filter parameters for listing tasks
type ListTasksFilter struct {
	ProjectID string
	Status    string
	Limit     int
	Offset    int
}

// RegisterNode registers a node in the database
func (db *PostgresDB) RegisterNode(ctx context.Context, node *NodeState) error {
	query := `
		INSERT INTO nodes (id, name, hostname, status, cpu_cores, memory_gb, gpu_count, disk_gb, labels, last_heartbeat, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			labels = EXCLUDED.labels,
			last_heartbeat = EXCLUDED.last_heartbeat,
			updated_at = NOW()
	`

	_, err := db.conn.ExecContext(ctx, query,
		node.ID,
		node.Hostname,
		node.Hostname,
		"healthy",
		node.CPUCores,
		node.MemoryGB,
		node.GPUCount,
		node.DiskGB,
		nil, // labels as hstore - simplified
		time.Now(),
		time.Now(),
		time.Now(),
	)

	return err
}

// UpdateNodeHeartbeat updates the last heartbeat time for a node
func (db *PostgresDB) UpdateNodeHeartbeat(ctx context.Context, nodeID string) error {
	query := `UPDATE nodes SET last_heartbeat = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := db.conn.ExecContext(ctx, query, nodeID)
	return err
}

// CreateTask creates a new task in the database
func (db *PostgresDB) CreateTask(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, project_id, name, status, input_data, timeout_seconds, priority, max_retries, labels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := db.conn.ExecContext(ctx, query,
		task.ID,
		task.ProjectID,
		task.Name,
		"pending",
		task.InputData,
		task.TimeoutSeconds,
		task.Priority,
		task.MaxRetries,
		nil, // labels
		time.Now(),
		time.Now(),
	)

	return err
}

// UpdateTaskStatus updates the status of a task
func (db *PostgresDB) UpdateTaskStatus(ctx context.Context, update *TaskStatusUpdate) error {
	var query string
	var args []interface{}

	if update.Status == "completed" {
		query = `
			UPDATE tasks
			SET status = $1, node_id = $2, output_data = $3, completed_at = $4, updated_at = NOW()
			WHERE id = $5
		`
		args = []interface{}{update.Status, update.NodeID, update.OutputData, time.Now(), update.TaskID}
	} else if update.Status == "failed" {
		query = `
			UPDATE tasks
			SET status = $1, node_id = $2, error_message = $3, completed_at = $4, updated_at = NOW()
			WHERE id = $5
		`
		args = []interface{}{update.Status, update.NodeID, update.Error, time.Now(), update.TaskID}
	} else {
		query = `
			UPDATE tasks
			SET status = $1, node_id = $2, started_at = COALESCE(started_at, $3), updated_at = NOW()
			WHERE id = $4
		`
		args = []interface{}{update.Status, update.NodeID, time.Now(), update.TaskID}
	}

	_, err := db.conn.ExecContext(ctx, query, args...)
	return err
}

// GetTask retrieves a task by ID
func (db *PostgresDB) GetTask(ctx context.Context, taskID string) (*Task, error) {
	query := `
		SELECT id, project_id, node_id, name, status, input_data, output_data, error_message,
		       retry_count, max_retries, timeout_seconds, priority, created_at, started_at, completed_at, updated_at
		FROM tasks WHERE id = $1
	`

	task := &Task{}
	var nodeID sql.NullString

	err := db.conn.QueryRowContext(ctx, query, taskID).Scan(
		&task.ID,
		&task.ProjectID,
		&nodeID,
		&task.Name,
		&task.Status,
		&task.InputData,
		&task.OutputData,
		&task.ErrorMessage,
		&task.RetryCount,
		&task.MaxRetries,
		&task.TimeoutSeconds,
		&task.Priority,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, err
	}

	if nodeID.Valid {
		task.NodeID = nodeID.String
	}

	return task, nil
}

// ListTasks lists tasks with filtering
func (db *PostgresDB) ListTasks(ctx context.Context, filter *ListTasksFilter) ([]*Task, int, error) {
	// Build dynamic query based on filters
	query := `
		SELECT id, project_id, node_id, name, status, input_data, output_data, error_message,
		       retry_count, max_retries, timeout_seconds, priority, created_at, started_at, completed_at, updated_at
		FROM tasks
		WHERE project_id = $1
	`
	args := []interface{}{filter.ProjectID}
	argCount := 2

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	// Add ordering and pagination
	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d OFFSET $%d", argCount, argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := make([]*Task, 0)
	for rows.Next() {
		task := &Task{}
		var nodeID sql.NullString

		err := rows.Scan(
			&task.ID, &task.ProjectID, &nodeID, &task.Name, &task.Status,
			&task.InputData, &task.OutputData, &task.ErrorMessage,
			&task.RetryCount, &task.MaxRetries, &task.TimeoutSeconds, &task.Priority,
			&task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if nodeID.Valid {
			task.NodeID = nodeID.String
		}

		tasks = append(tasks, task)
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM tasks WHERE project_id = $1`
	countArgs := []interface{}{filter.ProjectID}

	if filter.Status != "" {
		countQuery += ` AND status = $2`
		countArgs = append(countArgs, filter.Status)
	}

	var total int
	if err := db.conn.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetNode retrieves a node by ID
func (db *PostgresDB) GetNode(ctx context.Context, nodeID string) (*Node, error) {
	query := `
		SELECT id, name, hostname, status, cpu_cores, memory_gb, gpu_count, disk_gb, last_heartbeat, created_at, updated_at
		FROM nodes WHERE id = $1
	`

	node := &Node{}
	err := db.conn.QueryRowContext(ctx, query, nodeID).Scan(
		&node.ID, &node.Name, &node.Hostname, &node.Status,
		&node.CPUCores, &node.MemoryGB, &node.GPUCount, &node.DiskGB,
		&node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node not found")
	}
	return node, err
}

// ListNodes lists all nodes
func (db *PostgresDB) ListNodes(ctx context.Context, limit, offset int) ([]*Node, int, error) {
	query := `
		SELECT id, name, hostname, status, cpu_cores, memory_gb, gpu_count, disk_gb, last_heartbeat, created_at, updated_at
		FROM nodes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.conn.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	nodes := make([]*Node, 0)
	for rows.Next() {
		node := &Node{}
		err := rows.Scan(
			&node.ID, &node.Name, &node.Hostname, &node.Status,
			&node.CPUCores, &node.MemoryGB, &node.GPUCount, &node.DiskGB,
			&node.LastHeartbeat, &node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		nodes = append(nodes, node)
	}

	// Count total
	var total int
	if err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM nodes").Scan(&total); err != nil {
		return nil, 0, err
	}

	return nodes, total, nil
}

// Close closes the database connection
func (db *PostgresDB) Close() error {
	return db.conn.Close()
}
