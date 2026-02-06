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

// PromptFunc is a function that generates a command prompt for a session.
type PromptFunc func(session *SSHSession, executor *CommandExecutor) string

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

	// UI customization
	PromptFunc      PromptFunc    // Custom prompt function
	WelcomeBanner   string        // Welcome banner (displayed after login)
	MessageOfTheDay string        // MOTD (displayed after welcome)
	Banner          string        // Pre-authentication banner (RFC 4252)
	InitialHistory  []string      // Pre-populate command history with these commands

	// Session management
	IdleTimeout      time.Duration // Disconnect after inactivity (0 = disabled)
	MaxSessionTime   time.Duration // Max session duration (0 = unlimited)
	MaxConnections   int           // Max concurrent connections (0 = unlimited)
	MaxPerUser       int           // Max connections per user (0 = unlimited)

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

// ANSI color codes for terminal output
const (
	colorReset   = "\x1b[0m"
	colorRed     = "\x1b[31m"
	colorGreen   = "\x1b[32m"
	colorYellow  = "\x1b[33m"
	colorBlue    = "\x1b[34m"
	colorMagenta = "\x1b[35m"
	colorCyan    = "\x1b[36m"
	colorWhite   = "\x1b[37m"
	colorBold    = "\x1b[1m"
)

// SSHSession represents an active SSH connection.
type SSHSession struct {
	id           string
	user         string
	remoteIP     string
	channel      ssh.Channel
	requests     <-chan *ssh.Request
	env          map[string]string
	pty          *ptyInfo
	ctx          context.Context
	cancel       context.CancelFunc
	history      []string      // Command history
	historyPos   int           // Current position in history (-1 = not navigating)
	historyTemp  string        // Temporary storage for current line when navigating history
	startTime    time.Time     // Session start time
	lastActivity time.Time     // Last activity timestamp
	mu           sync.Mutex    // Mutex for updating timestamps
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

// SetCustomListener sets a custom listener.
func (h *SSHHandler) SetCustomListener(listener net.Listener) {
	h.listener = listener
}

// Start begins serving SSH connections (blocking).
func (h *SSHHandler) Start() error {
	// Create SSH server config
	sshConfig := &ssh.ServerConfig{
		NoClientAuth: h.authConfig != nil && h.authConfig.AllowAnonymous,
	}

	// Add pre-authentication banner if configured
	if h.Banner != "" {
		sshConfig.BannerCallback = func(conn ssh.ConnMetadata) string {
			return h.Banner
		}
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

		// Check connection limits
		h.sessionsMu.RLock()
		currentConnections := len(h.sessions)
		userConnections := 0
		for _, s := range h.sessions {
			if s.user == conn.User() {
				userConnections++
			}
		}
		h.sessionsMu.RUnlock()

		// Enforce max connections
		if h.MaxConnections > 0 && currentConnections >= h.MaxConnections {
			newChannel.Reject(ssh.ResourceShortage, "Maximum connections reached")
			continue
		}

		// Enforce max per user
		if h.MaxPerUser > 0 && userConnections >= h.MaxPerUser {
			newChannel.Reject(ssh.ResourceShortage, "Maximum connections per user reached")
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
		now := time.Now()

		// Initialize history with pre-populated commands if configured
		initialHistory := make([]string, 0, 100)
		if h.InitialHistory != nil && len(h.InitialHistory) > 0 {
			// Copy initial history to session (each session gets its own copy)
			initialHistory = make([]string, len(h.InitialHistory), 100)
			copy(initialHistory, h.InitialHistory)
		}

		session := &SSHSession{
			id:           sessionID,
			user:         conn.User(),
			remoteIP:     conn.RemoteAddr().String(),
			channel:      channel,
			requests:     requests,
			env:          make(map[string]string),
			ctx:          ctx,
			cancel:       cancel,
			history:      initialHistory,
			historyPos:   -1,
			historyTemp:  "",
			startTime:    now,
			lastActivity: now,
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

		// Log session end
		if h.executor.LogManager != nil && h.executor.LogManager.IsEnabled() {
			duration := time.Since(session.startTime)
			h.executor.LogManager.Log(AuditLog{
				Command:   fmt.Sprintf("[SSH Session End: %s from %s]", session.id, session.remoteIP),
				Timestamp: time.Now(),
				Duration:  duration,
				Success:   true,
				User:      session.user,
			})
		}

		fmt.Printf("[DEBUG] Session %s cleaned up\n", session.id)
	}()

	fmt.Printf("[DEBUG] handleSession started for %s\n", session.id)

	// Log session start
	if h.executor.LogManager != nil && h.executor.LogManager.IsEnabled() {
		h.executor.LogManager.Log(AuditLog{
			Command:   fmt.Sprintf("[SSH Session Start: %s from %s]", session.id, session.remoteIP),
			Timestamp: session.startTime,
			Duration:  0,
			Success:   true,
			User:      session.user,
		})
	}

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

// sessionWrite writes text to the SSH session channel, converting \n to \r\n
// when a PTY is allocated (raw terminal mode requires explicit carriage returns).
func (h *SSHHandler) sessionWrite(session *SSHSession, s string) {
	if session.pty != nil {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\n", "\r\n")
	}
	fmt.Fprint(session.channel, s)
}

// clearLine clears the current line buffer from the terminal.
func (h *SSHHandler) clearLine(session *SSHSession, line []byte, cursorPos int) {
	// Move cursor to start of line
	for i := 0; i < cursorPos; i++ {
		fmt.Fprint(session.channel, "\b")
	}

	// Clear entire line
	for i := 0; i < len(line); i++ {
		fmt.Fprint(session.channel, " ")
	}

	// Move cursor back to start
	for i := 0; i < len(line); i++ {
		fmt.Fprint(session.channel, "\b")
	}
}

// updateActivity updates the last activity timestamp for a session.
func (h *SSHHandler) updateActivity(session *SSHSession) {
	session.mu.Lock()
	session.lastActivity = time.Now()
	session.mu.Unlock()
}

// checkSessionTimeout checks if session has exceeded idle or max time limits.
// Returns true if session should be terminated.
func (h *SSHHandler) checkSessionTimeout(session *SSHSession) (bool, string) {
	session.mu.Lock()
	lastActivity := session.lastActivity
	startTime := session.startTime
	session.mu.Unlock()

	// Check max session time
	if h.MaxSessionTime > 0 {
		if time.Since(startTime) > h.MaxSessionTime {
			return true, "Maximum session time exceeded"
		}
	}

	// Check idle timeout
	if h.IdleTimeout > 0 {
		if time.Since(lastActivity) > h.IdleTimeout {
			return true, "Session idle timeout"
		}
	}

	return false, ""
}

// supportsColor checks if the session supports ANSI colors.
func (h *SSHHandler) supportsColor(session *SSHSession) bool {
	// Check NO_COLOR environment variable
	if noColor, ok := session.env["NO_COLOR"]; ok && noColor != "" {
		return false
	}

	// PTY sessions generally support colors
	return session.pty != nil
}

// colorize wraps text in ANSI color codes if colors are supported.
func (h *SSHHandler) colorize(session *SSHSession, text, color string) string {
	if !h.supportsColor(session) {
		return text
	}
	return color + text + colorReset
}

// handleShell runs an interactive shell session.
func (h *SSHHandler) handleShell(session *SSHSession) {
	fmt.Printf("[DEBUG] handleShell started for session %s\n", session.id)

	// Start session timeout monitor
	if h.IdleTimeout > 0 || h.MaxSessionTime > 0 {
		timeoutCtx, timeoutCancel := context.WithCancel(session.ctx)
		defer timeoutCancel()

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-timeoutCtx.Done():
					return
				case <-ticker.C:
					if expired, reason := h.checkSessionTimeout(session); expired {
						h.sessionWrite(session, fmt.Sprintf("\n\nSession terminated: %s\n", reason))
						session.cancel()
						return
					}
				}
			}
		}()
	}

	// Write welcome banner
	if h.WelcomeBanner != "" {
		h.sessionWrite(session, h.WelcomeBanner)
		if !strings.HasSuffix(h.WelcomeBanner, "\n") {
			h.sessionWrite(session, "\n")
		}
	} else {
		// Default welcome message
		h.sessionWrite(session, fmt.Sprintf("Welcome to %s SSH console\n", h.executor.AppName))
		h.sessionWrite(session, fmt.Sprintf("User: %s, Session: %s\n", session.user, session.id))
	}

	// Write message of the day
	if h.MessageOfTheDay != "" {
		h.sessionWrite(session, h.MessageOfTheDay)
		if !strings.HasSuffix(h.MessageOfTheDay, "\n") {
			h.sessionWrite(session, "\n")
		}
	}

	h.sessionWrite(session, "\n")

	// Generate prompt using custom function or default
	promptFunc := h.PromptFunc
	if promptFunc == nil {
		promptFunc = DefaultPrompt
	}
	prompt := promptFunc(session, h.executor)

	// Write initial prompt
	fmt.Fprint(session.channel, prompt)
	fmt.Printf("[DEBUG] Prompt written, starting interactive loop\n")

	// Read character-by-character with echo and cursor support
	var line []byte
	var cursorPos int // Current cursor position in the line buffer
	var multiLine []string // Accumulated lines for multi-line commands
	var inMultiLine bool
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

		// Update activity on input
		h.updateActivity(session)

		b := buf[0]

		switch b {
		case '\r', '\n':
			// Enter pressed - check for line continuation
			fmt.Fprint(session.channel, "\r\n")

			cmdLine := string(line)

			// Check for backslash continuation
			if strings.HasSuffix(cmdLine, "\\") {
				// Remove trailing backslash and add to multiLine
				cmdLine = strings.TrimSuffix(cmdLine, "\\")
				multiLine = append(multiLine, cmdLine)
				inMultiLine = true

				// Clear buffer and show continuation prompt
				line = line[:0]
				cursorPos = 0
				fmt.Fprint(session.channel, "> ")
				continue
			}

			// If in multi-line mode, accumulate and execute
			if inMultiLine {
				multiLine = append(multiLine, cmdLine)
				cmdLine = strings.Join(multiLine, " ")
				multiLine = multiLine[:0]
				inMultiLine = false
			}

			cmdLine = strings.TrimSpace(cmdLine)
			line = line[:0] // Clear buffer
			cursorPos = 0
			session.historyPos = -1   // Reset history position
			session.historyTemp = ""  // Clear history temp

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
				h.sessionWrite(session, "Goodbye!\n")
				return
			}

			// Add to history (avoid duplicates of last command)
			if len(session.history) == 0 || session.history[len(session.history)-1] != cmdLine {
				session.history = append(session.history, cmdLine)
				// Limit history size
				if len(session.history) > 1000 {
					session.history = session.history[1:]
				}
			}

			fmt.Printf("[DEBUG] Executing: %s\n", cmdLine)

			// Execute command and measure time
			startTime := time.Now()
			output, err := h.executeCommand(session, cmdLine)
			duration := time.Since(startTime)

			// Write output
			if output != "" {
				h.sessionWrite(session, output)
				if !strings.HasSuffix(output, "\n") {
					h.sessionWrite(session, "\n")
				}
			}

			// Write status indicator
			if err != nil {
				errorMsg := fmt.Sprintf("[ERROR] %v (%.2fs)\n", err, duration.Seconds())
				h.sessionWrite(session, h.colorize(session, errorMsg, colorRed))

				// Check if it's a "command not found" error and suggest alternatives
				if strings.Contains(err.Error(), "unknown command") ||
				   strings.Contains(err.Error(), "not found") {
					// Extract command name from line
					cmdName := strings.Fields(cmdLine)[0]
					suggestions := h.suggestCommands(cmdName)
					if len(suggestions) > 0 {
						h.sessionWrite(session, h.colorize(session,
							"Did you mean: "+strings.Join(suggestions, ", ")+"?\n",
							colorYellow))
					}
				}
			} else {
				// Only show [OK] for commands that produce output or take significant time
				if output != "" || duration > 100*time.Millisecond {
					okMsg := fmt.Sprintf("[OK] (%.2fs)\n", duration.Seconds())
					h.sessionWrite(session, h.colorize(session, okMsg, colorGreen))
				}
			}

			// Add empty line before next prompt
			h.sessionWrite(session, "\n")

			// Write next prompt
			fmt.Fprint(session.channel, prompt)

		case 127, 8: // Backspace or DEL
			if cursorPos > 0 {
				// Remove character at cursor position
				line = append(line[:cursorPos-1], line[cursorPos:]...)
				cursorPos--

				// Redraw line from cursor position
				fmt.Fprint(session.channel, "\b")                   // Move cursor back
				fmt.Fprint(session.channel, string(line[cursorPos:])) // Write rest of line
				fmt.Fprint(session.channel, " ")                     // Clear last character
				// Move cursor back to correct position
				for i := 0; i < len(line)-cursorPos+1; i++ {
					fmt.Fprint(session.channel, "\b")
				}
			}

		case 3: // Ctrl+C
			fmt.Fprint(session.channel, "^C\r\n")
			line = line[:0]
			cursorPos = 0
			fmt.Fprint(session.channel, prompt)

		case 4: // Ctrl+D (EOF)
			if len(line) == 0 {
				h.sessionWrite(session, "\nGoodbye!\n")
				return
			}

		case 1: // Ctrl+A - Move to start of line
			for cursorPos > 0 {
				fmt.Fprint(session.channel, "\b")
				cursorPos--
			}

		case 5: // Ctrl+E - Move to end of line
			for cursorPos < len(line) {
				fmt.Fprint(session.channel, string(line[cursorPos]))
				cursorPos++
			}

		case 11: // Ctrl+K - Delete from cursor to end of line
			if cursorPos < len(line) {
				// Clear from cursor to end
				deleted := len(line) - cursorPos
				line = line[:cursorPos]
				// Clear the rest of the line on screen
				for i := 0; i < deleted; i++ {
					fmt.Fprint(session.channel, " ")
				}
				// Move cursor back
				for i := 0; i < deleted; i++ {
					fmt.Fprint(session.channel, "\b")
				}
			}

		case 21: // Ctrl+U - Delete from cursor to start of line
			if cursorPos > 0 {
				// Save the part after cursor
				remaining := make([]byte, len(line)-cursorPos)
				copy(remaining, line[cursorPos:])

				// Move cursor to start
				for i := 0; i < cursorPos; i++ {
					fmt.Fprint(session.channel, "\b")
				}

				// Clear entire line
				for i := 0; i < len(line); i++ {
					fmt.Fprint(session.channel, " ")
				}
				for i := 0; i < len(line); i++ {
					fmt.Fprint(session.channel, "\b")
				}

				// Write remaining part
				line = remaining
				cursorPos = 0
				if len(remaining) > 0 {
					fmt.Fprint(session.channel, string(remaining))
					// Move cursor back to start
					for i := 0; i < len(remaining); i++ {
						fmt.Fprint(session.channel, "\b")
					}
				}
			}

		case 23: // Ctrl+W - Delete word before cursor
			if cursorPos > 0 {
				// Find start of word (skip trailing spaces first)
				wordStart := cursorPos - 1
				for wordStart > 0 && line[wordStart] == ' ' {
					wordStart--
				}
				// Find beginning of word
				for wordStart > 0 && line[wordStart-1] != ' ' {
					wordStart--
				}

				// Delete from wordStart to cursor
				deleted := cursorPos - wordStart
				remaining := append([]byte{}, line[cursorPos:]...)
				line = append(line[:wordStart], remaining...)

				// Move cursor back
				for i := 0; i < deleted; i++ {
					fmt.Fprint(session.channel, "\b")
				}

				// Redraw line
				fmt.Fprint(session.channel, string(line[wordStart:]))
				for i := 0; i < deleted; i++ {
					fmt.Fprint(session.channel, " ")
				}

				// Move cursor to correct position
				for i := 0; i < len(line)-wordStart+deleted; i++ {
					fmt.Fprint(session.channel, "\b")
				}

				cursorPos = wordStart
			}

		case 12: // Ctrl+L - Clear screen
			// Clear screen and move cursor to top
			fmt.Fprint(session.channel, "\x1b[2J\x1b[H")
			// Redraw prompt and current line
			fmt.Fprint(session.channel, prompt)
			fmt.Fprint(session.channel, string(line))
			// Move cursor to correct position
			for i := cursorPos; i < len(line); i++ {
				fmt.Fprint(session.channel, "\b")
			}

		case 9: // Tab - command completion
			if len(line) == 0 {
				continue
			}

			// Get the current word to complete
			cmdLine := string(line)
			words := strings.Fields(cmdLine)
			if len(words) == 0 {
				continue
			}

			// Complete the first word (command name)
			wordToComplete := words[0]
			if cursorPos < len(line) {
				// Find word at cursor position
				for _, word := range words {
					if strings.Index(cmdLine, word) <= cursorPos &&
						strings.Index(cmdLine, word)+len(word) >= cursorPos {
						wordToComplete = word
						break
					}
				}
			}

			// Get all matching commands
			allCommands := h.executor.GetAvailableCommands()
			matches := make([]string, 0)
			for _, cmd := range allCommands {
				if strings.HasPrefix(cmd, wordToComplete) {
					matches = append(matches, cmd)
				}
			}

			if len(matches) == 0 {
				// No matches - beep
				fmt.Fprint(session.channel, "\x07")
			} else if len(matches) == 1 {
				// Single match - complete it
				completion := matches[0][len(wordToComplete):]

				// Insert completion at cursor
				line = append(line[:cursorPos], append([]byte(completion), line[cursorPos:]...)...)

				// Echo completion
				fmt.Fprint(session.channel, completion)
				fmt.Fprint(session.channel, string(line[cursorPos+len(completion):]))

				cursorPos += len(completion)

				// Move cursor back to position
				for i := cursorPos; i < len(line); i++ {
					fmt.Fprint(session.channel, "\b")
				}
			} else {
				// Multiple matches - show them
				// Find common prefix
				commonPrefix := matches[0]
				for _, match := range matches[1:] {
					for i := 0; i < len(commonPrefix) && i < len(match); i++ {
						if commonPrefix[i] != match[i] {
							commonPrefix = commonPrefix[:i]
							break
						}
					}
				}

				// Complete to common prefix if longer than current word
				if len(commonPrefix) > len(wordToComplete) {
					completion := commonPrefix[len(wordToComplete):]
					line = append(line[:cursorPos], append([]byte(completion), line[cursorPos:]...)...)
					fmt.Fprint(session.channel, completion)
					fmt.Fprint(session.channel, string(line[cursorPos+len(completion):]))
					cursorPos += len(completion)
					for i := cursorPos; i < len(line); i++ {
						fmt.Fprint(session.channel, "\b")
					}
				} else {
					// Show all matches
					h.sessionWrite(session, "\r\n")
					for _, match := range matches {
						h.sessionWrite(session, match+"  ")
					}
					h.sessionWrite(session, "\r\n")

					// Redraw prompt and line
					fmt.Fprint(session.channel, prompt)
					fmt.Fprint(session.channel, string(line))

					// Move cursor to correct position
					for i := cursorPos; i < len(line); i++ {
						fmt.Fprint(session.channel, "\b")
					}
				}
			}

		case 27: // ESC - start of escape sequence
			// Read next two bytes to determine sequence type
			// Use a channel with timeout to handle SSH buffering issues
			escBuf := make([]byte, 2)

			// Read with a timeout to handle cases where bytes don't arrive atomically
			type readResult struct {
				n   int
				err error
			}
			readCh := make(chan readResult, 1)

			go func() {
				n, err := io.ReadFull(session.channel, escBuf)
				readCh <- readResult{n, err}
			}()

			// Wait for read with timeout
			select {
			case result := <-readCh:
				if result.err != nil || result.n != 2 {
					// Incomplete escape sequence - consume and ignore
					continue
				}
			case <-time.After(50 * time.Millisecond):
				// Timeout waiting for escape sequence completion
				// This is likely a standalone ESC key press
				continue
			}

			// Check for CSI sequence (ESC[)
			if escBuf[0] == '[' {
				switch escBuf[1] {
				case 'D': // Left arrow
					if cursorPos > 0 {
						cursorPos--
						fmt.Fprint(session.channel, "\x1b[D") // Move cursor left
					}

				case 'C': // Right arrow
					if cursorPos < len(line) {
						cursorPos++
						fmt.Fprint(session.channel, "\x1b[C") // Move cursor right
					}

				case 'A': // Up arrow - history previous
					if len(session.history) == 0 {
						continue
					}

					// Save current line if starting history navigation
					if session.historyPos == -1 {
						session.historyTemp = string(line)
						session.historyPos = len(session.history)
					}

					// Move to previous command
					if session.historyPos > 0 {
						session.historyPos--

						// Clear current line
						h.clearLine(session, line, cursorPos)

						// Load history command
						line = []byte(session.history[session.historyPos])
						cursorPos = len(line)

						// Display it
						fmt.Fprint(session.channel, string(line))
					}

				case 'B': // Down arrow - history next
					if session.historyPos == -1 {
						continue // Not in history navigation
					}

					// Move to next command
					if session.historyPos < len(session.history)-1 {
						session.historyPos++

						// Clear current line
						h.clearLine(session, line, cursorPos)

						// Load history command
						line = []byte(session.history[session.historyPos])
						cursorPos = len(line)

						// Display it
						fmt.Fprint(session.channel, string(line))
					} else {
						// Restore the original line
						h.clearLine(session, line, cursorPos)

						line = []byte(session.historyTemp)
						cursorPos = len(line)
						session.historyPos = -1

						// Display it
						fmt.Fprint(session.channel, string(line))
					}

				case 'H': // Home key
					// Move cursor to beginning
					for cursorPos > 0 {
						fmt.Fprint(session.channel, "\x1b[D")
						cursorPos--
					}

				case 'F': // End key
					// Move cursor to end
					for cursorPos < len(line) {
						fmt.Fprint(session.channel, "\x1b[C")
						cursorPos++
					}

				case '3': // Delete key (ESC[3~)
					// Read the trailing '~'
					tildeBuf := make([]byte, 1)
					if n, err := session.channel.Read(tildeBuf); err == nil && n == 1 && tildeBuf[0] == '~' {
						if cursorPos < len(line) {
							// Delete character at cursor
							line = append(line[:cursorPos], line[cursorPos+1:]...)
							// Redraw line
							fmt.Fprint(session.channel, string(line[cursorPos:]))
							fmt.Fprint(session.channel, " ") // Clear last character
							// Move cursor back
							for i := 0; i < len(line)-cursorPos+1; i++ {
								fmt.Fprint(session.channel, "\b")
							}
						}
					}
				}
			}

		default:
			// Printable character - insert at cursor position
			if b >= 32 && b < 127 {
				// Insert character at cursor position
				line = append(line[:cursorPos], append([]byte{b}, line[cursorPos:]...)...)

				// Write the new character and everything after it
				fmt.Fprint(session.channel, string(line[cursorPos:]))
				cursorPos++

				// Move cursor back to correct position
				for i := cursorPos; i < len(line); i++ {
					fmt.Fprint(session.channel, "\b")
				}
			}
		}
	}
}

// handleExec executes a single command and closes the session.
func (h *SSHHandler) handleExec(session *SSHSession, cmd string) {
	output, err := h.executeCommand(session, cmd)

	// Write output
	if output != "" {
		h.sessionWrite(session, output)
		if !strings.HasSuffix(output, "\n") {
			h.sessionWrite(session, "\n")
		}
	}

	// Write error
	if err != nil {
		errorMsg := fmt.Sprintf("Error: %v\n", err)
		h.sessionWrite(session, h.colorize(session, errorMsg, colorRed))
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

	// Log command execution (if logging is enabled)
	startTime := time.Now()

	// Execute command with session context
	output, err := h.executor.ExecuteWithContext(session.ctx, cmd, scope)

	// Log the execution result
	if h.executor.LogManager != nil && h.executor.LogManager.IsEnabled() {
		duration := time.Since(startTime)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		h.executor.LogManager.Log(AuditLog{
			Command:   fmt.Sprintf("[SSH:%s] %s", session.remoteIP, cmd),
			Timestamp: startTime,
			Duration:  duration,
			Success:   err == nil,
			Output:    output,
			Error:     errStr,
			User:      session.user,
		})
	}

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

// levenshteinDistance calculates the edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Calculate distances
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// suggestCommands finds similar command names for typos.
func (h *SSHHandler) suggestCommands(cmdName string) []string {
	// Get all available commands from executor
	commands := h.executor.GetAvailableCommands()

	// Calculate distances
	type suggestion struct {
		name     string
		distance int
	}

	suggestions := make([]suggestion, 0)
	for _, cmd := range commands {
		distance := levenshteinDistance(cmdName, cmd)
		// Only suggest if distance is reasonable (within 3 edits)
		if distance <= 3 && distance < len(cmdName) {
			suggestions = append(suggestions, suggestion{cmd, distance})
		}
	}

	// Sort by distance
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].distance < suggestions[i].distance {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Return top 3
	result := make([]string, 0, 3)
	for i := 0; i < len(suggestions) && i < 3; i++ {
		result = append(result, suggestions[i].name)
	}

	return result
}

// DefaultPrompt returns a simple prompt: "user@app > "
func DefaultPrompt(session *SSHSession, executor *CommandExecutor) string {
	return fmt.Sprintf("%s@%s > ", session.user, executor.AppName)
}

// DetailedPrompt returns a detailed prompt with session info: "[user@app:sessionID] > "
func DetailedPrompt(session *SSHSession, executor *CommandExecutor) string {
	return fmt.Sprintf("[%s@%s:%s] > ", session.user, executor.AppName, session.id[:8])
}

// MinimalPrompt returns a minimal prompt: "> "
func MinimalPrompt(session *SSHSession, executor *CommandExecutor) string {
	return "> "
}

// ColorPrompt returns a colored prompt (requires ANSI color support)
func ColorPrompt(session *SSHSession, executor *CommandExecutor) string {
	// Green user, cyan app name, white >
	return fmt.Sprintf("\x1b[32m%s\x1b[0m@\x1b[36m%s\x1b[0m > ", session.user, executor.AppName)
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
