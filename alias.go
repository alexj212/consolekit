package consolekit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func AddAlias(cli *CLI) func(cmd *cobra.Command) {

	return func(rootCmd *cobra.Command) {

		cli.LoadAliases(rootCmd)

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
				cli.aliases.Set(args[0], args[1])
				err := cli.SaveAliases(cmd)
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
				err := cli.SaveAliases(cmd)
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
					if cli.aliases.Len() == 0 {
						cmd.Printf("No aliases defined\n")
						return
					}
					cmd.Printf("Aliases:\n----------------------------------------\n")
					cli.aliases.ForEach(func(k string, v string) bool {
						cmd.Printf("%s=%s\n", k, v)
						return false
					})
					return
				}
				alias := args[0]
				value, ok := cli.aliases.Get(alias)
				if !ok {
					cmd.Print(cli.ErrorString("alias `%s` not found\n", alias))
					return
				}
				cmd.Printf("%s=%s\n", alias, value)
				return

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
				cli.aliases.Delete(alias)
				err := cli.SaveAliases(cmd)
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
				cli.aliases.SortedForEach(func(k string, v string) bool {
					cmd.Printf("%s=%s\n", k, v)
					return false
				})
				return

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
					cmd.Print(cli.ErrorString("unable to get home directory: %v\n", err))
					return
				}
				aliasesFilePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", cli.AppName))
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

func (c *CLI) LoadAliases(cmd *cobra.Command) {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Print(c.ErrorString("unable to get home directory, %v\n", err))
		return
	}

	// Construct the full path to the .aliases file

	aliasesFilePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", c.AppName))

	// Open the .aliases file
	file, err := os.Open(aliasesFilePath)
	if err != nil {
		fmt.Print(c.ErrorString("error opening alias file `%s`, %v\n", aliasesFilePath, err))
		c.AddDefaultAliases(cmd)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore empty lines and comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Split the line into name and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, line)
			continue
		}

		name := strings.TrimSpace(parts[0])
		if strings.Contains(name, " ") {
			fmt.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, name)
			continue
		}
		value := strings.TrimSpace(parts[1])
		if len(value) == 0 {
			fmt.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, line)
			continue
		}

		c.aliases.Set(name, value)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return
	}
	//fmt.Printf("loaded %d aliases from %s.\n", c.aliases.Len(), aliasesFilePath)
}

func (c *CLI) AddDefaultAlias(alias, expanded string) {
	c.aliases.Set(alias, expanded)
}

func (c *CLI) AddDefaultAliases(cmd *cobra.Command) {
	c.aliases.Set("pp", "print test")

	err := c.SaveAliases(cmd)
	if err != nil {
		cmd.Printf("error saving aliases, %v\n", err)
		return
	}
}

func (c *CLI) SaveAliases(cmd *cobra.Command) error {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {

		return err
	}

	filePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", c.AppName))

	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	// Write each key-value pair to the file
	writer := bufio.NewWriter(file)
	c.aliases.ForEach(func(name string, value string) bool {
		_, err := writer.WriteString(fmt.Sprintf("%s=%s\n", name, value))
		if err != nil {
			cmd.Print(c.ErrorString("error writing to `%s`, %v", name, err))
			return true
		}
		return false
	})

	cmd.Print(c.InfoString("aliases saved to `%s`\n", filePath))

	// Flush the buffered writer to ensure all data is written
	err = writer.Flush()
	if err != nil {
		return err
	}
	return nil
}
