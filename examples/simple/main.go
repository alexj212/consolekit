package main

import (
	"embed"
	"fmt"

	"github.com/alexj212/consolekit"
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

func main() {
	// Create command executor (core execution engine)
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

	executor, err := consolekit.NewCommandExecutor("simple", customizer)
	if err != nil {
		fmt.Printf("consolekit.NewCommandExecutor error, %v\n", err)
		return
	}

	// Create REPL handler (local terminal interface)
	handler := consolekit.NewREPLHandler(executor)

	// By default, NewREPLHandler() uses ReflectiveAdapter (reeflective/console)
	// To switch to a different display backend (e.g., bubbletea):
	//   adapter := consolekit.NewBubbletteaAdapter("simple")
	//   handler.SetDisplayAdapter(adapter)

	// Set prompt with leading newline (helps with terminal scrolling issues)
	handler.SetPrompt(func() string {
		return fmt.Sprintf("\nsimple > ")
	})

	// Run will execute command-line args if present, otherwise start REPL
	err = handler.Run()
	if err != nil {
		fmt.Printf("Error, %v\n", err)
	}
}
