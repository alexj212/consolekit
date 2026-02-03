package consolekit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// REPLHandler implements TransportHandler for local interactive REPL.
// It wraps a CommandExecutor and adds REPL-specific functionality like:
// - Interactive display (reeflective/console, bubbletea, etc.)
// - History file management
// - Color output
// - Prompt customization
// - TTY detection and batch mode
type REPLHandler struct {
	executor *CommandExecutor

	// Display adapter for REPL interaction
	display DisplayAdapter

	// REPL-specific state
	historyFile   string
	promptFunc    func() string // Dynamic prompt function
	pendingOutput string        // For pipe/redirect detection

	// Display formatting
	NoColor       bool
	InfoString    func(format string, a ...any) string
	ErrorString   func(format string, a ...any) string
	SuccessString func(format string, a ...any) string
	WarningString func(format string, a ...any) string

	// Lifecycle callback
	OnExit func(caller string, code int)
}

// NewREPLHandler creates a new REPL handler for the given executor.
func NewREPLHandler(executor *CommandExecutor) *REPLHandler {
	handler := &REPLHandler{
		executor:      executor,
		InfoString:    color.New(color.FgCyan).SprintfFunc(),
		ErrorString:   color.New(color.FgRed).SprintfFunc(),
		SuccessString: color.New(color.FgGreen).SprintfFunc(),
		WarningString: color.New(color.FgYellow).SprintfFunc(),
	}

	// Set default prompt function
	handler.promptFunc = func() string {
		return fmt.Sprintf("%s > ", executor.AppName)
	}

	// Detect TTY for color support
	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	handler.NoColor = os.Getenv("NO_COLOR") != "" || !isTTY

	// Sync NoColor setting with executor
	executor.NoColor = handler.NoColor

	if handler.NoColor {
		handler.InfoString = fmt.Sprintf
		handler.ErrorString = fmt.Sprintf
		handler.SuccessString = fmt.Sprintf
		handler.WarningString = fmt.Sprintf
		color.NoColor = true
	}

	// Set up history file path
	currentUser, err := user.Current()
	if err == nil {
		name := strings.ToLower(executor.AppName)
		fileName := fmt.Sprintf(".%s.history", name)
		handler.historyFile = filepath.Join(currentUser.HomeDir, fileName)
	}

	// Create and configure display adapter (default: reeflective/console)
	handler.display = NewReflectiveAdapter(executor.AppName)

	// Pass executor to adapter if it supports it (e.g., BubbletteaAdapter)
	if execSetter, ok := handler.display.(interface{ SetExecutor(*CommandExecutor) }); ok {
		execSetter.SetExecutor(executor)
	}

	// Configure display settings
	handler.display.Configure(DefaultDisplayConfig(executor.AppName))

	// Set commands using RootCmd with REPL-specific root command
	handler.display.SetCommands(handler.buildREPLRootCmd())

	// Configure history file if available
	if handler.historyFile != "" {
		handler.display.SetHistoryFile(handler.historyFile)
	}

	// Set the prompt using the stored prompt function
	handler.display.SetPrompt(handler.promptFunc)

	// Add a pre-command hook for alias expansion
	// Store line in pendingOutput for root command to check for pipes/redirects/@tokens
	handler.display.AddPreCommandHook(func(args []string) ([]string, error) {
		// Skip empty input
		if len(args) == 0 {
			return args, nil
		}

		// Reconstruct the line
		line := strings.Join(args, " ")

		// Skip comments
		if strings.HasPrefix(line, "#") {
			return nil, nil
		}

		// Store line for root command to check
		handler.pendingOutput = line

		// Check for alias expansion
		originalLine := line

		// Check aliases - exact match
		executor.aliases.ForEach(func(k string, v string) bool {
			if k == line {
				line = v
				return true
			}
			return false
		})

		// Check if first word matches an alias
		firstWord := line
		if idx := strings.IndexAny(line, " |>;"); idx != -1 {
			firstWord = line[:idx]
		}

		executor.aliases.ForEach(func(k string, v string) bool {
			if k == firstWord && k != line {
				line = v + line[len(firstWord):]
				return true
			}
			return false
		})

		// If alias changed, update stored line and re-split
		if line != originalLine {
			handler.pendingOutput = line
			return strings.Fields(line), nil
		}

		return args, nil
	})

	return handler
}

// buildREPLRootCmd creates a REPL-specific root command that handles pipes/redirects.
// This wraps the executor's RootCmd but adds REPL-specific behavior.
func (h *REPLHandler) buildREPLRootCmd() func() *cobra.Command {
	return func() *cobra.Command {
		baseCmd := h.executor.RootCmd()

		// Override root command RunE to handle pipes/redirects/@tokens
		baseCmd.RunE = func(cmd *cobra.Command, args []string) error {
			// Check if we have a line with pipes/redirects/@tokens
			line := h.pendingOutput
			if line != "" && (strings.Contains(line, "|") || strings.Contains(line, ">") || strings.Contains(line, "@")) {
				// Execute through ExecuteLine which handles pipes/redirects
				output, err := h.executor.Execute(line, nil)

				// Print output with guaranteed trailing newline
				if output != "" {
					cmd.Print(output)
					if !strings.HasSuffix(output, "\n") {
						cmd.Println()
					}
				}

				// Add extra newline to ensure cursor is not on last terminal row
				// This prevents readline prompt rendering issues when at bottom of screen
				cmd.Println()

				// Explicitly sync stdout to ensure terminal has processed all output
				// before readline tries to draw the prompt. Critical for last-row rendering.
				os.Stdout.Sync()

				h.pendingOutput = "" // Clear
				return err
			}

			// Otherwise show help (default root command behavior)
			return cmd.Help()
		}

		return baseCmd
	}
}

// Start begins the REPL loop (blocking).
// This is the main entry point for interactive REPL mode.
func (h *REPLHandler) Start() error {
	return h.display.Start()
}

// Stop gracefully shuts down the REPL.
func (h *REPLHandler) Stop() error {
	// REPL doesn't have active connections to close
	return nil
}

// Name returns the transport type.
func (h *REPLHandler) Name() string {
	return "repl"
}

// Run executes command-line arguments if present, otherwise starts the REPL.
// This is the recommended entry point for CLI applications.
// Detects if stdin is piped and runs in batch mode automatically.
func (h *REPLHandler) Run() error {
	// Check if this is an MCP server command - if so, execute it directly
	// regardless of stdin status (MCP needs to read JSON-RPC from stdin)
	if h.isMCPCommand() {
		return h.ExecuteArgs(os.Args[1:])
	}

	// Check if stdin is being piped (not a TTY) - enables script piping
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return h.RunBatch()
	}

	// Check if we have command-line arguments
	if len(os.Args) > 1 {
		// Check if we have an actual command or just flags
		hasCommand := h.hasNonFlagArgs()
		if hasCommand {
			// Execute command directly without entering REPL
			return h.ExecuteArgs(os.Args[1:])
		}
	}

	// No arguments or only flags, start REPL
	return h.Start()
}

// isMCPCommand checks if the command being run is the MCP server.
// MCP server needs special handling because it reads JSON-RPC from stdin.
func (h *REPLHandler) isMCPCommand() bool {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		// Skip flags
		if strings.HasPrefix(arg, "-") {
			// Skip flag and its value if it's a flag that takes a value
			if !strings.Contains(arg, "=") {
				// Common flags that take values - skip the next arg
				if i+1 < len(os.Args) &&
					(arg == "-c" || arg == "--config" ||
						arg == "-d" || arg == "--saveDir" ||
						arg == "-s" || arg == "--save" ||
						arg == "-o" || arg == "--output" ||
						arg == "-f" || arg == "--file" ||
						arg == "-S" ||
						arg == "--profile-port") {
					i++ // Skip the next arg (flag value)
				}
			}
			continue
		}
		// First non-flag argument is the command
		return arg == "mcp"
	}
	return false
}

// hasNonFlagArgs checks if command-line arguments contain an actual command (non-flag argument).
func (h *REPLHandler) hasNonFlagArgs() bool {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// Empty arg or starts with dash = flag
		if len(arg) == 0 || arg[0] == '-' {
			// Check if this flag has a value (not --flag=value style)
			if !strings.Contains(arg, "=") {
				// Common flags that take values - skip the next arg
				if i+1 < len(os.Args) &&
					(arg == "-c" || arg == "--config" ||
						arg == "-d" || arg == "--saveDir" ||
						arg == "-s" || arg == "--save" ||
						arg == "-o" || arg == "--output" ||
						arg == "-f" || arg == "--file" ||
						arg == "-S" ||
						arg == "--profile-port") {
					i++ // Skip the next arg (flag value)
				}
			}
			continue
		}

		// Found a non-flag argument, it's a command
		return true
	}
	return false
}

// ExecuteArgs executes command-line arguments directly using Cobra.
func (h *REPLHandler) ExecuteArgs(args []string) error {
	rootCmd := h.executor.RootCmd()
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// RunBatch reads commands from stdin and executes them line by line.
// This enables piping scripts for automated testing: cat script.run | ./app
func (h *REPLHandler) RunBatch() error {
	scanner := bufio.NewScanner(os.Stdin)
	lineNum := 0
	successCount := 0
	errorCount := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Show the command being executed
		fmt.Printf("%s\n", h.InfoString("→ %s", line))

		output, err := h.executor.Execute(line, nil)
		if output != "" {
			fmt.Print(output)
			if !strings.HasSuffix(output, "\n") {
				fmt.Println()
			}
		}

		if err != nil {
			errorCount++
			fmt.Fprintf(os.Stderr, "%s\n", h.ErrorString("✗ Error at line %d: %v", lineNum, err))
			// Continue on error to run all test commands
		} else {
			successCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}

	// Return error if any commands failed (for exit code)
	if errorCount > 0 {
		return fmt.Errorf("batch completed: %d succeeded, %d failed", successCount, errorCount)
	}

	return nil
}

// Exit handles program exit.
func (h *REPLHandler) Exit(caller string, code int) {
	if h.OnExit != nil {
		h.OnExit(caller, code)
	}

	if code != 0 {
		fmt.Printf("%s: exiting with code %d\n", caller, code)
	}

	// Console app automatically saves history on exit
	os.Exit(code)
}

// SetPrompt changes the prompt function.
func (h *REPLHandler) SetPrompt(s func() string) {
	// Store the prompt function
	h.promptFunc = s

	// Update the display adapter prompt immediately
	h.display.SetPrompt(s)
}

// GetDisplayAdapter returns the current display adapter.
// Advanced users can type-assert to access adapter-specific features.
//
// Example:
//
//	if reflective, ok := handler.GetDisplayAdapter().(*consolekit.ReflectiveAdapter); ok {
//	    app := reflective.GetConsole()
//	    // Use console-specific features
//	}
func (h *REPLHandler) GetDisplayAdapter() DisplayAdapter {
	return h.display
}

// SetDisplayAdapter changes the display backend.
// This allows switching between different REPL implementations (reeflective, custom, etc.).
//
// Example:
//
//	// Use a custom adapter (e.g., from examples/simple_bubbletea/)
//	adapter := NewCustomAdapter("myapp")
//	handler.SetDisplayAdapter(adapter)
func (h *REPLHandler) SetDisplayAdapter(adapter DisplayAdapter) {
	h.display = adapter

	// Re-apply configuration
	h.display.Configure(DefaultDisplayConfig(h.executor.AppName))

	h.display.SetCommands(h.buildREPLRootCmd())
	if h.historyFile != "" {
		h.display.SetHistoryFile(h.historyFile)
	}
	if h.promptFunc != nil {
		h.display.SetPrompt(h.promptFunc)
	}

	// Re-add pre-command hook for alias expansion
	h.display.AddPreCommandHook(func(args []string) ([]string, error) {
		// Skip empty input
		if len(args) == 0 {
			return args, nil
		}

		// Reconstruct the line
		line := strings.Join(args, " ")

		// Skip comments
		if strings.HasPrefix(line, "#") {
			return nil, nil
		}

		// Store line for root command to check
		h.pendingOutput = line

		// Check for alias expansion
		originalLine := line

		// Check aliases - exact match
		h.executor.aliases.ForEach(func(k string, v string) bool {
			if k == line {
				line = v
				return true
			}
			return false
		})

		// Check if first word matches an alias
		firstWord := line
		if idx := strings.IndexAny(line, " |>;"); idx != -1 {
			firstWord = line[:idx]
		}

		h.executor.aliases.ForEach(func(k string, v string) bool {
			if k == firstWord && k != line {
				line = v + line[len(firstWord):]
				return true
			}
			return false
		})

		// If alias changed, update stored line and re-split
		if line != originalLine {
			h.pendingOutput = line
			return strings.Fields(line), nil
		}

		return args, nil
	})
}

// History and bookmark methods (REPL-specific functionality)

// getHistory reads history from file and returns as slice.
func (h *REPLHandler) getHistory() []string {
	var history []string

	if h.historyFile == "" {
		return history
	}

	file, err := os.Open(h.historyFile)
	if err != nil {
		return history
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			history = append(history, line)
		}
	}

	return history
}

// getBookmarksFile returns the path to bookmarks file.
func (h *REPLHandler) getBookmarksFile() string {
	if h.historyFile == "" {
		return ""
	}
	dir := filepath.Dir(h.historyFile)
	return filepath.Join(dir, "."+strings.ToLower(h.executor.AppName)+".bookmarks")
}

// loadBookmarks loads bookmarks from file.
func (h *REPLHandler) loadBookmarks() (map[string]*HistoryBookmark, error) {
	bookmarks := make(map[string]*HistoryBookmark)
	bookmarksFile := h.getBookmarksFile()
	if bookmarksFile == "" {
		return bookmarks, nil
	}

	data, err := os.ReadFile(bookmarksFile)
	if err != nil {
		if os.IsNotExist(err) {
			return bookmarks, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &bookmarks); err != nil {
		return nil, err
	}

	return bookmarks, nil
}

// saveBookmarks saves bookmarks to file.
func (h *REPLHandler) saveBookmarks(bookmarks map[string]*HistoryBookmark) error {
	bookmarksFile := h.getBookmarksFile()
	if bookmarksFile == "" {
		return fmt.Errorf("bookmarks file not available")
	}

	// Create parent directory if needed
	currentUser, err := user.Current()
	if err == nil {
		name := strings.ToLower(h.executor.AppName)
		dir := filepath.Join(currentUser.HomeDir, fmt.Sprintf(".%s", name))
		_ = os.MkdirAll(dir, 0755)
	}

	data, err := json.MarshalIndent(bookmarks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bookmarksFile, data, 0644)
}

// Interactive prompt methods (REPL-specific functionality)

// Confirm prompts the user for a yes/no confirmation.
func (h *REPLHandler) Confirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// Prompt prompts the user for a string input.
func (h *REPLHandler) Prompt(message string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(response)
}

// PromptDefault prompts the user for a string input with a default value.
func (h *REPLHandler) PromptDefault(message string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultValue
	}

	return response
}

// PromptPassword prompts the user for a password (simplified version).
func (h *REPLHandler) PromptPassword(message string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(response)
}

// Select prompts the user to select one option from a list.
func (h *REPLHandler) Select(message string, options []string) string {
	if len(options) == 0 {
		return ""
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(message)
	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}
	fmt.Print("Select an option (1-", len(options), ", or 0 to cancel): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	response = strings.TrimSpace(response)
	choice, err := strconv.Atoi(response)
	if err != nil || choice < 0 || choice > len(options) {
		return ""
	}

	if choice == 0 {
		return ""
	}

	return options[choice-1]
}

// SelectWithDefault prompts the user to select one option with a default.
func (h *REPLHandler) SelectWithDefault(message string, options []string, defaultIdx int) string {
	if len(options) == 0 {
		return ""
	}

	if defaultIdx < 0 || defaultIdx >= len(options) {
		defaultIdx = 0
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(message)
	for i, option := range options {
		if i == defaultIdx {
			fmt.Printf("  %d) %s (default)\n", i+1, option)
		} else {
			fmt.Printf("  %d) %s\n", i+1, option)
		}
	}
	fmt.Printf("Select an option [%d]: ", defaultIdx+1)

	response, err := reader.ReadString('\n')
	if err != nil {
		return options[defaultIdx]
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return options[defaultIdx]
	}

	choice, err := strconv.Atoi(response)
	if err != nil || choice < 1 || choice > len(options) {
		return options[defaultIdx]
	}

	return options[choice-1]
}

// MultiSelect prompts the user to select multiple options from a list.
func (h *REPLHandler) MultiSelect(message string, options []string) []string {
	if len(options) == 0 {
		return []string{}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(message)
	fmt.Println("  (Enter numbers separated by spaces, e.g., '1 3 5', or 0 to cancel)")
	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}
	fmt.Print("Select options: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return []string{}
	}

	response = strings.TrimSpace(response)
	if response == "" || response == "0" {
		return []string{}
	}

	parts := strings.Fields(response)
	selected := make([]string, 0)
	seen := make(map[int]bool)

	for _, part := range parts {
		choice, err := strconv.Atoi(part)
		if err != nil || choice < 1 || choice > len(options) {
			continue
		}

		if !seen[choice] {
			selected = append(selected, options[choice-1])
			seen[choice] = true
		}
	}

	return selected
}

// ConfirmDestructive prompts for confirmation of a destructive operation.
func (h *REPLHandler) ConfirmDestructive(message string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s\n", h.ErrorString("WARNING: This is a destructive operation!"))
	fmt.Printf("%s [yes/NO]: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes"
}

// PromptInteger prompts the user for an integer value.
func (h *REPLHandler) PromptInteger(message string, defaultValue int) int {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [%d]: ", message, defaultValue)

	response, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	response = strings.TrimSpace(response)
	if response == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(response)
	if err != nil {
		return defaultValue
	}

	return value
}
