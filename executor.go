package consolekit

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alexj212/consolekit/parser"
	"github.com/alexj212/consolekit/safemap"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandExecutor handles pure command execution without any transport or display concerns.
// It is the core execution engine that can be used by any transport handler (REPL, SSH, HTTP, etc.).
type CommandExecutor struct {
	// Application metadata
	AppName string

	// Command state (thread-safe)
	Variables *safemap.SafeMap[string, string]
	VariableExpanders []func(string) (string, bool)
	aliases        *safemap.SafeMap[string, string] // Per-instance aliases

	// Command registration
	rootInit []func(*cobra.Command)

	// Managers (dependency injection)
	JobManager      *JobManager
	Config          *Config
	LogManager      *LogManager
	TemplateManager *TemplateManager
	NotificationManager   *NotificationManager
	HistoryManager  *HistoryManager

	// Recursion protection
	maxExecDepth int32
	execDepth    atomic.Int32

	// File handling (can be overridden for SSH chroot, etc.)
	FileHandler FileHandler

	// Embedded filesystem for scripts (nil if no embedded scripts)
	Scripts *embed.FS

	// Display settings
	NoColor bool // Disable color output (respects NO_COLOR env var)
}

// FileHandler abstracts file I/O for redirection and script loading.
// This allows different transports to override file access (e.g., SSH with chroot).
type FileHandler interface {
	WriteFile(path string, content string) error
	ReadFile(path string) (string, error)
}

// LocalFileHandler implements FileHandler for local file system access.
type LocalFileHandler struct{}

func (h *LocalFileHandler) WriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (h *LocalFileHandler) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ExecutionResult contains command output and metadata.
type ExecutionResult struct {
	Output      string
	Error       error
	Success     bool
	Duration    time.Duration
	CommandLine string
}

// NewCommandExecutor creates a new command executor with the given application name.
// The customizer function is called to configure the executor (register commands, set defaults, etc.).
func NewCommandExecutor(appName string, customizer func(*CommandExecutor) error) (*CommandExecutor, error) {
	// Initialize configuration
	config, err := NewConfig(appName)
	if err != nil {
		fmt.Printf("Warning: unable to initialize config: %v\n", err)
	}

	// Try to load existing configuration
	if config != nil {
		_ = config.Load() // Ignore errors if config doesn't exist
	}

	// Set up log file path from config or default
	var logFile string
	var templatesDir string
	var historyFile string
	if config != nil && config.Logging.LogFile != "" {
		logFile = config.Logging.LogFile
	}
	if currentUser, err := user.Current(); err == nil {
		name := strings.ToLower(appName)
		appDir := filepath.Join(currentUser.HomeDir, fmt.Sprintf(".%s", name))
		if logFile == "" {
			logFile = filepath.Join(appDir, "audit.log")
		}
		templatesDir = filepath.Join(appDir, "templates")
		historyFile = filepath.Join(currentUser.HomeDir, fmt.Sprintf(".%s.history", name))
	}

	exec := &CommandExecutor{
		AppName:         appName,
		Variables: safemap.New[string, string](),
		aliases:         safemap.New[string, string](),
		JobManager:      NewJobManager(),
		Config:          config,
		LogManager:      NewLogManager(logFile),
		TemplateManager: NewTemplateManager(templatesDir, embed.FS{}),
		NotificationManager:   NewNotificationManager(),
		HistoryManager:  NewHistoryManager(appName, historyFile),
		maxExecDepth:    10, // Prevent infinite recursion
		FileHandler:     &LocalFileHandler{},
		NoColor:         os.Getenv("NO_COLOR") != "", // Respect NO_COLOR env var
	}

	// Apply logging configuration from config file
	if config != nil {
		exec.applyLoggingConfig()
		exec.applyNotificationConfig()
	}

	// Call customizer to configure the executor
	if customizer != nil {
		err = customizer(exec)
		if err != nil {
			return nil, fmt.Errorf("customizer error: %w", err)
		}
	}

	return exec, nil
}

// AddBuiltinCommands registers all built-in commands.
// This provides backward compatibility and convenience.
// For fine-grained control, use the individual command group functions instead.
func (e *CommandExecutor) AddBuiltinCommands() {
	e.AddCommands(AddAllCmds(e))
}

// AddCommands registers a command customizer function.
func (e *CommandExecutor) AddCommands(cmds func(*cobra.Command)) {
	e.rootInit = append(e.rootInit, cmds)
}

// ExecuteLine executes a command line and returns the output.
// This is a convenience wrapper around ExecuteWithContext using context.Background().
func (e *CommandExecutor) Execute(line string, scope *safemap.SafeMap[string, string]) (string, error) {
	return e.ExecuteWithContext(context.Background(), line, scope)
}

// ExecuteWithContext executes a command line with context support for cancellation and timeout.
func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, line string, scope *safemap.SafeMap[string, string]) (string, error) {
	// Track command execution time for logging
	startTime := time.Now()

	// Track recursion depth to prevent infinite loops (thread-safe)
	depth := e.execDepth.Add(1)
	defer e.execDepth.Add(-1)

	if depth > e.maxExecDepth {
		return "", fmt.Errorf("maximum execution depth exceeded (%d) - possible infinite recursion", e.maxExecDepth)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("command cancelled: %w", ctx.Err())
	default:
	}

	rootCmd := e.RootCmd()
	line = e.ExpandCommand(rootCmd, scope, line)

	outputFile, commands, err := parser.ParseCommands(line)
	if err != nil {
		// Log failed command
		if e.LogManager.IsEnabled() && depth == 1 {
			_ = e.LogManager.Log(AuditLog{
				Timestamp: startTime,
				User:      e.getCurrentUser(),
				Command:   line,
				Duration:  time.Since(startTime),
				Success:   false,
				Error:     err.Error(),
			})
		}
		return "", err
	}

	output, err := e.executeCommandsWithContext(ctx, rootCmd, commands)

	// Log command execution (only log top-level commands, not recursive calls)
	if e.LogManager.IsEnabled() && depth == 1 {
		logEntry := AuditLog{
			Timestamp: startTime,
			User:      e.getCurrentUser(),
			Command:   line,
			Output:    output,
			Duration:  time.Since(startTime),
			Success:   err == nil,
		}
		if err != nil {
			logEntry.Error = err.Error()
		}
		_ = e.LogManager.Log(logEntry)
	}

	if err != nil {
		return output, err
	}

	// Handle file redirection if specified
	if outputFile != "" {
		err = e.FileHandler.WriteFile(outputFile, output)
		if err != nil {
			return output, fmt.Errorf("failed to write to file %s: %w", outputFile, err)
		}
	}

	return output, nil
}

// executeCommandsWithContext executes parsed commands with context support for cancellation.
func (e *CommandExecutor) executeCommandsWithContext(ctx context.Context, rootCmd *cobra.Command, commands []*parser.ExecCmd) (string, error) {
	var output bytes.Buffer
	var input io.Reader

	for _, cmd := range commands {
		// Check for cancellation before each command
		select {
		case <-ctx.Done():
			return output.String(), fmt.Errorf("command execution cancelled: %w", ctx.Err())
		default:
		}

		var buf bytes.Buffer
		curCmd := cmd

		for curCmd != nil {
			// Check for cancellation in pipeline
			select {
			case <-ctx.Done():
				return output.String(), fmt.Errorf("command execution cancelled: %w", ctx.Err())
			default:
			}

			args := append([]string{curCmd.Cmd}, curCmd.Args...)

			rootCmd.SetArgs(args)
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			if input != nil {
				rootCmd.SetIn(input)
			}

			if err := rootCmd.Execute(); err != nil {
				return buf.String(), err
			}

			input = &buf
			curCmd = curCmd.Pipe
		}

		output.Write(buf.Bytes())
	}

	return output.String(), nil
}

// ExpandCommand performs token replacement including aliases, defaults, and custom replacers.
// Use this for full command line processing before execution.
func (e *CommandExecutor) ExpandCommand(cmd *cobra.Command, scope *safemap.SafeMap[string, string], input string) string {
	// Check if entire line matches an alias
	e.aliases.ForEach(func(k string, v string) bool {
		if k == input {
			input = v
			return true
		}
		return false
	})

	// Also check if first word matches an alias (for cases like "pp|grep u")
	// Split by first space or special chars to get the first command
	firstWord := input
	if idx := strings.IndexAny(input, " |>;"); idx != -1 {
		firstWord = input[:idx]
	}

	e.aliases.ForEach(func(k string, v string) bool {
		if k == firstWord && k != input { // Don't double-replace exact matches
			// Replace first word with alias value
			input = v + input[len(firstWord):]
			return true
		}
		return false
	})

	e.Variables.ForEach(func(k string, v string) bool {
		input = strings.ReplaceAll(input, k, v)
		return false
	})

	// Replace scoped variables (scope) before custom replacers
	if scope != nil {
		scope.ForEach(func(k string, v string) bool {
			input = strings.ReplaceAll(input, k, v)
			return false
		})
	}

	for _, replacer := range e.VariableExpanders {
		input, stop := replacer(input)
		if stop {
			return input
		}
	}
	input = e.replaceToken(scope, input)

	return input
}

// ExpandVariables replaces only variables (@tokens), NOT aliases.
// Use this for processing command arguments to prevent alias expansion in the middle of commands.
func (e *CommandExecutor) ExpandVariables(cmd *cobra.Command, scope *safemap.SafeMap[string, string], input string) string {
	// Replace variables from Defaults (with @ prefix)
	e.Variables.ForEach(func(k string, v string) bool {
		input = strings.ReplaceAll(input, k, v)
		return false
	})

	// Replace scoped variables (scope)
	if scope != nil {
		scope.ForEach(func(k string, v string) bool {
			input = strings.ReplaceAll(input, k, v)
			return false
		})
	}

	// Apply custom token replacers
	for _, replacer := range e.VariableExpanders {
		input, stop := replacer(input)
		if stop {
			return input
		}
	}

	// Replace built-in tokens (@env:, @exec:, etc.)
	input = e.replaceToken(scope, input)

	return input
}

// replaceToken handles token replacement for environment variables, command execution, and defaults.
func (e *CommandExecutor) replaceToken(scope *safemap.SafeMap[string, string], token string) string {
	if strings.HasPrefix(token, "@env:") {
		envVar := strings.TrimPrefix(token, "@env:")
		if value, exists := os.LookupEnv(envVar); exists {
			return value
		}
		return token
	}

	if strings.HasPrefix(token, "@exec:") {
		toExec := strings.TrimPrefix(token, "@exec:")
		res, _ := e.Execute(toExec, scope)
		return res
	}

	v, ok := e.Variables.Get(token)
	if ok {
		return v
	}

	if scope != nil {
		v, ok := scope.Get(token)
		if ok {
			return v
		}
	}

	return token
}

// RootCmd creates a new root Cobra command with all registered subcommands.
// Returns a fresh command instance on each call.
func (e *CommandExecutor) RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:                "",
		Short:              "modular consolekit",
		DisableFlagParsing: true,
		SilenceErrors:      true, // Prevent duplicate error printing
		// Root command shows help by default
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	for _, init := range e.rootInit {
		init(rootCmd)
	}

	SetRecursiveHelpFunc(rootCmd)
	return rootCmd
}

// GetAvailableCommands returns a list of all available command names.
// This includes all registered commands and their subcommands.
func (e *CommandExecutor) GetAvailableCommands() []string {
	root := e.RootCmd()
	commands := make([]string, 0)

	// Helper function to recursively collect command names
	var collectCommands func(*cobra.Command, string)
	collectCommands = func(cmd *cobra.Command, prefix string) {
		for _, subCmd := range cmd.Commands() {
			// Skip hidden commands
			if subCmd.Hidden {
				continue
			}

			name := subCmd.Name()
			if prefix != "" {
				name = prefix + " " + name
			}

			// Add command if it has a Run function (is executable)
			if subCmd.Run != nil || subCmd.RunE != nil {
				commands = append(commands, name)
			}

			// Recursively collect subcommands
			collectCommands(subCmd, name)
		}
	}

	collectCommands(root, "")

	// Also include aliases
	e.aliases.SortedForEach(func(alias, _ string) bool {
		commands = append(commands, alias)
		return true // Continue iteration
	})

	return commands
}

// getCurrentUser returns the current username for logging.
func (e *CommandExecutor) getCurrentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "unknown"
}

// applyLoggingConfig applies logging configuration from config file.
func (e *CommandExecutor) applyLoggingConfig() {
	if e.Config == nil || e.LogManager == nil {
		return
	}

	cfg := e.Config.Logging

	// Set enabled state
	if cfg.Enabled {
		e.LogManager.Enable()
	} else {
		e.LogManager.Disable()
	}

	// Apply other settings
	if cfg.LogFile != "" {
		e.LogManager.SetLogFile(cfg.LogFile)
	}
	e.LogManager.SetLogSuccess(cfg.LogSuccess)
	e.LogManager.SetLogFailures(cfg.LogFailures)
	if cfg.MaxSizeMB > 0 {
		e.LogManager.SetMaxSize(int64(cfg.MaxSizeMB))
	}
	if cfg.RetentionDays > 0 {
		e.LogManager.SetRetention(cfg.RetentionDays)
	}
}

// applyNotificationConfig applies notification configuration from config file.
func (e *CommandExecutor) applyNotificationConfig() {
	if e.Config == nil || e.NotificationManager == nil {
		return
	}

	cfg := e.Config.Notification

	// Set webhook URL if configured
	if cfg.WebhookURL != "" {
		e.NotificationManager.SetWebhook(cfg.WebhookURL)
	}
}

// LoadAliases loads aliases from the ~/.{appname}.aliases file.
func (e *CommandExecutor) LoadAliases() error {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get home directory: %w", err)
	}

	// Construct the full path to the .aliases file
	aliasesFilePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", strings.ToLower(e.AppName)))

	// Open the .aliases file
	file, err := os.Open(aliasesFilePath)
	if err != nil {
		// File doesn't exist yet - add default aliases
		e.AddDefaultAliases()
		return nil
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore empty lines and comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Split the line into name and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, line)
			continue
		}

		name := strings.TrimSpace(parts[0])
		if strings.Contains(name, " ") {
			fmt.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, name)
			continue
		}
		value := strings.TrimSpace(parts[1])
		if len(value) == 0 {
			fmt.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, line)
			continue
		}

		e.aliases.Set(name, value)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading aliases file: %w", err)
	}

	return nil
}

// SaveAliases saves aliases to the ~/.{appname}.aliases file.
func (e *CommandExecutor) SaveAliases() error {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get home directory: %w", err)
	}

	filePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", strings.ToLower(e.AppName)))

	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create aliases file: %w", err)
	}
	defer file.Close()

	// Write each key-value pair to the file
	writer := bufio.NewWriter(file)
	e.aliases.ForEach(func(name string, value string) bool {
		_, err := writer.WriteString(fmt.Sprintf("%s=%s\n", name, value))
		if err != nil {
			fmt.Printf("error writing alias %s: %v\n", name, err)
		}
		return false
	})

	// Flush the writer to ensure all data is written
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush aliases file: %w", err)
	}

	return nil
}

// AddDefaultAlias adds a single default alias.
func (e *CommandExecutor) AddDefaultAlias(alias, expanded string) {
	e.aliases.Set(alias, expanded)
}

// AddDefaultAliases adds default aliases and saves them.
func (e *CommandExecutor) AddDefaultAliases() {
	e.aliases.Set("pp", "print test")
	_ = e.SaveAliases() // Ignore error
}

// Deprecated: Interactive prompt methods moved to REPLHandler.
// These stub methods are kept for backward compatibility.
// They print the prompt but return default values - use REPLHandler methods for actual interaction.
// Interactive prompts only work in REPL mode, not over SSH or other transports.
func (e *CommandExecutor) Confirm(message string) bool {
	fmt.Printf("%s [y/N]: (interactive prompts not available)\n", message)
	return false
}

func (e *CommandExecutor) Prompt(message string) string {
	fmt.Printf("%s: (interactive prompts not available)\n", message)
	return ""
}

func (e *CommandExecutor) PromptDefault(message string, defaultValue string) string {
	fmt.Printf("%s [%s]: (interactive prompts not available, using default)\n", message, defaultValue)
	return defaultValue
}

func (e *CommandExecutor) PromptPassword(message string) string {
	fmt.Printf("%s: (interactive prompts not available)\n", message)
	return ""
}

func (e *CommandExecutor) Select(message string, options []string) string {
	fmt.Println(message, "(interactive selection not available)")
	if len(options) > 0 {
		fmt.Printf("Using first option: %s\n", options[0])
		return options[0]
	}
	return ""
}

func (e *CommandExecutor) SelectWithDefault(message string, options []string, defaultIdx int) string {
	fmt.Println(message, "(interactive selection not available)")
	if defaultIdx >= 0 && defaultIdx < len(options) {
		fmt.Printf("Using default option: %s\n", options[defaultIdx])
		return options[defaultIdx]
	}
	return ""
}

func (e *CommandExecutor) MultiSelect(message string, options []string) []string {
	fmt.Println(message, "(interactive selection not available)")
	return []string{}
}

func (e *CommandExecutor) ConfirmDestructive(message string) bool {
	fmt.Printf("%s (type 'yes' to confirm): (interactive prompts not available)\n", message)
	return false
}

func (e *CommandExecutor) PromptInteger(message string, defaultValue int) int {
	fmt.Printf("%s [%d]: (interactive prompts not available, using default)\n", message, defaultValue)
	return defaultValue
}
