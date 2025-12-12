package consolekit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// HistoryBookmark represents a bookmarked command
type HistoryBookmark struct {
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

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

// getBookmarksFile returns the path to bookmarks file
func (c *CLI) getBookmarksFile() string {
	if c.historyFile == "" {
		return ""
	}
	dir := filepath.Dir(c.historyFile)
	return filepath.Join(dir, "."+strings.ToLower(c.AppName)+".bookmarks")
}

// loadBookmarks loads bookmarks from file
func (c *CLI) loadBookmarks() (map[string]*HistoryBookmark, error) {
	bookmarks := make(map[string]*HistoryBookmark)
	bookmarksFile := c.getBookmarksFile()
	if bookmarksFile == "" {
		return bookmarks, nil
	}

	data, err := os.ReadFile(bookmarksFile)
	if err != nil {
		if os.IsNotExist(err) {
			return bookmarks, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &bookmarks); err != nil {
		return nil, err
	}

	return bookmarks, nil
}

// saveBookmarks saves bookmarks to file
func (c *CLI) saveBookmarks(bookmarks map[string]*HistoryBookmark) error {
	bookmarksFile := c.getBookmarksFile()
	if bookmarksFile == "" {
		return fmt.Errorf("bookmarks file not available")
	}

	// Create parent directory if needed
	currentUser, err := user.Current()
	if err == nil {
		name := strings.ToLower(c.AppName)
		dir := filepath.Join(currentUser.HomeDir, fmt.Sprintf(".%s", name))
		_ = os.MkdirAll(dir, 0755)
	}

	data, err := json.MarshalIndent(bookmarks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bookmarksFile, data, 0644)
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

		// history bookmark - bookmark a command
		var bookmarkAddCmd = &cobra.Command{
			Use:   "add [name] [command]",
			Short: "Bookmark a command",
			Args:  cobra.MinimumNArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				name := args[0]
				command := strings.Join(args[1:], " ")
				desc, _ := cmd.Flags().GetString("description")

				bookmarks, err := cli.loadBookmarks()
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to load bookmarks: %v", err)))
					return
				}

				bookmarks[name] = &HistoryBookmark{
					Name:        name,
					Command:     command,
					Description: desc,
					CreatedAt:   time.Now(),
				}

				if err := cli.saveBookmarks(bookmarks); err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to save bookmark: %v", err)))
					return
				}

				cmd.Println(cli.SuccessString(fmt.Sprintf("Bookmarked '%s'", name)))
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		bookmarkAddCmd.Flags().StringP("description", "d", "", "Description of the bookmark")

		// history bookmark list
		var bookmarkListCmd = &cobra.Command{
			Use:     "list",
			Short:   "List all bookmarks",
			Aliases: []string{"ls"},
			Run: func(cmd *cobra.Command, args []string) {
				bookmarks, err := cli.loadBookmarks()
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to load bookmarks: %v", err)))
					return
				}

				if len(bookmarks) == 0 {
					cmd.Println("No bookmarks")
					return
				}

				cmd.Println("Bookmarks:")
				for name, bm := range bookmarks {
					if bm.Description != "" {
						cmd.Printf("  %s: %s (%s)\n", name, bm.Command, bm.Description)
					} else {
						cmd.Printf("  %s: %s\n", name, bm.Command)
					}
				}
			},
		}

		// history bookmark run
		var bookmarkRunCmd = &cobra.Command{
			Use:   "run [name]",
			Short: "Run a bookmarked command",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				name := args[0]

				bookmarks, err := cli.loadBookmarks()
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to load bookmarks: %v", err)))
					return
				}

				bm, ok := bookmarks[name]
				if !ok {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Bookmark '%s' not found", name)))
					return
				}

				output, err := cli.ExecuteLine(bm.Command, nil)
				if output != "" {
					cmd.Print(output)
					if !strings.HasSuffix(output, "\n") {
						cmd.Println()
					}
				}
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Error: %v", err)))
				}
			},
		}

		// history bookmark remove
		var bookmarkRemoveCmd = &cobra.Command{
			Use:     "remove [name]",
			Short:   "Remove a bookmark",
			Aliases: []string{"rm"},
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				name := args[0]

				bookmarks, err := cli.loadBookmarks()
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to load bookmarks: %v", err)))
					return
				}

				if _, ok := bookmarks[name]; !ok {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Bookmark '%s' not found", name)))
					return
				}

				delete(bookmarks, name)

				if err := cli.saveBookmarks(bookmarks); err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to save bookmarks: %v", err)))
					return
				}

				cmd.Println(cli.SuccessString(fmt.Sprintf("Removed bookmark '%s'", name)))
			},
		}

		var bookmarkCmd = &cobra.Command{
			Use:   "bookmark",
			Short: "Manage history bookmarks",
		}
		bookmarkCmd.AddCommand(bookmarkAddCmd)
		bookmarkCmd.AddCommand(bookmarkListCmd)
		bookmarkCmd.AddCommand(bookmarkRunCmd)
		bookmarkCmd.AddCommand(bookmarkRemoveCmd)

		// history replay - re-execute a command from history
		var replayCmd = &cobra.Command{
			Use:   "replay [index]",
			Short: "Re-execute a command from history",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				index, err := strconv.Atoi(args[0])
				if err != nil {
					cmd.PrintErrln(cli.ErrorString("Invalid history index"))
					return
				}

				history := cli.getHistory()
				if index < 0 || index >= len(history) {
					cmd.PrintErrln(cli.ErrorString("History index out of range"))
					return
				}

				command := history[index]
				cmd.Println(cli.InfoString(fmt.Sprintf("Replaying: %s", command)))

				output, err := cli.ExecuteLine(command, nil)
				if output != "" {
					cmd.Print(output)
					if !strings.HasSuffix(output, "\n") {
						cmd.Println()
					}
				}
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Error: %v", err)))
				}
			},
		}

		// history stats - show statistics
		var statsCmd = &cobra.Command{
			Use:   "stats",
			Short: "Show history statistics",
			Run: func(cmd *cobra.Command, args []string) {
				history := cli.getHistory()
				total := len(history)

				if total == 0 {
					cmd.Println("No history")
					return
				}

				// Count unique commands
				unique := make(map[string]int)
				for _, line := range history {
					unique[line]++
				}

				// Find most used commands
				type cmdCount struct {
					cmd   string
					count int
				}
				var counts []cmdCount
				for cmd, count := range unique {
					counts = append(counts, cmdCount{cmd, count})
				}

				// Simple sort by count (bubble sort for small data)
				for i := 0; i < len(counts); i++ {
					for j := i + 1; j < len(counts); j++ {
						if counts[j].count > counts[i].count {
							counts[i], counts[j] = counts[j], counts[i]
						}
					}
				}

				cmd.Println(fmt.Sprintf("Total commands: %d", total))
				cmd.Println(fmt.Sprintf("Unique commands: %d", len(unique)))
				cmd.Println(fmt.Sprintf("Duplicates: %d", total-len(unique)))
				cmd.Println()
				cmd.Println("Top 10 most used commands:")

				limit := 10
				if len(counts) < limit {
					limit = len(counts)
				}

				for i := 0; i < limit; i++ {
					cmd.Printf("  %d. %s (%d times)\n", i+1, counts[i].cmd, counts[i].count)
				}
			},
		}

		historyCmd.AddCommand(historyClearCmd)
		historyCmd.AddCommand(historySearchCmd)
		historyCmd.AddCommand(historyLsCmd)
		historyCmd.AddCommand(bookmarkCmd)
		historyCmd.AddCommand(replayCmd)
		historyCmd.AddCommand(statsCmd)

		historyLsCmd.Flags().BoolP("show_dupes", "d", false, "Show duplicate commands in the history. Example: 'ls --show_dupes'")
		historyLsCmd.Flags().IntVarP(new(int), "limit", "", 100, "limit records")
		historySearchCmd.Flags().BoolP("show_dupes", "d", false, "Show duplicate commands in the history. Example: 'ls --show_dupes'")

		rootCmd.AddCommand(historyCmd)
	}
}
