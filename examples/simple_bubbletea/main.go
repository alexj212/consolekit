package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/alexj212/consolekit"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

//go:embed *.run
var Data embed.FS

var (
	// BuildDate date string of when build was performed filled in by -X compile flag
	BuildDate string

	// LatestCommit date string of when build was performed filled in by -X compile flag
	LatestCommit string

	// Version string of build filled in by -X compile flag
	Version string

	// GitRepo string of the git repo url when build was performed filled in by -X compile flag
	GitRepo string

	// GitBranch string of branch in the git repo filled in by -X compile flag
	GitBranch string
)

// Styles
var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	inputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	outputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
)

type model struct {
	executor *consolekit.CommandExecutor
	input    string
	output   []string
	history  []string
	histIdx  int
	cursor   int
	quitting bool
}

func initialModel(executor *consolekit.CommandExecutor) model {
	return model{
		executor: executor,
		input:    "",
		output:   []string{},
		history:  []string{},
		histIdx:  -1,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.output = append(m.output, promptStyle.Render("simple > ")+inputStyle.Render(m.input))

			result, err := m.executor.Execute(m.input, nil)
			if err != nil {
				m.output = append(m.output, errorStyle.Render("Error: "+err.Error()))
			} else if result != "" {
				// Split output into lines for better display
				lines := strings.Split(strings.TrimSpace(result), "\n")
				for _, line := range lines {
					m.output = append(m.output, outputStyle.Render(line))
				}
			}

			// Keep last 100 lines
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
			// Add printable characters to input
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return successStyle.Render("Goodbye!\n")
	}

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("╔═══════════════════════════════════════════════════════════╗") + "\n")
	b.WriteString(headerStyle.Render("║         ConsoleKit Bubbletea REPL Example                ║") + "\n")
	b.WriteString(headerStyle.Render("╚═══════════════════════════════════════════════════════════╝") + "\n\n")

	// Output history (show last 20 lines)
	start := 0
	if len(m.output) > 20 {
		start = len(m.output) - 20
	}
	for i := start; i < len(m.output); i++ {
		b.WriteString(m.output[i] + "\n")
	}

	// Current prompt and input
	b.WriteString("\n" + promptStyle.Render("simple > ") + inputStyle.Render(m.input) + "█\n\n")

	// Help text
	helpText := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		"↑/↓: History • Enter: Execute • Ctrl+U: Clear • Ctrl+C/D: Quit",
	)
	b.WriteString(helpText + "\n")

	return b.String()
}

func main() {
	// If there are command-line arguments, execute them directly (non-interactive)
	if len(os.Args) > 1 {
		runNonInteractive()
		return
	}

	// Interactive mode with bubbletea
	runInteractive()
}

func runNonInteractive() {
	// Create command executor
	executor, err := createExecutor()
	if err != nil {
		fmt.Printf("Error creating executor: %v\n", err)
		os.Exit(1)
	}

	// Execute command-line arguments
	handler := consolekit.NewREPLHandler(executor)
	if err := handler.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runInteractive() {
	// Create command executor
	executor, err := createExecutor()
	if err != nil {
		fmt.Printf("Error creating executor: %v\n", err)
		os.Exit(1)
	}

	// Start bubbletea program
	p := tea.NewProgram(initialModel(executor), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func createExecutor() (*consolekit.CommandExecutor, error) {
	customizer := func(exec *consolekit.CommandExecutor) error {
		exec.Scripts = Data
		exec.AddBuiltinCommands()
		exec.AddCommands(consolekit.AddRun(exec, Data))

		// Add custom version command
		var verCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf("BuildDate    : %s\n", BuildDate)
			cmd.Printf("LatestCommit : %s\n", LatestCommit)
			cmd.Printf("Version      : %s\n", Version)
			cmd.Printf("GitRepo      : %s\n", GitRepo)
			cmd.Printf("GitBranch    : %s\n", GitBranch)
		}

		var verCmd = &cobra.Command{
			Use:     "version",
			Aliases: []string{"v", "ver"},
			Short:   "Show version info",
			Run:     verCmdFunc,
		}
		exec.AddCommands(func(rootCmd *cobra.Command) {
			rootCmd.AddCommand(verCmd)
		})

		return nil
	}

	return consolekit.NewCommandExecutor("simple", customizer)
}
