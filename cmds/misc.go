package cmds

import (
	"fmt"
	"github.com/alexj212/consolekit"
	"io"
	"os"

	"strings"

	"github.com/spf13/cobra"
)

// AddMisc adds the commands echo and cat

func AddMisc(cli *consolekit.CLI) {

	var catCmd = &cobra.Command{
		Use:   "cat [file]",
		Short: "Displays the contents of a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("could not read file: %s error: %v", args[0], err)
			}
			cmd.Printf("%s\n", string(content))
			return nil
		},
	}
	cli.AddCommand(catCmd)
	var grepCmd = &cobra.Command{
		Use:   "grep <expression>",
		Short: "Grep with optional inverse and insensitive flags",
		Args:  cobra.ExactArgs(1),
		PostRun: func(cmd *cobra.Command, args []string) {
			consolekit.ResetHelpFlagRecursively(cmd)
			consolekit.ResetAllFlags(cmd)
		},

		Run: func(cmd *cobra.Command, args []string) {

			expression := args[0]

			inverse, _ := cmd.Flags().GetBool("inverse")
			insensitive, _ := cmd.Flags().GetBool("insensitive")

			if insensitive {
				expression = strings.ToLower(expression)
			}

			inputBytes, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				cmd.Print("Error reading input: ", err)
				return
			}

			//cmd.Printf("inputBytes: %s\n", string(inputBytes))
			//cmd.Printf("expression: %s\n", expression)
			//cmd.Printf("inverse: %t\n", inverse)
			//cmd.Printf("insensitive: %t\n", insensitive)
			input := string(inputBytes)
			lines := strings.Split(input, "\n")

			for _, line := range lines {
				if len(line) == 0 {
					continue
				}
				compareLine := line
				if insensitive {
					compareLine = strings.ToLower(line)
				}

				contains := strings.Contains(compareLine, expression)
				if contains && !inverse {
					cmd.Println(line)
					continue
				}
				if !contains && inverse {
					cmd.Println(line)
					continue
				}

			}
		},
	}

	grepCmd.Flags().BoolP("inverse", "v", false, "Inverse match")
	grepCmd.Flags().BoolP("insensitive", "i", false, "Case insensitive match")

	cli.AddCommand(grepCmd)
}
