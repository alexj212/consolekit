package consolekit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// AddLogCommands adds log management commands to the CLI
func AddLogCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
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
				exec.LogManager.Enable()
				cmd.Println(fmt.Sprintf("Command logging enabled"))
			},
		}

		// log disable
		var disableCmd = &cobra.Command{
			Use:   "disable",
			Short: "Disable command logging",
			Run: func(cmd *cobra.Command, args []string) {
				exec.LogManager.Disable()
				cmd.Println(fmt.Sprintf("Command logging disabled"))
			},
		}

		// log status
		var statusCmd = &cobra.Command{
			Use:   "status",
			Short: "Show logging status",
			Run: func(cmd *cobra.Command, args []string) {
				enabled := exec.LogManager.IsEnabled()
				logFile := exec.LogManager.GetLogFile()

				if enabled {
					cmd.Println(fmt.Sprintf("Logging: ENABLED"))
				} else {
					cmd.Println(fmt.Sprintf("Logging: DISABLED"))
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
					logs = exec.LogManager.GetFailedLogs()
				} else if showSearch != "" {
					logs = exec.LogManager.SearchLogs(showSearch)
				} else if showSince != "" {
					since, err := time.Parse(time.RFC3339, showSince)
					if err != nil {
						// Try parsing as date only
						since, err = time.Parse("2006-01-02", showSince)
						if err != nil {
							cmd.PrintErrln(fmt.Sprintf("Invalid date format: %v", err))
							return
						}
					}
					logs = exec.LogManager.GetLogsSince(since)
				} else if showLast > 0 {
					logs = exec.LogManager.GetRecentLogs(showLast)
				} else {
					logs = exec.LogManager.GetLogs()
				}

				if len(logs) == 0 {
					cmd.Println(fmt.Sprintf("No logs found"))
					return
				}

				// Output format
				if showJSON {
					data, err := json.MarshalIndent(logs, "", "  ")
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Failed to marshal JSON: %v", err))
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
							cmd.Printf(" - %s", log.Error)
						}
						cmd.Println()
					}

					cmd.Printf("\n%s\n", fmt.Sprintf("Total: %d logs", len(logs)))
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
				exec.LogManager.Clear()
				cmd.Println(fmt.Sprintf("Logs cleared"))
			},
		}

		// log export
		var exportFormat string
		var exportCmd = &cobra.Command{
			Use:   "export",
			Short: "Export logs to file",
			Long:  "Export command logs to JSON format",
			Run: func(cmd *cobra.Command, args []string) {
				data, err := exec.LogManager.ExportJSON()
				if err != nil {
					cmd.PrintErrln(fmt.Sprintf("Export failed: %v", err))
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
				if err := exec.LogManager.LoadFromFile(); err != nil {
					cmd.PrintErrln(fmt.Sprintf("Failed to load logs: %v", err))
					return
				}

				logs := exec.LogManager.GetLogs()
				cmd.Println(fmt.Sprintf("Loaded %d logs from file", len(logs)))
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
					cmd.Printf("  Enabled: %v\n", exec.LogManager.IsEnabled())
					cmd.Printf("  Log file: %s\n", exec.LogManager.GetLogFile())
					return
				}

				setting := args[0]
				if len(args) == 1 {
					cmd.PrintErrln(fmt.Sprintf("Value required"))
					return
				}

				value := args[1]

				switch setting {
				case "max_size":
					size, err := strconv.ParseInt(value, 10, 64)
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Invalid size: %v", err))
						return
					}
					exec.LogManager.SetMaxSize(size)
					cmd.Println(fmt.Sprintf("Max log size set to %d MB", size))

				case "retention":
					days, err := strconv.Atoi(value)
					if err != nil {
						cmd.PrintErrln(fmt.Sprintf("Invalid days: %v", err))
						return
					}
					exec.LogManager.SetRetention(days)
					cmd.Println(fmt.Sprintf("Log retention set to %d days", days))

				case "log_success":
					enable := value == "true" || value == "1"
					exec.LogManager.SetLogSuccess(enable)
					cmd.Println(fmt.Sprintf("Log success commands: %v", enable))

				case "log_failures":
					enable := value == "true" || value == "1"
					exec.LogManager.SetLogFailures(enable)
					cmd.Println(fmt.Sprintf("Log failed commands: %v", enable))

				default:
					cmd.PrintErrln(fmt.Sprintf("Unknown setting: %s", setting))
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
