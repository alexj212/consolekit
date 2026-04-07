package consolekit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
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

// historyEntry represents a JSON history entry written by reeflective/console.
type historyEntry struct {
	DateTime string `json:"datetime"`
	Block    string `json:"block"`
}

// GetHistory reads history from file and returns as slice.
// Handles both plain text lines and reeflective JSON format ({"datetime":"...","block":"..."}).
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
		if line == "" {
			continue
		}

		// Try to parse as JSON history entry
		if strings.HasPrefix(line, "{") {
			var entry historyEntry
			if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.Block != "" {
				history = append(history, entry.Block)
				continue
			}
		}

		// Plain text line
		history = append(history, line)
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

// GetRawLines reads the raw lines from the history file without parsing.
func (hm *HistoryManager) GetRawLines() []string {
	var lines []string

	if hm.historyFile == "" {
		return lines
	}

	file, err := os.Open(hm.historyFile)
	if err != nil {
		return lines
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

// parseRawLine extracts the command from a raw history line (JSON or plain text).
func parseRawLine(line string) string {
	if strings.HasPrefix(line, "{") {
		var entry historyEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.Block != "" {
			return entry.Block
		}
	}
	return line
}

// WriteRawLines writes raw lines back to the history file.
func (hm *HistoryManager) WriteRawLines(lines []string) error {
	if hm.historyFile == "" {
		return fmt.Errorf("history file not configured")
	}

	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	return os.WriteFile(hm.historyFile, []byte(buf.String()), 0644)
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
			Use:     "search {filter} [--show-dupes]",
			Short:   "Search history",
			Aliases: []string{"s"},
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				showDupes, _ := cmd.Flags().GetBool("show-dupes")
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
		historySearchCmd.Flags().BoolP("show-dupes", "d", false, "Show duplicate commands")

		// history list
		var historyLsCmd = &cobra.Command{
			Use:     "list [prefix] [--limit={n}] [--show-dupes]",
			Short:   "Show history, optionally filtered by prefix",
			Args:    cobra.MaximumNArgs(1),
			Aliases: []string{"ls", "l"},
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				history := exec.HistoryManager.GetHistory()
				lines := len(history)
				cmd.Printf("History: %d entries\n\n", lines)

				showDupes, _ := cmd.Flags().GetBool("show-dupes")
				limit, _ := cmd.Flags().GetInt("limit")

				var prefix string
				if len(args) > 0 {
					prefix = strings.ToLower(args[0])
				}

				// When prefix is set, collect unique sorted matches
				if prefix != "" {
					seen := make(map[string]bool)
					var matches []string

					for i := 0; i < lines; i++ {
						line := history[i]
						if !strings.HasPrefix(strings.ToLower(line), prefix) {
							continue
						}
						if !showDupes && seen[line] {
							continue
						}
						seen[line] = true
						matches = append(matches, line)
					}

					sort.Strings(matches)

					cnt := 0
					for _, line := range matches {
						cmd.Printf("%s\n", line)
						cnt++
						if cnt >= limit {
							break
						}
					}

					if cnt == 0 {
						cmd.Println("No matches found")
					}
				} else {
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
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		historyLsCmd.Flags().BoolP("show-dupes", "d", false, "Show duplicate commands")
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

		// history last
		var lastCmd = &cobra.Command{
			Use:   "last [n]",
			Short: "Show or replay the last N commands",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				history := exec.HistoryManager.GetHistory()
				if len(history) == 0 {
					cmd.Println("No history")
					return
				}

				n := 1
				if len(args) > 0 {
					var err error
					n, err = strconv.Atoi(args[0])
					if err != nil || n < 1 {
						cmd.PrintErrln("Invalid count")
						return
					}
				}

				rerun, _ := cmd.Flags().GetBool("exec")

				start := len(history) - n
				if start < 0 {
					start = 0
				}

				for i := start; i < len(history); i++ {
					command := history[i]
					if rerun {
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
					} else {
						cmd.Printf("%d: %s\n", i, command)
					}
				}
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		lastCmd.Flags().BoolP("exec", "x", false, "Re-execute the last command(s)")

		// history delete
		var deleteCmd = &cobra.Command{
			Use:     "delete [index...]",
			Short:   "Delete history entries by index",
			Aliases: []string{"del", "rm"},
			Args:    cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				rawLines := exec.HistoryManager.GetRawLines()
				if len(rawLines) == 0 {
					cmd.Println("No history")
					return
				}

				// Parse indices to delete
				toDelete := make(map[int]bool)
				for _, arg := range args {
					idx, err := strconv.Atoi(arg)
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Invalid index: %s", arg))
						return
					}
					if idx < 0 || idx >= len(rawLines) {
						cmd.PrintErrln(fmt.Sprintf("Index out of range: %d (0-%d)", idx, len(rawLines)-1))
						return
					}
					toDelete[idx] = true
				}

				var remaining []string
				for i, line := range rawLines {
					if !toDelete[i] {
						remaining = append(remaining, line)
					}
				}

				if err := exec.HistoryManager.WriteRawLines(remaining); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to write history: %v", err))
					return
				}

				cmd.Printf("Deleted %d entries (%d remaining)\n", len(toDelete), len(remaining))
			},
		}

		// history dedupe
		var dedupeCmd = &cobra.Command{
			Use:   "dedupe",
			Short: "Remove duplicate entries from history",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				rawLines := exec.HistoryManager.GetRawLines()
				if len(rawLines) == 0 {
					cmd.Println("No history")
					return
				}

				consecutive, _ := cmd.Flags().GetBool("consecutive")

				var deduped []string
				if consecutive {
					// Remove only consecutive duplicates
					var prev string
					for _, line := range rawLines {
						command := parseRawLine(line)
						if command != prev {
							deduped = append(deduped, line)
						}
						prev = command
					}
				} else {
					// Remove all duplicates, keeping last occurrence
					seen := make(map[string]int) // command -> last index
					for i, line := range rawLines {
						command := parseRawLine(line)
						seen[command] = i
					}
					for i, line := range rawLines {
						command := parseRawLine(line)
						if seen[command] == i {
							deduped = append(deduped, line)
						}
					}
				}

				removed := len(rawLines) - len(deduped)
				if removed == 0 {
					cmd.Println("No duplicates found")
					return
				}

				if err := exec.HistoryManager.WriteRawLines(deduped); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to write history: %v", err))
					return
				}

				cmd.Printf("Removed %d duplicates (%d entries remaining)\n", removed, len(deduped))
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		dedupeCmd.Flags().BoolP("consecutive", "c", false, "Only remove consecutive duplicates")

		// history export
		var exportCmd = &cobra.Command{
			Use:   "export [file]",
			Short: "Export history to a plain text file",
			Args:  cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				history := exec.HistoryManager.GetHistory()
				if len(history) == 0 {
					cmd.Println("No history to export")
					return
				}

				var out *os.File
				if len(args) > 0 {
					var err error
					out, err = os.Create(args[0])
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Failed to create file: %v", err))
						return
					}
					defer out.Close()
				} else {
					// Print to stdout
					for _, line := range history {
						cmd.Println(line)
					}
					return
				}

				for _, line := range history {
					fmt.Fprintln(out, line)
				}

				cmd.Printf("Exported %d entries to %s\n", len(history), args[0])
			},
		}

		// history trim
		var trimCmd = &cobra.Command{
			Use:   "trim [n]",
			Short: "Keep only the last N entries",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if exec.HistoryManager == nil {
					cmd.PrintErrln("History not available")
					return
				}

				n, err := strconv.Atoi(args[0])
				if err != nil || n < 0 {
					cmd.PrintErrln("Invalid count")
					return
				}

				rawLines := exec.HistoryManager.GetRawLines()
				total := len(rawLines)

				if total == 0 {
					cmd.Println("No history")
					return
				}

				if n >= total {
					cmd.Printf("History has only %d entries, nothing to trim\n", total)
					return
				}

				trimmed := rawLines[total-n:]
				if err := exec.HistoryManager.WriteRawLines(trimmed); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to write history: %v", err))
					return
				}

				cmd.Printf("Trimmed %d entries, kept last %d\n", total-n, n)
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
		historyCmd.AddCommand(lastCmd)
		historyCmd.AddCommand(deleteCmd)
		historyCmd.AddCommand(dedupeCmd)
		historyCmd.AddCommand(exportCmd)
		historyCmd.AddCommand(trimCmd)

		rootCmd.AddCommand(historyCmd)
	}
}
