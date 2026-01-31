// +build bubbletea

package consolekit

import "github.com/spf13/cobra"

// BubbletteaAdapter is a STUB implementation of DisplayAdapter for bubbletea.
// This adapter is not yet implemented and will panic if used.
//
// Purpose: This file exists to demonstrate the architecture and prove that
// the DisplayAdapter interface can support multiple backends.
//
// TODO: Implement full bubbletea integration
//   - Create bubbletea model for REPL
//   - Implement command input with history
//   - Integrate Cobra command execution
//   - Add tab completion support
//   - Handle pre-command hooks
//
// To use this adapter in the future:
//   adapter := consolekit.NewBubbletteaAdapter("myapp")
//   cli.SetDisplayAdapter(adapter)
type BubbletteaAdapter struct {
	appName     string
	historyFile string
	promptFunc  func() string
	config      DisplayConfig
	buildCmd    func() *cobra.Command
	hooks       []func([]string) ([]string, error)
}

// NewBubbletteaAdapter creates a bubbletea adapter (STUB).
// This is not yet implemented and will panic if Start() is called.
func NewBubbletteaAdapter(appName string) *BubbletteaAdapter {
	return &BubbletteaAdapter{
		appName: appName,
		config:  DefaultDisplayConfig(appName),
		promptFunc: func() string {
			return appName + " > "
		},
		hooks: make([]func([]string) ([]string, error), 0),
	}
}

// Start begins the interactive REPL loop (NOT IMPLEMENTED).
func (b *BubbletteaAdapter) Start() error {
	panic("BubbletteaAdapter.Start() not implemented - this is a stub for future development")
}

// SetPrompt configures the prompt function (STUB).
func (b *BubbletteaAdapter) SetPrompt(fn func() string) {
	b.promptFunc = fn
	// TODO: Update bubbletea model prompt
}

// AddPreCommandHook registers a pre-command hook (STUB).
func (b *BubbletteaAdapter) AddPreCommandHook(hook func([]string) ([]string, error)) {
	b.hooks = append(b.hooks, hook)
	// TODO: Apply hooks before command execution in bubbletea model
}

// SetCommands registers the Cobra root command (STUB).
func (b *BubbletteaAdapter) SetCommands(buildCmd func() *cobra.Command) {
	b.buildCmd = buildCmd
	// TODO: Integrate with bubbletea command execution
}

// SetHistoryFile configures command history persistence (STUB).
func (b *BubbletteaAdapter) SetHistoryFile(path string) {
	b.historyFile = path
	// TODO: Load/save history from file in bubbletea model
}

// Configure applies display-specific options (STUB).
func (b *BubbletteaAdapter) Configure(config DisplayConfig) {
	b.config = config
	// TODO: Apply config to bubbletea model (styles, behavior, etc.)
}
