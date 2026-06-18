package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/workflow-engine/internal/engine"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/workflow-engine/internal/storage"
	"google.golang.org/grpc"
)

func main() {
	// Initialize storage
	db := storage.NewInMemoryDB()
	log.Println("✓ Workflow engine storage initialized")

	// Create workflow engine
	workflowEngine := engine.NewWorkflowEngine(db, 10) // 10 workers
	log.Println("✓ Workflow engine initialized")

	// Start engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workflowEngine.Start(ctx)

	// Setup gRPC server
	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// Register WorkflowEngineService
	// pb.RegisterWorkflowEngineServiceServer(grpcServer, workflowEngine)

	go func() {
		log.Printf("✓ gRPC server listening on :50052")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down workflow engine...")
	grpcServer.GracefulStop()
	cancel()
	log.Println("✓ Workflow engine stopped")
}
