package consolekit

import (
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

// ReflectiveAdapter wraps reeflective/console to implement DisplayAdapter.
// This is the default display backend for ConsoleKit, providing:
//   - Interactive readline-based REPL
//   - Command history with arrow key navigation
//   - Tab completion via Cobra integration
//   - Customizable prompts and hooks
type ReflectiveAdapter struct {
	app         *console.Console
	historyFile string
	promptFunc  func() string
	config      DisplayConfig
	buildCmd    func() *cobra.Command
}

// NewReflectiveAdapter creates a reeflective/console adapter.
// This adapter provides a full-featured REPL with readline support.
func NewReflectiveAdapter(appName string) *ReflectiveAdapter {
	adapter := &ReflectiveAdapter{
		app:    console.New(appName),
		config: DefaultDisplayConfig(appName),
	}

	// Set default prompt
	adapter.promptFunc = func() string {
		return appName + " > "
	}

	return adapter
}

// Start begins the interactive REPL loop.
func (r *ReflectiveAdapter) Start() error {
	return r.app.Start()
}

// SetPrompt configures the prompt function.
func (r *ReflectiveAdapter) SetPrompt(fn func() string) {
	r.promptFunc = fn

	// Update the active prompt immediately
	menu := r.app.ActiveMenu()
	prompt := menu.Prompt()
	prompt.Primary = fn
}

// AddPreCommandHook registers a hook that runs before command execution.
func (r *ReflectiveAdapter) AddPreCommandHook(hook func([]string) ([]string, error)) {
	r.app.PreCmdRunLineHooks = append(r.app.PreCmdRunLineHooks, hook)
}

// SetCommands registers the Cobra root command for the REPL.
func (r *ReflectiveAdapter) SetCommands(buildCmd func() *cobra.Command) {
	r.buildCmd = buildCmd

	// Register commands with the console menu
	menu := r.app.ActiveMenu()
	menu.SetCommands(buildCmd)
}

// SetHistoryFile configures command history persistence.
func (r *ReflectiveAdapter) SetHistoryFile(path string) {
	r.historyFile = path

	// Add history source to the console menu
	if path != "" {
		menu := r.app.ActiveMenu()
		menu.AddHistorySourceFile("main", path)
	}
}

// Configure applies display-specific options.
func (r *ReflectiveAdapter) Configure(config DisplayConfig) {
	r.config = config

	// Apply newline settings
	r.app.NewlineAfter = config.NewlineAfter
	r.app.NewlineBefore = config.NewlineBefore
	r.app.NewlineWhenEmpty = config.NewlineWhenEmpty

	// Configure shell settings
	if config.DisableAutoPairs {
		shell := r.app.Shell()
		shell.Config.Set("autopairs", false)
	}

	// Update prompt if needed
	if r.promptFunc != nil {
		menu := r.app.ActiveMenu()
		prompt := menu.Prompt()
		prompt.Primary = r.promptFunc
	}
}

// GetConsole returns the underlying console instance.
// This is provided for advanced users who need direct access to
// reeflective/console features not exposed by DisplayAdapter.
//
// Usage:
//
//	if reflective, ok := cli.GetDisplayAdapter().(*consolekit.ReflectiveAdapter); ok {
//	    app := reflective.GetConsole()
//	    // Use console-specific features
//	}
func (r *ReflectiveAdapter) GetConsole() *console.Console {
	return r.app
}
