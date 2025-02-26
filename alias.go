package consolekit

import (
	"bufio"
	"fmt"
	"github.com/alexj212/consolekit/safemap"

	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

var aliases = safemap.New[string, string]()

func (c *CLI) AddAlias() {
	c.loadAliases(c.AppName)
	aliasCmd.AddCommand(AliasAddCmd)
	aliasCmd.AddCommand(aliasDeleteCmd)
	aliasCmd.AddCommand(aliasDefaultsCmd)
	aliasCmd.AddCommand(aliasSaveCmd)

	aliasCmd.AddCommand(aliasListCmd)
	aliasCmd.AddCommand(aliasPrintCmd)
	c.AddCommand(aliasCmd)
}

func (c *CLI) loadAliases(Name string) {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		c.Repl.Printf(cli.ErrorString("unable to get home directory, %v\n", err))
		return
	}

	// Construct the full path to the .aliases file

	aliasesFilePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", Name))

	// Open the .aliases file
	file, err := os.Open(aliasesFilePath)
	if err != nil {
		c.Repl.Printf(cli.ErrorString("error opening alias file `%s`, %v\n", aliasesFilePath, err))
		setupDefaultAliases()
		return
	}
	defer file.Close()

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
			c.Repl.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, line)
			continue
		}

		name := strings.TrimSpace(parts[0])
		if strings.Contains(name, " ") {
			c.Repl.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, name)
			continue
		}
		value := strings.TrimSpace(parts[1])
		if len(value) == 0 {
			c.Repl.Printf("Skipping invalid alias - file `%s` line: %s\n", aliasesFilePath, line)
			continue
		}

		// here to replace the old password with the new one
		value = strings.ReplaceAll(value, "tipTopMagoo", "KillMenOw")

		aliases.Set(name, value)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		c.Repl.Printf("Error reading file: %v\n", err)
		return
	}

	if aliases.Len() == 0 {
		setupDefaultAliases()
		c.Repl.Printf("No aliases found in %s, setting defaults.\n", aliasesFilePath)
		return
	}
}

func setupDefaultAliases() {
	aliases.Set("lsu", "service list user")
	aliases.Set("s", "service list")
	aliases.Set("lsp", "service list proto")
	aliases.Set("lsx", "service list proxy")
	aliases.Set("who", "remote 'who -r'")
	aliases.Set("wbot", "remote 'who -r --bot'")
	aliases.Set("bots", "remote 'who -r --bot'")
	aliases.Set("wb", "remote 'who -r --bot'")
	aliases.Set("w", "remote 'who -r'")
	aliases.Set("expr", "client expr")
	aliases.Set("abdicate", "remote 'system abdicate'")
	aliases.Set("kill", "service kill --password=qaKillMenOw! --force ")
	saveAliases(nil)
}

func AddAlias(k, v string) {
	aliases.Set(k, v)
	saveAliases(nil)
}

func saveAliases(c *cobra.Command) {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		c.Printf(cli.ErrorString("error getting home directory, %v\n", err))
		return
	}

	filePath := filepath.Join(homeDir, fmt.Sprintf(".%s.aliases", cli.AppName))

	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		c.Printf(cli.ErrorString("error creating `%s`, %v", filePath, err))
		return
	}
	defer file.Close()

	// Write each key-value pair to the file
	writer := bufio.NewWriter(file)
	aliases.ForEach(func(name string, value string) bool {
		_, err := writer.WriteString(fmt.Sprintf("%s=%s\n", name, value))
		if err != nil {
			c.Printf(cli.ErrorString("error writing to `%s`, %v", name, err))
			return true
		}
		return false
	})

	// Flush the buffered writer to ensure all data is written
	err = writer.Flush()
	if err != nil {
		c.Printf(cli.ErrorString("error flushing writer %v", err))
		return
	}
	c.Printf("aliases saved to %s\n", filePath)
}

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
		aliases.Set(args[0], args[1])
		saveAliases(cmd)
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
		saveAliases(nil)
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
		setupDefaultAliases()
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
		aliases.Delete(alias)
		saveAliases(nil)

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
			if aliases.Len() == 0 {
				cmd.Printf("No aliases defined\n")
				return
			}
			cmd.Printf("Aliases:\n----------------------------------------\n")
			aliases.ForEach(func(k string, v string) bool {
				cmd.Printf("%s=%s\n", k, v)
				return false
			})
			return
		}
		alias := args[0]
		value, ok := aliases.Get(alias)
		if !ok {
			cmd.Printf(cli.ErrorString("alias `%s` not found\n", alias))
			return
		}
		cmd.Printf("%s=%s\n", alias, value)
		return

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
		aliases.SortedForEach(func(k string, v string) bool {
			cmd.Printf("%s=%s\n", k, v)
			return false
		})
		return

	},
}
