package consolekit

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/alexj212/consolekit/parser"
	"github.com/alexj212/consolekit/safemap"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
)

type CLI struct {
	NoColor        bool
	rootInit       []func(c *cobra.Command)
	AppName        string
	OnExit         func(caller string, code int)
	InfoString     func(format string, a ...any) string
	ErrorString    func(format string, a ...any) string
	Scripts        embed.FS
	Defaults       *safemap.SafeMap[string, string]
	TokenReplacers []func(string) (string, bool)

	// readline specific fields
	rl          *readline.Instance
	historyFile string
}

func NewCLI(AppName string, customizer func(*CLI) error) (*CLI, error) {
	cli := &CLI{
		AppName:     AppName,
		InfoString:  color.New(color.FgWhite).SprintfFunc(),
		ErrorString: color.New(color.FgRed).SprintfFunc(),
		Defaults:    safemap.New[string, string](),
	}

	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	cli.NoColor = os.Getenv("NO_COLOR") != "" || !isTTY

	if cli.NoColor {
		cli.InfoString = fmt.Sprintf
		cli.ErrorString = fmt.Sprintf
		color.NoColor = true
	}

	// Set up history file
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

func (c *CLI) AddCommands(cmds func(*cobra.Command)) {
	c.rootInit = append(c.rootInit, cmds)
}

func (c *CLI) ReplaceDefaults(cmd *cobra.Command, defs *safemap.SafeMap[string, string], input string) string {

	aliases.ForEach(func(k string, v string) bool {
		if k == input {
			input = v
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
	rootCmd := c.BuildRootCmd()()
	line = c.ReplaceDefaults(rootCmd, defs, line)

	_, commands, err := parser.ParseCommands(line)
	if err != nil {
		return "", err
	}

	return c.executeCommands(rootCmd, commands)
}

// executeCommands executes parsed commands with piping support
func (c *CLI) executeCommands(rootCmd *cobra.Command, commands []*parser.ExecCmd) (string, error) {
	var output bytes.Buffer
	var input io.Reader

	for _, cmd := range commands {
		var buf bytes.Buffer
		curCmd := cmd

		for curCmd != nil {
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

		for _, init := range c.rootInit {
			init(rootCmd)
		}

		SetRecursiveHelpFunc(rootCmd)
		return rootCmd
	}
}

// completer provides auto-completion suggestions for readline
func (c *CLI) completer(line string) []string {
	var values []string

	// Build suggestions from cobra commands
	rootCmd := c.BuildRootCmd()()
	word := strings.TrimSpace(line)

	for _, cmd := range rootCmd.Commands() {
		// Add main command
		if strings.HasPrefix(cmd.Use, word) || word == "" {
			values = append(values, cmd.Use)
		}

		// Add aliases
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, word) || word == "" {
				values = append(values, alias)
			}
		}
	}

	return values
}

// AppBlock starts the REPL loop (maintains API compatibility)
func (c *CLI) AppBlock() error {
	// Configure readline with better tab completion
	config := &readline.Config{
		Prompt:          fmt.Sprintf("%s > ", c.AppName),
		HistoryFile:     c.historyFile,
		AutoComplete:    &completerAdapter{cli: c},
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	}

	// Create readline instance
	var err error
	c.rl, err = readline.NewEx(config)
	if err != nil {
		return fmt.Errorf("failed to initialize readline: %w", err)
	}
	defer c.rl.Close()

	// Set up signal handler to save history and exit gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigChan
		c.rl.Close()
		os.Exit(0)
	}()

	// Main REPL loop
	for {
		line, err := c.rl.Readline()

		// Handle EOF (Ctrl+D) or Ctrl+C
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				fmt.Println("\nUse 'exit' or 'quit' to exit")
				continue
			}
			continue
		} else if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// Trim input
		input := strings.TrimSpace(line)

		// Skip empty input
		if input == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(input, "#") {
			continue
		}

		// Execute the command
		output, err := c.ExecuteLine(input, nil)

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
	}

	return nil
}

// completerAdapter adapts CLI completer to readline.AutoCompleter interface
type completerAdapter struct {
	cli *CLI
}

func (ca *completerAdapter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// Get completions from CLI completer
	completions := ca.cli.completer(string(line[:pos]))

	// Convert to readline format
	var suggestions [][]rune
	for _, c := range completions {
		suggestions = append(suggestions, []rune(c))
	}

	return suggestions, len(line) - pos
}

// Exit handles program exit
func (c *CLI) Exit(caller string, code int) {
	if c.OnExit != nil {
		c.OnExit(caller, code)
	}

	if code != 0 {
		fmt.Printf("%s: exiting with code %d\n", caller, code)
	}

	// Close readline if it exists (history is saved automatically)
	if c.rl != nil {
		c.rl.Close()
	}

	os.Exit(code)
}

// Execute starts the CLI (alternative entry point for compatibility)
func (c *CLI) Execute() {
	if err := c.AppBlock(); err != nil {
		fmt.Printf(c.ErrorString("error executing CLI: %v\n", err))
		os.Exit(1)
	}
}

func (c *CLI) SetPrompt(s string) {
	if c.rl != nil {
		c.rl.SetPrompt(s)
	}
}
