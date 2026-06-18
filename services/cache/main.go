package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/cache/internal/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Redis configuration
	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	grpcPort := getEnv("GRPC_PORT", "50056")

	redisAddr := redisHost + ":" + redisPort

	log.Println("✓ Initializing Redis cache client")

	// Initialize Redis client
	cacheClient, err := redis.NewClient(ctx, redisAddr)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer cacheClient.Close()

	log.Println("✓ Redis cache client initialized and connected")

	// Start gRPC server
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	// Register services
	// pb.RegisterCacheServiceServer(grpcServer, cacheClient)

	go func() {
		log.Printf("✓ gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	log.Printf("✓ Cache service running")
	log.Printf("  Redis: %s", redisAddr)
	log.Printf("  gRPC: :%s", grpcPort)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("✓ Shutting down Cache service")
	grpcServer.GracefulStop()
	cancel()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
