package consolekit

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ResetAllFlags Function to reset all flags to their default values
func ResetAllFlags(cmd *cobra.Command) {
	//Printf("LocalRootReplCmd resetAllFlags %s\n", cmd.Use)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
	})
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
	})

	for _, subCmd := range cmd.Commands() {
		ResetAllFlags(subCmd)
	}
}

// SetRecursiveHelpFunc function to set custom HelpFunc for a command and all its subcommands
func SetRecursiveHelpFunc(cmd *cobra.Command) {
	// Store the original help function
	originalHelpFunc := cmd.HelpFunc()

	// Set a new HelpFunc that includes resetting flags after help is shown
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		// Display the original help message
		originalHelpFunc(c, args)
		// After help is displayed, reset all flags to their default values
		ResetAllFlags(c)
	})

	// Recursively apply this to all subcommands
	for _, subCmd := range cmd.Commands() {
		SetRecursiveHelpFunc(subCmd)
	}
}

// ResetHelpFlag resets the help flag for a single command
func ResetHelpFlag(cmd *cobra.Command) {
	helpFlag := cmd.Flags().Lookup("help")
	if helpFlag != nil && helpFlag.Changed {
		_ = helpFlag.Value.Set("false")
		helpFlag.Changed = false
	}
}

// ResetHelpFlagRecursively resets the help flag for a command and all of its children
func ResetHelpFlagRecursively(cmd *cobra.Command) {
	ResetHelpFlag(cmd) // Reset help flag for the current command

	for _, childCmd := range cmd.Commands() {
		ResetHelpFlagRecursively(childCmd) // Recursively reset help flag for each child
	}
}

