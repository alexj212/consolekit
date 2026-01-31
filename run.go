package consolekit

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexj212/consolekit/safemap"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

func AddRun(exec *CommandExecutor, scripts embed.FS) func(cmd *cobra.Command) {

	return func(rootCmd *cobra.Command) {

		var lsCmdFunc = func(cmd *cobra.Command, args []string) {
			dir := "."

			if len(args) > 0 {
				dir = args[0]
			}

			// Check if listing embedded files (@ prefix)
			if strings.HasPrefix(dir, "@") {
				// Remove @ prefix and get directory path (default to root)
				embedPath := strings.TrimPrefix(dir, "@")
				if embedPath == "" {
					embedPath = "."
				}

				entries, err := scripts.ReadDir(embedPath)
				if err != nil {
					cmd.Printf("Error reading embedded files: %v\n", err)
					return
				}

				cmd.Printf("%s\n", fmt.Sprintf("Listing embedded files in: @%s", embedPath))
				cmd.Printf("%s\n", strings.Repeat("-", 80))

				dirCount := 0
				fileCount := 0

				for _, entry := range entries {
					if entry.IsDir() {
						// Format the path: use @dirname/ for root, @path/dirname/ for subdirs
						dirPath := "@" + entry.Name() + "/"
						if embedPath != "." {
							dirPath = "@" + embedPath + "/" + entry.Name() + "/"
						}
						cmd.Printf("%s  %s\n", "[DIR]", dirPath)
						dirCount++
					} else {
						// Get file info for size and timestamp
						info, err := entry.Info()
						var sizeStr string
						var timeStr string
						if err == nil {
							sizeStr = fmt.Sprintf("%10d bytes", info.Size())
							timeStr = info.ModTime().Format("2006-01-02 15:04:05")
						} else {
							sizeStr = "          -"
							timeStr = "                   -"
						}
						// Format the path: use @filename for root, @path/filename for subdirs
						filePath := "@" + entry.Name()
						if embedPath != "." {
							filePath = "@" + embedPath + "/" + entry.Name()
						}
						cmd.Printf("%s  %-20s  %-19s  %s\n",
							"[FILE]", sizeStr, timeStr, filePath)
						fileCount++
					}
				}

				cmd.Printf("%s\n", strings.Repeat("-", 80))
				cmd.Printf("Total: %d directories, %d files\n", dirCount, fileCount)
				return
			}

			// Check if path exists and whether it's a file or directory
			fileInfo, err := os.Stat(dir)
			if err != nil {
				cmd.Printf("Error accessing path: %v\n", err)
				return
			}

			// If it's a file, just show info about that file
			if !fileInfo.IsDir() {
				absPath, err := filepath.Abs(dir)
				if err != nil {
					absPath = dir
				}

				cmd.Printf("%s\n", fmt.Sprintf("File information: %s", absPath))
				cmd.Printf("%s\n", strings.Repeat("-", 80))
				cmd.Printf("%s  %10d bytes  %-19s  %s\n",
					"[FILE]",
					fileInfo.Size(),
					fileInfo.ModTime().Format("2006-01-02 15:04:05"),
					absPath)
				cmd.Printf("%s\n", strings.Repeat("-", 80))
				return
			}

			// List regular filesystem directory contents
			cmd.Printf("%s\n", fmt.Sprintf("Listing files in: %s", dir))
			files, err := ListFiles(dir, "")
			if err != nil {
				cmd.Printf("Error listing files: %v\n", err)
				return
			}

			cmd.Printf("%s\n", strings.Repeat("-", 80))

			dirCount := 0
			fileCount := 0

			// Print the sorted files with better formatting
			for _, file := range files {
				if file.IsDir {
					cmd.Printf("%s  %s  %s\n",
						"[DIR]",
						file.Timestamp.Format("2006-01-02 15:04:05"),
						file.FullPath+"/")
					dirCount++
				} else {
					cmd.Printf("%s  %10d bytes  %-19s  %s\n",
						"[FILE]",
						file.Size,
						file.Timestamp.Format("2006-01-02 15:04:05"),
						file.FullPath)
					fileCount++
				}
			}

			cmd.Printf("%s\n", strings.Repeat("-", 80))
			cmd.Printf("Total: %d directories, %d files\n", dirCount, fileCount)
		} // lsCmdFunc

		var lsCmd = &cobra.Command{
			Use:     "ls [path | @[path]]",
			Aliases: []string{"list", "l"},
			Short:   "List directory contents or show file information",
			Long: `List files and directories in the filesystem or embedded files.

When no argument is provided, lists the current directory.
When a directory is provided, lists its contents.
When a file is provided, shows information about that specific file.
Use '@' prefix to list embedded files from the scripts filesystem.
Directories are displayed with a '/' suffix and shown before files.
Full paths are displayed for all entries.`,
			Example: `  # List current directory
  ls

  # Show info about a specific file
  ls myfile.txt

  # List embedded scripts
  ls @

  # List files in specific directory
  ls /path/to/directory

  # List embedded subdirectory
  ls @subdir`,
			Run: lsCmdFunc,
		}
		rootCmd.AddCommand(lsCmd)

		var viewScriptCmdFunc = func(cmd *cobra.Command, args []string) {

			cmds, err := LoadScript(scripts, cmd, args[0])
			if err != nil {
				cmd.Print(fmt.Sprintf("error loading file: %s, %s\n", args[0], err))
				return
			}

			// Helper function to check if a command exists
			commandExists := func(cmdName string) bool {
				rootCmd := cmd.Root()
				for _, c := range rootCmd.Commands() {
					if c.Name() == cmdName {
						return true
					}
					// Check aliases too
					for _, alias := range c.Aliases {
						if alias == cmdName {
							return true
						}
					}
				}
				return false
			}

			cmd.Printf("Script file: %s - %d commands\n", args[0], len(cmds))
			for _, cmdLine := range cmds {
				if cmdLine == "" {
					continue
				}
				cmdLine = strings.TrimSpace(cmdLine)

				// Parse the first token to check if it's a valid command
				tokens, err := shellquote.Split(cmdLine)
				if err != nil || len(tokens) == 0 {
					// If parsing fails or no tokens, print as-is
					cmd.Printf("%s\n", cmdLine)
					continue
				}

				firstToken := tokens[0]
				if commandExists(firstToken) {
					// Valid command - print in green
					cmd.Printf("%s\n", fmt.Sprintf("%s", cmdLine))
				} else {
					// Invalid command - print in red
					cmd.Printf("%s\n", fmt.Sprintf("%s", cmdLine))
				}
			}
		}
		var viewScriptCmd = &cobra.Command{
			Use:     "vs {file | @}",
			Aliases: []string{"view-script"},
			Short:   "view script file, pass @ to list all scripts",
			Args:    cobra.ExactArgs(1),

			Run: viewScriptCmdFunc,
		}

		var runScriptCmdFunc = func(cmd *cobra.Command, args []string) {
			// Create scoped defaults for script arguments to avoid leakage
			scriptDefs := safemap.New[string, string]()
			for i, arg := range args[1:] {
				scriptDefs.Set(fmt.Sprintf("@arg%d", i), arg)
			}

			if args[0] == "@" {
				f, err := scripts.ReadDir(".")
				if err != nil {
					cmd.Printf("Error reading scripts: %v\n", err)
					return
				}

				cmd.Printf("Scripts Available:\n")
				for _, script := range f {
					if strings.HasSuffix(script.Name(), ".go") {
						continue
					}
					cmd.Printf("@%s\n", script.Name())
				}
				return
			}

			cmds, err := LoadScript(scripts, cmd, args[0])
			if err != nil {
				cmd.Print(fmt.Sprintf("error loading file %s, %s\n", args[0], err))
				return
			}

			spawn, err := cmd.Flags().GetBool("spawn")
			if err != nil {
				cmd.Print(fmt.Sprintf("unable to get flag spawn, %v\n", err))
				return
			}

			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				cmd.Print(fmt.Sprintf("unable to get flag quiet, %v\n", err))
				return
			}

			doExec := func() {
				startTime := time.Now()
				if !quiet {
					cmd.Printf("%s\n", fmt.Sprintf("▶ Executing file: %s - %d commands", args[0], len(cmds)))
				}

				execCount := 0
				for _, cmdLine := range cmds {
					if cmdLine == "" {
						continue
					}

					// Show the command being executed with arrow prefix (unless quiet)
					if !quiet {
						cmd.Printf("%s\n", fmt.Sprintf("  → %s", strings.TrimSpace(cmdLine)))
					}
					
					res, err := exec.Execute(cmdLine, scriptDefs)
					if res != "" {
						cmd.Printf("%s\n", res)
					}
					if err != nil {
						if !quiet {
							cmd.Printf("%s\n", fmt.Sprintf("  ✗ Error executing command: %s", err))
						}
						break
					}
					execCount++
				}
				
				if !quiet {
					elapsed := time.Since(startTime)
					timeSince := HumanizeDuration(elapsed, false)
					if execCount == len(cmds) {
						cmd.Printf("%s\n", fmt.Sprintf("✓ Script '%s' completed successfully - %d commands in %s", args[0], execCount, timeSince))
					} else {
						cmd.Printf("%s\n", fmt.Sprintf("⚠ Script '%s' partially completed - %d/%d commands in %s", args[0], execCount, len(cmds), timeSince))
					}
				}
			}

			if spawn {
				go doExec()
				return
			}
			doExec()
		}

		var runScriptCmd = &cobra.Command{
			Use:   "run [--spawn] [--quiet] {file | @file | @ } [args]...",
			Short: "exec script file, use `@name` files for internal scripts. pass args that can be referenced in script as @arg0, @arg1, ...",
			Long: `Execute a script file with optional flags.

Flags:
  --spawn    Run the script in the background
  --quiet    Suppress execution headers and command echoing, only show command output

Arguments can be passed after the filename and referenced in the script as @arg0, @arg1, etc.`,
			Args:  cobra.MinimumNArgs(1),
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetHelpFlagRecursively(cmd)
				ResetAllFlags(cmd)
			},

			Run: runScriptCmdFunc,
		}

		var spawnScriptCmdFunc = func(cmd *cobra.Command, args []string) {

			go func() {
				cmdLine := strings.Join(args, " ")
				cmdLineArgs, err := shellquote.Split(cmdLine)
				if err != nil {
					cmd.Print(fmt.Sprintf("error splitting cmd `%s`, %s\n", cmdLine, err))
					return
				}
				cmd.Printf("spawn cmd: %s | %s\n", rootCmd.Use, cmdLine)
				rootCmd.SetArgs(cmdLineArgs)
				if err := rootCmd.Execute(); err != nil {
					cmd.Print(fmt.Sprintf("error %s executing command: %s, %s\n", rootCmd.Name(), cmdLine, err))
					return
				}
			}()

		}

		var spawnScriptCmd = &cobra.Command{
			Use:   "spawn {cmd}",
			Short: "exec command",
			Args:  cobra.ExactArgs(1),

			Run: spawnScriptCmdFunc,
		}

		rootCmd.AddCommand(viewScriptCmd)
		rootCmd.AddCommand(runScriptCmd)
		rootCmd.AddCommand(spawnScriptCmd)

		runScriptCmd.Flags().Bool("spawn", false, "run script in background")
		runScriptCmd.Flags().BoolP("quiet", "q", false, "suppress execution headers and command echoing")
	}
}

func HumanizeDuration(duration time.Duration, showMs bool) string {
	ms := duration.Milliseconds() % 1000
	totalSeconds := int(duration.Seconds()) // Convert to int
	minutes := totalSeconds / 60
	hours := minutes / 60
	minutes = minutes % 60
	seconds := totalSeconds % 60
	if hours < 0 {
		hours = 0
	}
	if minutes < 0 {
		minutes = 0
	}
	if seconds < 0 {
		seconds = 0
	}

	if showMs {
		return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, ms)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func ReadLines(rdr io.Reader) ([]string, error) {

	// Prepare to read lines and accumulate multi-line commands
	scanner := bufio.NewScanner(rdr)
	var commandBuilder strings.Builder
	results := make([]string, 0)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line ends with a backslash, indicating a multi-line command
		if strings.HasSuffix(line, "\\") {
			// Replace the trailing backslash with a newline character
			commandBuilder.WriteString(strings.TrimSuffix(line, "\\"))
		} else {
			commandBuilder.WriteString(line + "\n")
			command := commandBuilder.String()
			commandBuilder.Reset() // Clear the builder for the next command
			// Execute the command
			results = append(results, command)
		}
	}
	return results, scanner.Err()
}

func LoadScript(scripts embed.FS, cmd *cobra.Command, filename string) ([]string, error) {

	if len(filename) == 0 {
		return nil, fmt.Errorf("no filename provided")
	}

	if strings.HasPrefix(filename, "@") {

		// Remove the @ prefix
		// Read the file content
		content, err := scripts.ReadFile(filename[1:])
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Convert the content into a string
		text := string(content)

		// Split the text into lines
		return ReadLines(strings.NewReader(text))
	}

	// Read the entire file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("LoadScript failed to read file: %w", err)
	}
	return ReadLines(strings.NewReader(string(content)))
}
