package consolekit

import (
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// AddConfigCommands adds configuration management commands
func AddConfigCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {

		// config command - main configuration command
		configCmd := &cobra.Command{
			Use:   "config [action]",
			Short: "Manage application configuration",
			Long: `Manage application configuration.

Actions:
  get [key]        - Get a configuration value
  set [key] [val]  - Set a configuration value
  edit             - Open config file in $EDITOR
  reload           - Reload configuration from file
  show             - Show all configuration
  path             - Show config file path
  save             - Save current configuration`,
		}

		// config get
		getCmd := &cobra.Command{
			Use:   "get [key]",
			Short: "Get a configuration value",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if cli.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				value, err := cli.Config.GetString(args[0])
				if err != nil {
					cmd.PrintErrf("Error: %v\n", err)
					return
				}

				cmd.Printf("%s = %s\n", args[0], value)
			},
		}

		// config set
		setCmd := &cobra.Command{
			Use:   "set [key] [value]",
			Short: "Set a configuration value",
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				if cli.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				err := cli.Config.SetString(args[0], args[1])
				if err != nil {
					cmd.PrintErrf("Error: %v\n", err)
					return
				}

				// Save the configuration
				err = cli.Config.Save()
				if err != nil {
					cmd.PrintErrf("Error saving config: %v\n", err)
					return
				}

				cmd.Printf("Set %s = %s\n", args[0], args[1])
			},
		}

		// config edit
		editCmd := &cobra.Command{
			Use:   "edit",
			Short: "Open config file in $EDITOR",
			Run: func(cmd *cobra.Command, args []string) {
				if cli.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi" // Default to vi
				}

				// Ensure config file exists
				if err := cli.Config.Save(); err != nil {
					cmd.PrintErrf("Error creating config file: %v\n", err)
					return
				}

				// Open editor
				editCmd := exec.Command(editor, cli.Config.FilePath())
				editCmd.Stdin = os.Stdin
				editCmd.Stdout = os.Stdout
				editCmd.Stderr = os.Stderr

				if err := editCmd.Run(); err != nil {
					cmd.PrintErrf("Error opening editor: %v\n", err)
					return
				}

				cmd.Println("Config file edited. Use 'config reload' to apply changes.")
			},
		}

		// config reload
		reloadCmd := &cobra.Command{
			Use:   "reload",
			Short: "Reload configuration from file",
			Run: func(cmd *cobra.Command, args []string) {
				if cli.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				err := cli.Config.Load()
				if err != nil {
					cmd.PrintErrf("Error reloading config: %v\n", err)
					return
				}

				// Apply configuration
				applyConfig(cli)

				cmd.Println("Configuration reloaded")
			},
		}

		// config show
		showCmd := &cobra.Command{
			Use:   "show",
			Short: "Show all configuration",
			Run: func(cmd *cobra.Command, args []string) {
				if cli.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				cmd.Println("Configuration:")
				cmd.Println(strings.Repeat("=", 60))

				cmd.Println("\n[settings]")
				cmd.Printf("  history_size = %d\n", cli.Config.Settings.HistorySize)
				cmd.Printf("  prompt = %q\n", cli.Config.Settings.Prompt)
				cmd.Printf("  color = %t\n", cli.Config.Settings.Color)
				cmd.Printf("  pager = %q\n", cli.Config.Settings.Pager)

				if len(cli.Config.Aliases) > 0 {
					cmd.Println("\n[aliases]")
					for k, v := range cli.Config.Aliases {
						cmd.Printf("  %s = %q\n", k, v)
					}
				}

				if len(cli.Config.Variables) > 0 {
					cmd.Println("\n[variables]")
					for k, v := range cli.Config.Variables {
						cmd.Printf("  %s = %q\n", k, v)
					}
				}

				cmd.Println("\n[hooks]")
				if cli.Config.Hooks.OnStartup != "" {
					cmd.Printf("  on_startup = %q\n", cli.Config.Hooks.OnStartup)
				}
				if cli.Config.Hooks.OnExit != "" {
					cmd.Printf("  on_exit = %q\n", cli.Config.Hooks.OnExit)
				}
				if cli.Config.Hooks.BeforeCommand != "" {
					cmd.Printf("  before_command = %q\n", cli.Config.Hooks.BeforeCommand)
				}
				if cli.Config.Hooks.AfterCommand != "" {
					cmd.Printf("  after_command = %q\n", cli.Config.Hooks.AfterCommand)
				}

				cmd.Println("\n[logging]")
				cmd.Printf("  enabled = %t\n", cli.Config.Logging.Enabled)
				cmd.Printf("  log_file = %q\n", cli.Config.Logging.LogFile)
				cmd.Printf("  log_success = %t\n", cli.Config.Logging.LogSuccess)
				cmd.Printf("  log_failures = %t\n", cli.Config.Logging.LogFailures)
				cmd.Printf("  max_size_mb = %d\n", cli.Config.Logging.MaxSizeMB)
				cmd.Printf("  retention_days = %d\n", cli.Config.Logging.RetentionDays)

				cmd.Println(strings.Repeat("=", 60))
			},
		}

		// config path
		pathCmd := &cobra.Command{
			Use:   "path",
			Short: "Show config file path",
			Run: func(cmd *cobra.Command, args []string) {
				if cli.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				cmd.Println(cli.Config.FilePath())
			},
		}

		// config save
		saveCmd := &cobra.Command{
			Use:   "save",
			Short: "Save current configuration to file",
			Run: func(cmd *cobra.Command, args []string) {
				if cli.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				// Sync current state to config
				syncStateToConfig(cli)

				err := cli.Config.Save()
				if err != nil {
					cmd.PrintErrf("Error saving config: %v\n", err)
					return
				}

				cmd.Printf("Configuration saved to: %s\n", cli.Config.FilePath())
			},
		}

		configCmd.AddCommand(getCmd)
		configCmd.AddCommand(setCmd)
		configCmd.AddCommand(editCmd)
		configCmd.AddCommand(reloadCmd)
		configCmd.AddCommand(showCmd)
		configCmd.AddCommand(pathCmd)
		configCmd.AddCommand(saveCmd)

		rootCmd.AddCommand(configCmd)
	}
}

// applyConfig applies configuration settings to the CLI
func applyConfig(cli *CLI) {
	if cli.Config == nil {
		return
	}

	// Apply aliases from config
	for k, v := range cli.Config.Aliases {
		cli.Defaults.Set(k, v)
	}

	// Apply variables from config
	for k, v := range cli.Config.Variables {
		if !strings.HasPrefix(k, "@") {
			k = "@" + k
		}
		cli.Defaults.Set(k, v)
	}

	// Apply color setting
	if !cli.Config.Settings.Color {
		cli.NoColor = true
	}

	// Execute startup hook if configured
	if cli.Config.Hooks.OnStartup != "" {
		_, _ = cli.ExecuteLine(cli.Config.Hooks.OnStartup, nil)
	}
}

// syncStateToConfig syncs current CLI state to config
func syncStateToConfig(cli *CLI) {
	if cli.Config == nil {
		return
	}

	// Sync aliases
	cli.Config.Aliases = make(map[string]string)
	// Note: aliases are stored in cli.aliases SafeMap (not in Defaults)
	// We would need to add an export method to sync them

	// Sync variables
	cli.Config.Variables = make(map[string]string)
	cli.Defaults.ForEach(func(k, v string) bool {
		if strings.HasPrefix(k, "@") && !strings.HasPrefix(k, "@arg") && !strings.HasPrefix(k, "@env:") && !strings.HasPrefix(k, "@exec:") {
			varName := strings.TrimPrefix(k, "@")
			cli.Config.Variables[varName] = v
		}
		return false
	})
}
