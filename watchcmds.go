package consolekit

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// AddWatchCommand adds a watch command that repeatedly executes a command
func AddWatchCommand(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		var interval time.Duration
		var count int
		var clearScreen bool

		var watchCmd = &cobra.Command{
			Use:   "watch [command]",
			Short: "Execute a command repeatedly",
			Long: `Execute a command repeatedly at a specified interval.
By default, runs indefinitely until interrupted (Ctrl+C).
Use --count to limit the number of executions.

Examples:
  watch "date"                    # Run every 2 seconds (default)
  watch --interval 5s "jobs"      # Run every 5 seconds
  watch --count 10 "date"         # Run 10 times
  watch --clear "date"            # Clear screen before each run`,
			Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				command := args[0]
				iteration := 0

				// Print header
				cmd.Println(cli.InfoString(fmt.Sprintf("Every %s: %s", interval, command)))
				cmd.Println()

				// Main watch loop
				for {
					iteration++

					// Clear screen if requested
					if clearScreen {
						fmt.Print("\033[H\033[2J")
					}

					// Print iteration info
					cmd.Println(cli.InfoString(fmt.Sprintf("Iteration %d at %s:", iteration, time.Now().Format("15:04:05"))))
					cmd.Println(strings.Repeat("-", 60))

					// Execute command
					output, err := cli.ExecuteLine(command, nil)
					if err != nil {
						cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Error: %v", err)))
					} else if output != "" {
						cmd.Print(output)
						if !strings.HasSuffix(output, "\n") {
							cmd.Println()
						}
					}

					// Check if we've reached the count limit
					if count > 0 && iteration >= count {
						break
					}

					// Wait for interval
					time.Sleep(interval)
				}

				cmd.Println()
				cmd.Println(cli.SuccessString(fmt.Sprintf("Watch completed after %d iterations", iteration)))
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}

		watchCmd.Flags().DurationVarP(&interval, "interval", "n", 2*time.Second, "Interval between executions (e.g., 2s, 500ms, 1m)")
		watchCmd.Flags().IntVarP(&count, "count", "c", 0, "Number of times to execute (0 = infinite)")
		watchCmd.Flags().BoolVar(&clearScreen, "clear", false, "Clear screen before each execution")

		rootCmd.AddCommand(watchCmd)
	}
}
