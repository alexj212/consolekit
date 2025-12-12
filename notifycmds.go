package consolekit

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AddNotifyCommands adds notification commands
func AddNotifyCommands(cli *CLI) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {
		var notifyCmd = &cobra.Command{
			Use:   "notify",
			Short: "Send notifications",
			Long:  "Send desktop notifications or webhook notifications",
		}

		// notify send - send a desktop notification
		var urgency string
		var webhook bool
		var sendCmd = &cobra.Command{
			Use:   "send [title] [message]",
			Short: "Send a desktop or webhook notification",
			Long: `Send a notification to the desktop or to a webhook.
Desktop notifications use notify-send (Linux), osascript (macOS), or PowerShell (Windows).
Webhook notifications require webhook URL to be configured.

Examples:
  notify send "Build Complete" "The build finished successfully"
  notify send "Error" "Deployment failed" --urgency critical
  notify send "Alert" "Server down" --webhook`,
			Args: cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				title := args[0]
				message := args[1]

				if webhook {
					err := cli.NotifyManager.SendWebhook(title, message)
					if err != nil {
						cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to send webhook: %v", err)))
						return
					}
					cmd.Println(cli.SuccessString("Webhook notification sent"))
				} else {
					err := cli.NotifyManager.Send(title, message, urgency)
					if err != nil {
						cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Failed to send notification: %v", err)))
						return
					}
					cmd.Println(cli.SuccessString("Desktop notification sent"))
				}
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				ResetAllFlags(cmd)
			},
		}
		sendCmd.Flags().StringVarP(&urgency, "urgency", "u", "normal", "Urgency level: low, normal, critical")
		sendCmd.Flags().BoolVar(&webhook, "webhook", false, "Send to webhook instead of desktop")

		// notify config - configure webhook
		var configCmd = &cobra.Command{
			Use:   "config [webhook_url]",
			Short: "Configure notification webhook URL",
			Long:  "Set the webhook URL for webhook notifications",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				webhookURL := args[0]
				cli.NotifyManager.SetWebhook(webhookURL)

				// Also save to config if available
				if cli.Config != nil {
					if cli.Config.Notification.WebhookURL != webhookURL {
						cli.Config.Notification.WebhookURL = webhookURL
						if err := cli.Config.Save(); err != nil {
							cmd.PrintErrln(cli.ErrorString(fmt.Sprintf("Warning: failed to save config: %v", err)))
						} else {
							cmd.Println(cli.SuccessString("Webhook URL saved to config"))
						}
					}
				}

				cmd.Println(cli.SuccessString(fmt.Sprintf("Webhook URL set to: %s", webhookURL)))
			},
		}

		notifyCmd.AddCommand(sendCmd)
		notifyCmd.AddCommand(configCmd)

		rootCmd.AddCommand(notifyCmd)
	}
}
