package consolekit

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AddPromptCommands adds interactive prompt demonstration commands
func AddPromptCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Demo command to show various prompt types
		var promptDemoCmd = &cobra.Command{
			Use:   "prompt-demo",
			Short: "Demonstrate interactive prompts",
			Long:  "Show examples of different types of interactive prompts available",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Println(cli.InfoString("=== Interactive Prompts Demo ==="))
				cmd.Println()

				// 1. Simple confirmation
				cmd.Println(cli.InfoString("1. Simple Confirmation"))
				if cli.Confirm("Do you want to continue?") {
					cmd.Println(cli.SuccessString("✓ User confirmed"))
				} else {
					cmd.Println(cli.InfoString("User declined"))
				}
				cmd.Println()

				// 2. String input
				cmd.Println(cli.InfoString("2. String Input"))
				name := cli.Prompt("Enter your name")
				if name != "" {
					cmd.Printf("Hello, %s!\n", name)
				}
				cmd.Println()

				// 3. String input with default
				cmd.Println(cli.InfoString("3. String Input with Default"))
				color := cli.PromptDefault("Choose a color", "blue")
				cmd.Printf("You selected: %s\n", color)
				cmd.Println()

				// 4. Single selection
				cmd.Println(cli.InfoString("4. Single Selection"))
				options := []string{"Red", "Green", "Blue", "Yellow"}
				choice := cli.Select("Choose your favorite color", options)
				if choice != "" {
					cmd.Printf("You chose: %s\n", choice)
				} else {
					cmd.Println("No selection made")
				}
				cmd.Println()

				// 5. Selection with default
				cmd.Println(cli.InfoString("5. Selection with Default"))
				environments := []string{"Development", "Staging", "Production"}
				env := cli.SelectWithDefault("Select deployment environment", environments, 0)
				cmd.Printf("Selected environment: %s\n", env)
				cmd.Println()

				// 6. Multi-selection
				cmd.Println(cli.InfoString("6. Multi-Selection"))
				features := []string{"Authentication", "Logging", "Caching", "Monitoring"}
				selected := cli.MultiSelect("Select features to enable", features)
				if len(selected) > 0 {
					cmd.Println("Selected features:")
					for _, feature := range selected {
						cmd.Printf("  - %s\n", feature)
					}
				} else {
					cmd.Println("No features selected")
				}
				cmd.Println()

				// 7. Integer input
				cmd.Println(cli.InfoString("7. Integer Input"))
				count := cli.PromptInteger("Enter a number", 10)
				cmd.Printf("You entered: %d\n", count)
				cmd.Println()

				cmd.Println(cli.SuccessString("=== Demo Complete ==="))
			},
		}

		// Confirm command - generic confirmation command
		var confirmCmd = &cobra.Command{
			Use:   "confirm [message]",
			Short: "Prompt for confirmation",
			Long:  "Ask the user for yes/no confirmation",
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				message := args[0]
				if cli.Confirm(message) {
					cmd.Println("yes")
					cmd.SetContext(cmd.Context()) // Success
				} else {
					cmd.Println("no")
					cmd.SetContext(cmd.Context()) // No error, just declined
				}
			},
		}

		// Input command - prompt for string input
		var inputDefault string
		var inputCmd = &cobra.Command{
			Use:   "input [message]",
			Short: "Prompt for text input",
			Long:  "Ask the user to enter text",
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				message := args[0]
				var result string
				if inputDefault != "" {
					result = cli.PromptDefault(message, inputDefault)
				} else {
					result = cli.Prompt(message)
				}
				cmd.Println(result)
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		inputCmd.Flags().StringVar(&inputDefault, "default", "", "Default value if user enters nothing")

		// Select command - single selection from list
		var selectDefault int
		var selectCmd = &cobra.Command{
			Use:   "select [message] [option1] [option2] ...",
			Short: "Prompt for single selection",
			Long:  "Ask the user to select one option from a list",
			Args:  cobra.MinimumNArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				message := args[0]
				options := args[1:]

				var result string
				if selectDefault >= 0 {
					result = cli.SelectWithDefault(message, options, selectDefault)
				} else {
					result = cli.Select(message, options)
				}

				if result != "" {
					cmd.Println(result)
				}
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		selectCmd.Flags().IntVar(&selectDefault, "default", -1, "Default option index (0-based)")

		// Multi-select command
		var multiSelectCmd = &cobra.Command{
			Use:   "multiselect [message] [option1] [option2] ...",
			Short: "Prompt for multiple selections",
			Long:  "Ask the user to select multiple options from a list",
			Args:  cobra.MinimumNArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				message := args[0]
				options := args[1:]

				results := cli.MultiSelect(message, options)
				for _, result := range results {
					cmd.Println(result)
				}
			},
		}

		rootCmd.AddCommand(promptDemoCmd)
		rootCmd.AddCommand(confirmCmd)
		rootCmd.AddCommand(inputCmd)
		rootCmd.AddCommand(selectCmd)
		rootCmd.AddCommand(multiSelectCmd)
	}
}

// AddYesFlag adds a --yes flag to a command to skip confirmation prompts
func AddYesFlag(cmd *cobra.Command, yesVar *bool) {
	cmd.Flags().BoolVar(yesVar, "yes", false, "Skip confirmation prompts")
}

// AddDryRunFlag adds a --dry-run flag to a command to simulate without executing
func AddDryRunFlag(cmd *cobra.Command, dryRunVar *bool) {
	cmd.Flags().BoolVar(dryRunVar, "dry-run", false, "Show what would happen without executing")
}

// ConfirmOrSkip checks if the --yes flag is set, or prompts for confirmation
// Returns true if confirmed or yes flag is set
func ConfirmOrSkip(cli *CLI, yesFlag bool, message string) bool {
	if yesFlag {
		return true
	}
	return cli.Confirm(message)
}

// ConfirmDestructiveOrSkip checks if the --yes flag is set, or prompts for destructive confirmation
// Returns true if confirmed or yes flag is set
func ConfirmDestructiveOrSkip(cli *CLI, yesFlag bool, message string) bool {
	if yesFlag {
		return true
	}
	return cli.ConfirmDestructive(message)
}

// ShowDryRun prints a dry-run message if the flag is set
func ShowDryRun(cmd *cobra.Command, cli *CLI, dryRun bool, action string) {
	if dryRun {
		cmd.Println(cli.InfoString(fmt.Sprintf("[DRY RUN] Would %s", action)))
	}
}
