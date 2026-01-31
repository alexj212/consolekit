package consolekit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/alexj212/consolekit/safemap"
	"golang.org/x/crypto/ssh"
)

// SSHHandler implements TransportHandler for SSH server.
// Provides multi-session SSH access to ConsoleKit commands with:
// - Public key and password authentication
// - Interactive shell sessions
// - Single command execution (exec)
// - PTY support for terminal applications
// - Per-session context and logging
type SSHHandler struct {
	executor *CommandExecutor
	config   *TransportConfig

	// SSH server config
	addr       string       // Listen address (e.g., ":2222")
	hostKey    ssh.Signer   // Server host key
	authConfig *SSHAuthConfig

	// Server instance
	listener   net.Listener
	sessions   map[string]*SSHSession
	sessionsMu sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// SSHAuthConfig configures SSH authentication methods.
type SSHAuthConfig struct {
	// PublicKeyAuth validates public key authentication.
	// Return nil error and permissions on success.
	PublicKeyAuth func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

	// PasswordAuth validates password authentication.
	// Return nil error and permissions on success.
	PasswordAuth func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)

	// AllowAnonymous allows connections without authentication (development only).
	AllowAnonymous bool
}

// SSHSession represents an active SSH connection.
type SSHSession struct {
	id       string
	user     string
	remoteIP string
	channel  ssh.Channel
	requests <-chan *ssh.Request
	env      map[string]string
	pty      *ptyInfo
	ctx      context.Context
	cancel   context.CancelFunc
}

// ptyInfo stores PTY configuration.
type ptyInfo struct {
	term   string
	width  uint32
	height uint32
}

// NewSSHHandler creates an SSH server handler.
func NewSSHHandler(executor *CommandExecutor, addr string, hostKey ssh.Signer) *SSHHandler {
	return &SSHHandler{
		executor: executor,
		config: &TransportConfig{
			Executor: executor,
		},
		addr:     addr,
		hostKey:  hostKey,
		sessions: make(map[string]*SSHSession),
		stopCh:   make(chan struct{}),
	}
}

// SetAuthConfig configures SSH authentication.
func (h *SSHHandler) SetAuthConfig(config *SSHAuthConfig) {
	h.authConfig = config
}

// SetTransportConfig sets the transport configuration.
func (h *SSHHandler) SetTransportConfig(config *TransportConfig) {
	h.config = config
}

// SetCustomListener sets a custom listener (e.g., Tailscale).
func (h *SSHHandler) SetCustomListener(listener net.Listener) {
	h.listener = listener
}

// Start begins serving SSH connections (blocking).
func (h *SSHHandler) Start() error {
	// Create SSH server config
	sshConfig := &ssh.ServerConfig{
		NoClientAuth: h.authConfig != nil && h.authConfig.AllowAnonymous,
	}

	// Add authentication methods
	if h.authConfig != nil {
		if h.authConfig.PublicKeyAuth != nil {
			sshConfig.PublicKeyCallback = h.authConfig.PublicKeyAuth
		}
		if h.authConfig.PasswordAuth != nil {
			sshConfig.PasswordCallback = h.authConfig.PasswordAuth
		}
	}

	// Add host key
	sshConfig.AddHostKey(h.hostKey)

	// Start listening
	listener, err := net.Listen("tcp", h.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", h.addr, err)
	}
	h.listener = listener

	fmt.Printf("SSH server listening on %s\n", h.addr)

	// Accept connections
	for {
		select {
		case <-h.stopCh:
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-h.stopCh:
				return nil
			default:
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}
		}

		h.wg.Add(1)
		go h.handleConnection(conn, sshConfig)
	}
}

// Stop gracefully shuts down the SSH server.
func (h *SSHHandler) Stop() error {
	close(h.stopCh)

	// Close listener to stop accepting new connections
	if h.listener != nil {
		h.listener.Close()
	}

	// Close all active sessions
	h.sessionsMu.Lock()
	for _, session := range h.sessions {
		if session.cancel != nil {
			session.cancel()
		}
		if session.channel != nil {
			session.channel.Close()
		}
	}
	h.sessionsMu.Unlock()

	// Wait for all sessions to complete (with timeout)
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("shutdown timeout waiting for sessions to complete")
	}
}

// Name returns the transport type.
func (h *SSHHandler) Name() string {
	return "ssh"
}

// handleConnection processes a new SSH connection.
func (h *SSHHandler) handleConnection(nConn net.Conn, config *ssh.ServerConfig) {
	defer h.wg.Done()

	// Perform SSH handshake
	conn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		fmt.Printf("Failed to handshake: %v\n", err)
		return
	}
	defer conn.Close()

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Printf("Could not accept channel: %v\n", err)
			continue
		}

		// Create session
		sessionID := fmt.Sprintf("ssh-%d", time.Now().UnixNano())
		ctx, cancel := context.WithCancel(context.Background())
		session := &SSHSession{
			id:       sessionID,
			user:     conn.User(),
			remoteIP: conn.RemoteAddr().String(),
			channel:  channel,
			requests: requests,
			env:      make(map[string]string),
			ctx:      ctx,
			cancel:   cancel,
		}

		// Store session
		h.sessionsMu.Lock()
		h.sessions[sessionID] = session
		h.sessionsMu.Unlock()

		// Handle session in goroutine
		h.wg.Add(1)
		go h.handleSession(session)
	}
}

// handleSession processes an SSH session (shell or exec).
func (h *SSHHandler) handleSession(session *SSHSession) {
	defer h.wg.Done()
	defer session.channel.Close()
	defer func() {
		h.sessionsMu.Lock()
		delete(h.sessions, session.id)
		h.sessionsMu.Unlock()
		fmt.Printf("[DEBUG] Session %s cleaned up\n", session.id)
	}()

	fmt.Printf("[DEBUG] handleSession started for %s\n", session.id)

	// Process session requests
	for req := range session.requests {
		fmt.Printf("[DEBUG] Received request type: %s\n", req.Type)
		switch req.Type {
		case "pty-req":
			// Parse PTY request
			session.pty = h.parsePtyRequest(req.Payload)
			req.Reply(true, nil)

		case "env":
			// Parse environment variable
			key, value := h.parseEnvRequest(req.Payload)
			session.env[key] = value
			req.Reply(true, nil)

		case "shell":
			// Interactive shell
			fmt.Printf("[DEBUG] Shell request received, starting shell\n")
			req.Reply(true, nil)
			h.handleShell(session)
			fmt.Printf("[DEBUG] handleShell returned, exiting handleSession\n")
			return

		case "exec":
			// Single command execution
			cmd := h.parseExecRequest(req.Payload)
			req.Reply(true, nil)
			h.handleExec(session, cmd)
			return

		case "window-change":
			// Update PTY size
			if session.pty != nil {
				session.pty = h.parsePtyRequest(req.Payload)
			}
			req.Reply(true, nil)

		default:
			req.Reply(false, nil)
		}
	}
}

// handleShell runs an interactive shell session.
func (h *SSHHandler) handleShell(session *SSHSession) {
	fmt.Printf("[DEBUG] handleShell started for session %s\n", session.id)

	// Write welcome message
	fmt.Fprintf(session.channel, "Welcome to %s SSH console\n", h.executor.AppName)
	fmt.Fprintf(session.channel, "User: %s, Session: %s\n\n", session.user, session.id)

	prompt := fmt.Sprintf("%s@%s > ", session.user, h.executor.AppName)

	// Write initial prompt
	fmt.Fprint(session.channel, prompt)
	fmt.Printf("[DEBUG] Prompt written, starting interactive loop\n")

	// Read character-by-character with echo
	var line []byte
	buf := make([]byte, 1)

	for {
		// Read one byte at a time
		n, err := session.channel.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[DEBUG] Read error: %v\n", err)
			} else {
				fmt.Printf("[DEBUG] EOF received\n")
			}
			return
		}

		if n == 0 {
			continue
		}

		b := buf[0]

		switch b {
		case '\r', '\n':
			// Enter pressed - execute command
			fmt.Fprint(session.channel, "\r\n")

			cmdLine := strings.TrimSpace(string(line))
			line = line[:0] // Clear buffer

			// Skip empty lines
			if cmdLine == "" {
				fmt.Fprint(session.channel, prompt)
				continue
			}

			// Skip comments
			if strings.HasPrefix(cmdLine, "#") {
				fmt.Fprint(session.channel, prompt)
				continue
			}

			// Handle exit command
			if cmdLine == "exit" || cmdLine == "quit" {
				fmt.Fprintln(session.channel, "Goodbye!")
				return
			}

			fmt.Printf("[DEBUG] Executing: %s\n", cmdLine)

			// Execute command
			output, err := h.executeCommand(session, cmdLine)

			// Write output
			if output != "" {
				fmt.Fprint(session.channel, output)
				if !strings.HasSuffix(output, "\n") {
					fmt.Fprintln(session.channel)
				}
			}

			// Write error
			if err != nil {
				fmt.Fprintf(session.channel, "Error: %v\n", err)
			}

			// Write next prompt
			fmt.Fprint(session.channel, prompt)

		case 127, 8: // Backspace or DEL
			if len(line) > 0 {
				line = line[:len(line)-1]
				// Erase character: backspace, space, backspace
				fmt.Fprint(session.channel, "\b \b")
			}

		case 3: // Ctrl+C
			fmt.Fprint(session.channel, "^C\r\n")
			line = line[:0]
			fmt.Fprint(session.channel, prompt)

		case 4: // Ctrl+D (EOF)
			if len(line) == 0 {
				fmt.Fprintln(session.channel, "\nGoodbye!")
				return
			}

		default:
			// Echo character back and add to buffer
			if b >= 32 && b < 127 {
				session.channel.Write([]byte{b})
				line = append(line, b)
			}
		}
	}
}

// handleExec executes a single command and closes the session.
func (h *SSHHandler) handleExec(session *SSHSession, cmd string) {
	output, err := h.executeCommand(session, cmd)

	// Write output
	if output != "" {
		fmt.Fprint(session.channel, output)
		if !strings.HasSuffix(output, "\n") {
			fmt.Fprintln(session.channel)
		}
	}

	// Write error
	if err != nil {
		fmt.Fprintf(session.channel, "Error: %v\n", err)
		session.channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{1}))
		return
	}

	// Send success exit status
	session.channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
}

// executeCommand runs a command in the session context.
func (h *SSHHandler) executeCommand(session *SSHSession, cmd string) (string, error) {
	// Check command filtering
	cmdName := cmd
	if idx := strings.IndexAny(cmd, " |>;"); idx != -1 {
		cmdName = cmd[:idx]
	}

	if h.config != nil && !h.config.IsCommandAllowed(cmdName) {
		return "", fmt.Errorf("command '%s' is not allowed", cmdName)
	}

	// Create session-specific defaults (for environment variables, etc.)
	scope := safemap.New[string, string]()

	// Add session environment variables as @ssh:VARNAME tokens
	for k, v := range session.env {
		scope.Set(fmt.Sprintf("@ssh:%s", k), v)
	}

	// Add session metadata
	scope.Set("@ssh:user", session.user)
	scope.Set("@ssh:remote_ip", session.remoteIP)
	scope.Set("@ssh:session_id", session.id)

	// Execute command with session context
	output, err := h.executor.ExecuteWithContext(session.ctx, cmd, scope)

	return output, err
}

// parsePtyRequest parses a PTY request payload.
func (h *SSHHandler) parsePtyRequest(payload []byte) *ptyInfo {
	if len(payload) < 16 {
		return nil
	}

	termLen := int(payload[3])
	if len(payload) < 4+termLen+8 {
		return nil
	}

	term := string(payload[4 : 4+termLen])
	width := uint32(payload[4+termLen])<<24 | uint32(payload[4+termLen+1])<<16 |
		uint32(payload[4+termLen+2])<<8 | uint32(payload[4+termLen+3])
	height := uint32(payload[4+termLen+4])<<24 | uint32(payload[4+termLen+5])<<16 |
		uint32(payload[4+termLen+6])<<8 | uint32(payload[4+termLen+7])

	return &ptyInfo{
		term:   term,
		width:  width,
		height: height,
	}
}

// parseEnvRequest parses an environment variable request.
func (h *SSHHandler) parseEnvRequest(payload []byte) (string, string) {
	if len(payload) < 8 {
		return "", ""
	}

	keyLen := int(payload[3])
	if len(payload) < 8+keyLen {
		return "", ""
	}

	key := string(payload[4 : 4+keyLen])
	valueLen := int(payload[4+keyLen+3])
	if len(payload) < 8+keyLen+valueLen {
		return key, ""
	}

	value := string(payload[8+keyLen : 8+keyLen+valueLen])
	return key, value
}

// parseExecRequest parses an exec request payload.
func (h *SSHHandler) parseExecRequest(payload []byte) string {
	if len(payload) < 4 {
		return ""
	}

	cmdLen := int(payload[3])
	if len(payload) < 4+cmdLen {
		return ""
	}

	return string(payload[4 : 4+cmdLen])
}

// GetActiveSessions returns a list of active session IDs.
func (h *SSHHandler) GetActiveSessions() []string {
	h.sessionsMu.RLock()
	defer h.sessionsMu.RUnlock()

	sessions := make([]string, 0, len(h.sessions))
	for id := range h.sessions {
		sessions = append(sessions, id)
	}
	return sessions
}

// GetSessionInfo returns information about a specific session.
func (h *SSHHandler) GetSessionInfo(sessionID string) (user, remoteIP string, ok bool) {
	h.sessionsMu.RLock()
	defer h.sessionsMu.RUnlock()

	session, exists := h.sessions[sessionID]
	if !exists {
		return "", "", false
	}

	return session.user, session.remoteIP, true
}

// GenerateHostKey generates a new RSA host key for testing.
// For production use, load a persistent key from disk.
func GenerateHostKey() (ssh.Signer, error) {
	// Generate 2048-bit RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create SSH signer from private key
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return signer, nil
}

// LoadHostKeyFromFile loads an RSA private key from a PEM file.
// import "crypto/x509"
//
// func generateHostKey() (ssh.Signer, error) {
//     privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
//     if err != nil {
//         return nil, err
//     }
//
//     privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
//     signer, err := ssh.NewSignerFromKey(privateKey)
//     if err != nil {
//         return nil, err
//     }
//
//     return signer, nil
// }
