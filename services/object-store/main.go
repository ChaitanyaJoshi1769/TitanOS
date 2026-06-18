package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/object-store/internal/s3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// S3 configuration
	s3Endpoint := getEnv("S3_ENDPOINT", "minio:9000")
	s3AccessKey := getEnv("S3_ACCESS_KEY", "minioadmin")
	s3SecretKey := getEnv("S3_SECRET_KEY", "minioadmin")
	s3UseSSL := getEnv("S3_USE_SSL", "false") == "true"
	grpcPort := getEnv("GRPC_PORT", "50055")

	log.Println("✓ Initializing S3-compatible object storage client")

	// Initialize S3 client
	s3Client, err := s3.NewClient(ctx, s3Endpoint, s3AccessKey, s3SecretKey, s3UseSSL)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}
	defer s3Client.Close()

	log.Println("✓ S3 client initialized")

	// Create default buckets
	defaultBuckets := []string{
		"titan-artifacts",
		"titan-workflows",
		"titan-logs",
		"titan-models",
	}

	for _, bucket := range defaultBuckets {
		if err := s3Client.CreateBucketIfNotExists(ctx, bucket); err != nil {
			log.Fatalf("Failed to create bucket %s: %v", bucket, err)
		}
		log.Printf("✓ Bucket '%s' ready", bucket)
	}

	// Start gRPC server
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	// Register services
	// pb.RegisterObjectStoreServiceServer(grpcServer, s3Client)

	go func() {
		log.Printf("✓ gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	log.Printf("✓ Object Store service running")
	log.Printf("  S3: %s", s3Endpoint)
	log.Printf("  gRPC: :%s", grpcPort)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("✓ Shutting down Object Store service")
	grpcServer.GracefulStop()
	cancel()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
