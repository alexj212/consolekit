package consolekit

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexj212/consolekit/safemap"
)

// SocketHandler implements TransportHandler for Unix/TCP socket server.
// Provides a lightweight JSON-line protocol for programmatic access to commands.
// Designed for integration with external tools, scripts, and Claude Code skills.
//
// Protocol: Newline-delimited JSON (NDJSON)
//
//	Request:  {"id":"optional","command":"help","token":"for-tcp"}
//	Response: {"id":"optional","output":"...","error":"","success":true}
type SocketHandler struct {
	executor *CommandExecutor
	config   *TransportConfig

	// Socket config
	network   string // "unix" or "tcp"
	addr      string // socket path or host:port
	authToken string // required for TCP, empty for unix

	// Connection management
	listener    net.Listener
	connections *safemap.SafeMap[string, *SocketConnection]
	stopCh      chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
	isRunning   bool

	// Limits
	MaxConnections int           // 0 = unlimited
	IdleTimeout    time.Duration // 0 = disabled
	SocketMode     os.FileMode   // Unix socket permissions (default 0600)

	// Discovery
	InfoFile string // Path to write socket info JSON for skill/tool discovery
}

// SocketConnection represents an active socket connection.
type SocketConnection struct {
	id            string
	conn          net.Conn
	remoteAddr    string
	authenticated bool
	ctx           context.Context
	cancel        context.CancelFunc
	startTime     time.Time
	lastActivity  time.Time
	mu            sync.Mutex
}

// SocketRequest is the JSON request format for the socket protocol.
type SocketRequest struct {
	ID      string `json:"id,omitempty"`      // Optional correlation ID, echoed in response
	Command string `json:"command"`           // Command line to execute
	Token   string `json:"token,omitempty"`   // Auth token (required for TCP)
	Timeout int    `json:"timeout,omitempty"` // Optional per-request timeout in seconds (0 = no timeout)
}

// SocketResponse is the JSON response format for the socket protocol.
type SocketResponse struct {
	ID      string `json:"id,omitempty"`      // Echoed from request
	Output  string `json:"output"`            // Command output
	Error   string `json:"error,omitempty"`   // Error message, empty on success
	Success bool   `json:"success"`           // True if command succeeded
}

// NewSocketHandler creates a socket server handler.
// network is "unix" for Unix domain sockets or "tcp" for TCP.
// addr is the socket path (for Unix) or host:port (for TCP).
func NewSocketHandler(executor *CommandExecutor, network, addr string) *SocketHandler {
	return &SocketHandler{
		executor:    executor,
		network:     network,
		addr:        addr,
		connections: safemap.New[string, *SocketConnection](),
		stopCh:      make(chan struct{}),
		SocketMode:  0600,
	}
}

// SetTransportConfig sets the transport configuration (allow/deny lists, logging).
func (h *SocketHandler) SetTransportConfig(config *TransportConfig) {
	h.config = config
}

// SetAuthToken sets the required authentication token for TCP connections.
func (h *SocketHandler) SetAuthToken(token string) {
	h.authToken = token
}

// ActualAddr returns the listener's actual address, useful when binding to port 0.
// Returns empty string if the server is not running.
func (h *SocketHandler) ActualAddr() string {
	if h.listener == nil {
		return h.addr
	}
	return h.listener.Addr().String()
}

// SocketInfo contains connection details written to the info file for discovery.
type SocketInfo struct {
	Network string `json:"network"`          // "unix" or "tcp"
	Addr    string `json:"addr"`             // actual listen address
	Token   string `json:"token,omitempty"`  // auth token (TCP only)
	PID     int    `json:"pid"`              // server process ID
	App     string `json:"app,omitempty"`    // application name
}

// DefaultSocketInfoPath returns the default path for the socket info file.
func DefaultSocketInfoPath(appName string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.sockinfo.json", strings.ToLower(appName)))
}

// writeInfoFile writes socket connection details to a JSON file for discovery by tools/skills.
func (h *SocketHandler) writeInfoFile() error {
	if h.InfoFile == "" {
		return nil
	}
	info := SocketInfo{
		Network: h.network,
		Addr:    h.ActualAddr(),
		Token:   h.authToken,
		PID:     os.Getpid(),
		App:     h.executor.AppName,
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.InfoFile, data, 0600)
}

// removeInfoFile removes the socket info file on shutdown.
func (h *SocketHandler) removeInfoFile() {
	if h.InfoFile != "" {
		os.Remove(h.InfoFile)
	}
}

// DefaultNetwork returns the default network type for the current platform.
// Returns "tcp" on Windows, "unix" on all other platforms.
func DefaultNetwork() string {
	if runtime.GOOS == "windows" {
		return "tcp"
	}
	return "unix"
}

// DefaultAddr returns the default listen address for the given network and app name.
func DefaultAddr(network, appName string) string {
	if network == "tcp" {
		return "127.0.0.1:0"
	}
	return DefaultSocketPath(appName)
}

// Start begins serving commands on the socket. Blocks until stopped.
func (h *SocketHandler) Start() error {
	h.mu.Lock()
	if h.isRunning {
		h.mu.Unlock()
		return fmt.Errorf("socket server already running")
	}
	h.isRunning = true
	h.mu.Unlock()

	// For Unix sockets: remove stale socket file
	if h.network == "unix" {
		// Check if something is already listening
		testConn, err := net.DialTimeout("unix", h.addr, 500*time.Millisecond)
		if err == nil {
			testConn.Close()
			h.mu.Lock()
			h.isRunning = false
			h.mu.Unlock()
			return fmt.Errorf("another instance is already listening on %s", h.addr)
		}
		// Stale socket file, safe to remove
		os.Remove(h.addr)
	}

	listener, err := net.Listen(h.network, h.addr)
	if err != nil {
		h.mu.Lock()
		h.isRunning = false
		h.mu.Unlock()
		return fmt.Errorf("failed to listen on %s %s: %w", h.network, h.addr, err)
	}
	h.listener = listener

	// Set Unix socket permissions
	if h.network == "unix" {
		mode := h.SocketMode
		if mode == 0 {
			mode = 0600
		}
		os.Chmod(h.addr, mode)
	}

	// Update addr to actual address (important for port 0)
	h.addr = h.listener.Addr().String()

	log.Printf("Socket server listening on %s %s\n", h.network, h.addr)

	// Write info file for tool/skill discovery
	if err := h.writeInfoFile(); err != nil {
		log.Printf("Warning: failed to write socket info file: %v", err)
	}

	// Accept loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-h.stopCh:
				// Clean shutdown
				if h.network == "unix" {
					os.Remove(h.addr)
				}
				h.removeInfoFile()
				h.mu.Lock()
				h.isRunning = false
				h.mu.Unlock()
				return nil
			default:
				log.Printf("Socket accept error: %v", err)
				continue
			}
		}

		// Enable TCP keepalive to detect half-open connections
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(30 * time.Second)
		}

		// Check connection limit
		if h.MaxConnections > 0 && h.connections.Len() >= h.MaxConnections {
			conn.Close()
			continue
		}

		h.wg.Add(1)
		go h.handleConnection(conn)
	}
}

// Stop gracefully shuts down the socket server.
func (h *SocketHandler) Stop() error {
	h.mu.Lock()
	if !h.isRunning {
		h.mu.Unlock()
		return nil
	}
	h.mu.Unlock()

	close(h.stopCh)

	if h.listener != nil {
		h.listener.Close()
	}

	// Cancel and close all active connections
	h.connections.ForEach(func(id string, sc *SocketConnection) bool {
		sc.cancel()
		sc.conn.Close()
		return true
	})

	// Wait for connections to finish with timeout
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		log.Printf("Socket server shutdown timed out, forcing close")
	}

	if h.network == "unix" {
		os.Remove(h.addr)
	}

	h.removeInfoFile()

	log.Printf("Socket server stopped\n")
	return nil
}

// Name returns the transport type.
func (h *SocketHandler) Name() string {
	return "socket"
}

// handleConnection processes a single socket connection.
func (h *SocketHandler) handleConnection(conn net.Conn) {
	defer h.wg.Done()
	defer conn.Close()

	connID := h.generateConnID()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc := &SocketConnection{
		id:            connID,
		conn:          conn,
		remoteAddr:    conn.RemoteAddr().String(),
		authenticated: h.network == "unix", // Unix sockets are pre-authenticated
		ctx:           ctx,
		cancel:        cancel,
		startTime:     time.Now(),
		lastActivity:  time.Now(),
	}
	h.connections.Set(connID, sc)
	defer h.connections.Delete(connID)

	log.Printf("New socket connection %s from %s\n", connID, sc.remoteAddr)

	// Idle timeout enforcement
	if h.IdleTimeout > 0 {
		go func() {
			ticker := time.NewTicker(h.IdleTimeout / 2)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					sc.mu.Lock()
					idle := time.Since(sc.lastActivity)
					sc.mu.Unlock()
					if idle > h.IdleTimeout {
						log.Printf("Socket connection %s idle timeout (%v), closing\n", connID, h.IdleTimeout)
						cancel()
						conn.Close()
						return
					}
				case <-ctx.Done():
					return
				case <-h.stopCh:
					return
				}
			}
		}()
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max line

	for scanner.Scan() {
		select {
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req SocketRequest
		if err := json.Unmarshal(line, &req); err != nil {
			h.writeResponse(conn, SocketResponse{
				Error:   "invalid JSON: " + err.Error(),
				Success: false,
			})
			continue
		}

		// Handle TCP authentication
		if !sc.authenticated {
			if h.authToken == "" || req.Token != h.authToken {
				h.writeResponse(conn, SocketResponse{
					ID:      req.ID,
					Error:   "authentication required",
					Success: false,
				})
				continue
			}
			sc.authenticated = true
		}

		// Built-in ping/health check — bypasses executor and allow/deny
		if req.Command == "ping" {
			sc.mu.Lock()
			sc.lastActivity = time.Now()
			sc.mu.Unlock()
			h.writeResponse(conn, SocketResponse{
				ID:      req.ID,
				Output:  "pong",
				Success: true,
			})
			continue
		}

		// Built-in connection info
		if req.Command == "conninfo" {
			sc.mu.Lock()
			sc.lastActivity = time.Now()
			uptime := time.Since(sc.startTime).Round(time.Second)
			idle := time.Since(sc.lastActivity).Round(time.Second)
			sc.mu.Unlock()
			info := fmt.Sprintf("conn_id=%s remote=%s uptime=%v idle=%v auth=%v",
				sc.id, sc.remoteAddr, uptime, idle, sc.authenticated)
			h.writeResponse(conn, SocketResponse{
				ID:      req.ID,
				Output:  info,
				Success: true,
			})
			continue
		}

		// Check command allow/deny
		cmdName := req.Command
		if idx := strings.IndexAny(cmdName, " |>;"); idx != -1 {
			cmdName = cmdName[:idx]
		}
		if h.config != nil && !h.config.IsCommandAllowed(cmdName) {
			h.writeResponse(conn, SocketResponse{
				ID:      req.ID,
				Error:   fmt.Sprintf("command '%s' is not allowed", cmdName),
				Success: false,
			})
			continue
		}

		// Update activity
		sc.mu.Lock()
		sc.lastActivity = time.Now()
		sc.mu.Unlock()

		resp := h.runCommand(sc, req)
		h.writeResponse(conn, resp)
	}

	log.Printf("Socket connection %s closed\n", connID)
}

// runCommand executes a command and returns the response.
func (h *SocketHandler) runCommand(sc *SocketConnection, req SocketRequest) SocketResponse {
	// Create session-specific scope
	scope := safemap.New[string, string]()
	scope.Set("@socket:conn_id", sc.id)
	scope.Set("@socket:remote_addr", sc.remoteAddr)
	scope.Set("@socket:network", h.network)

	startTime := time.Now()

	// Apply per-request timeout if specified
	execCtx := sc.ctx
	if req.Timeout > 0 {
		var timeoutCancel context.CancelFunc
		execCtx, timeoutCancel = context.WithTimeout(sc.ctx, time.Duration(req.Timeout)*time.Second)
		defer timeoutCancel()
	}

	output, err := h.executor.ExecuteWithContext(execCtx, req.Command, scope)

	// Audit log
	if h.executor.LogManager != nil && h.executor.LogManager.IsEnabled() {
		duration := time.Since(startTime)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		h.executor.LogManager.Log(AuditLog{
			Command:   fmt.Sprintf("[Socket:%s] %s", sc.remoteAddr, req.Command),
			Timestamp: startTime,
			Duration:  duration,
			Success:   err == nil,
			Output:    output,
			Error:     errStr,
		})
	}

	if err != nil {
		return SocketResponse{
			ID:      req.ID,
			Output:  output,
			Error:   err.Error(),
			Success: false,
		}
	}

	return SocketResponse{
		ID:      req.ID,
		Output:  output,
		Success: true,
	}
}

// writeResponse encodes and writes a JSON response followed by newline.
func (h *SocketHandler) writeResponse(conn net.Conn, resp SocketResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Socket response marshal error: %v", err)
		return
	}
	data = append(data, '\n')
	conn.Write(data)
}

// ActiveConnections returns a snapshot of active connection info.
func (h *SocketHandler) ActiveConnections() []map[string]string {
	var result []map[string]string
	h.connections.ForEach(func(id string, sc *SocketConnection) bool {
		sc.mu.Lock()
		info := map[string]string{
			"id":         sc.id,
			"remote":     sc.remoteAddr,
			"uptime":     time.Since(sc.startTime).Round(time.Second).String(),
			"idle":       time.Since(sc.lastActivity).Round(time.Second).String(),
			"auth":       fmt.Sprintf("%v", sc.authenticated),
		}
		sc.mu.Unlock()
		result = append(result, info)
		return true
	})
	return result
}

// generateConnID creates a unique connection identifier.
func (h *SocketHandler) generateConnID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("conn_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ReadSocketInfo reads and parses a socket info file.
func ReadSocketInfo(path string) (*SocketInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info SocketInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("invalid socket info: %w", err)
	}
	return &info, nil
}

// IsServerRunning checks if a socket server is running by reading its info file
// and verifying the process is alive.
func IsServerRunning(infoPath string) (*SocketInfo, bool) {
	info, err := ReadSocketInfo(infoPath)
	if err != nil {
		return nil, false
	}
	// Check if process is still alive
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return info, false
	}
	// On Unix, FindProcess always succeeds. Send signal 0 to check liveness.
	// On Windows, FindProcess fails if process doesn't exist.
	if runtime.GOOS != "windows" {
		err = proc.Signal(syscall.Signal(0))
		if err != nil {
			return info, false
		}
	}
	return info, true
}

// StopServer attempts to stop a running socket server by sending SIGTERM to its process.
func StopServer(infoPath string) error {
	info, running := IsServerRunning(infoPath)
	if !running {
		// Clean up stale info file
		if info != nil {
			os.Remove(infoPath)
		}
		return fmt.Errorf("no running server found")
	}

	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return fmt.Errorf("cannot find process %d: %w", info.PID, err)
	}

	if runtime.GOOS == "windows" {
		return proc.Kill()
	}
	return proc.Signal(syscall.SIGTERM)
}
