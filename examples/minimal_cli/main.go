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
)

func main() {
	// This example demonstrates the modular command selection system.
	// You can choose which command groups to include in your CLI.

	customizer := func(exec *consolekit.CommandExecutor) error {
		// Option 1: Include only minimal commands (core + variables + scripting)
		// exec.AddCommands(consolekit.AddMinimalCmds(exec))

		// Option 2: Include standard recommended commands
		// exec.AddCommands(consolekit.AddStandardCmds(exec))

		// Option 3: Include all commands
		// exec.AddCommands(consolekit.AddAllCmds(exec))

		// Option 4: Selectively pick specific command groups (shown below)
		// This gives you complete control over which features to include

		// Core essentials (cls, exit, print, date)
		exec.AddCommands(consolekit.AddCoreCmds(exec))

		// Variable management (let, unset, vars, inc, dec)
		exec.AddCommands(consolekit.AddVariableCmds(exec))

		// Scripting support (run command)
		exec.AddCommands(consolekit.AddRun(exec, &Data))

		// Control flow (if, repeat, while, for, case, test)
		exec.AddCommands(consolekit.AddControlFlowCmds(exec))

		// Optional: Add specific feature groups as needed
		// exec.AddCommands(consolekit.AddNetworkCmds(exec))      // http
		// exec.AddCommands(consolekit.AddTimeCmds(exec))         // sleep, wait, watch
		// exec.AddCommands(consolekit.AddOSExecCmds(exec))       // osexec
		// exec.AddCommands(consolekit.AddJobCmds(exec))          // jobs, job, killall
		// exec.AddCommands(consolekit.AddAliasCmds(exec))        // alias management
		// exec.AddCommands(consolekit.AddHistoryCmds(exec))      // history
		// exec.AddCommands(consolekit.AddConfigCmds(exec))       // config
		// exec.AddCommands(consolekit.AddDataManipulationCmds(exec)) // json, yaml, csv
		// exec.AddCommands(consolekit.AddFormatCmds(exec))       // table, column, highlight
		// exec.AddCommands(consolekit.AddTemplateCmds(exec))     // templates
		// exec.AddCommands(consolekit.AddInteractiveCmds(exec))  // prompts
		// exec.AddCommands(consolekit.AddLoggingCmds(exec))      // audit logging
		// exec.AddCommands(consolekit.AddNotificationCmds(exec)) // notifications
		// exec.AddCommands(consolekit.AddScheduleCmds(exec))     // scheduled tasks
		// exec.AddCommands(consolekit.AddClipboardCmds(exec))    // clipboard
		// exec.AddCommands(consolekit.AddMCPCmds(exec))          // MCP integration

		// Add custom version command
		var verCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf("Minimal CLI Example\n")
			cmd.Printf("BuildDate    : %s\n", BuildDate)
			cmd.Printf("LatestCommit : %s\n", LatestCommit)
			cmd.Printf("Version      : %s\n", Version)
			cmd.Printf("\nThis example demonstrates modular command selection.\n")
			cmd.Printf("Only selected command groups are included.\n")
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

	executor, err := consolekit.NewCommandExecutor("minimalcli", customizer)
	if err != nil {
		fmt.Printf("consolekit.NewCommandExecutor error, %v\n", err)
		return
	}

	// Create REPL handler
	handler := consolekit.NewREPLHandler(executor)

	// Set prompt
	handler.SetPrompt(func() string {
		return fmt.Sprintf("\nminimal > ")
	})

	// Run will execute command-line args if present, otherwise start REPL
	err = handler.Run()
	if err != nil {
		fmt.Printf("Error, %v\n", err)
	}
}
