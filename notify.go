package consolekit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// NotifyManager handles system notifications
type NotifyManager struct {
	webhookURL string
}

// NewNotifyManager creates a new notification manager
func NewNotifyManager() *NotifyManager {
	return &NotifyManager{}
}

// SetWebhook sets a webhook URL for notifications
func (nm *NotifyManager) SetWebhook(url string) {
	nm.webhookURL = url
}

// Send sends a desktop notification
func (nm *NotifyManager) Send(title string, message string, urgency string) error {
	switch runtime.GOOS {
	case "linux":
		return nm.sendLinux(title, message, urgency)
	case "darwin":
		return nm.sendMac(title, message)
	case "windows":
		return nm.sendWindows(title, message)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// sendLinux sends notification via notify-send
func (nm *NotifyManager) sendLinux(title string, message string, urgency string) error {
	args := []string{title, message}
	if urgency != "" {
		args = append([]string{"-u", urgency}, args...)
	}

	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}

// sendMac sends notification via osascript
func (nm *NotifyManager) sendMac(title string, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// sendWindows sends notification via PowerShell
func (nm *NotifyManager) sendWindows(title string, message string) error {
	script := fmt.Sprintf(`
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		$Template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
		$RawXml = [xml] $Template.GetXml()
		($RawXml.toast.visual.binding.text|where {$_.id -eq "1"}).AppendChild($RawXml.CreateTextNode("%s")) | Out-Null
		($RawXml.toast.visual.binding.text|where {$_.id -eq "2"}).AppendChild($RawXml.CreateTextNode("%s")) | Out-Null
		$SerializedXml = New-Object Windows.Data.Xml.Dom.XmlDocument
		$SerializedXml.LoadXml($RawXml.OuterXml)
		$Toast = [Windows.UI.Notifications.ToastNotification]::new($SerializedXml)
		$Toast.Tag = "PowerShell"
		$Toast.Group = "PowerShell"
		[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("PowerShell").Show($Toast)
	`, title, message)

	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

// SendWebhook sends a notification to a webhook URL
func (nm *NotifyManager) SendWebhook(title string, message string) error {
	if nm.webhookURL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	payload := map[string]interface{}{
		"title":     title,
		"message":   message,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	resp, err := http.Post(nm.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}
