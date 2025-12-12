package consolekit

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// AddPipelineCommands adds pipeline enhancement commands
func AddPipelineCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		// tee command - read from stdin and write to both stdout and file
		var teeAppend bool
		var teeCmd = &cobra.Command{
			Use:   "tee [file]",
			Short: "Read from stdin and write to both stdout and file(s)",
			Long: `Read from standard input and write to both standard output and one or more files.
This allows you to capture output while still seeing it on screen.

Examples:
  env | tee output.txt              # Save to file and display
  env | tee output.txt | grep PATH  # Save and continue pipeline
  env | tee -a output.txt           # Append to file
  env | tee file1.txt file2.txt     # Write to multiple files`,
			Args: cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				// Open all output files
				var writers []io.Writer
				writers = append(writers, cmd.OutOrStdout())

				var files []*os.File
				defer func() {
					for _, f := range files {
						f.Close()
					}
				}()

				// Open each file
				for _, filename := range args {
					var file *os.File
					var err error

					if teeAppend {
						file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					} else {
						file, err = os.Create(filename)
					}

					if err != nil {
						cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to open %s: %v", filename, err)))
						return
					}

					files = append(files, file)
					writers = append(writers, file)
				}

				// Create MultiWriter to write to all destinations
				multiWriter := io.MultiWriter(writers...)

				// Copy stdin to all writers
				_, err := io.Copy(multiWriter, cmd.InOrStdin())
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to copy: %v", err)))
					return
				}
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		teeCmd.Flags().BoolVarP(&teeAppend, "append", "a", false, "Append to file instead of overwriting")

		rootCmd.AddCommand(teeCmd)
	}
}
