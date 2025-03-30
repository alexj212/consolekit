package consolekit

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"github.com/alexj212/console/parser"
	"github.com/alexj212/consolekit/safemap"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"regexp"

	"github.com/mattn/go-isatty"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexj212/console"

	"github.com/spf13/cobra"
)

type CLI struct {
	NoColor bool
	//rootCmd     *cobra.Command
	rootInit       []func(c *cobra.Command)
	Repl           *console.Console
	LocalMenu      *console.Menu
	LocalPrompt    *console.Prompt
	AppName        string
	OnExit         func(caller string, code int)
	InfoString     func(format string, a ...any) string
	ErrorString    func(format string, a ...any) string
	Scripts        embed.FS
	Defaults       *safemap.SafeMap[string, string]
	TokenReplacers []func(string) (string, bool)
}

func NewCLI(AppName string, customizer func(*CLI) error) (*CLI, error) {
	cli := &CLI{
		AppName:     AppName,
		InfoString:  color.New(color.FgWhite).SprintfFunc(),
		ErrorString: color.New(color.FgRed).SprintfFunc(),
		Defaults:    safemap.New[string, string](),
	}

	//cli.cmdBuilders = []func() *cobra.Command{}
	cli.Repl = console.New(AppName)

	console.DisableParse = true

	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	cli.NoColor = os.Getenv("NO_COLOR") != "" || !isTTY

	if cli.NoColor {
		cli.InfoString = fmt.Sprintf
		cli.ErrorString = fmt.Sprintf
	}
	cli.Repl.NewlineAfter = true

	_ = cli.Repl.Shell().Config.Set("editing-mode", "vi")
	_ = cli.Repl.Shell().Config.Set("history-autosuggest", true)
	_ = cli.Repl.Shell().Config.Set("usage-hint-always", true)

	name := strings.ToLower(cli.AppName)
	fileName := fmt.Sprintf(".%s.local.repl.history", name)

	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("unable to get current user: %v\n", err)
	}

	filePath := filepath.Join(currentUser.HomeDir, fileName)
	cli.LocalMenu = cli.Repl.NewMenu("local")
	cli.LocalMenu.AddHistorySourceFile(fmt.Sprintf("%s-local", cli.AppName), filePath)
	cli.LocalMenu.AddInterrupt(io.EOF, cli.ExitCtrlD)
	cli.LocalMenu.AddInterrupt(errors.New(os.Interrupt.String()), cli.ExitCtrlD)
	rootcmd := cli.BuildRootCmd()
	cli.LocalMenu.SetCommands(rootcmd)

	cli.LocalPrompt = cli.LocalMenu.Prompt()

	cli.LocalPrompt.Primary = func() string {
		return cli.InfoString("local >") + " "
	}
	cli.LocalPrompt.Right = func() string {
		return cli.InfoString(time.Now().Format("03:04:05.000"))
	}
	cli.LocalPrompt.Transient = func() string { return "\x1b[1;30m>> \x1b[0m" }
	cli.Repl.SwitchMenu("local")

	if customizer != nil {
		err := customizer(cli)
		if err != nil {
			fmt.Printf("customizer error: %v\n", err)
			return nil, errors.New("customizer error")
		}
	}

	return cli, nil
}

func (c *CLI) AddCommands(cmds func(*cobra.Command)) {
	c.rootInit = append(c.rootInit, cmds)
}

func (c *CLI) ReplaceDefaults(cmd *cobra.Command, input string) string {
	for _, e := range c.TokenReplacers {
		input, stop := e(input)
		if stop {
			//fmt.Printf("ReplaceDefaults A: %s\n", input)
			return input
		}
	}

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

	// Regular expression to split by spaces, but keep quoted sections intact
	re := regexp.MustCompile(`"[^"]*"|\S+`)
	words := re.FindAllString(input, -1)

	for i, word := range words {
		// Check for tokens inside quotes but retain quotes
		if strings.HasPrefix(word, `"@`) {
			// Remove quotes, replace token, then re-wrap with quotes
			cleanWord := strings.Trim(word, `"`)
			words[i] = `"` + c.ReplaceToken(cmd, cleanWord) + `"`
		} else if strings.HasPrefix(word, "@") {
			// Replace token directly if not in quotes
			words[i] = c.ReplaceToken(cmd, word)
		}
	}
	return strings.Join(words, " ")
} //ReplaceDefaults

// ReplaceToken handles token replacement. Modify this function as needed.
func (c *CLI) ReplaceToken(cmd *cobra.Command, token string) string {
	//cmd.Printf("ReplaceToken: %s\n", token)
	//cmd.Printf("\n")
	if strings.HasPrefix(token, "@env:") {
		envVar := strings.TrimPrefix(token, "@env:")
		if value, exists := os.LookupEnv(envVar); exists {
			return value
		}
		return token
	}

	if strings.HasPrefix(token, "@exec:") {
		toExec := strings.TrimPrefix(token, "@exec:")
		//fmt.Printf("exec: %s\n", toExec)
		res, _ := c.ExecuteLine(toExec)
		//fmt.Printf("exec result: %s\n", res)
		return res
	}

	v, ok := c.Defaults.Get(token)
	if ok {
		return v
	}
	return token
}

func (c *CLI) ExecuteLine(line string) (string, error) {
	rootCmd := c.BuildRootCmd()()
	line = c.ReplaceDefaults(rootCmd, line)

	//fmt.Printf("ExecuteLine : %s\n", line)
	_, commands, err := parser.ParseCommands(line)
	if err != nil {
		fmt.Printf("ExecuteLine ParseCommands err: %s\n", err)
		return "", err
	}

	//fmt.Printf("commands: %d\n", len(commands))
	return c.Repl.ExecuteCommand(rootCmd, commands)
}

func (c *CLI) BuildRootCmd() console.Commands {
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

// ExecuteCommand executes parsed commands using a Cobra root command with piped execution
func ExecuteCommand(rootCmd *cobra.Command, commands []*parser.ExecCmd) (string, error) {

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
				return "", err
			}

			input = &buf
			curCmd = curCmd.Pipe
		}

		output.Write(buf.Bytes())
	}

	return output.String(), nil
}
