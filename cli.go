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
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/reeflective/console"
	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
)

type CLI struct {
	NoColor         bool
	rootInit        []func(c *cobra.Command)
	AppName         string
	OnExit          func(caller string, code int)
	InfoString      func(format string, a ...any) string
	ErrorString     func(format string, a ...any) string
	SuccessString   func(format string, a ...any) string
	WarningString   func(format string, a ...any) string
	Scripts         embed.FS
	Defaults        *safemap.SafeMap[string, string]
	TokenReplacers  []func(string) (string, bool)
	aliases         *safemap.SafeMap[string, string] // Per-instance aliases
	JobManager      *JobManager                      // Background job management
	Config          *Config                          // Application configuration
	LogManager      *LogManager                      // Command logging and audit trail
	TemplateManager *TemplateManager                 // Template system
	NotifyManager   *NotifyManager                   // Notification system

	// console specific fields
	app         *console.Console
	historyFile string
	promptFunc  func() string // Store prompt function for dynamic updates

	// recursion protection (thread-safe)
	execDepth    atomic.Int32
	maxExecDepth int32
}

func NewCLI(AppName string, customizer func(*CLI) error) (*CLI, error) {
	// Initialize configuration
	config, err := NewConfig(AppName)
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
	if config != nil && config.Logging.LogFile != "" {
		logFile = config.Logging.LogFile
	}
	if currentUser, err := user.Current(); err == nil {
		name := strings.ToLower(AppName)
		appDir := filepath.Join(currentUser.HomeDir, fmt.Sprintf(".%s", name))
		if logFile == "" {
			logFile = filepath.Join(appDir, "audit.log")
		}
		templatesDir = filepath.Join(appDir, "templates")
	}

	cli := &CLI{
		AppName:         AppName,
		InfoString:      color.New(color.FgCyan).SprintfFunc(),
		ErrorString:     color.New(color.FgRed).SprintfFunc(),
		SuccessString:   color.New(color.FgGreen).SprintfFunc(),
		WarningString:   color.New(color.FgYellow).SprintfFunc(),
		Defaults:        safemap.New[string, string](),
		aliases:         safemap.New[string, string](),                // Per-instance aliases
		JobManager:      NewJobManager(),                              // Initialize job manager
		Config:          config,                                       // Initialize configuration
		LogManager:      NewLogManager(logFile),                       // Initialize log manager
		TemplateManager: NewTemplateManager(templatesDir, embed.FS{}), // Initialize template manager
		NotifyManager:   NewNotifyManager(),                           // Initialize notify manager
		maxExecDepth:    10,                                           // Prevent infinite recursion
	}

	// Apply logging configuration from config file
	if config != nil {
		cli.applyLoggingConfig()
		cli.applyNotificationConfig()
	}

	// Set default prompt function
	cli.promptFunc = func() string {
		return fmt.Sprintf("%s > ", cli.AppName)
	}

	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	cli.NoColor = os.Getenv("NO_COLOR") != "" || !isTTY

	if cli.NoColor {
		cli.InfoString = fmt.Sprintf
		cli.ErrorString = fmt.Sprintf
		cli.SuccessString = fmt.Sprintf
		cli.WarningString = fmt.Sprintf
		color.NoColor = true
	}

	// Set up history file path
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("unable to get current user: %v\n", err)
	} else {
		name := strings.ToLower(cli.AppName)
		fileName := fmt.Sprintf(".%s.history", name)
		cli.historyFile = filepath.Join(currentUser.HomeDir, fileName)
	}

	if customizer != nil {
		err := customizer(cli)
		if err != nil {
			fmt.Printf("customizer error: %v\n", err)
			return nil, fmt.Errorf("customizer error: %w", err)
		}
	}

	return cli, nil
}

func (c *CLI) AddAll() {
	c.AddCommands(AddAlias(c))
	c.AddCommands(AddOSExec(c))
	c.AddCommands(AddHistory(c))
	c.AddCommands(AddMisc(c))
	c.AddCommands(AddBaseCmds(c))
	c.AddCommands(AddScriptingCmds(c))
	c.AddCommands(AddJobCommands(c))
	c.AddCommands(AddVariableCommands(c))
	c.AddCommands(AddConfigCommands(c))
	c.AddCommands(AddUtilityCommands(c))
	c.AddCommands(AddLogCommands(c))
	c.AddCommands(AddPromptCommands(c))
	c.AddCommands(AddTemplateCommands(c))
	c.AddCommands(AddDataManipulationCommands(c))
	c.AddCommands(AddWatchCommand(c))
	c.AddCommands(AddPipelineCommands(c))
	c.AddCommands(AddControlFlowCommands(c))
	c.AddCommands(AddNotifyCommands(c))
	c.AddCommands(AddScheduleCommands(c))
	c.AddCommands(AddFormatCommands(c))
	c.AddCommands(AddClipboardCommands(c))
	c.AddCommands(AddMCPCommands(c))

}

func (c *CLI) AddCommands(cmds func(*cobra.Command)) {
	c.rootInit = append(c.rootInit, cmds)
}

func (c *CLI) ReplaceDefaults(cmd *cobra.Command, defs *safemap.SafeMap[string, string], input string) string {

	// Check if entire line matches an alias
	c.aliases.ForEach(func(k string, v string) bool {
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

	c.aliases.ForEach(func(k string, v string) bool {
		if k == firstWord && k != input { // Don't double-replace exact matches
			// Replace first word with alias value
			input = v + input[len(firstWord):]
			return true
		}
		return false
	})

	c.Defaults.ForEach(func(k string, v string) bool {
		input = strings.ReplaceAll(input, k, v)
		return false
	})

	for _, e := range c.TokenReplacers {
		input, stop := e(input)
		if stop {
			return input
		}
	}
	input = c.replaceToken(cmd, defs, input)

	return input
}

// ReplaceTokens replaces only variables (@tokens), NOT aliases.
// Use this for processing command arguments to prevent alias expansion in the middle of commands.
func (c *CLI) ReplaceTokens(cmd *cobra.Command, defs *safemap.SafeMap[string, string], input string) string {
	// Replace variables from Defaults (with @ prefix)
	c.Defaults.ForEach(func(k string, v string) bool {
		input = strings.ReplaceAll(input, k, v)
		return false
	})

	// Apply custom token replacers
	for _, e := range c.TokenReplacers {
		input, stop := e(input)
		if stop {
			return input
		}
	}

	// Replace built-in tokens (@env:, @exec:, etc.)
	input = c.replaceToken(cmd, defs, input)

	return input
}

// replaceToken handles token replacement
func (c *CLI) replaceToken(cmd *cobra.Command, defs *safemap.SafeMap[string, string], token string) string {
	if strings.HasPrefix(token, "@env:") {
		envVar := strings.TrimPrefix(token, "@env:")
		if value, exists := os.LookupEnv(envVar); exists {
			return value
		}
		return token
	}

	if strings.HasPrefix(token, "@exec:") {
		toExec := strings.TrimPrefix(token, "@exec:")
		res, _ := c.ExecuteLine(toExec, defs)
		return res
	}

	v, ok := c.Defaults.Get(token)
	if ok {
		return v
	}

	if defs != nil {
		v, ok := defs.Get(token)
		if ok {
			return v
		}
	}

	return token
}

func (c *CLI) ExecuteLine(line string, defs *safemap.SafeMap[string, string]) (string, error) {
	return c.ExecuteLineWithContext(context.Background(), line, defs)
}

// ExecuteLineWithContext executes a command line with context support for cancellation and timeout
func (c *CLI) ExecuteLineWithContext(ctx context.Context, line string, defs *safemap.SafeMap[string, string]) (string, error) {
	// Track command execution time for logging
	startTime := time.Now()

	// Track recursion depth to prevent infinite loops (thread-safe)
	depth := c.execDepth.Add(1)
	defer c.execDepth.Add(-1)

	if depth > c.maxExecDepth {
		return "", fmt.Errorf("maximum execution depth exceeded (%d) - possible infinite recursion", c.maxExecDepth)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("command cancelled: %w", ctx.Err())
	default:
	}

	rootCmd := c.BuildRootCmd()()
	line = c.ReplaceDefaults(rootCmd, defs, line)

	outputFile, commands, err := parser.ParseCommands(line)
	if err != nil {
		// Log failed command
		if c.LogManager.IsEnabled() && depth == 1 {
			_ = c.LogManager.Log(AuditLog{
				Timestamp: startTime,
				User:      c.getCurrentUser(),
				Command:   line,
				Duration:  time.Since(startTime),
				Success:   false,
				Error:     err.Error(),
			})
		}
		return "", err
	}

	output, err := c.executeCommandsWithContext(ctx, rootCmd, commands)

	// Log command execution (only log top-level commands, not recursive calls)
	if c.LogManager.IsEnabled() && depth == 1 {
		logEntry := AuditLog{
			Timestamp: startTime,
			User:      c.getCurrentUser(),
			Command:   line,
			Output:    output,
			Duration:  time.Since(startTime),
			Success:   err == nil,
		}
		if err != nil {
			logEntry.Error = err.Error()
		}
		_ = c.LogManager.Log(logEntry)
	}

	if err != nil {
		return output, err
	}

	// Handle file redirection if specified
	if outputFile != "" {
		err = c.writeToFile(outputFile, output)
		if err != nil {
			return output, fmt.Errorf("failed to write to file %s: %w", outputFile, err)
		}
	}

	return output, nil
}

// getCurrentUser returns the current username for logging
func (c *CLI) getCurrentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "unknown"
}

// applyLoggingConfig applies logging configuration from config file
func (c *CLI) applyLoggingConfig() {
	if c.Config == nil || c.LogManager == nil {
		return
	}

	cfg := c.Config.Logging

	// Set enabled state
	if cfg.Enabled {
		c.LogManager.Enable()
	} else {
		c.LogManager.Disable()
	}

	// Apply other settings
	if cfg.LogFile != "" {
		c.LogManager.SetLogFile(cfg.LogFile)
	}
	c.LogManager.SetLogSuccess(cfg.LogSuccess)
	c.LogManager.SetLogFailures(cfg.LogFailures)
	if cfg.MaxSizeMB > 0 {
		c.LogManager.SetMaxSize(int64(cfg.MaxSizeMB))
	}
	if cfg.RetentionDays > 0 {
		c.LogManager.SetRetention(cfg.RetentionDays)
	}
}

// applyNotificationConfig applies notification configuration from config file
func (c *CLI) applyNotificationConfig() {
	if c.Config == nil || c.NotifyManager == nil {
		return
	}

	cfg := c.Config.Notification

	// Set webhook URL if configured
	if cfg.WebhookURL != "" {
		c.NotifyManager.SetWebhook(cfg.WebhookURL)
	}
}

// writeToFile writes content to a file
func (c *CLI) writeToFile(filename string, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// executeCommands executes parsed commands with piping support
func (c *CLI) executeCommands(rootCmd *cobra.Command, commands []*parser.ExecCmd) (string, error) {
	return c.executeCommandsWithContext(context.Background(), rootCmd, commands)
}

// executeCommandsWithContext executes parsed commands with context support for cancellation
func (c *CLI) executeCommandsWithContext(ctx context.Context, rootCmd *cobra.Command, commands []*parser.ExecCmd) (string, error) {
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

func (c *CLI) BuildRootCmd() func() *cobra.Command {
	return func() *cobra.Command {

		rootCmd := &cobra.Command{
			Use:                "",
			Short:              "modular consolekit",
			DisableFlagParsing: true,
		}

		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

		// Add a hidden noop command that's used internally to suppress execution
		// after we've already handled pipes/redirects in the PreCmdRunLineHooks
		noopCmd := &cobra.Command{
			Use:    "__noop__",
			Hidden: true,
			Run:    func(cmd *cobra.Command, args []string) {}, // Do nothing
		}
		rootCmd.AddCommand(noopCmd)

		for _, init := range c.rootInit {
			init(rootCmd)
		}

		SetRecursiveHelpFunc(rootCmd)
		return rootCmd
	}
}

// Run executes command-line arguments if present, otherwise starts the REPL
// This is the recommended entry point for CLI applications
// Detects if stdin is piped and runs in batch mode automatically
func (c *CLI) Run() error {
	// Check if stdin is being piped (not a TTY) - enables script piping
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return c.RunBatch()
	}

	// Check if we have command-line arguments
	if len(os.Args) > 1 {
		// Check if we have an actual command or just flags
		hasCommand := c.hasNonFlagArgs()
		if hasCommand {
			// Execute command directly without entering REPL
			return c.ExecuteArgs(os.Args[1:])
		}
	}

	// No arguments or only flags, start REPL
	return c.AppBlock()
}

// hasNonFlagArgs checks if command-line arguments contain an actual command (non-flag argument)
func (c *CLI) hasNonFlagArgs() bool {
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
				    arg == "-f" || arg == "--file") {
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

// ExecuteArgs executes command-line arguments directly using Cobra
func (c *CLI) ExecuteArgs(args []string) error {
	rootCmd := c.BuildRootCmd()()
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// RunBatch reads commands from stdin and executes them line by line
// This enables piping scripts for automated testing: cat script.run | ./app
func (c *CLI) RunBatch() error {
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
		fmt.Printf("%s\n", c.InfoString("→ %s", line))

		output, err := c.ExecuteLine(line, nil)
		if output != "" {
			fmt.Print(output)
			if !strings.HasSuffix(output, "\n") {
				fmt.Println()
			}
		}

		if err != nil {
			errorCount++
			fmt.Fprintf(os.Stderr, "%s\n", c.ErrorString("✗ Error at line %d: %v", lineNum, err))
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

// AppBlock starts the REPL loop (maintains API compatibility)
func (c *CLI) AppBlock() error {
	// Create a new console application
	c.app = console.New(c.AppName)

	// Disable automatic quote/bracket pairing in readline
	shell := c.app.Shell()
	shell.Config.Set("autopairs", false)

	// Get the active menu (default menu)
	menu := c.app.ActiveMenu()

	// Set commands using our BuildRootCmd function
	menu.SetCommands(c.BuildRootCmd())

	// Configure history file if available
	if c.historyFile != "" {
		menu.AddHistorySourceFile("main", c.historyFile)
	}

	// Set the prompt using the stored prompt function
	prompt := menu.Prompt()
	prompt.Primary = c.promptFunc

	// Add a pre-command hook to handle token replacement and piping
	c.app.PreCmdRunLineHooks = append(c.app.PreCmdRunLineHooks, func(args []string) ([]string, error) {
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

		// Check if we need full custom handling (pipes, redirects, or @ tokens)
		// These need ExecuteLine which handles both alias/token replacement AND piping/redirection
		if strings.Contains(line, "|") || strings.Contains(line, ">") || strings.Contains(line, "@") {
			// Execute through our custom ExecuteLine which handles everything
			output, err := c.ExecuteLine(line, nil)

			// Print output
			if output != "" {
				fmt.Print(output)
				if !strings.HasSuffix(output, "\n") {
					fmt.Println()
				}
			}

			if err != nil {
				fmt.Printf("%s\n", c.ErrorString("Error: %v", err))
			}

			// Return the noop command to prevent further execution
			return []string{"__noop__"}, nil
		}

		// For simple commands without pipes/redirects, do alias replacement only
		originalLine := line

		// Check aliases - note: only checks exact match of the whole line
		c.aliases.ForEach(func(k string, v string) bool {
			if k == line {
				line = v
				return true
			}
			return false
		})

		// If alias changed the line, re-split and return
		if line != originalLine {
			newArgs := strings.Fields(line)
			return newArgs, nil
		}

		// No changes, let console handle it normally
		return args, nil
	})

	// Start the console REPL
	return c.app.Start()
}

// Exit handles program exit
func (c *CLI) Exit(caller string, code int) {
	if c.OnExit != nil {
		c.OnExit(caller, code)
	}

	if code != 0 {
		fmt.Printf("%s: exiting with code %d\n", caller, code)
	}

	// Console app automatically saves history on exit
	os.Exit(code)
}

// Execute starts the CLI (alternative entry point for compatibility)
func (c *CLI) Execute() {
	if err := c.AppBlock(); err != nil {
		fmt.Print(c.ErrorString("error executing CLI: %v\n", err))
		os.Exit(1)
	}
}

func (c *CLI) SetPrompt(s func() string) {
	// Store the prompt function
	c.promptFunc = s

	// Update the active prompt if the app is already running
	if c.app != nil {
		menu := c.app.ActiveMenu()
		if menu != nil {
			prompt := menu.Prompt()
			prompt.Primary = s
		}
	}
}
