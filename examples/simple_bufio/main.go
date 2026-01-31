package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alexj212/consolekit"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

//go:embed *.run
var Data embed.FS

var (
	// BuildDate date string of when build was performed filled in by -X compile flag
	BuildDate string

	// LatestCommit date string of when build was performed filled in by -X compile flag
	LatestCommit string

	// Version string of build filled in by -X compile flag
	Version string

	// GitRepo string of the git repo url when build was performed filled in by -X compile flag
	GitRepo string

	// GitBranch string of branch in the git repo filled in by -X compile flag
	GitBranch string
)

func main() {
	// Create command executor (core execution engine)
	customizer := func(exec *consolekit.CommandExecutor) error {
		exec.Scripts = Data
		exec.AddBuiltinCommands()
		exec.AddCommands(consolekit.AddRun(exec, Data))

		// Add custom version command
		var verCmdFunc = func(cmd *cobra.Command, args []string) {
			cmd.Printf("BuildDate    : %s\n", BuildDate)
			cmd.Printf("LatestCommit : %s\n", LatestCommit)
			cmd.Printf("Version      : %s\n", Version)
			cmd.Printf("GitRepo      : %s\n", GitRepo)
			cmd.Printf("GitBranch    : %s\n", GitBranch)
		}

		var verCmd = &cobra.Command{
			Use:     "version",
			Aliases: []string{"v", "ver"},
			Short:   "Show version info",
			Run:     verCmdFunc,
		}
		exec.AddCommands(func(rootCmd *cobra.Command) {
			rootCmd.AddCommand(verCmd)
		})

		return nil
	}

	executor, err := consolekit.NewCommandExecutor("simple-bufio", customizer)
	if err != nil {
		fmt.Printf("consolekit.NewCommandExecutor error, %v\n", err)
		return
	}

	// If command-line arguments provided, execute them directly
	if len(os.Args) > 1 {
		// Join args into a command line
		cmdLine := strings.Join(os.Args[1:], " ")
		output, err := executor.Execute(cmdLine, nil)
		if output != "" {
			fmt.Print(output)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// No arguments - start interactive REPL with history
	fmt.Println("ConsoleKit Simple Bufio Example (with history)")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println("Use ↑/↓ arrows for history navigation")
	fmt.Println()

	// Check if we're in a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Println("Not running in a terminal, falling back to line-buffered mode")
		runSimpleMode(executor)
		return
	}

	// Run interactive mode with history and arrow keys
	runInteractiveMode(executor)
}

// runSimpleMode runs in simple line-buffered mode (for pipes/non-TTY)
func runSimpleMode(executor *consolekit.CommandExecutor) {
	// Simple line-by-line reading for piped input
	var line string
	prompt := "simple-bufio > "

	for {
		fmt.Print(prompt)
		_, err := fmt.Scanln(&line)
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "exit" || line == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		output, execErr := executor.Execute(line, nil)
		if output != "" {
			fmt.Print(output)
			if !strings.HasSuffix(output, "\n") {
				fmt.Println()
			}
		}
		if execErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", execErr)
		}
	}
}

// runInteractiveMode runs with history and arrow key support
func runInteractiveMode(executor *consolekit.CommandExecutor) {
	// Save terminal state and restore on exit
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		runSimpleMode(executor)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	prompt := "simple-bufio > "
	history := []string{}
	historyIndex := -1

	for {
		// Print prompt
		fmt.Print("\r" + prompt)

		// Read and edit line with history support
		line, exit := readLineWithHistory(&history, &historyIndex, prompt)

		if exit {
			fmt.Print("\r\nGoodbye!\r\n")
			break
		}

		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Handle exit
		if line == "exit" || line == "quit" {
			fmt.Print("\r\nGoodbye!\r\n")
			break
		}

		// Add to history
		if len(history) == 0 || history[len(history)-1] != line {
			history = append(history, line)
		}
		historyIndex = len(history)

		// Execute command
		fmt.Print("\r\n")
		output, err := executor.Execute(line, nil)

		// Print output
		if output != "" {
			// Replace \n with \r\n for raw mode
			output = strings.ReplaceAll(output, "\n", "\r\n")
			fmt.Print(output)
		}

		// Print error (convert \n to \r\n for raw mode)
		if err != nil {
			errMsg := strings.ReplaceAll(fmt.Sprintf("Error: %v", err), "\n", "\r\n")
			fmt.Fprint(os.Stderr, errMsg+"\r\n")
		}
	}
}

// readLineWithHistory reads a line with arrow key support for history navigation
func readLineWithHistory(history *[]string, historyIndex *int, prompt string) (string, bool) {
	line := []rune{}
	cursor := 0
	buf := make([]byte, 1)
	escapeSeq := []byte{}

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return string(line), true
		}

		b := buf[0]

		// Handle escape sequences
		if len(escapeSeq) > 0 || b == 0x1b {
			escapeSeq = append(escapeSeq, b)

			// Arrow keys: ESC [ A/B/C/D
			if len(escapeSeq) == 3 && escapeSeq[0] == 0x1b && escapeSeq[1] == '[' {
				switch escapeSeq[2] {
				case 'A': // Up arrow
					if len(*history) > 0 {
						if *historyIndex > 0 {
							*historyIndex--
						}
						if *historyIndex >= 0 && *historyIndex < len(*history) {
							// Clear current line
							clearLine(prompt, line, cursor)
							// Load history entry
							line = []rune((*history)[*historyIndex])
							cursor = len(line)
							// Display line
							fmt.Print(string(line))
						}
					}

				case 'B': // Down arrow
					if len(*history) > 0 {
						if *historyIndex < len(*history)-1 {
							*historyIndex++
							// Clear current line
							clearLine(prompt, line, cursor)
							// Load history entry
							line = []rune((*history)[*historyIndex])
							cursor = len(line)
							fmt.Print(string(line))
						} else {
							// At end of history, clear line
							*historyIndex = len(*history)
							clearLine(prompt, line, cursor)
							line = []rune{}
							cursor = 0
						}
					}

				case 'C': // Right arrow
					if cursor < len(line) {
						cursor++
						fmt.Print("\x1b[C")
					}

				case 'D': // Left arrow
					if cursor > 0 {
						cursor--
						fmt.Print("\x1b[D")
					}
				}
				escapeSeq = escapeSeq[:0]
				continue
			}

			// Delete key: ESC [ 3 ~
			if len(escapeSeq) == 4 && escapeSeq[0] == 0x1b && escapeSeq[1] == '[' &&
			   escapeSeq[2] == '3' && escapeSeq[3] == '~' {
				if cursor < len(line) {
					line = append(line[:cursor], line[cursor+1:]...)
					// Redraw from cursor to end
					fmt.Print(string(line[cursor:]) + " ")
					// Move cursor back
					if len(line) > cursor {
						fmt.Printf("\x1b[%dD", len(line)-cursor+1)
					} else {
						fmt.Print("\b")
					}
				}
				escapeSeq = escapeSeq[:0]
				continue
			}

			// If not complete escape sequence yet, continue reading
			if len(escapeSeq) < 6 {
				continue
			}
			// Unknown escape, ignore
			escapeSeq = escapeSeq[:0]
			continue
		}

		switch b {
		case 13, 10: // Enter (CR or LF)
			return string(line), false

		case 127, 8: // Backspace or DEL
			if cursor > 0 {
				// Remove character before cursor
				line = append(line[:cursor-1], line[cursor:]...)
				cursor--
				// Redraw line from cursor position
				fmt.Print("\b")
				fmt.Print(string(line[cursor:]) + " ")
				// Move cursor back to position
				if len(line) > cursor {
					fmt.Printf("\x1b[%dD", len(line)-cursor+1)
				} else {
					fmt.Print("\b")
				}
			}

		case 3: // Ctrl+C
			fmt.Print("^C\r\n")
			return "", false

		case 4: // Ctrl+D (EOF)
			if len(line) == 0 {
				return "", true
			}

		case 12: // Ctrl+L (clear screen)
			fmt.Print("\x1b[2J\x1b[H")
			fmt.Print(prompt + string(line))
			if cursor < len(line) {
				fmt.Printf("\x1b[%dD", len(line)-cursor)
			}

		default:
			// Printable character - insert at cursor
			if b >= 32 && b < 127 {
				if cursor < len(line) {
					// Insert in middle
					line = append(line[:cursor], append([]rune{rune(b)}, line[cursor:]...)...)
					cursor++
					// Redraw from cursor to end
					fmt.Print(string(line[cursor-1:]))
					// Move cursor back to position
					if len(line) > cursor {
						fmt.Printf("\x1b[%dD", len(line)-cursor)
					}
				} else {
					// Append at end
					line = append(line, rune(b))
					cursor++
					fmt.Printf("%c", b)
				}
			}
		}
	}
}

// clearLine clears the current input line
func clearLine(prompt string, line []rune, cursor int) {
	// Move to beginning of line
	fmt.Print("\r")
	// Clear entire line
	fmt.Print("\x1b[K")
	// Reprint prompt
	fmt.Print(prompt)
}
