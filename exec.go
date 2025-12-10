package consolekit

import (
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func AddOSExec() func(cmd *cobra.Command) {

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

				if !showOutput {
					osCmd.Stdout = nil
					osCmd.Stderr = nil
				} else {
					osCmd.Stdout = os.Stdout
					osCmd.Stderr = os.Stderr
				}

				if background {
					if err := osCmd.Start(); err != nil {
						cmd.Printf("Error starting command in background: %v\n", err)
						return
					}
					cmd.Printf("Command started in background with PID %d\n", osCmd.Process.Pid)
				} else {
					if err := osCmd.Run(); err != nil {
						cmd.Printf("Error executing command: %v\n", err)
					}
				}
			},
		}
		rootCmd.AddCommand(osexecCmd)
	}
}
