package consolekit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func AddAlias(exec *CommandExecutor) func(cmd *cobra.Command) {

	return func(rootCmd *cobra.Command) {

		_ = exec.LoadAliases() // Load aliases on startup

		// aliasCmd represents the alias command
		var aliasCmd = &cobra.Command{
			Use:   "alias",
			Short: "Manage aliases",
			Long:  `Alias management for adding, deleting, and listing aliases.`,
		}

		// AliasAddCmd represents the add subcommand
		var AliasAddCmd = &cobra.Command{
			Use:     "add [alias] [command]",
			Short:   "Add a new alias",
			Aliases: []string{"a"},
			Long:    `Add a new alias to the system.`,
			Args:    cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				exec.aliases.Set(args[0], args[1])
				err := exec.SaveAliases()
				if err != nil {
					cmd.Printf("error saving aliases, %v\n", err)
					return
				}
				cmd.Printf("Setting alias, `%s` command: `%s`\n", args[0], args[1])
			},
		}

		// aliasSaveCmd represents the add subcommand
		var aliasSaveCmd = &cobra.Command{
			Use:     "save",
			Short:   "save aliases",
			Aliases: []string{"s"},
			Args:    cobra.ExactArgs(0),
			Run: func(cmd *cobra.Command, args []string) {
				err := exec.SaveAliases()
				if err != nil {
					cmd.Printf("error saving aliases, %v\n", err)
					return
				}
				cmd.Printf("saved aliases\n")

			},
		}

		// aliasPrintCmd represents the print subcommand
		var aliasPrintCmd = &cobra.Command{
			Use:     "print {alias}",
			Short:   "Print an alias",
			Aliases: []string{"p"},
			Long:    `Print an existing alias from the system.`,
			Args:    cobra.MaximumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if len(args) == 0 {
					if exec.aliases.Len() == 0 {
						cmd.Printf("No aliases defined\n")
						return
					}
					cmd.Printf("Aliases:\n----------------------------------------\n")
					exec.aliases.ForEach(func(k string, v string) bool {
						cmd.Printf("%s=%s\n", k, v)
						return false
					})
					return
				}
				alias := args[0]
				value, ok := exec.aliases.Get(alias)
				if !ok {
					cmd.Printf("alias `%s` not found\n", alias)
					return
				}
				cmd.Printf("%s=%s\n", alias, value)
			},
		}

		// aliasDefaultsCmd represents the add subcommand
		var aliasDefaultsCmd = &cobra.Command{
			Use:     "defaults",
			Short:   "Add default aliases",
			Aliases: []string{"d"},
			Long:    "Add default aliases",
			Args:    cobra.ExactArgs(0),
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Printf("adding default aliases, current list\n")
				aliasPrintCmd.Run(cmd, args)
			},
		}

		// aliasDeleteCmd alias delete subcommand
		var aliasDeleteCmd = &cobra.Command{
			Use:     "delete [alias]",
			Aliases: []string{"del"},
			Short:   "Delete an alias",
			Long:    `Delete an existing alias from the system.`,
			Args:    cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				alias := args[0]
				cmd.Printf("removing alias `%s`\n", alias)
				exec.aliases.Delete(alias)
				err := exec.SaveAliases()
				if err != nil {
					cmd.Printf("error saving aliases, %v\n", err)
					return
				}
				cmd.Printf("saved aliases\n")
			},
		}

		// aliasListCmd represents the list subcommand
		var aliasListCmd = &cobra.Command{
			Use:     "list",
			Aliases: []string{"ls"},
			Short:   "List all aliases",
			Long:    `List all aliases currently available in the system.`,
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Printf("Aliases:\n----------------------------------------\n")
				exec.aliases.SortedForEach(func(k string, v string) bool {
					cmd.Printf("%s=%s\n", k, v)
					return false
				})
			},
		}

		// aliasPathCmd shows the path to the aliases file
		var aliasPathCmd = &cobra.Command{
			Use:   "path",
			Short: "Show the path to the aliases file",
			Long:  `Display the full path to the aliases file location.`,
			Run: func(cmd *cobra.Command, args []string) {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					cmd.Printf("unable to get home directory: %v\n", err)
					return
				}
				aliasesFilePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", exec.AppName))
				cmd.Printf("%s\n", aliasesFilePath)
			},
		}

		aliasCmd.AddCommand(AliasAddCmd)
		aliasCmd.AddCommand(aliasDeleteCmd)
		aliasCmd.AddCommand(aliasDefaultsCmd)
		aliasCmd.AddCommand(aliasSaveCmd)

		aliasCmd.AddCommand(aliasListCmd)
		aliasCmd.AddCommand(aliasPrintCmd)
		aliasCmd.AddCommand(aliasPathCmd)

		rootCmd.AddCommand(aliasCmd)

	}
}
