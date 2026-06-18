package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/node-agent/internal/agent"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/node-agent/internal/executor"
)

func main() {
	// Configuration
	nodeID := getEnv("NODE_ID", fmt.Sprintf("node-%d", os.Getpid()))
	hostname := getEnv("HOSTNAME", "localhost")
	schedulerAddress := getEnv("SCHEDULER_ADDRESS", "localhost:50051")
	heartbeatInterval := getDurationEnv("HEARTBEAT_INTERVAL", 10*time.Second)

	log.Printf("Starting Node Agent %s", nodeID)
	log.Printf("Connecting to Scheduler at %s", schedulerAddress)

	// Create node agent
	config := &agent.Config{
		NodeID:            nodeID,
		Hostname:          hostname,
		SchedulerAddress:  schedulerAddress,
		HeartbeatInterval: heartbeatInterval,
		CPUCores:          getIntEnv("CPU_CORES", 4),
		MemoryGB:          getIntEnv("MEMORY_GB", 8),
		GPUCount:          getIntEnv("GPU_COUNT", 0),
		DiskGB:            getIntEnv("DISK_GB", 100),
	}

	nodeAgent, err := agent.NewNodeAgent(config)
	if err != nil {
		log.Fatalf("Failed to create node agent: %v", err)
	}

	log.Println("✓ Node Agent initialized")

	// Create task executor
	taskExecutor := executor.NewTaskExecutor(nodeAgent)

	// Start the agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := nodeAgent.Run(ctx); err != nil {
			log.Fatalf("Node agent error: %v", err)
		}
	}()

	log.Println("✓ Node Agent running - press Ctrl+C to stop")

	// Keep running
	select {}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
