package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/event-bus/internal/kafka"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/event-bus/internal/webhook"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kafkaHost := getEnv("KAFKA_HOST", "kafka:9092")
	grpcPort := getEnv("GRPC_PORT", "50053")

	log.Println("✓ Starting Event Bus service")

	// Initialize Kafka producer
	producer, err := kafka.NewProducer(kafkaHost)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()
	log.Println("✓ Kafka producer initialized")

	// Initialize Kafka consumer
	consumer, err := kafka.NewConsumer(kafkaHost, "event-bus-group")
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()
	log.Println("✓ Kafka consumer initialized")

	// Initialize webhook manager
	webhookMgr := webhook.NewManager(producer, consumer)
	log.Println("✓ Webhook manager initialized")

	// Start gRPC server
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	// Register services
	// pb.RegisterEventBusServiceServer(grpcServer, &grpcServer)
	// pb.RegisterWebhookServiceServer(grpcServer, webhookMgr)

	go func() {
		log.Printf("✓ gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	log.Printf("✓ Event Bus service running")
	log.Printf("  Kafka: %s", kafkaHost)
	log.Printf("  gRPC: :%s", grpcPort)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("✓ Shutting down Event Bus service")
	grpcServer.GracefulStop()
	cancel()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
