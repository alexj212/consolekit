package cmds

import (
	"bufio"
	"github.com/alexj212/consolekit"
	"strings"

	"github.com/spf13/cobra"
)

// EchoCommand returns a simple echo command
func EchoCommand(cli *consolekit.CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "echo [text]",
		Aliases: []string{"print"},
		Short:   "Echoes the input text",

		Run: func(cmd *cobra.Command, args []string) {
			input := ""
			if len(args) > 0 {
				input = strings.Join(args, " ")
			} else {
				input = cli.ReadFromPipe(cmd)
			}
			cmd.Printf("%s\n", input)
		},
	}
}

// GrepCommand returns a command for pattern matching
func GrepCommand(cli *consolekit.CLI) *cobra.Command {
	var invertMatch bool

	cmd := &cobra.Command{
		Use:   "grep [-v] [pattern]",
		Short: "Search for PATTERN in the input. Supports -v for inverted matches.",
		Long:  `The grep command searches the input for lines that match the specified PATTERN. Supports -v to select non-matching lines.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]
			scanner := bufio.NewScanner(cmd.InOrStdin())
			for scanner.Scan() {
				line := scanner.Text()
				matched := strings.Contains(line, pattern)
				if (matched && !invertMatch) || (!matched && invertMatch) {
					cmd.Printf(line)
				}
			}
			return scanner.Err()
		},
	}

	cmd.Flags().BoolVarP(&invertMatch, "invert-match", "v", false, "Select non-matching lines")

	return cmd
}
