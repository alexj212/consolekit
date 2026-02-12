package consolekit

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alexj212/consolekit/safemap"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//go:embed web/*
var embeddedWebFiles embed.FS

// HTTPHandler implements TransportHandler for HTTP/WebSocket server.
// Provides:
// - HTTP API endpoints for command execution
// - WebSocket REPL terminal (with xterm.js)
// - Session-based authentication
// - Embedded web UI or serve from local directory
type HTTPHandler struct {
	executor *CommandExecutor
	config   *TransportConfig

	// HTTP server config
	addr         string
	httpUser     string
	httpPassword string

	// UI customization
	AppName         string   // Application name (default: ConsoleKit)
	PageTitle       string   // HTML page title (default: ConsoleKit Web Service)
	WelcomeBanner   string   // Welcome banner for web terminal
	MessageOfTheDay string   // MOTD for web terminal
	InitialHistory  []string // Pre-populate command history with these commands

	// Session management
	IdleTimeout    time.Duration // Disconnect after inactivity (0 = disabled)
	MaxSessionTime time.Duration // Max session duration (0 = unlimited)
	MaxConnections int           // Max concurrent WebSocket connections (0 = unlimited)

	// Server instance
	server    *http.Server
	router    *mux.Router
	sessions  *safemap.SafeMap[string, *WebSession]
	upgrader  websocket.Upgrader
	once      sync.Once
	isRunning bool
	mu        sync.Mutex
	startTime time.Time // Server start time for uptime calculation

	// Optional custom listener
	customListener net.Listener

	// Custom route registration callback
	customRoutesFn func(*mux.Router)
}

// WebSession represents an authenticated web session.
type WebSession struct {
	Username     string
	Expires      time.Time
	SessionID    string
	CreatedAt    time.Time
	LastActivity time.Time
	mu           sync.Mutex
}

// ReplMessage represents a WebSocket REPL message.
type ReplMessage struct {
	Type    string `json:"type"`    // "input", "output", "error"
	Message string `json:"message"` // Command or result
}

// NewHTTPHandler creates an HTTP/WebSocket server handler.
func NewHTTPHandler(executor *CommandExecutor, addr, httpUser, httpPassword string) *HTTPHandler {
	return &HTTPHandler{
		executor:     executor,
		addr:         addr,
		httpUser:     httpUser,
		httpPassword: httpPassword,
		config: &TransportConfig{
			Executor: executor,
		},
		sessions: safemap.New[string, *WebSession](),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins - modify for production
			},
		},
	}
}

// SetTransportConfig sets the transport configuration.
func (h *HTTPHandler) SetTransportConfig(config *TransportConfig) {
	h.config = config
}

// SetCustomListener sets a custom network listener.
func (h *HTTPHandler) SetCustomListener(listener net.Listener) {
	h.customListener = listener
}

// RegisterCustomRoutes registers a callback to add custom routes to the router.
// The callback is invoked during setupRoutes(), allowing routes to be added
// before the catch-all PathPrefix handler.
func (h *HTTPHandler) RegisterCustomRoutes(fn func(*mux.Router)) {
	h.customRoutesFn = fn
}

// Start begins serving HTTP/WebSocket connections (blocking).
func (h *HTTPHandler) Start() error {
	h.mu.Lock()
	if h.isRunning {
		h.mu.Unlock()
		return fmt.Errorf("HTTP server already running")
	}
	h.isRunning = true
	h.startTime = time.Now()
	h.mu.Unlock()

	// Setup routes
	h.setupRoutes()

	// Start session cleanup
	h.startSessionCleanup()

	// Create server
	h.server = &http.Server{
		Addr:              h.addr,
		Handler:           h.router,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       300 * time.Second,
	}

	// Use custom listener if provided
	var listener net.Listener
	var err error

	if h.customListener != nil {
		listener = h.customListener
		log.Printf("HTTP server using custom listener\n")
	} else {
		listener, err = net.Listen("tcp", h.addr)
		if err != nil {
			h.mu.Lock()
			h.isRunning = false
			h.mu.Unlock()
			return fmt.Errorf("failed to listen on %s: %w", h.addr, err)
		}
		log.Printf("HTTP server listening on %s\n", h.addr)
	}

	// Serve
	err = h.server.Serve(listener)
	h.mu.Lock()
	h.isRunning = false
	h.mu.Unlock()

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (h *HTTPHandler) Stop() error {
	h.mu.Lock()
	if !h.isRunning {
		h.mu.Unlock()
		return nil
	}
	h.mu.Unlock()

	if h.server == nil {
		return nil
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return h.server.Shutdown(ctx)
}

// Name returns the transport type.
func (h *HTTPHandler) Name() string {
	return "http"
}

// setupRoutes configures HTTP routes.
func (h *HTTPHandler) setupRoutes() {
	h.router = mux.NewRouter()

	// Add security headers
	h.router.Use(h.securityHeadersMiddleware)

	// API endpoints
	h.router.HandleFunc("/login", h.loginHandler).Methods("POST")
	h.router.HandleFunc("/logout", h.logoutHandler).Methods("POST")
	h.router.HandleFunc("/config", h.configHandler).Methods("GET")  // UI configuration
	h.router.HandleFunc("/repl", h.replHandler).Methods("GET")      // WebSocket REPL

	// Register custom routes (if provided) before catch-all handler
	if h.customRoutesFn != nil {
		h.customRoutesFn(h.router)
	}

	// Serve web UI (embedded or local directory) - must be last (catch-all)
	h.router.PathPrefix("/").Handler(h.fsHandler())
}

// securityHeadersMiddleware adds security headers to all responses.
func (h *HTTPHandler) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		w.Header().Set("Content-Security-Policy",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https: http:; "+
				"script-src-elem 'self' 'unsafe-inline' https://cdn.jsdelivr.net; "+
				"script-src-attr 'self' 'unsafe-inline'; "+
				"media-src 'self' https: http: blob:;")

		// Security headers
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")

		// CORS headers (modify for production)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		next.ServeHTTP(w, r)
	})
}

// fsHandler serves web files (embedded or local directory).
func (h *HTTPHandler) fsHandler() http.Handler {
	// Check if local web directory exists
	info, err := os.Stat("./web")
	if err == nil && info.IsDir() {
		log.Printf("Serving web files from local directory ./web\n")
		return http.FileServer(http.Dir("./web"))
	}

	// Serve embedded files
	sub, err := fs.Sub(embeddedWebFiles, "web")
	if err != nil {
		log.Fatalf("Unable to create embedded file system: %v", err)
	}
	log.Printf("Serving web files from embedded files\n")
	return http.FileServer(http.FS(sub))
}

// loginHandler handles login requests.
func (h *HTTPHandler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Validate credentials
	if h.httpUser != "" && h.httpPassword != "" &&
		creds.Username == h.httpUser && creds.Password == h.httpPassword {

		// Create session
		sessionToken := h.generateSessionToken()
		now := time.Now()
		session := &WebSession{
			Username:     h.httpUser,
			SessionID:    sessionToken,
			CreatedAt:    now,
			LastActivity: now,
			Expires:      time.Now().Add(24 * time.Hour),
		}
		h.sessions.Set(sessionToken, session)

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Expires:  session.Expires,
		})

		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// logoutHandler handles logout requests.
func (h *HTTPHandler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.sessions.Delete(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	w.WriteHeader(http.StatusOK)
}

// configHandler returns UI configuration as JSON.
func (h *HTTPHandler) configHandler(w http.ResponseWriter, r *http.Request) {
	appName := h.AppName
	pageTitle := h.PageTitle
	welcome := h.WelcomeBanner

	// Set defaults if not configured
	if appName == "" {
		appName = "ConsoleKit"
	}
	if pageTitle == "" {
		pageTitle = "ConsoleKit Web Service"
	}
	if welcome == "" {
		welcome = "Welcome to ConsoleKit Web Terminal"
	}

	// Calculate uptime
	uptime := time.Since(h.startTime)

	config := map[string]interface{}{
		"appName":        appName,
		"pageTitle":      pageTitle,
		"welcome":        welcome,
		"motd":           h.MessageOfTheDay,
		"initialHistory": h.InitialHistory, // Pre-populate command history
		"uptimeSeconds":  int(uptime.Seconds()),
		"uptimeString":   formatUptime(uptime),
		"startTime":      h.startTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// formatUptime formats a duration into a human-readable uptime string
func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// replHandler handles WebSocket REPL connections.
func (h *HTTPHandler) replHandler(w http.ResponseWriter, r *http.Request) {
	// Check session authentication
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	session, ok := h.sessions.Get(cookie.Value)
	if !ok || time.Now().After(session.Expires) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket REPL connection from %s (user: %s)\n",
		r.RemoteAddr, session.Username)

	// Handle WebSocket messages
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		var msg ReplMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.sendJSON(conn, ReplMessage{
				Type:    "error",
				Message: "Invalid JSON format",
			})
			continue
		}

		switch msg.Type {
		case "input":
			output, err := h.runCommand(session, msg.Message)
			if err != nil {
				h.sendJSON(conn, ReplMessage{
					Type:    "error",
					Message: err.Error(),
				})
			} else {
				h.sendJSON(conn, ReplMessage{
					Type:    "output",
					Message: output,
				})
			}

		default:
			h.sendJSON(conn, ReplMessage{
				Type:    "error",
				Message: "Unknown message type: " + msg.Type,
			})
		}
	}

	log.Printf("WebSocket REPL connection closed for %s\n", session.Username)
}

// runCommand executes a command and returns the output.
func (h *HTTPHandler) runCommand(session *WebSession, input string) (string, error) {
	// Update activity timestamp
	session.mu.Lock()
	session.LastActivity = time.Now()
	session.mu.Unlock()

	// Create session-specific defaults
	scope := safemap.New[string, string]()
	scope.Set("@http:user", session.Username)
	scope.Set("@http:session_id", session.SessionID)

	// Log command execution
	startTime := time.Now()

	// Execute command
	output, err := h.executor.Execute(input, scope)

	// Log the execution result
	if h.executor.LogManager != nil && h.executor.LogManager.IsEnabled() {
		duration := time.Since(startTime)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		h.executor.LogManager.Log(AuditLog{
			Command:   fmt.Sprintf("[WebSocket] %s", input),
			Timestamp: startTime,
			Duration:  duration,
			Success:   err == nil,
			Output:    output,
			Error:     errStr,
			User:      session.Username,
		})
	}

	if err != nil {
		return "", err
	}

	return output, nil
}

// sendJSON sends a JSON message over WebSocket.
func (h *HTTPHandler) sendJSON(conn *websocket.Conn, msg ReplMessage) {
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

// generateSessionToken generates a random session token.
func (h *HTTPHandler) generateSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Failed to generate random session token: %v", err)
		return fmt.Sprintf("fallback_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// startSessionCleanup starts periodic session cleanup.
func (h *HTTPHandler) startSessionCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			now := time.Now()
			h.sessions.Remove(func(token string, session *WebSession) bool {
				return session.Expires.Before(now)
			})
		}
	}()
}
