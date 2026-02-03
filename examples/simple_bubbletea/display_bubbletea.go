package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/alexj212/consolekit"
	"github.com/spf13/cobra"
)

// BubbletteaAdapter implements DisplayAdapter using Bubbletea TUI framework.
// Provides a beautiful terminal user interface with styled output, command history,
// and keyboard navigation.
type BubbletteaAdapter struct {
	appName     string
	historyFile string
	promptFunc  func() string
	config      consolekit.DisplayConfig
	buildCmd    func() *cobra.Command
	hooks       []func([]string) ([]string, error)
	executor    *consolekit.CommandExecutor
}

// Bubbletea styles
var (
	btPromptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	btInputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	btOutputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	btErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	btSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	btHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	btHelpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// bubbletteaModel implements the Bubbletea model interface
type bubbletteaModel struct {
	adapter  *BubbletteaAdapter
	input    string
	output   []string
	history  []string
	histIdx  int
	quitting bool
}

// NewBubbletteaAdapter creates a new Bubbletea display adapter.
func NewBubbletteaAdapter(appName string) *BubbletteaAdapter {
	return &BubbletteaAdapter{
		appName: appName,
		config:  consolekit.DefaultDisplayConfig(appName),
		promptFunc: func() string {
			return appName + " > "
		},
		hooks: make([]func([]string) ([]string, error), 0),
	}
}

// Start begins the interactive REPL loop using Bubbletea.
func (b *BubbletteaAdapter) Start() error {
	// Load history if file is configured
	history := []string{}
	if b.historyFile != "" {
		history = b.loadHistory()
	}

	// Create initial model
	model := bubbletteaModel{
		adapter: b,
		input:   "",
		output:  []string{},
		history: history,
		histIdx: len(history),
	}

	// Run Bubbletea program
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()

	// Save history on exit
	if b.historyFile != "" && len(model.history) > 0 {
		b.saveHistory(model.history)
	}

	return err
}

// SetPrompt configures the prompt function.
func (b *BubbletteaAdapter) SetPrompt(fn func() string) {
	b.promptFunc = fn
}

// AddPreCommandHook registers a pre-command hook.
func (b *BubbletteaAdapter) AddPreCommandHook(hook func([]string) ([]string, error)) {
	b.hooks = append(b.hooks, hook)
}

// SetCommands registers the Cobra root command builder.
func (b *BubbletteaAdapter) SetCommands(buildCmd func() *cobra.Command) {
	b.buildCmd = buildCmd
}

// SetHistoryFile configures command history persistence.
func (b *BubbletteaAdapter) SetHistoryFile(path string) {
	b.historyFile = path
}

// Configure applies display-specific options.
func (b *BubbletteaAdapter) Configure(config consolekit.DisplayConfig) {
	b.config = config
}

// SetExecutor sets the command executor (called by REPLHandler)
func (b *BubbletteaAdapter) SetExecutor(exec *consolekit.CommandExecutor) {
	b.executor = exec
}

// loadHistory loads command history from file
func (b *BubbletteaAdapter) loadHistory() []string {
	file, err := os.Open(b.historyFile)
	if err != nil {
		return []string{}
	}
	defer file.Close()

	history := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			history = append(history, line)
		}
	}

	// Keep last 1000 commands
	if len(history) > 1000 {
		history = history[len(history)-1000:]
	}

	return history
}

// saveHistory saves command history to file
func (b *BubbletteaAdapter) saveHistory(history []string) error {
	file, err := os.Create(b.historyFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Save last 1000 commands
	start := 0
	if len(history) > 1000 {
		start = len(history) - 1000
	}

	for i := start; i < len(history); i++ {
		fmt.Fprintln(file, history[i])
	}

	return nil
}

// Bubbletea Model Implementation

func (m bubbletteaModel) Init() tea.Cmd {
	return nil
}

func (m bubbletteaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if m.input == "" {
				return m, nil
			}

			// Add to history
			m.history = append(m.history, m.input)
			m.histIdx = len(m.history)

			// Handle exit/quit
			if m.input == "exit" || m.input == "quit" {
				m.quitting = true
				return m, tea.Quit
			}

			// Execute command
			m.output = append(m.output, btPromptStyle.Render(m.adapter.promptFunc())+btInputStyle.Render(m.input))

			if m.adapter.executor != nil {
				result, err := m.adapter.executor.Execute(m.input, nil)
				if err != nil {
					m.output = append(m.output, btErrorStyle.Render("Error: "+err.Error()))
				} else if result != "" {
					lines := strings.Split(strings.TrimSpace(result), "\n")
					for _, line := range lines {
						m.output = append(m.output, btOutputStyle.Render(line))
					}
				}
			} else {
				m.output = append(m.output, btErrorStyle.Render("Error: executor not configured"))
			}

			// Keep last 100 output lines
			if len(m.output) > 100 {
				m.output = m.output[len(m.output)-100:]
			}

			m.input = ""
			return m, nil

		case "up":
			if m.histIdx > 0 {
				m.histIdx--
				m.input = m.history[m.histIdx]
			}
			return m, nil

		case "down":
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.input = m.history[m.histIdx]
			} else {
				m.histIdx = len(m.history)
				m.input = ""
			}
			return m, nil

		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
			return m, nil

		case "ctrl+u":
			m.input = ""
			return m, nil

		default:
			// Add printable characters
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m bubbletteaModel) View() string {
	if m.quitting {
		return btSuccessStyle.Render("Goodbye!\n")
	}

	var b strings.Builder

	// Header
	b.WriteString(btHeaderStyle.Render("╔═══════════════════════════════════════════════════════════╗") + "\n")
	b.WriteString(btHeaderStyle.Render(fmt.Sprintf("║  %s%-55s║", m.adapter.appName+" Console", strings.Repeat(" ", 55-len(m.adapter.appName)-8))) + "\n")
	b.WriteString(btHeaderStyle.Render("╚═══════════════════════════════════════════════════════════╝") + "\n\n")

	// Output history (show last 20 lines)
	start := 0
	if len(m.output) > 20 {
		start = len(m.output) - 20
	}
	for i := start; i < len(m.output); i++ {
		b.WriteString(m.output[i] + "\n")
	}

	// Current prompt and input
	b.WriteString("\n" + btPromptStyle.Render(m.adapter.promptFunc()) + btInputStyle.Render(m.input) + "█\n\n")

	// Help text
	helpText := btHelpStyle.Render("↑/↓: History • Enter: Execute • Ctrl+U: Clear • Ctrl+C/D: Quit")
	b.WriteString(helpText + "\n")

	return b.String()
}
