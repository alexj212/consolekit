package consolekit

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// AddUtilityCommands adds utility commands like which, time, etc.
func AddUtilityCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {

		// which command - shows if a command is an alias or built-in
		whichCmd := &cobra.Command{
			Use:   "which [command]",
			Short: "Show information about a command",
			Long:  `Show if a command is an alias, variable, or built-in command`,
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				cmdName := args[0]

				// Check if it's an alias
				if val, ok := cli.aliases.Get(cmdName); ok {
					cmd.Printf("%s: alias for %q\n", cmdName, val)
					return
				}

				// Check if it's a variable
				varName := "@" + cmdName
				if val, ok := cli.Defaults.Get(varName); ok {
					cmd.Printf("%s: variable with value %q\n", cmdName, val)
					return
				}

				// Check if it's a built-in command by trying to find it
				found := false
				rootCmd.VisitParents(func(c *cobra.Command) {
					if c.Name() == cmdName {
						found = true
					}
				})

				// Check root command's children
				for _, c := range rootCmd.Commands() {
					if c.Name() == cmdName || contains(c.Aliases, cmdName) {
						cmd.Printf("%s: built-in command\n", cmdName)
						return
					}
				}

				if !found {
					cmd.Printf("%s: not found\n", cmdName)
				}
			},
		}
		rootCmd.AddCommand(whichCmd)

		// time command - measure execution time of a command
		timeCmd := &cobra.Command{
			Use:   "time [command...]",
			Short: "Measure execution time of a command",
			Long:  `Execute a command and report how long it took`,
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				// Reconstruct the command line
				line := strings.Join(args, " ")

				// Measure execution time
				start := time.Now()
				output, err := cli.ExecuteLine(line, nil)
				duration := time.Since(start)

				// Print command output
				if output != "" {
					cmd.Print(output)
					if !strings.HasSuffix(output, "\n") {
						cmd.Println()
					}
				}

				// Print timing information
				if err != nil {
					cmd.Printf("\nCommand failed after %v: %v\n", duration, err)
				} else {
					cmd.Printf("\nCommand completed in %v\n", duration)
				}
			},
		}
		rootCmd.AddCommand(timeCmd)

		// dot command - execute script in current context (like bash source)
		dotCmd := &cobra.Command{
			Use:   ". [script]",
			Short: "Execute script in current context",
			Long:  `Execute a script file in the current context, similar to bash 'source' command`,
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				scriptPath := args[0]
				scriptArgs := args[1:]

				// Load the script
				lines, err := LoadScript(cli.Scripts, cmd, scriptPath)
				if err != nil {
					cmd.PrintErrf("Error loading script: %v\n", err)
					return
				}

				// Execute each line in the current context (no scoped defaults)
				// This means variables set in the script remain after execution
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}

					// Replace script arguments
					for i, arg := range scriptArgs {
						argToken := fmt.Sprintf("@arg%d", i)
						line = strings.ReplaceAll(line, argToken, arg)
					}

					// Execute in current context
					output, err := cli.ExecuteLine(line, nil)
					if output != "" {
						cmd.Print(output)
						if !strings.HasSuffix(output, "\n") {
							cmd.Println()
						}
					}
					if err != nil {
						cmd.PrintErrf("Error executing line: %v\n", err)
						return
					}
				}
			},
		}
		rootCmd.AddCommand(dotCmd)
	}
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
