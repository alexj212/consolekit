package consolekit

import (
	"fmt"
	"io"
	"os"

	"strings"

	"github.com/spf13/cobra"
)

// AddMisc adds the commands echo and cat
func AddMisc() func(cmd *cobra.Command) {

	return func(rootCmd *cobra.Command) {

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

		rootCmd.AddCommand(catCmd)
		var grepCmd = &cobra.Command{
			Use:   "grep [--inverse | -v] [--insensitive | -i] {expression}",
			Short: "Grep with optional inverse and insensitive flags",
			Args:  cobra.ExactArgs(1),
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetHelpFlagRecursively(cmd)
				ResetAllFlags(cmd)
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

		rootCmd.AddCommand(grepCmd)

		var envCmd = &cobra.Command{
			Use:   "env [key]",
			Short: "Displays environment variables and also specific var.",
			RunE: func(cmd *cobra.Command, args []string) error {

				if len(args) == 0 {
					envKv := os.Environ()
					for _, kv := range envKv {
						parts := strings.Split(kv, "=")
						if len(parts) == 2 {
							cmd.Printf("%-30s %s\n", parts[0], parts[1])
							continue
						}
						cmd.Printf("%-30s\n", parts[0])
					}
					return nil
				}

				val, ok := os.LookupEnv(args[0])
				if !ok {
					return fmt.Errorf("environment variable %s not found", args[0])
				}
				parts := strings.Split(val, "=")
				cmd.Printf("%-30s %s\n", parts[0], parts[1])
				return nil
			},
		}
		rootCmd.AddCommand(envCmd)

	}
}
