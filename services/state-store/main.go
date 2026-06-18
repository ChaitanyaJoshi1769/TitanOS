package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/state-store/internal/postgres"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/state-store/internal/storage"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database configuration
	dbHost := getEnv("DB_HOST", "postgres")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "titan")
	dbPassword := getEnv("DB_PASSWORD", "titan_dev_password")
	dbName := getEnv("DB_NAME", "titan_db")
	grpcPort := getEnv("GRPC_PORT", "50054")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	log.Println("✓ Connecting to PostgreSQL database")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("✓ Database connection established")

	// Initialize pool
	pool := postgres.NewPool(db)
	log.Println("✓ Connection pool initialized")

	// Initialize storage layer
	stateStore := storage.NewStateStore(pool)
	log.Println("✓ State store initialized")

	// Run migrations
	if err := pool.RunMigrations(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// Start gRPC server
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	// Register services
	// pb.RegisterStateStoreServiceServer(grpcServer, stateStore)

	go func() {
		log.Printf("✓ gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	log.Printf("✓ State Store service running")
	log.Printf("  Database: %s:%s/%s", dbHost, dbPort, dbName)
	log.Printf("  gRPC: :%s", grpcPort)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("✓ Shutting down State Store service")
	grpcServer.GracefulStop()
	cancel()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
