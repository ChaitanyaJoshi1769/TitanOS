package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Pool struct {
	db               *sql.DB
	mu               sync.RWMutex
	maxConnections   int
	connectionTimeout time.Duration
}

func NewPool(db *sql.DB) *Pool {
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	return &Pool{
		db:                db,
		maxConnections:    100,
		connectionTimeout: 30 * time.Second,
	}
}

func (p *Pool) GetConnection(ctx context.Context) (*sql.Conn, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	return conn, nil
}

func (p *Pool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

func (p *Pool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

func (p *Pool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

func (p *Pool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, opts)
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.db.Close()
}

func (p *Pool) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *Pool) Stats() sql.DBStats {
	return p.db.Stats()
}

func (p *Pool) RunMigrations(ctx context.Context) error {
	migrations := []string{
		// Already defined in packages/database/schema.sql
		// State store specific migrations
		`CREATE TABLE IF NOT EXISTS key_value_store (
			key TEXT PRIMARY KEY,
			value JSONB NOT NULL,
			version INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ttl_expires_at TIMESTAMP,
			INDEX idx_updated_at (updated_at)
		) WITH (fillfactor=70);`,

		`CREATE TABLE IF NOT EXISTS transactions (
			id UUID PRIMARY KEY,
			status VARCHAR(50) NOT NULL,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			operations JSONB,
			results JSONB,
			error_message TEXT,
			INDEX idx_status (status),
			INDEX idx_created_at (started_at)
		) WITH (fillfactor=70);`,

		`CREATE TABLE IF NOT EXISTS distributed_locks (
			resource_id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			acquired_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			INDEX idx_expires_at (expires_at)
		) WITH (fillfactor=70);`,

		`CREATE TABLE IF NOT EXISTS snapshots (
			id UUID PRIMARY KEY,
			resource_type VARCHAR(100) NOT NULL,
			resource_id VARCHAR(100) NOT NULL,
			snapshot_data JSONB NOT NULL,
			version INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(resource_type, resource_id, version),
			INDEX idx_resource (resource_type, resource_id)
		) WITH (fillfactor=70);`,

		`CREATE TABLE IF NOT EXISTS backups (
			id UUID PRIMARY KEY,
			backup_type VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			size_bytes BIGINT,
			location TEXT,
			error_message TEXT,
			INDEX idx_status (status),
			INDEX idx_created_at (started_at)
		) WITH (fillfactor=70);`,
	}

	for _, migration := range migrations {
		if _, err := p.db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}
