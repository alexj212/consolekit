package consolekit

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
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

var cli *CLI

type CLI struct {
	RootCmd                                                       *cobra.Command
	MainCmd                                                       *cobra.Command
	Repl                                                          *console.Console
	LocalMenu                                                     *console.Menu
	LocalPrompt                                                   *console.Prompt
	AppName, BuildDate, LatestCommit, Version, GitRepo, GitBranch string
	OnExit                                                        func(caller string, code int)
	InfoString, ErrorString                                       func(format string, a ...any) string
	scripts                                                       embed.FS
}

func NewCLI(AppName, BuildDate, LatestCommit, Version, GitRepo, GitBranch string, customizer func(*CLI) error) *CLI {
	rootCmd := &cobra.Command{
		Use:   "",
		Short: "Modular CLI application with REPL, piping, chaining, and file redirection",
		RunE: func(cmd *cobra.Command, args []string) error {
			commandStr := strings.Join(args, " ")
			if commandStr == "" {
				return nil
			}
			cmd.Printf("rootCmd [command]: %s\n", commandStr)

			res, err := cli.ExecuteCommands(commandStr)
			if err != nil {
				cmd.Printf("rootCmd error: %v\n", err)
				return err
			}
			fmt.Printf("rootCmd %s\n", res)

			return nil
		},
	}

	repl := console.New(AppName)
	cli = &CLI{
		RootCmd:      rootCmd,
		Repl:         repl,
		AppName:      AppName,
		BuildDate:    BuildDate,
		LatestCommit: LatestCommit,
		Version:      Version,
		GitRepo:      GitRepo,
		GitBranch:    GitBranch,
		InfoString:   color.New(color.FgWhite).SprintfFunc(),
		ErrorString:  color.New(color.FgRed).SprintfFunc(),
	}
	console.DisableParse = true
	cli.MainCmd = &cobra.Command{
		Use:   "",
		Short: "Modular CLI application with REPL, piping, chaining, and file redirection",
		RunE: func(cmd *cobra.Command, args []string) error {

			commandStr := strings.Join(args, " ")
			if commandStr == "" {
				return nil
			}

			res, err := cli.ExecuteCommands(commandStr)
			if err != nil {
				cmd.Printf("MainCmd ExecuteCommandsNew %s error: %v\n", commandStr, err)
				return err
			}

			fmt.Printf("%s\n", res)
			return err
		},
	}

	// Check for terminal support and NO_COLOR
	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	noColor := os.Getenv("NO_COLOR") != "" || !isTTY

	if noColor {
		cli.InfoString = fmt.Sprintf
		cli.ErrorString = fmt.Sprintf
	}
	repl.NewlineAfter = true

	//_ = repl.Shell().Config.Set("editing-mode", "vi")
	_ = repl.Shell().Config.Set("history-autosuggest", true)
	_ = repl.Shell().Config.Set("usage-hint-always", true)

	name := strings.ToLower(cli.AppName)
	fileName := fmt.Sprintf(".%s.local.repl.history", name)

	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("unable to get current user: %v\n", err)
	}

	filePath := filepath.Join(currentUser.HomeDir, fileName)
	cli.LocalMenu = repl.NewMenu("local")
	cli.LocalMenu.AddHistorySourceFile(fmt.Sprintf("%s-local", cli.AppName), filePath)
	cli.LocalMenu.AddInterrupt(io.EOF, cli.ExitCtrlD)
	cli.LocalMenu.AddInterrupt(errors.New(os.Interrupt.String()), cli.ExitCtrlD)
	cli.LocalMenu.SetCommands(func() *cobra.Command { return cli.MainCmd })
	SetRecursiveHelpFunc(cli.RootCmd)
	cli.LocalPrompt = cli.LocalMenu.Prompt()

	cli.LocalPrompt.Primary = func() string {
		return cli.InfoString("local >") + " "
	}
	cli.LocalPrompt.Right = func() string {
		return cli.InfoString(time.Now().Format("03:04:05.000"))
	}
	cli.LocalPrompt.Transient = func() string { return "\x1b[1;30m>> \x1b[0m" }

	repl.SwitchMenu("local")

	if customizer != nil {
		err := customizer(cli)
		if err != nil {
			fmt.Printf("customizer error: %v\n", err)
		}
	}

	cli.AddCommand(&cobra.Command{
		Use:   "exec [command]",
		Short: "Execute a command in REPL mode",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commandStr := strings.Join(args, " ")
			if commandStr == "" {
				return nil
			}
			cmd.Printf("exec [command]: %s\n", commandStr)
			res, err := cli.ExecuteCommands(commandStr)
			if err != nil {
				cmd.Printf("exec error: %v\n", err)
				return err
			}
			fmt.Print(res)
			return nil
		},
	})

	return cli
}

func (c *CLI) AddCommand(cmd *cobra.Command) {
	c.RootCmd.AddCommand(cmd)
}

func (c *CLI) ReadFromPipe(cmd *cobra.Command) string {
	var input strings.Builder
	reader := bufio.NewReader(cmd.InOrStdin())
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		input.WriteString(line)
		if err == io.EOF {
			break
		}
	}
	return strings.TrimSpace(input.String())
}

func isInputFromPipe(cmd *cobra.Command) bool {
	input := cmd.InOrStdin()
	if file, ok := input.(*os.File); ok {
		fileInfo, err := file.Stat()
		if err != nil {
			return false
		}
		return (fileInfo.Mode() & os.ModeCharDevice) == 0
	}
	return false
}

func (c *CLI) ExecuteCommands(commandStr string) (string, error) {
	//fmt.Printf("[DEBUG] %s: ExecuteCommands: %s\n", time.Now().Format("15:04:05.000"), commandStr)

	var filteredCommands []string
	lines := strings.Split(commandStr, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "#") || trimmedLine == "" {
			continue
		}
		if commentIndex := strings.Index(trimmedLine, "#"); commentIndex != -1 {
			trimmedLine = strings.TrimSpace(trimmedLine[:commentIndex])
		}
		if trimmedLine != "" {
			filteredCommands = append(filteredCommands, trimmedLine)
		}
	}
	commandStr = strings.Join(filteredCommands, ";")

	commands := strings.Split(commandStr, ";")
	var finalOutput bytes.Buffer

	for _, cmdStr := range commands {
		cmdStr = strings.TrimSpace(cmdStr)
		if cmdStr == "" {
			continue
		}

		var fileOutput *os.File
		redirMatch := regexp.MustCompile(`(.*)>(.*)`).FindStringSubmatch(cmdStr)
		if len(redirMatch) == 3 {
			cmdStr = strings.TrimSpace(redirMatch[1])
			filePath := strings.TrimSpace(redirMatch[2])
			var err error
			fileOutput, err = os.Create(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to create file: %w", err)
			}
			defer fileOutput.Close()
		}

		//fmt.Printf("[DEBUG] %s: cmdStr: %s\n", time.Now().Format("15:04:05.000"), cmdStr)
		pipeCommands := strings.Split(cmdStr, "|")

		var inputBuffer *bytes.Buffer

		for i, pipeCmd := range pipeCommands {
			pipeCmd = strings.TrimSpace(pipeCmd)
			//fmt.Printf("[DEBUG] %s: pipeCmd[%d]: %s\n", time.Now().Format("15:04:05.000"), i, pipeCmd)
			args := strings.Split(pipeCmd, " ")
			//fmt.Printf("[DEBUG] %s: args: %v\n", time.Now().Format("15:04:05.000"), args)

			cmd := c.RootCmd
			cmd.SetArgs(args)

			var outputBuffer bytes.Buffer

			if finalOutput.Len() > 0 {
				//fmt.Printf("[DEBUG] %s: setting cmd input from final output buffer\n", time.Now().Format("15:04:05.000"))
				cmd.SetIn(&finalOutput)
			} else if inputBuffer != nil {
				//fmt.Printf("[DEBUG] %s: setting cmd input from previous output buffer\n", time.Now().Format("15:04:05.000"))
				cmd.SetIn(inputBuffer)
			} else if i == 0 && isInputFromPipe(cmd) {
				//fmt.Printf("[DEBUG] %s: piping from stdin\n", time.Now().Format("15:04:05.000"))
				cmd.SetIn(cmd.InOrStdin())
			}

			cmd.SetOut(&outputBuffer)

			if err := cmd.Execute(); err != nil {
				return "", fmt.Errorf("failed to execute command '%s': %w", pipeCmd, err)
			}

			// Append output to the final output buffer
			io.Copy(&finalOutput, &outputBuffer)
			// Prepare the output for piping to the next command if needed
			inputBuffer = &outputBuffer
		}

		// If file redirection is set, write to the file
		if fileOutput != nil && inputBuffer != nil {
			inputBuffer.WriteTo(fileOutput)
		}
	}

	return finalOutput.String(), nil
}
