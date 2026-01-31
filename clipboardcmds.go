package consolekit

import (
	"bufio"
	"fmt"
	"io"
	osexec "os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// AddClipboardCommands adds clipboard integration commands
func AddClipboardCommands(executor *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// clip command - copy to clipboard
		var clipCmd = &cobra.Command{
			Use:   "clip",
			Short: "Copy input to clipboard",
			Long: `Read from stdin and copy to system clipboard.
Platform support: Linux (xclip/xsel), macOS (pbcopy), Windows (clip.exe)

Examples:
  print "Hello World" | clip
  cat file.txt | clip`,
			Run: func(cmd *cobra.Command, args []string) {
				var clipCmd *osexec.Cmd

				switch runtime.GOOS {
				case "linux":
					// Try xclip first, fall back to xsel
					if _, err := osexec.LookPath("xclip"); err == nil {
						clipCmd = osexec.Command("xclip", "-selection", "clipboard")
					} else if _, err := osexec.LookPath("xsel"); err == nil {
						clipCmd = osexec.Command("xsel", "--clipboard", "--input")
					} else {
						cmd.PrintErrln(fmt.Sprintf("Clipboard support requires xclip or xsel"))
						return
					}
				case "darwin":
					clipCmd = osexec.Command("pbcopy")
				case "windows":
					clipCmd = osexec.Command("clip")
				default:
					cmd.PrintErrln(fmt.Sprintf("Clipboard not supported on %s", runtime.GOOS))
					return
				}

				// Read from stdin
				var content strings.Builder
				scanner := bufio.NewScanner(cmd.InOrStdin())
				for scanner.Scan() {
					content.WriteString(scanner.Text())
					content.WriteString("\n")
				}

				if err := scanner.Err(); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error reading input: %v", err))
					return
				}

				// Write to clipboard
				stdin, err := clipCmd.StdinPipe()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error: %v", err))
					return
				}

				if err := clipCmd.Start(); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error: %v", err))
					return
				}

				if _, err := io.WriteString(stdin, content.String()); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error writing to clipboard: %v", err))
					stdin.Close()
					return
				}

				stdin.Close()

				if err := clipCmd.Wait(); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error: %v", err))
					return
				}

				cmd.Println(fmt.Sprintf("Copied to clipboard"))
			},
		}

		// paste command - paste from clipboard
		var pasteCmd = &cobra.Command{
			Use:   "paste",
			Short: "Paste from clipboard",
			Long: `Output clipboard contents to stdout.
Platform support: Linux (xclip/xsel), macOS (pbpaste), Windows (PowerShell)

Examples:
  paste
  paste | grep "pattern"`,
			Run: func(cmd *cobra.Command, args []string) {
				var pasteCmd *osexec.Cmd

				switch runtime.GOOS {
				case "linux":
					// Try xclip first, fall back to xsel
					if _, err := osexec.LookPath("xclip"); err == nil {
						pasteCmd = osexec.Command("xclip", "-selection", "clipboard", "-o")
					} else if _, err := osexec.LookPath("xsel"); err == nil {
						pasteCmd = osexec.Command("xsel", "--clipboard", "--output")
					} else {
						cmd.PrintErrln(fmt.Sprintf("Clipboard support requires xclip or xsel"))
						return
					}
				case "darwin":
					pasteCmd = osexec.Command("pbpaste")
				case "windows":
					pasteCmd = osexec.Command("powershell", "-command", "Get-Clipboard")
				default:
					cmd.PrintErrln(fmt.Sprintf("Clipboard not supported on %s", runtime.GOOS))
					return
				}

				output, err := pasteCmd.Output()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Error: %v", err))
					return
				}

				cmd.Print(string(output))
			},
		}

		rootCmd.AddCommand(clipCmd)
		rootCmd.AddCommand(pasteCmd)
	}
}
