package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/gateway/internal/auth"
	"github.com/ChaitanyaJoshi1769/TitanOS/services/gateway/internal/gateway"
)

func main() {
	// Initialize authentication
	authManager := auth.NewAuthManager()
	log.Println("✓ Authentication manager initialized")

	// Initialize gateway
	gw := gateway.NewGateway(authManager)
	log.Println("✓ API Gateway initialized")

	// Setup routes
	setupRoutes(gw)

	// Start server
	port := getEnv("PORT", "8000")
	log.Printf("✓ API Gateway listening on :%s", port)

	if err := http.ListenAndServe(":"+port, gw.Router()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func setupRoutes(gw *gateway.Gateway) {
	// Task endpoints
	gw.POST("/api/v1/tasks", gw.SubmitTask)
	gw.GET("/api/v1/tasks/:taskId", gw.GetTask)
	gw.GET("/api/v1/tasks", gw.ListTasks)

	// Node endpoints
	gw.GET("/api/v1/nodes", gw.ListNodes)
	gw.GET("/api/v1/nodes/:nodeId", gw.GetNode)

	// Workflow endpoints
	gw.POST("/api/v1/workflows", gw.CreateWorkflow)
	gw.GET("/api/v1/workflows/:workflowId", gw.GetWorkflow)
	gw.POST("/api/v1/workflows/:workflowId/execute", gw.ExecuteWorkflow)

	// Agent endpoints
	gw.POST("/api/v1/agents", gw.CreateAgent)
	gw.GET("/api/v1/agents/:agentId", gw.GetAgent)
	gw.POST("/api/v1/agents/:agentId/tools/:toolName", gw.ExecuteAgentTool)

	// Health check
	gw.GET("/health", gw.Health)
	gw.GET("/metrics", gw.Metrics)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
