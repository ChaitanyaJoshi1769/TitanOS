package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/internal/server"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/internal/storage"
)

func main() {
	// Configuration
	grpcPort := getEnv("GRPC_PORT", "50051")
	metricsPort := getEnv("METRICS_PORT", "8001")
	dbURL := getEnv("DATABASE_URL", "postgres://titan:titan_dev_password@postgres:5432/titan_db")

	// Initialize database
	db, err := storage.NewPostgresDB(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("✓ Database connected")

	// Initialize scheduler
	scheduler := server.NewScheduler(db)
	log.Println("✓ Scheduler initialized")

	// Start metrics server
	go func() {
		if err := server.StartMetricsServer(metricsPort); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()
	log.Printf("✓ Metrics server listening on :%s", metricsPort)

	// Start gRPC server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	grpcServer := server.NewGRPCServer(scheduler)
	go func() {
		log.Printf("✓ gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down scheduler service...")
	grpcServer.GracefulStop()
	log.Println("✓ Scheduler service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
