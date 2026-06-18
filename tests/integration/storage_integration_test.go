package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/state-store/internal/postgres"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/state-store/internal/storage"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	dsn := "host=localhost port=5432 user=titan password=titan_dev_password dbname=titan_db sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestKeyValueSet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	key := "test-key-1"
	value := map[string]interface{}{
		"name": "Test Value",
		"id":   "123",
	}

	err := store.Set(ctx, key, value, nil)
	if err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	t.Logf("✓ Key-value set operation successful")
}

func TestKeyValueGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	key := "test-key-2"
	value := map[string]interface{}{
		"name": "Test Value",
		"id":   "456",
	}

	err := store.Set(ctx, key, value, nil)
	if err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	retrieved, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get key: %v", err)
	}

	if retrieved.Key != key {
		t.Errorf("Expected key %s, got %s", key, retrieved.Key)
	}

	t.Logf("✓ Key-value get operation successful")
}

func TestKeyValueDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	key := "test-key-3"
	value := map[string]interface{}{"data": "test"}

	err := store.Set(ctx, key, value, nil)
	if err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}

	t.Logf("✓ Key-value delete operation successful")
}

func TestRangeQuery(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	// Set multiple keys with common prefix
	for i := 0; i < 5; i++ {
		key := "prefix:key-" + string(rune(i))
		value := map[string]interface{}{"index": i}
		store.Set(ctx, key, value, nil)
	}

	results, err := store.RangeQuery(ctx, "prefix:")
	if err != nil {
		t.Fatalf("Failed to query range: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	t.Logf("✓ Range query operation successful, returned %d results", len(results))
}

func TestTransactionCommit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	tx, err := store.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	operations := []storage.Operation{
		{
			Type:  "set",
			Key:   "tx-key-1",
			Value: map[string]interface{}{"data": "value1"},
		},
		{
			Type:  "set",
			Key:   "tx-key-2",
			Value: map[string]interface{}{"data": "value2"},
		},
	}

	err = store.CommitTransaction(ctx, tx.ID, operations)
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	if tx.Status != "committed" {
		t.Errorf("Expected status 'committed', got '%s'", tx.Status)
	}

	t.Logf("✓ Transaction commit successful")
}

func TestDistributedLock(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	resourceID := "resource-1"
	ownerID := "owner-1"

	err := store.AcquireLock(ctx, resourceID, ownerID, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if !store.IsLocked(ctx, resourceID) {
		t.Error("Expected resource to be locked")
	}

	err = store.ReleaseLock(ctx, resourceID, ownerID)
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	if store.IsLocked(ctx, resourceID) {
		t.Error("Expected resource to be unlocked")
	}

	t.Logf("✓ Distributed lock operations successful")
}

func TestSnapshot(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	resourceType := "workflow"
	resourceID := "wf-1"
	data := map[string]interface{}{
		"status": "running",
		"tasks":  []string{"task-1", "task-2"},
	}

	snapshot, err := store.CreateSnapshot(ctx, resourceType, resourceID, data)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if snapshot.Version != 1 {
		t.Errorf("Expected version 1, got %d", snapshot.Version)
	}

	retrieved, err := store.GetSnapshot(ctx, resourceType, resourceID, 1)
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}

	if retrieved.ID != snapshot.ID {
		t.Errorf("Expected snapshot ID %s, got %s", snapshot.ID, retrieved.ID)
	}

	t.Logf("✓ Snapshot operations successful")
}

func TestTTLExpiration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	ctx := context.Background()

	key := "ttl-key"
	value := map[string]interface{}{"data": "test"}
	ttl := 1 * time.Second

	err := store.Set(ctx, key, value, &ttl)
	if err != nil {
		t.Fatalf("Failed to set key with TTL: %v", err)
	}

	// Should be retrievable immediately
	_, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get key: %v", err)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should not be retrievable after expiration
	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Expected error when getting expired key")
	}

	t.Logf("✓ TTL expiration works correctly")
}

// Benchmark tests

func BenchmarkKeyValueSet(b *testing.B) {
	db, cleanup := setupTestDB(&testing.T{})
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)
	ctx := context.Background()

	value := map[string]interface{}{"data": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Set(ctx, "bench-key", value, nil)
	}
}

func BenchmarkKeyValueGet(b *testing.B) {
	db, cleanup := setupTestDB(&testing.T{})
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)
	ctx := context.Background()

	value := map[string]interface{}{"data": "test"}
	store.Set(ctx, "bench-key-get", value, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get(ctx, "bench-key-get")
	}
}

func BenchmarkRangeQuery(b *testing.B) {
	db, cleanup := setupTestDB(&testing.T{})
	defer cleanup()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)
	ctx := context.Background()

	// Setup: insert 100 keys
	for i := 0; i < 100; i++ {
		key := "bench:key-" + string(rune(i))
		value := map[string]interface{}{"index": i}
		store.Set(ctx, key, value, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.RangeQuery(ctx, "bench:")
	}
}
