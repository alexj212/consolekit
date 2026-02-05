package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alexj212/consolekit"
	"github.com/alexj212/consolekit/safemap"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

//go:embed scripts/*.run
var Data embed.FS

var (
	BuildDate    string
	LatestCommit string
	Version      string
)

func main() {
	// Create command executor
	customizer := func(exec *consolekit.CommandExecutor) error {
		exec.Scripts = &Data
		exec.AddBuiltinCommands()
		exec.AddCommands(consolekit.AddRun(exec, &Data))

		// Add version command
		var verCmd = &cobra.Command{
			Use:     "version",
			Aliases: []string{"v", "ver"},
			Short:   "Show version info",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Printf("REST API Example\n")
				cmd.Printf("BuildDate    : %s\n", BuildDate)
				cmd.Printf("LatestCommit : %s\n", LatestCommit)
				cmd.Printf("Version      : %s\n", Version)
			},
		}
		exec.AddCommands(func(rootCmd *cobra.Command) {
			rootCmd.AddCommand(verCmd)
		})

		return nil
	}

	executor, err := consolekit.NewCommandExecutor("rest-api", customizer)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Create custom HTTP router with REST API endpoints
	router := mux.NewRouter()

	// Add REST API endpoints
	api := &APIServer{executor: executor}
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Command execution endpoint
	apiRouter.HandleFunc("/execute", api.executeCommand).Methods("POST")

	// Job management endpoints
	apiRouter.HandleFunc("/jobs", api.listJobs).Methods("GET")
	apiRouter.HandleFunc("/jobs/{id}", api.getJob).Methods("GET")
	apiRouter.HandleFunc("/jobs/{id}/kill", api.killJob).Methods("POST")

	// Variable management endpoints
	apiRouter.HandleFunc("/variables", api.listVariables).Methods("GET")
	apiRouter.HandleFunc("/variables/{name}", api.getVariable).Methods("GET")
	apiRouter.HandleFunc("/variables/{name}", api.setVariable).Methods("PUT")
	apiRouter.HandleFunc("/variables/{name}", api.deleteVariable).Methods("DELETE")

	// Health check endpoint
	apiRouter.HandleFunc("/health", api.healthCheck).Methods("GET")

	// System info endpoint
	apiRouter.HandleFunc("/info", api.systemInfo).Methods("GET")

	// Create standalone HTTP server for REST API
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Print API information
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           ConsoleKit REST API Server Example              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println()
	fmt.Println("Web Terminal:")
	fmt.Println("  http://localhost:8080/admin")
	fmt.Println("  Username: admin")
	fmt.Println("  Password: secret123")
	fmt.Println()
	fmt.Println("REST API Endpoints:")
	fmt.Println("  POST   /api/v1/execute              - Execute command")
	fmt.Println("  GET    /api/v1/jobs                 - List background jobs")
	fmt.Println("  GET    /api/v1/jobs/{id}            - Get job details")
	fmt.Println("  POST   /api/v1/jobs/{id}/kill       - Kill a job")
	fmt.Println("  GET    /api/v1/variables            - List variables")
	fmt.Println("  GET    /api/v1/variables/{name}     - Get variable")
	fmt.Println("  PUT    /api/v1/variables/{name}     - Set variable")
	fmt.Println("  DELETE /api/v1/variables/{name}     - Delete variable")
	fmt.Println("  GET    /api/v1/health               - Health check")
	fmt.Println("  GET    /api/v1/info                 - System info")
	fmt.Println()
	fmt.Println("Example API calls:")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/execute \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"command\": \"print \\\"Hello from API\\\"\"}'")
	fmt.Println()
	fmt.Println("  curl http://localhost:8080/api/v1/health")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()

	// Start server in goroutine
	go func() {
		log.Printf("Starting REST API server on http://localhost:8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

// APIServer handles REST API requests
type APIServer struct {
	executor *consolekit.CommandExecutor
}

// ExecuteRequest represents a command execution request
type ExecuteRequest struct {
	Command string            `json:"command"`
	Timeout int               `json:"timeout,omitempty"` // seconds
	Env     map[string]string `json:"env,omitempty"`
}

// ExecuteResponse represents a command execution response
type ExecuteResponse struct {
	Output   string  `json:"output"`
	Error    string  `json:"error,omitempty"`
	Duration float64 `json:"duration_ms"`
	Success  bool    `json:"success"`
}

// executeCommand handles POST /api/v1/execute
func (api *APIServer) executeCommand(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, "Command is required", http.StatusBadRequest)
		return
	}

	// Create context with timeout
	ctx := context.Background()
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}

	// Create scoped defaults for environment variables
	scope := safemap.New[string, string]()
	for k, v := range req.Env {
		scope.Set(fmt.Sprintf("@%s", k), v)
	}

	// Execute command
	start := time.Now()
	output, err := api.executor.ExecuteWithContext(ctx, req.Command, scope)
	duration := time.Since(start)

	// Build response
	resp := ExecuteResponse{
		Output:   output,
		Duration: float64(duration.Milliseconds()),
		Success:  err == nil,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// listJobs handles GET /api/v1/jobs
func (api *APIServer) listJobs(w http.ResponseWriter, r *http.Request) {
	jobs := api.executor.JobManager.List()

	jobList := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		jobList = append(jobList, map[string]interface{}{
			"id":      job.ID,
			"command": job.Command,
			"status":  job.Status,
			"pid":     job.PID,
			"started": job.StartTime,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs":  jobList,
		"count": len(jobList),
	})
}

// getJob handles GET /api/v1/jobs/{id}
func (api *APIServer) getJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, ok := api.executor.JobManager.Get(id)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       job.ID,
		"command":  job.Command,
		"status":   job.Status,
		"pid":      job.PID,
		"started":  job.StartTime,
		"finished": job.EndTime,
		"output":   job.Output.String(),
	})
}

// killJob handles POST /api/v1/jobs/{id}/kill
func (api *APIServer) killJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	if err := api.executor.JobManager.Kill(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Job killed",
	})
}

// listVariables handles GET /api/v1/variables
func (api *APIServer) listVariables(w http.ResponseWriter, r *http.Request) {
	variables := make(map[string]string)
	api.executor.Variables.ForEach(func(k string, v string) bool {
		variables[k] = v
		return false
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"variables": variables,
		"count":     len(variables),
	})
}

// getVariable handles GET /api/v1/variables/{name}
func (api *APIServer) getVariable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	value, ok := api.executor.Variables.Get("@" + name)
	if !ok {
		http.Error(w, "Variable not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":  name,
		"value": value,
	})
}

// setVariable handles PUT /api/v1/variables/{name}
func (api *APIServer) setVariable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api.executor.Variables.Set("@"+name, req.Value)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"name":    name,
		"value":   req.Value,
	})
}

// deleteVariable handles DELETE /api/v1/variables/{name}
func (api *APIServer) deleteVariable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	api.executor.Variables.Delete("@" + name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Variable deleted",
	})
}

// healthCheck handles GET /api/v1/health
func (api *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": Version,
		"uptime":  time.Since(startTime).String(),
	})
}

var startTime = time.Now()

// systemInfo handles GET /api/v1/info
func (api *APIServer) systemInfo(w http.ResponseWriter, r *http.Request) {
	jobs := api.executor.JobManager.List()
	runningJobs := 0
	for _, job := range jobs {
		if job.Status == "running" {
			runningJobs++
		}
	}

	varCount := 0
	api.executor.Variables.ForEach(func(k string, v string) bool {
		varCount++
		return false
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"app_name":      api.executor.AppName,
		"version":       Version,
		"build_date":    BuildDate,
		"commit":        LatestCommit,
		"uptime":        time.Since(startTime).String(),
		"jobs_total":    len(jobs),
		"jobs_running":  runningJobs,
		"variables":     varCount,
		"logging":       api.executor.LogManager.IsEnabled(),
		"max_recursion": 10,
	})
}
