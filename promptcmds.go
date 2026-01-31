package consolekit

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AddPromptCommands adds interactive prompt demonstration commands
func AddPromptCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// Demo command to show various prompt types
		var promptDemoCmd = &cobra.Command{
			Use:   "prompt-demo",
			Short: "Demonstrate interactive prompts",
			Long:  "Show examples of different types of interactive prompts available",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Println(fmt.Sprintf("=== Interactive Prompts Demo ==="))
				cmd.Println()

				// 1. Simple confirmation
				cmd.Println(fmt.Sprintf("1. Simple Confirmation"))
				if exec.Confirm("Do you want to continue?") {
					cmd.Println(fmt.Sprintf("✓ User confirmed"))
				} else {
					cmd.Println(fmt.Sprintf("User declined"))
				}
				cmd.Println()

				// 2. String input
				cmd.Println(fmt.Sprintf("2. String Input"))
				name := exec.Prompt("Enter your name")
				if name != "" {
					cmd.Printf("Hello, %s!\n", name)
				}
				cmd.Println()

				// 3. String input with default
				cmd.Println(fmt.Sprintf("3. String Input with Default"))
				color := exec.PromptDefault("Choose a color", "blue")
				cmd.Printf("You selected: %s\n", color)
				cmd.Println()

				// 4. Single selection
				cmd.Println(fmt.Sprintf("4. Single Selection"))
				options := []string{"Red", "Green", "Blue", "Yellow"}
				choice := exec.Select("Choose your favorite color", options)
				if choice != "" {
					cmd.Printf("You chose: %s\n", choice)
				} else {
					cmd.Println("No selection made")
				}
				cmd.Println()

				// 5. Selection with default
				cmd.Println(fmt.Sprintf("5. Selection with Default"))
				environments := []string{"Development", "Staging", "Production"}
				env := exec.SelectWithDefault("Select deployment environment", environments, 0)
				cmd.Printf("Selected environment: %s\n", env)
				cmd.Println()

				// 6. Multi-selection
				cmd.Println(fmt.Sprintf("6. Multi-Selection"))
				features := []string{"Authentication", "Logging", "Caching", "Monitoring"}
				selected := exec.MultiSelect("Select features to enable", features)
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
				cmd.Println(fmt.Sprintf("7. Integer Input"))
				count := exec.PromptInteger("Enter a number", 10)
				cmd.Printf("You entered: %d\n", count)
				cmd.Println()

				cmd.Println(fmt.Sprintf("=== Demo Complete ==="))
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
				if exec.Confirm(message) {
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
					result = exec.PromptDefault(message, inputDefault)
				} else {
					result = exec.Prompt(message)
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
					result = exec.SelectWithDefault(message, options, selectDefault)
				} else {
					result = exec.Select(message, options)
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

				results := exec.MultiSelect(message, options)
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
func ConfirmOrSkip(exec *CommandExecutor, yesFlag bool, message string) bool {
	if yesFlag {
		return true
	}
	return exec.Confirm(message)
}

// ConfirmDestructiveOrSkip checks if the --yes flag is set, or prompts for destructive confirmation
// Returns true if confirmed or yes flag is set
func ConfirmDestructiveOrSkip(exec *CommandExecutor, yesFlag bool, message string) bool {
	if yesFlag {
		return true
	}
	return exec.ConfirmDestructive(message)
}

// ShowDryRun prints a dry-run message if the flag is set
func ShowDryRun(cmd *cobra.Command, exec *CommandExecutor, dryRun bool, action string) {
	if dryRun {
		cmd.Println(fmt.Sprintf("[DRY RUN] Would %s", action))
	}
}
