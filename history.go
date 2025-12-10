package consolekit

import (
	"bufio"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// getHistory reads history from file and returns as slice
func (c *CLI) getHistory() []string {
	var history []string

	if c.historyFile == "" {
		return history
	}

	file, err := os.Open(c.historyFile)
	if err != nil {
		return history
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			history = append(history, line)
		}
	}

	return history
}

func AddHistory(cli *CLI) func(cmd *cobra.Command) {

	return func(rootCmd *cobra.Command) {

		var historyCmd = &cobra.Command{
			Use:   "history",
			Short: "history related commands",
		}
		var historyClearCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf("clearing history not available\n")
		}

		var historyClearCmd = &cobra.Command{
			Use:   "clear",
			Short: "clear history",
			Run:   historyClearCmdFunc,
		}
		var historySearchCmdFunc = func(cmd *cobra.Command, args []string) {

			showDupes, _ := cmd.Flags().GetBool("show_dupes")
			filter := strings.ToLower(args[0])

			// Get history from file
			history := cli.getHistory()
			lines := len(history)
			cmd.Printf("History: %d\n\n", lines)

			cnt := 0
			seen := make(map[string]bool)

			for i := 0; i < lines; i++ {
				line := history[i]

				if !strings.Contains(strings.ToLower(line), filter) {
					continue
				}

				if !showDupes {
					if seen[line] {
						continue
					}
					seen[line] = true
				}
				cmd.Printf("%d: %s\n", i, line)
				cnt++
			}

		}

		var historySearchCmd = &cobra.Command{
			Use:     "search {filter} [--show_dupes]",
			Short:   "show history",
			Aliases: []string{"s"},
			Args:    cobra.ExactArgs(1),
			Run:     historySearchCmdFunc,
		}

		var historyLsCmdFunc = func(cmd *cobra.Command, args []string) {
			// Get history from file
			history := cli.getHistory()
			lines := len(history)
			cmd.Printf("History: %d\n\n", lines)
			showDupes, _ := cmd.Flags().GetBool("show_dupes")
			limit, _ := cmd.Flags().GetInt("limit")

			cnt := 0
			seen := make(map[string]bool)

			start := lines - limit
			if start < 0 {
				start = 0
			}

			for i := start; i < lines; i++ {
				line := history[i]
				if !showDupes {
					if seen[line] {
						continue
					}
					seen[line] = true
				}
				cmd.Printf("%d: %s\n", i, line)
				cnt++
				if cnt >= limit {
					break
				}
			}
		}

		var historyLsCmd = &cobra.Command{
			Use:     "list [--limit={n}] [--show_dupes]",
			Short:   "show history",
			Args:    cobra.NoArgs,
			Aliases: []string{"ls", "l"},
			Run:     historyLsCmdFunc,
		}

		historyCmd.AddCommand(historyClearCmd)
		historyCmd.AddCommand(historySearchCmd)
		historyCmd.AddCommand(historyLsCmd)

		historyLsCmd.Flags().BoolP("show_dupes", "d", false, "Show duplicate commands in the history. Example: 'ls --show_dupes'")
		historyLsCmd.Flags().IntVarP(new(int), "limit", "", 100, "limit records")
		historySearchCmd.Flags().BoolP("show_dupes", "d", false, "Show duplicate commands in the history. Example: 'ls --show_dupes'")

		rootCmd.AddCommand(historyCmd)
	}
}
