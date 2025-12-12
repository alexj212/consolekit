package consolekit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// AddLogCommands adds log management commands to the CLI
func AddLogCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		var logCmd = &cobra.Command{
			Use:   "log",
			Short: "Manage command logging and audit trail",
			Long:  "Enable, disable, and query command execution logs for debugging and compliance",
		}

		// log enable
		var enableCmd = &cobra.Command{
			Use:   "enable",
			Short: "Enable command logging",
			Run: func(cmd *cobra.Command, args []string) {
				cli.LogManager.Enable()
				cmd.Println(cli.SuccessString("Command logging enabled"))
			},
		}

		// log disable
		var disableCmd = &cobra.Command{
			Use:   "disable",
			Short: "Disable command logging",
			Run: func(cmd *cobra.Command, args []string) {
				cli.LogManager.Disable()
				cmd.Println(cli.InfoString("Command logging disabled"))
			},
		}

		// log status
		var statusCmd = &cobra.Command{
			Use:   "status",
			Short: "Show logging status",
			Run: func(cmd *cobra.Command, args []string) {
				enabled := cli.LogManager.IsEnabled()
				logFile := cli.LogManager.GetLogFile()

				if enabled {
					cmd.Println(cli.SuccessString("Logging: ENABLED"))
				} else {
					cmd.Println(cli.InfoString("Logging: DISABLED"))
				}

				if logFile != "" {
					cmd.Printf("Log file: %s\n", logFile)
				} else {
					cmd.Println("Log file: (in-memory only)")
				}
			},
		}

		// log show
		var (
			showLast   int
			showFailed bool
			showSearch string
			showSince  string
			showJSON   bool
		)
		var showCmd = &cobra.Command{
			Use:   "show",
			Short: "Show command logs",
			Long:  "Display command execution logs with optional filtering",
			Run: func(cmd *cobra.Command, args []string) {
				var logs []AuditLog

				// Apply filters
				if showFailed {
					logs = cli.LogManager.GetFailedLogs()
				} else if showSearch != "" {
					logs = cli.LogManager.SearchLogs(showSearch)
				} else if showSince != "" {
					since, err := time.Parse(time.RFC3339, showSince)
					if err != nil {
						// Try parsing as date only
						since, err = time.Parse("2006-01-02", showSince)
						if err != nil {
							cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Invalid date format: %v", err)))
							return
						}
					}
					logs = cli.LogManager.GetLogsSince(since)
				} else if showLast > 0 {
					logs = cli.LogManager.GetRecentLogs(showLast)
				} else {
					logs = cli.LogManager.GetLogs()
				}

				if len(logs) == 0 {
					cmd.Println(cli.InfoString("No logs found"))
					return
				}

				// Output format
				if showJSON {
					data, err := json.MarshalIndent(logs, "", "  ")
					if err != nil {
						cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to marshal JSON: %v", err)))
						return
					}
					cmd.Println(string(data))
				} else {
					// Text format
					for _, log := range logs {
						status := "✓"
						if !log.Success {
							status = "✗"
						}

						duration := log.Duration.Round(time.Millisecond)
						timestamp := log.Timestamp.Format("2006-01-02 15:04:05")

						cmd.Printf("%s %s [%s] %s", status, timestamp, duration, log.Command)

						if !log.Success && log.Error != "" {
							cmd.Printf(" - %s", cli.ErrorString(log.Error))
						}
						cmd.Println()
					}

					cmd.Printf("\n%s\n", cli.InfoString(fmt.Sprintf("Total: %d logs", len(logs))))
				}
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		showCmd.Flags().IntVar(&showLast, "last", 0, "Show last N logs")
		showCmd.Flags().BoolVar(&showFailed, "failed", false, "Show only failed commands")
		showCmd.Flags().StringVar(&showSearch, "search", "", "Search logs by command text")
		showCmd.Flags().StringVar(&showSince, "since", "", "Show logs since date (YYYY-MM-DD or RFC3339)")
		showCmd.Flags().BoolVar(&showJSON, "json", false, "Output in JSON format")

		// log clear
		var clearCmd = &cobra.Command{
			Use:   "clear",
			Short: "Clear all in-memory logs",
			Run: func(cmd *cobra.Command, args []string) {
				cli.LogManager.Clear()
				cmd.Println(cli.SuccessString("Logs cleared"))
			},
		}

		// log export
		var exportFormat string
		var exportCmd = &cobra.Command{
			Use:   "export",
			Short: "Export logs to file",
			Long:  "Export command logs to JSON format",
			Run: func(cmd *cobra.Command, args []string) {
				data, err := cli.LogManager.ExportJSON()
				if err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Export failed: %v", err)))
					return
				}

				cmd.Println(string(data))
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format (json)")

		// log load
		var loadCmd = &cobra.Command{
			Use:   "load",
			Short: "Load logs from file",
			Long:  "Load command logs from the configured log file",
			Run: func(cmd *cobra.Command, args []string) {
				if err := cli.LogManager.LoadFromFile(); err != nil {
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to load logs: %v", err)))
					return
				}

				logs := cli.LogManager.GetLogs()
				cmd.Println(cli.SuccessString(fmt.Sprintf("Loaded %d logs from file", len(logs))))
			},
		}

		// log config
		var configCmd = &cobra.Command{
			Use:   "config [setting] [value]",
			Short: "Configure logging settings",
			Long:  "Configure logging settings: max_size, retention, log_success, log_failures",
			Args:  cobra.RangeArgs(0, 2),
			Run: func(cmd *cobra.Command, args []string) {
				if len(args) == 0 {
					// Show current config
					cmd.Println("Logging Configuration:")
					cmd.Printf("  Enabled: %v\n", cli.LogManager.IsEnabled())
					cmd.Printf("  Log file: %s\n", cli.LogManager.GetLogFile())
					return
				}

				setting := args[0]
				if len(args) == 1 {
					cmd.PrintErrln(cli.ErrorString("Value required"))
					return
				}

				value := args[1]

				switch setting {
				case "max_size":
					size, err := strconv.ParseInt(value, 10, 64)
					if err != nil {
						cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Invalid size: %v", err)))
						return
					}
					cli.LogManager.SetMaxSize(size)
					cmd.Println(cli.SuccessString(fmt.Sprintf("Max log size set to %d MB", size)))

				case "retention":
					days, err := strconv.Atoi(value)
					if err != nil {
						cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Invalid days: %v", err)))
						return
					}
					cli.LogManager.SetRetention(days)
					cmd.Println(cli.SuccessString(fmt.Sprintf("Log retention set to %d days", days)))

				case "log_success":
					enable := value == "true" || value == "1"
					cli.LogManager.SetLogSuccess(enable)
					cmd.Println(cli.SuccessString(fmt.Sprintf("Log success commands: %v", enable)))

				case "log_failures":
					enable := value == "true" || value == "1"
					cli.LogManager.SetLogFailures(enable)
					cmd.Println(cli.SuccessString(fmt.Sprintf("Log failed commands: %v", enable)))

				default:
					cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Unknown setting: %s", setting)))
					cmd.Println("Available settings: max_size, retention, log_success, log_failures")
				}
			},
		}

		// Add subcommands
		logCmd.AddCommand(enableCmd)
		logCmd.AddCommand(disableCmd)
		logCmd.AddCommand(statusCmd)
		logCmd.AddCommand(showCmd)
		logCmd.AddCommand(clearCmd)
		logCmd.AddCommand(exportCmd)
		logCmd.AddCommand(loadCmd)
		logCmd.AddCommand(configCmd)

		rootCmd.AddCommand(logCmd)
	}
}
