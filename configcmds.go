package consolekit

import (
	"os"
	osexec "os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// AddConfigCommands adds configuration management commands
func AddConfigCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
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
				if exec.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				value, err := exec.Config.GetString(args[0])
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
				if exec.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				err := exec.Config.SetString(args[0], args[1])
				if err != nil {
					cmd.PrintErrf("Error: %v\n", err)
					return
				}

				// Save the configuration
				err = exec.Config.Save()
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
				if exec.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi" // Default to vi
				}

				// Ensure config file exists
				if err := exec.Config.Save(); err != nil {
					cmd.PrintErrf("Error creating config file: %v\n", err)
					return
				}

				// Open editor
				editCmd := osexec.Command(editor, exec.Config.FilePath())
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
				if exec.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				err := exec.Config.Load()
				if err != nil {
					cmd.PrintErrf("Error reloading config: %v\n", err)
					return
				}

				// Apply configuration
				applyConfig(exec)

				cmd.Println("Configuration reloaded")
			},
		}

		// config show
		showCmd := &cobra.Command{
			Use:   "show",
			Short: "Show all configuration",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				cmd.Println("Configuration:")
				cmd.Println(strings.Repeat("=", 60))

				cmd.Println("\n[settings]")
				cmd.Printf("  history_size = %d\n", exec.Config.Settings.HistorySize)
				cmd.Printf("  prompt = %q\n", exec.Config.Settings.Prompt)
				cmd.Printf("  color = %t\n", exec.Config.Settings.Color)
				cmd.Printf("  pager = %q\n", exec.Config.Settings.Pager)

				if len(exec.Config.Aliases) > 0 {
					cmd.Println("\n[aliases]")
					for k, v := range exec.Config.Aliases {
						cmd.Printf("  %s = %q\n", k, v)
					}
				}

				if len(exec.Config.Variables) > 0 {
					cmd.Println("\n[variables]")
					for k, v := range exec.Config.Variables {
						cmd.Printf("  %s = %q\n", k, v)
					}
				}

				cmd.Println("\n[hooks]")
				if exec.Config.Hooks.OnStartup != "" {
					cmd.Printf("  on_startup = %q\n", exec.Config.Hooks.OnStartup)
				}
				if exec.Config.Hooks.OnExit != "" {
					cmd.Printf("  on_exit = %q\n", exec.Config.Hooks.OnExit)
				}
				if exec.Config.Hooks.BeforeCommand != "" {
					cmd.Printf("  before_command = %q\n", exec.Config.Hooks.BeforeCommand)
				}
				if exec.Config.Hooks.AfterCommand != "" {
					cmd.Printf("  after_command = %q\n", exec.Config.Hooks.AfterCommand)
				}

				cmd.Println("\n[logging]")
				cmd.Printf("  enabled = %t\n", exec.Config.Logging.Enabled)
				cmd.Printf("  log_file = %q\n", exec.Config.Logging.LogFile)
				cmd.Printf("  log_success = %t\n", exec.Config.Logging.LogSuccess)
				cmd.Printf("  log_failures = %t\n", exec.Config.Logging.LogFailures)
				cmd.Printf("  max_size_mb = %d\n", exec.Config.Logging.MaxSizeMB)
				cmd.Printf("  retention_days = %d\n", exec.Config.Logging.RetentionDays)

				cmd.Println(strings.Repeat("=", 60))
			},
		}

		// config path
		pathCmd := &cobra.Command{
			Use:   "path",
			Short: "Show config file path",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				cmd.Println(exec.Config.FilePath())
			},
		}

		// config save
		saveCmd := &cobra.Command{
			Use:   "save",
			Short: "Save current configuration to file",
			Run: func(cmd *cobra.Command, args []string) {
				if exec.Config == nil {
					cmd.Println("Configuration not initialized")
					return
				}

				// Sync current state to config
				syncStateToConfig(exec)

				err := exec.Config.Save()
				if err != nil {
					cmd.PrintErrf("Error saving config: %v\n", err)
					return
				}

				cmd.Printf("Configuration saved to: %s\n", exec.Config.FilePath())
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
func applyConfig(exec *CommandExecutor) {
	if exec.Config == nil {
		return
	}

	// Apply aliases from config
	for k, v := range exec.Config.Aliases {
		exec.aliases.Set(k, v)
	}

	// Apply variables from config
	for k, v := range exec.Config.Variables {
		if !strings.HasPrefix(k, "@") {
			k = "@" + k
		}
		exec.Variables.Set(k, v)
	}

	// Apply color setting
	if !exec.Config.Settings.Color {
		exec.NoColor = true
	}

	// Execute startup hook if configured
	if exec.Config.Hooks.OnStartup != "" {
		_, _ = exec.Execute(exec.Config.Hooks.OnStartup, nil)
	}
}

// syncStateToConfig syncs current CLI state to config
func syncStateToConfig(exec *CommandExecutor) {
	if exec.Config == nil {
		return
	}

	// Sync aliases
	exec.Config.Aliases = make(map[string]string)
	exec.aliases.ForEach(func(k, v string) bool {
		exec.Config.Aliases[k] = v
		return false
	})

	// Sync variables
	exec.Config.Variables = make(map[string]string)
	exec.Variables.ForEach(func(k, v string) bool {
		if strings.HasPrefix(k, "@") && !strings.HasPrefix(k, "@arg") && !strings.HasPrefix(k, "@env:") && !strings.HasPrefix(k, "@exec:") {
			varName := strings.TrimPrefix(k, "@")
			exec.Config.Variables[varName] = v
		}
		return false
	})
}
