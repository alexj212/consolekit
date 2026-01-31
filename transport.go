package consolekit

// TransportHandler defines how commands are delivered to the executor.
// Different implementations can serve commands via different protocols:
// - REPLHandler: Interactive terminal REPL
// - SSHHandler: SSH server
// - HTTPHandler: HTTP API server
// - WebSocketHandler: WebSocket server
type TransportHandler interface {
	// Start begins serving commands on this transport.
	// Blocks until transport is closed or encounters fatal error.
	Start() error

	// Stop gracefully shuts down the transport.
	// Should wait for active sessions/requests to complete (with timeout).
	Stop() error

	// Name returns the transport type (e.g., "repl", "ssh", "http").
	Name() string
}

// TransportConfig holds common transport configuration.
type TransportConfig struct {
	// Executor is the command execution engine
	Executor *CommandExecutor

	// SessionLogger logs transport-level events
	SessionLogger *LogManager

	// AllowedCommands restricts which commands can run (nil = all allowed)
	// If specified, only commands in this list are permitted
	AllowedCommands []string

	// DeniedCommands prevents specific commands (nil = none denied)
	// Takes precedence over AllowedCommands
	DeniedCommands []string
}

// IsCommandAllowed checks if a command is permitted based on allow/deny lists.
// Returns true if the command is allowed, false otherwise.
func (c *TransportConfig) IsCommandAllowed(commandName string) bool {
	// Check deny list first (takes precedence)
	if c.DeniedCommands != nil {
		for _, denied := range c.DeniedCommands {
			if denied == commandName {
				return false
			}
		}
	}

	// If allow list is specified, command must be in it
	if c.AllowedCommands != nil {
		for _, allowed := range c.AllowedCommands {
			if allowed == commandName {
				return true
			}
		}
		return false // Not in allow list
	}

	// No restrictions, allow by default
	return true
}
