package consolekit

import (
	"embed"
	"errors"
	"fmt"
	"github.com/alexj212/consolekit/safemap"
	"github.com/fatih/color"
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
	NoColor     bool
	RootCmd     *cobra.Command
	Repl        *console.Console
	LocalMenu   *console.Menu
	LocalPrompt *console.Prompt
	AppName     string
	OnExit      func(caller string, code int)
	InfoString  func(format string, a ...any) string
	ErrorString func(format string, a ...any) string
	Scripts     embed.FS
	Defaults    *safemap.SafeMap[string, string]
}

func NewCLI(AppName string, customizer func(*CLI) error) (*CLI, error) {
	cli := &CLI{
		AppName:     AppName,
		InfoString:  color.New(color.FgWhite).SprintfFunc(),
		ErrorString: color.New(color.FgRed).SprintfFunc(),
		Defaults:    safemap.New[string, string](),
	}

	cli.RootCmd = &cobra.Command{
		Use:                "",
		Short:              "modular consolekit",
		DisableFlagParsing: true,
	}
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
	cli.LocalMenu.SetCommands(func() *cobra.Command { return cli.RootCmd })

	SetRecursiveHelpFunc(cli.RootCmd)
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

func (c *CLI) AddCommand(cmd *cobra.Command) {
	c.RootCmd.AddCommand(cmd)
}

func (c *CLI) ReplaceDefaults(cmd *cobra.Command, input string) string {
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
			words[i] = `"` + c.replaceToken(cmd, cleanWord) + `"`
		} else if strings.HasPrefix(word, "@") {
			// Replace token directly if not in quotes
			words[i] = c.replaceToken(cmd, word)
		}
	}

	return strings.Join(words, " ")
} //ReplaceDefaults

// replaceToken handles token replacement. Modify this function as needed.
func (c *CLI) replaceToken(cmd *cobra.Command, token string) string {

	v, ok := c.Defaults.Get(token)
	if ok {
		cmd.Printf("replaceToken: %s -> %s\n", token, v)
		return v
	}
	return token
}
