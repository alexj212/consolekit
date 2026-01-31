package consolekit

import "github.com/spf13/cobra"

// DisplayAdapter abstracts REPL/display functionality from the command system.
// This interface allows ConsoleKit to work with different display backends
// (reeflective/console, bubbletea, go-prompt, etc.) while keeping Cobra
// for command management.
//
// Implementations should handle:
//   - Interactive REPL loop with line editing
//   - Command history persistence
//   - Tab completion (via Cobra integration)
//   - Prompt customization
//   - Pre-command hooks for input preprocessing
type DisplayAdapter interface {
	// Start begins the interactive REPL loop.
	// This method blocks until the user exits the REPL.
	// Returns an error if the REPL fails to start or encounters a fatal error.
	Start() error

	// SetPrompt configures the prompt function.
	// The function is called before each input line to generate the prompt string.
	// Example: func() string { return "myapp > " }
	SetPrompt(fn func() string)

	// AddPreCommandHook registers a hook that runs before command execution.
	// Hooks receive the parsed command arguments and can modify them or return an error.
	// Multiple hooks can be registered and will be called in registration order.
	//
	// Hooks are useful for:
	//   - Alias expansion
	//   - Input validation
	//   - Logging/auditing
	//
	// If a hook returns an error, command execution is aborted.
	AddPreCommandHook(hook func([]string) ([]string, error))

	// SetCommands registers the Cobra root command for the REPL.
	// The buildCmd function should return a fresh root command instance each time.
	// This enables proper flag reset between command executions in the REPL.
	SetCommands(buildCmd func() *cobra.Command)

	// SetHistoryFile configures command history persistence.
	// The history file stores previously executed commands for recall via arrow keys.
	// Empty string disables history persistence.
	SetHistoryFile(path string)

	// Configure applies display-specific options.
	// This allows customization of output formatting, line editing behavior, etc.
	Configure(config DisplayConfig)
}

// DisplayConfig holds display-specific configuration options.
// Not all backends may support all options. Implementations should
// document which options they respect.
type DisplayConfig struct {
	// AppName is the application name shown in prompts and window titles
	AppName string

	// NewlineAfter adds a newline after command output (default: true)
	NewlineAfter bool

	// NewlineBefore adds a newline before command output (default: true)
	NewlineBefore bool

	// NewlineWhenEmpty controls newlines for empty command output (default: false)
	NewlineWhenEmpty bool

	// DisableAutoPairs prevents automatic quote/bracket pairing in line editor (default: false)
	// This is useful when special characters like |, >, ; are used for command chaining
	DisableAutoPairs bool
}

// DefaultDisplayConfig returns the recommended default configuration.
func DefaultDisplayConfig(appName string) DisplayConfig {
	return DisplayConfig{
		AppName:          appName,
		NewlineAfter:     true,
		NewlineBefore:    true,
		NewlineWhenEmpty: false,
		DisableAutoPairs: true,
	}
}
