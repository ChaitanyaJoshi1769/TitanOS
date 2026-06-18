package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/agent-runtime/internal/agent"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/agent-runtime/internal/memory"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/agent-runtime/internal/storage"
)

func main() {
	// Initialize storage and memory
	stateStore := storage.NewInMemoryDatabase()
	memoryStore := memory.NewInMemoryStore()
	log.Println("✓ Agent runtime storage initialized")

	// Create runtime
	runtime := agent.NewAgentRuntime(stateStore, memoryStore)
	log.Println("✓ Agent runtime initialized")

	// Register default tools
	registerDefaultTools(runtime)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("\nShutting down agent runtime...")
	log.Println("✓ Agent runtime stopped")
}

func registerDefaultTools(runtime *agent.AgentRuntime) {
	// Register a simple echo tool for testing
	runtime.RegisterTool(&agent.ToolDefinition{
		Name:        "echo",
		Description: "Echo tool for testing",
		InputSchema: map[string]interface{}{
			"message": "string",
		},
		OutputSchema: map[string]interface{}{
			"result": "string",
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			message, ok := input["message"].(string)
			if !ok {
				message = "no message"
			}
			return map[string]interface{}{
				"result": message,
			}, nil
		},
	})

	log.Println("✓ Default tools registered")
}
