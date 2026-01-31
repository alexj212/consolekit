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

// HistoryBookmark represents a bookmarked command.
type HistoryBookmark struct {
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// HistoryManager provides transport-agnostic command history management.
// Each transport can configure its own history file path.
type HistoryManager struct {
	historyFile string
	appName     string
}

// NewHistoryManager creates a new history manager.
func NewHistoryManager(appName, historyFile string) *HistoryManager {
	return &HistoryManager{
		appName:     appName,
		historyFile: historyFile,
	}
}

// SetHistoryFile sets the history file path.
func (hm *HistoryManager) SetHistoryFile(path string) {
	hm.historyFile = path
}

// GetHistory reads history from file and returns as slice.
func (hm *HistoryManager) GetHistory() []string {
	var history []string

	if hm.historyFile == "" {
		return history
	}

	file, err := os.Open(hm.historyFile)
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

// AppendHistory adds a command to the history file.
func (hm *HistoryManager) AppendHistory(command string) error {
	if hm.historyFile == "" {
		return nil // History disabled
	}

	// Create parent directory if needed
	dir := filepath.Dir(hm.historyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	f, err := os.OpenFile(hm.historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(command + "\n"); err != nil {
		return fmt.Errorf("failed to write to history: %w", err)
	}

	return nil
}

// ClearHistory removes all history.
func (hm *HistoryManager) ClearHistory() error {
	if hm.historyFile == "" {
		return nil
	}

	return os.WriteFile(hm.historyFile, []byte{}, 0644)
}

// GetBookmarksFile returns the path to bookmarks file.
func (hm *HistoryManager) GetBookmarksFile() string {
	if hm.historyFile == "" {
		return ""
	}
	dir := filepath.Dir(hm.historyFile)
	return filepath.Join(dir, "."+strings.ToLower(hm.appName)+".bookmarks")
}

// LoadBookmarks loads bookmarks from file.
func (hm *HistoryManager) LoadBookmarks() (map[string]*HistoryBookmark, error) {
	bookmarks := make(map[string]*HistoryBookmark)
	bookmarksFile := hm.GetBookmarksFile()
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

// SaveBookmarks saves bookmarks to file.
func (hm *HistoryManager) SaveBookmarks(bookmarks map[string]*HistoryBookmark) error {
	bookmarksFile := hm.GetBookmarksFile()
	if bookmarksFile == "" {
		return fmt.Errorf("bookmarks file not available")
	}

	// Create parent directory if needed
	currentUser, err := user.Current()
	if err == nil {
		name := strings.ToLower(hm.appName)
		dir := filepath.Join(currentUser.HomeDir, fmt.Sprintf(".%s", name))
		_ = os.MkdirAll(dir, 0755)
	}

	data, err := json.MarshalIndent(bookmarks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(bookmarksFile, data, 0644)
}

// AddHistory registers history-related commands.
func AddHistory(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {

		var historyCmd = &cobra.Command{
			Use:   "history",
			Short: "History related commands",
		}

		// history clear
		var historyClearCmd = &cobra.Command{
			Use:   "clear",
			Short: "Clear history",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				if err := exec.HistoryManager.ClearHistory(); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to clear history: %v", err))
					return
				}

				cmd.Println("History cleared")
			},
		}

		// history search
		var historySearchCmd = &cobra.Command{
			Use:     "search {filter} [--show_dupes]",
			Short:   "Search history",
			Aliases: []string{"s"},
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				showDupes, _ := cmd.Flags().GetBool("show_dupes")
				filter := strings.ToLower(args[0])

				history := exec.HistoryManager.GetHistory()
				lines := len(history)
				cmd.Printf("History: %d entries\n\n", lines)

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

				if cnt == 0 {
					cmd.Println("No matches found")
				}
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		historySearchCmd.Flags().BoolP("show_dupes", "d", false, "Show duplicate commands")

		// history list
		var historyLsCmd = &cobra.Command{
			Use:     "list [--limit={n}] [--show_dupes]",
			Short:   "Show history",
			Args:    cobra.NoArgs,
			Aliases: []string{"ls", "l"},
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				history := exec.HistoryManager.GetHistory()
				lines := len(history)
				cmd.Printf("History: %d entries\n\n", lines)

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
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		historyLsCmd.Flags().BoolP("show_dupes", "d", false, "Show duplicate commands")
		historyLsCmd.Flags().IntP("limit", "n", 100, "Limit number of entries")

		// history replay
		var replayCmd = &cobra.Command{
			Use:   "replay [index]",
			Short: "Re-execute a command from history",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				index, err := strconv.Atoi(args[0])
				if err != nil {
					cmd.PrintErrln("Invalid history index")
					return
				}

				history := exec.HistoryManager.GetHistory()
				if index < 0 || index >= len(history) {
					cmd.PrintErrln("History index out of range")
					return
				}

				command := history[index]
				cmd.Printf("Replaying: %s\n", command)

				output, err := exec.Execute(command, nil)
				if output != "" {
					cmd.Print(output)
					if !strings.HasSuffix(output, "\n") {
						cmd.Println()
					}
				}
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error: %v", err))
				}
			},
		}

		// history stats
		var statsCmd = &cobra.Command{
			Use:   "stats",
			Short: "Show history statistics",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				history := exec.HistoryManager.GetHistory()
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

				// Simple sort by count (bubble sort)
				for i := 0; i < len(counts); i++ {
					for j := i + 1; j < len(counts); j++ {
						if counts[j].count > counts[i].count {
							counts[i], counts[j] = counts[j], counts[i]
						}
					}
				}

				cmd.Printf("Total commands: %d\n", total)
				cmd.Printf("Unique commands: %d\n", len(unique))
				cmd.Printf("Duplicates: %d\n\n", total-len(unique))
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

		// Bookmark subcommands
		var bookmarkCmd = &cobra.Command{
			Use:   "bookmark",
			Short: "Manage history bookmarks",
		}

		var bookmarkAddCmd = &cobra.Command{
			Use:   "add [name] [command...]",
			Short: "Bookmark a command",
			Args:  cobra.MinimumNArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				name := args[0]
				command := strings.Join(args[1:], " ")
				desc, _ := cmd.Flags().GetString("description")

				bookmarks, err := exec.HistoryManager.LoadBookmarks()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to load bookmarks: %v", err))
					return
				}

				bookmarks[name] = &HistoryBookmark{
					Name:        name,
					Command:     command,
					Description: desc,
					CreatedAt:   time.Now(),
				}

				if err := exec.HistoryManager.SaveBookmarks(bookmarks); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to save bookmark: %v", err))
					return
				}

				cmd.Printf("Bookmarked '%s'\n", name)
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		bookmarkAddCmd.Flags().StringP("description", "d", "", "Description of the bookmark")

		var bookmarkListCmd = &cobra.Command{
			Use:     "list",
			Short:   "List all bookmarks",
			Aliases: []string{"ls"},
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				bookmarks, err := exec.HistoryManager.LoadBookmarks()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to load bookmarks: %v", err))
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

		var bookmarkRunCmd = &cobra.Command{
			Use:   "run [name]",
			Short: "Run a bookmarked command",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				name := args[0]

				bookmarks, err := exec.HistoryManager.LoadBookmarks()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to load bookmarks: %v", err))
					return
				}

				bm, ok := bookmarks[name]
				if !ok {
					cmd.PrintErrln(fmt.Sprintf("Bookmark '%s' not found", name))
					return
				}

				output, err := exec.Execute(bm.Command, nil)
				if output != "" {
					cmd.Print(output)
					if !strings.HasSuffix(output, "\n") {
						cmd.Println()
					}
				}
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error: %v", err))
				}
			},
		}

		var bookmarkRemoveCmd = &cobra.Command{
			Use:     "remove [name]",
			Short:   "Remove a bookmark",
			Aliases: []string{"rm"},
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				name := args[0]

				bookmarks, err := exec.HistoryManager.LoadBookmarks()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to load bookmarks: %v", err))
					return
				}

				if _, ok := bookmarks[name]; !ok {
					cmd.PrintErrln(fmt.Sprintf("Bookmark '%s' not found", name))
					return
				}

				delete(bookmarks, name)

				if err := exec.HistoryManager.SaveBookmarks(bookmarks); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to save bookmarks: %v", err))
					return
				}

				cmd.Printf("Removed bookmark '%s'\n", name)
			},
		}

		bookmarkCmd.AddCommand(bookmarkAddCmd)
		bookmarkCmd.AddCommand(bookmarkListCmd)
		bookmarkCmd.AddCommand(bookmarkRunCmd)
		bookmarkCmd.AddCommand(bookmarkRemoveCmd)

		historyCmd.AddCommand(historyClearCmd)
		historyCmd.AddCommand(historySearchCmd)
		historyCmd.AddCommand(historyLsCmd)
		historyCmd.AddCommand(bookmarkCmd)
		historyCmd.AddCommand(replayCmd)
		historyCmd.AddCommand(statsCmd)

		rootCmd.AddCommand(historyCmd)
	}
}
