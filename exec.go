package consolekit

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func AddOSExec(cli *CLI) func(cmd *cobra.Command) {

	return func(rootCmd *cobra.Command) {
		// osexecCmd represents the exec command
		var osexecCmd = &cobra.Command{
			Use:   "osexec [--out] [--background] [command] ",
			Short: "Executes a command with options to run in the background and hide/show output",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				// Local flags for the command
				background, _ := cmd.Flags().GetBool("background")
				showOutput, _ := cmd.Flags().GetBool("out")

				cmdLine := strings.Fields(args[0])
				if len(cmdLine) == 0 {
					cmd.Printf("No command provided\n")
					return
				}

				osCmd := exec.Command(cmdLine[0], cmdLine[1:]...)

				if background {
					// Create context for cancellation
					ctx, cancel := context.WithCancel(context.Background())
					osCmd = exec.CommandContext(ctx, cmdLine[0], cmdLine[1:]...)

					// Create output buffer for job tracking
					outputBuf := &bytes.Buffer{}

					if showOutput {
						// Tee output to both stdout and buffer
						osCmd.Stdout = io.MultiWriter(os.Stdout, outputBuf)
						osCmd.Stderr = io.MultiWriter(os.Stderr, outputBuf)
					} else {
						// Only capture to buffer
						osCmd.Stdout = outputBuf
						osCmd.Stderr = outputBuf
					}

					if err := osCmd.Start(); err != nil {
						cmd.Printf("Error starting command in background: %v\n", err)
						cancel()
						return
					}

					// Add to job manager
					jobID := cli.JobManager.Add(args[0], ctx, cancel, osCmd)
					cmd.Printf("Command started in background with PID %d (Job ID: %d)\n", osCmd.Process.Pid, jobID)

					// Update the job's output buffer reference
					if job, ok := cli.JobManager.Get(jobID); ok {
						job.mu.Lock()
						job.Output = outputBuf
						job.mu.Unlock()
					}
				} else {
					// Foreground execution
					if !showOutput {
						osCmd.Stdout = io.Discard
						osCmd.Stderr = io.Discard
					} else {
						osCmd.Stdout = os.Stdout
						osCmd.Stderr = os.Stderr
					}

					if err := osCmd.Run(); err != nil {
						cmd.Printf("Error executing command: %v\n", err)
					}
				}
			},
		}
		osexecCmd.Flags().BoolP("background", "b", false, "Run command in background")
		osexecCmd.Flags().BoolP("out", "o", false, "Show command output")

		rootCmd.AddCommand(osexecCmd)
	}
}
