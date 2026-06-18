package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/security/internal/auth"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/security/internal/secrets"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vaultAddr := getEnv("VAULT_ADDR", "http://vault:8200")
	vaultToken := getEnv("VAULT_TOKEN", "")
	grpcPort := getEnv("GRPC_PORT", "50057")

	log.Println("✓ Initializing Security service")

	// Initialize secret manager
	secretMgr, err := secrets.NewVaultManager(ctx, vaultAddr, vaultToken)
	if err != nil {
		log.Fatalf("Failed to initialize Vault: %v", err)
	}
	defer secretMgr.Close()
	log.Println("✓ Vault integration initialized")

	// Initialize auth manager
	authMgr := auth.NewAuthManager(secretMgr)
	log.Println("✓ Auth manager initialized")

	// Start gRPC server
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	// Register services
	// pb.RegisterSecurityServiceServer(grpcServer, authMgr)

	go func() {
		log.Printf("✓ gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	log.Printf("✓ Security service running\n  Vault: %s\n  gRPC: :%s", vaultAddr, grpcPort)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("✓ Shutting down Security service")
	grpcServer.GracefulStop()
	cancel()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
