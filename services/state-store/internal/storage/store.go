package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/state-store/internal/postgres"
	"github.com/google/uuid"
)

type StateStore struct {
	pool            *postgres.Pool
	mu              sync.RWMutex
	locks           map[string]*DistributedLock
	transactionLog  []*Transaction
	lockManager     *LockManager
}

type KeyValueEntry struct {
	Key         string                 `json:"key"`
	Value       map[string]interface{} `json:"value"`
	Version     int                    `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	TTLExpiresAt *time.Time             `json:"ttl_expires_at,omitempty"`
}

type Transaction struct {
	ID           string                   `json:"id"`
	Status       string                   `json:"status"`
	StartedAt    time.Time                `json:"started_at"`
	CompletedAt  *time.Time               `json:"completed_at,omitempty"`
	Operations   []Operation              `json:"operations"`
	Results      []interface{}            `json:"results,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

type Operation struct {
	Type  string      `json:"type"`
	Key   string      `json:"key"`
	Value interface{} `json:"value,omitempty"`
}

type DistributedLock struct {
	ResourceID string
	OwnerID    string
	AcquiredAt time.Time
	ExpiresAt  time.Time
}

type Snapshot struct {
	ID           string
	ResourceType string
	ResourceID   string
	SnapshotData map[string]interface{}
	Version      int
	CreatedAt    time.Time
}

type LockManager struct {
	locks map[string]*DistributedLock
	mu    sync.RWMutex
}

func NewStateStore(pool *postgres.Pool) *StateStore {
	return &StateStore{
		pool:           pool,
		locks:          make(map[string]*DistributedLock),
		transactionLog: make([]*Transaction, 0),
		lockManager:    NewLockManager(),
	}
}

func NewLockManager() *LockManager {
	return &LockManager{
		locks: make(map[string]*DistributedLock),
	}
}

// Key-Value Operations

func (s *StateStore) Set(ctx context.Context, key string, value map[string]interface{}, ttl *time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	var ttlExpiresAt *time.Time
	if ttl != nil {
		expiry := time.Now().Add(*ttl)
		ttlExpiresAt = &expiry
	}

	query := `
		INSERT INTO key_value_store (key, value, ttl_expires_at, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = $2,
			version = version + 1,
			ttl_expires_at = $3,
			updated_at = NOW()
	`

	_, err = s.pool.Exec(ctx, query, key, valueJSON, ttlExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

func (s *StateStore) Get(ctx context.Context, key string) (*KeyValueEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT key, value, version, created_at, updated_at, ttl_expires_at
		FROM key_value_store
		WHERE key = $1 AND (ttl_expires_at IS NULL OR ttl_expires_at > NOW())
	`

	var entry KeyValueEntry
	var valueJSON []byte

	row := s.pool.QueryRow(ctx, query, key)
	err := row.Scan(&entry.Key, &valueJSON, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt, &entry.TTLExpiresAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	err = json.Unmarshal(valueJSON, &entry.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return &entry, nil
}

func (s *StateStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := "DELETE FROM key_value_store WHERE key = $1"
	_, err := s.pool.Exec(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

func (s *StateStore) RangeQuery(ctx context.Context, keyPrefix string) ([]KeyValueEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT key, value, version, created_at, updated_at, ttl_expires_at
		FROM key_value_store
		WHERE key LIKE $1 AND (ttl_expires_at IS NULL OR ttl_expires_at > NOW())
		ORDER BY key
	`

	rows, err := s.pool.Query(ctx, query, keyPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query range: %w", err)
	}
	defer rows.Close()

	var entries []KeyValueEntry
	for rows.Next() {
		var entry KeyValueEntry
		var valueJSON []byte

		err := rows.Scan(&entry.Key, &valueJSON, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt, &entry.TTLExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		err = json.Unmarshal(valueJSON, &entry.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal value: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// Transaction Operations

func (s *StateStore) BeginTransaction(ctx context.Context) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx := &Transaction{
		ID:        uuid.New().String(),
		Status:    "active",
		StartedAt: time.Now(),
		Operations: make([]Operation, 0),
	}

	s.transactionLog = append(s.transactionLog, tx)
	return tx, nil
}

func (s *StateStore) CommitTransaction(ctx context.Context, txID string, operations []Operation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.getTransaction(txID)
	if err != nil {
		return err
	}

	sqlTx, err := s.pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin SQL transaction: %w", err)
	}

	for _, op := range operations {
		if op.Type == "set" {
			valueJSON, _ := json.Marshal(op.Value)
			query := "INSERT INTO key_value_store (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2, version = version + 1"
			if _, err := sqlTx.ExecContext(ctx, query, op.Key, valueJSON); err != nil {
				sqlTx.Rollback()
				tx.Status = "failed"
				tx.ErrorMessage = err.Error()
				return fmt.Errorf("failed to execute operation: %w", err)
			}
		} else if op.Type == "delete" {
			query := "DELETE FROM key_value_store WHERE key = $1"
			if _, err := sqlTx.ExecContext(ctx, query, op.Key); err != nil {
				sqlTx.Rollback()
				tx.Status = "failed"
				tx.ErrorMessage = err.Error()
				return fmt.Errorf("failed to execute operation: %w", err)
			}
		}
	}

	if err := sqlTx.Commit(); err != nil {
		tx.Status = "failed"
		tx.ErrorMessage = err.Error()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	now := time.Now()
	tx.Status = "committed"
	tx.CompletedAt = &now

	return nil
}

func (s *StateStore) RollbackTransaction(ctx context.Context, txID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.getTransaction(txID)
	if err != nil {
		return err
	}

	now := time.Now()
	tx.Status = "rolled_back"
	tx.CompletedAt = &now

	return nil
}

// Distributed Locks

func (s *StateStore) AcquireLock(ctx context.Context, resourceID string, ownerID string, ttl time.Duration) error {
	return s.lockManager.AcquireLock(resourceID, ownerID, ttl)
}

func (s *StateStore) ReleaseLock(ctx context.Context, resourceID string, ownerID string) error {
	return s.lockManager.ReleaseLock(resourceID, ownerID)
}

func (s *StateStore) IsLocked(ctx context.Context, resourceID string) bool {
	return s.lockManager.IsLocked(resourceID)
}

// Snapshots

func (s *StateStore) CreateSnapshot(ctx context.Context, resourceType string, resourceID string, data map[string]interface{}) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshotID := uuid.New().String()
	dataJSON, _ := json.Marshal(data)

	query := `
		INSERT INTO snapshots (id, resource_type, resource_id, snapshot_data, version, created_at)
		VALUES ($1, $2, $3, $4, 1, NOW())
	`

	_, err := s.pool.Exec(ctx, query, snapshotID, resourceType, resourceID, dataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	return &Snapshot{
		ID:           snapshotID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		SnapshotData: data,
		Version:      1,
		CreatedAt:    time.Now(),
	}, nil
}

func (s *StateStore) GetSnapshot(ctx context.Context, resourceType string, resourceID string, version int) (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, resource_type, resource_id, snapshot_data, version, created_at
		FROM snapshots
		WHERE resource_type = $1 AND resource_id = $2 AND version = $3
	`

	var snapshot Snapshot
	var dataJSON []byte

	row := s.pool.QueryRow(ctx, query, resourceType, resourceID, version)
	err := row.Scan(&snapshot.ID, &snapshot.ResourceType, &snapshot.ResourceID, &dataJSON, &snapshot.Version, &snapshot.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	json.Unmarshal(dataJSON, &snapshot.SnapshotData)
	return &snapshot, nil
}

// Helper methods

func (s *StateStore) getTransaction(txID string) (*Transaction, error) {
	for _, tx := range s.transactionLog {
		if tx.ID == txID {
			return tx, nil
		}
	}
	return nil, fmt.Errorf("transaction not found: %s", txID)
}

// LockManager methods

func (lm *LockManager) AcquireLock(resourceID string, ownerID string, ttl time.Duration) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if _, exists := lm.locks[resourceID]; exists {
		return fmt.Errorf("resource is already locked: %s", resourceID)
	}

	lm.locks[resourceID] = &DistributedLock{
		ResourceID: resourceID,
		OwnerID:    ownerID,
		AcquiredAt: time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
	}

	return nil
}

func (lm *LockManager) ReleaseLock(resourceID string, ownerID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock, exists := lm.locks[resourceID]
	if !exists {
		return fmt.Errorf("lock not found: %s", resourceID)
	}

	if lock.OwnerID != ownerID {
		return fmt.Errorf("not the owner of the lock: %s", resourceID)
	}

	delete(lm.locks, resourceID)
	return nil
}

func (lm *LockManager) IsLocked(resourceID string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lock, exists := lm.locks[resourceID]
	if !exists {
		return false
	}

	if time.Now().After(lock.ExpiresAt) {
		return false
	}

	return true
}
