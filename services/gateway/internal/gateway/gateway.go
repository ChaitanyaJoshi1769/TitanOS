package gateway

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/gateway/internal/auth"
)

type Gateway struct {
	authManager *auth.AuthManager
	routes      map[string]Handler
	middleware  []Middleware
	metrics     *Metrics
}

type Handler func(w http.ResponseWriter, r *http.Request)
type Middleware func(Handler) Handler

type Metrics struct {
	RequestsTotal   int64
	RequestsSuccess int64
	RequestsFailed  int64
	AvgLatency      float64
}

func NewGateway(authManager *auth.AuthManager) *Gateway {
	return &Gateway{
		authManager: authManager,
		routes:      make(map[string]Handler),
		middleware:  make([]Middleware, 0),
		metrics:     &Metrics{},
	}
}

func (g *Gateway) Router() http.Handler {
	mux := http.NewServeMux()

	// Auth middleware wrapper
	mux.HandleFunc("/api/v1/", g.withAuth(g.handleAPI))
	mux.HandleFunc("/health", g.Health)
	mux.HandleFunc("/metrics", g.Metrics)

	return mux
}

func (g *Gateway) withAuth(next Handler) Handler {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !g.authManager.ValidateToken(token) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (g *Gateway) handleAPI(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	g.metrics.RequestsTotal++

	w.Header().Set("Content-Type", "application/json")

	// Route handling
	switch {
	case r.Method == "POST" && r.URL.Path == "/api/v1/tasks":
		g.SubmitTask(w, r)
	case r.Method == "GET" && len(r.URL.Path) > len("/api/v1/tasks/"):
		g.GetTask(w, r)
	case r.Method == "GET" && r.URL.Path == "/api/v1/tasks":
		g.ListTasks(w, r)
	default:
		http.NotFound(w, r)
		return
	}

	g.metrics.AvgLatency = time.Since(start).Seconds()
}

func (g *Gateway) SubmitTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID  string                 `json:"taskId"`
		Project string                 `json:"projectId"`
		Name    string                 `json:"name"`
		Input   map[string]interface{} `json:"inputData"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Forward to scheduler (simplified - direct response)
	response := map[string]interface{}{
		"taskId":  req.TaskID,
		"success": true,
		"message": "Task submitted",
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) GetTask(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"id":     "test-task",
		"status": "pending",
		"name":   "Test Task",
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) ListTasks(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"tasks": []interface{}{},
		"total": 0,
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) ListNodes(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"nodes": []interface{}{},
		"total": 0,
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) GetNode(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"id":   "node-1",
		"status": "healthy",
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"workflowId": "workflow-1",
		"success":    true,
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"id":   "workflow-1",
		"name": "Test Workflow",
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"executionId": "exec-1",
		"success":     true,
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) CreateAgent(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"agentId": "agent-1",
		"success": true,
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) GetAgent(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"id":     "agent-1",
		"status": "online",
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) ExecuteAgentTool(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"result": "Tool executed",
		"success": true,
	}

	json.NewEncoder(w).Encode(response)
	g.metrics.RequestsSuccess++
}

func (g *Gateway) POST(path string, handler Handler) {
	g.routes["POST:"+path] = handler
}

func (g *Gateway) GET(path string, handler Handler) {
	g.routes["GET:"+path] = handler
}

func (g *Gateway) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

func (g *Gateway) Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g.metrics)
}
