package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/state-store/internal/postgres"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/state-store/internal/storage"
	_ "github.com/lib/pq"
)

func main() {
	ctx := context.Background()

	// Connect to database
	dsn := "host=localhost port=5432 user=titan password=titan_dev_password dbname=titan_db sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	pool := postgres.NewPool(db)
	store := storage.NewStateStore(pool)

	// Run migrations
	pool.RunMigrations(ctx)

	fmt.Println("=== Titan OS Storage Examples ===\n")

	// Example 1: Basic key-value operations
	basicKeyValueOps(ctx, store)

	// Example 2: Range queries
	rangeQueryExample(ctx, store)

	// Example 3: Transactions
	transactionExample(ctx, store)

	// Example 4: Distributed locks
	distributedLockExample(ctx, store)

	// Example 5: Snapshots
	snapshotExample(ctx, store)

	// Example 6: TTL expiration
	ttlExample(ctx, store)
}

func basicKeyValueOps(ctx context.Context, store *storage.StateStore) {
	fmt.Println("1. Basic Key-Value Operations")
	fmt.Println("------------------------------")

	// Set a value
	key := "user:1"
	value := map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   30,
	}

	err := store.Set(ctx, key, value, nil)
	if err != nil {
		log.Printf("Error setting key: %v", err)
		return
	}
	fmt.Printf("✓ Set key '%s' successfully\n", key)

	// Get the value
	entry, err := store.Get(ctx, key)
	if err != nil {
		log.Printf("Error getting key: %v", err)
		return
	}
	fmt.Printf("✓ Retrieved key '%s'\n", entry.Key)
	fmt.Printf("  Value: %v\n", entry.Value)
	fmt.Printf("  Version: %d\n", entry.Version)
	fmt.Printf("  Updated: %s\n\n", entry.UpdatedAt)
}

func rangeQueryExample(ctx context.Context, store *storage.StateStore) {
	fmt.Println("2. Range Queries")
	fmt.Println("----------------")

	// Set multiple related keys
	prefix := "project:1:task:"
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("%s%d", prefix, i)
		value := map[string]interface{}{
			"id":    i,
			"name":  fmt.Sprintf("Task %d", i),
			"status": "pending",
		}
		store.Set(ctx, key, value, nil)
	}
	fmt.Printf("✓ Created 5 tasks with prefix '%s'\n", prefix)

	// Query by prefix
	results, err := store.RangeQuery(ctx, prefix)
	if err != nil {
		log.Printf("Error querying range: %v", err)
		return
	}

	fmt.Printf("✓ Range query returned %d results\n", len(results))
	for _, result := range results {
		fmt.Printf("  - %s (v%d)\n", result.Key, result.Version)
	}
	fmt.Println()
}

func transactionExample(ctx context.Context, store *storage.StateStore) {
	fmt.Println("3. ACID Transactions")
	fmt.Println("--------------------")

	// Begin transaction
	tx, err := store.BeginTransaction(ctx)
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		return
	}
	fmt.Printf("✓ Transaction started: %s\n", tx.ID)

	// Define operations
	operations := []storage.Operation{
		{
			Type: "set",
			Key:  "account:1000",
			Value: map[string]interface{}{
				"balance": 1000,
				"currency": "USD",
			},
		},
		{
			Type: "set",
			Key:  "account:2000",
			Value: map[string]interface{}{
				"balance": 500,
				"currency": "USD",
			},
		},
	}

	// Commit transaction
	err = store.CommitTransaction(ctx, tx.ID, operations)
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}

	fmt.Printf("✓ Transaction committed with %d operations\n", len(operations))
	fmt.Printf("  Status: %s\n\n", tx.Status)
}

func distributedLockExample(ctx context.Context, store *storage.StateStore) {
	fmt.Println("4. Distributed Locks")
	fmt.Println("--------------------")

	resourceID := "workflow:123"
	ownerID := "executor:5"
	ttl := 30 * time.Second

	// Acquire lock
	err := store.AcquireLock(ctx, resourceID, ownerID, ttl)
	if err != nil {
		log.Printf("Error acquiring lock: %v", err)
		return
	}
	fmt.Printf("✓ Lock acquired for resource '%s' by '%s'\n", resourceID, ownerID)
	fmt.Printf("  TTL: %s\n", ttl)

	// Check if locked
	if store.IsLocked(ctx, resourceID) {
		fmt.Printf("✓ Resource '%s' is locked\n", resourceID)
	}

	// Release lock
	err = store.ReleaseLock(ctx, resourceID, ownerID)
	if err != nil {
		log.Printf("Error releasing lock: %v", err)
		return
	}
	fmt.Printf("✓ Lock released\n\n")
}

func snapshotExample(ctx context.Context, store *storage.StateStore) {
	fmt.Println("5. Snapshots")
	fmt.Println("------------")

	resourceType := "workflow_execution"
	resourceID := "exec:456"
	snapshotData := map[string]interface{}{
		"status": "running",
		"progress": 0.5,
		"tasks_completed": 2,
		"total_tasks": 4,
		"started_at": time.Now().String(),
	}

	// Create snapshot
	snapshot, err := store.CreateSnapshot(ctx, resourceType, resourceID, snapshotData)
	if err != nil {
		log.Printf("Error creating snapshot: %v", err)
		return
	}
	fmt.Printf("✓ Snapshot created\n")
	fmt.Printf("  ID: %s\n", snapshot.ID)
	fmt.Printf("  Resource: %s:%s\n", resourceType, resourceID)
	fmt.Printf("  Version: %d\n", snapshot.Version)

	// Retrieve snapshot
	retrieved, err := store.GetSnapshot(ctx, resourceType, resourceID, snapshot.Version)
	if err != nil {
		log.Printf("Error retrieving snapshot: %v", err)
		return
	}
	fmt.Printf("✓ Snapshot retrieved\n")
	fmt.Printf("  Status: %v\n\n", retrieved.SnapshotData["status"])
}

func ttlExample(ctx context.Context, store *storage.StateStore) {
	fmt.Println("6. TTL (Time-To-Live) Expiration")
	fmt.Println("--------------------------------")

	key := "temporary:session:abc123"
	value := map[string]interface{}{
		"user_id": "user:1",
		"created_at": time.Now().String(),
	}
	ttl := 5 * time.Second

	// Set with expiration
	err := store.Set(ctx, key, value, &ttl)
	if err != nil {
		log.Printf("Error setting key with TTL: %v", err)
		return
	}
	fmt.Printf("✓ Key set with TTL of %s\n", ttl)

	// Retrieve before expiration
	entry, err := store.Get(ctx, key)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("✓ Key retrieved before expiration\n")
	if entry.TTLExpiresAt != nil {
		fmt.Printf("  Expires at: %s\n", entry.TTLExpiresAt)
	}

	fmt.Println("  Waiting for expiration...")
	time.Sleep(6 * time.Second)

	// Try to retrieve after expiration
	_, err = store.Get(ctx, key)
	if err != nil {
		fmt.Printf("✓ Key expired as expected: %v\n\n", err)
	}
}
