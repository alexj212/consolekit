package consolekit

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config represents the application configuration
type Config struct {
	Settings     SettingsConfig     `toml:"settings"`
	Aliases      map[string]string  `toml:"aliases"`
	Variables    map[string]string  `toml:"variables"`
	Hooks        HooksConfig        `toml:"hooks"`
	Logging      LoggingConfig      `toml:"logging"`
	Notification NotificationConfig `toml:"notification"`
	filePath     string             // Path to config file
}

// SettingsConfig contains general settings
type SettingsConfig struct {
	HistorySize int    `toml:"history_size"`
	Prompt      string `toml:"prompt"`
	Color       bool   `toml:"color"`
	Pager       string `toml:"pager"`
}

// HooksConfig contains lifecycle hooks
type HooksConfig struct {
	OnStartup      string `toml:"on_startup"`
	OnExit         string `toml:"on_exit"`
	BeforeCommand  string `toml:"before_command"`
	AfterCommand   string `toml:"after_command"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Enabled        bool   `toml:"enabled"`
	LogFile        string `toml:"log_file"`
	LogSuccess     bool   `toml:"log_success"`
	LogFailures    bool   `toml:"log_failures"`
	MaxSizeMB      int    `toml:"max_size_mb"`
	RetentionDays  int    `toml:"retention_days"`
}

// NotificationConfig contains notification settings
type NotificationConfig struct {
	WebhookURL string `toml:"webhook_url"`
}

// NewConfig creates a new config with defaults
func NewConfig(appName string) (*Config, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to get current user: %w", err)
	}

	configDir := filepath.Join(currentUser.HomeDir, fmt.Sprintf(".%s", strings.ToLower(appName)))
	configPath := filepath.Join(configDir, "config.toml")

	config := &Config{
		Settings: SettingsConfig{
			HistorySize: 10000,
			Prompt:      "%s > ",
			Color:       true,
			Pager:       "less -R",
		},
		Aliases:   make(map[string]string),
		Variables: make(map[string]string),
		Hooks: HooksConfig{
			OnStartup:     "",
			OnExit:        "",
			BeforeCommand: "",
			AfterCommand:  "",
		},
		Logging: LoggingConfig{
			Enabled:       false,
			LogFile:       filepath.Join(configDir, "audit.log"),
			LogSuccess:    true,
			LogFailures:   true,
			MaxSizeMB:     100,
			RetentionDays: 90,
		},
		Notification: NotificationConfig{
			WebhookURL: "",
		},
		filePath: configPath,
	}

	return config, nil
}

// Load reads the configuration from file
func (c *Config) Load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, use defaults
			return nil
		}
		return fmt.Errorf("error reading config file: %w", err)
	}

	err = toml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	return nil
}

// Save writes the configuration to file
func (c *Config) Save() error {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	err = os.WriteFile(c.filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// GetString retrieves a config value by path (e.g., "settings.history_size")
func (c *Config) GetString(path string) (string, error) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid config path: %s", path)
	}

	section := parts[0]
	key := parts[1]

	switch section {
	case "settings":
		switch key {
		case "history_size":
			return fmt.Sprintf("%d", c.Settings.HistorySize), nil
		case "prompt":
			return c.Settings.Prompt, nil
		case "color":
			return fmt.Sprintf("%t", c.Settings.Color), nil
		case "pager":
			return c.Settings.Pager, nil
		}
	case "hooks":
		switch key {
		case "on_startup":
			return c.Hooks.OnStartup, nil
		case "on_exit":
			return c.Hooks.OnExit, nil
		case "before_command":
			return c.Hooks.BeforeCommand, nil
		case "after_command":
			return c.Hooks.AfterCommand, nil
		}
	case "logging":
		switch key {
		case "enabled":
			return fmt.Sprintf("%t", c.Logging.Enabled), nil
		case "log_file":
			return c.Logging.LogFile, nil
		case "log_success":
			return fmt.Sprintf("%t", c.Logging.LogSuccess), nil
		case "log_failures":
			return fmt.Sprintf("%t", c.Logging.LogFailures), nil
		case "max_size_mb":
			return fmt.Sprintf("%d", c.Logging.MaxSizeMB), nil
		case "retention_days":
			return fmt.Sprintf("%d", c.Logging.RetentionDays), nil
		}
	}

	return "", fmt.Errorf("unknown config key: %s", path)
}

// SetString sets a config value by path
func (c *Config) SetString(path string, value string) error {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid config path: %s", path)
	}

	section := parts[0]
	key := parts[1]

	switch section {
	case "settings":
		switch key {
		case "history_size":
			var v int
			_, err := fmt.Sscanf(value, "%d", &v)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", value)
			}
			c.Settings.HistorySize = v
			return nil
		case "prompt":
			c.Settings.Prompt = value
			return nil
		case "color":
			var v bool
			_, err := fmt.Sscanf(value, "%t", &v)
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", value)
			}
			c.Settings.Color = v
			return nil
		case "pager":
			c.Settings.Pager = value
			return nil
		}
	case "hooks":
		switch key {
		case "on_startup":
			c.Hooks.OnStartup = value
			return nil
		case "on_exit":
			c.Hooks.OnExit = value
			return nil
		case "before_command":
			c.Hooks.BeforeCommand = value
			return nil
		case "after_command":
			c.Hooks.AfterCommand = value
			return nil
		}
	case "logging":
		switch key {
		case "enabled":
			var v bool
			_, err := fmt.Sscanf(value, "%t", &v)
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", value)
			}
			c.Logging.Enabled = v
			return nil
		case "log_file":
			c.Logging.LogFile = value
			return nil
		case "log_success":
			var v bool
			_, err := fmt.Sscanf(value, "%t", &v)
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", value)
			}
			c.Logging.LogSuccess = v
			return nil
		case "log_failures":
			var v bool
			_, err := fmt.Sscanf(value, "%t", &v)
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", value)
			}
			c.Logging.LogFailures = v
			return nil
		case "max_size_mb":
			var v int
			_, err := fmt.Sscanf(value, "%d", &v)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", value)
			}
			c.Logging.MaxSizeMB = v
			return nil
		case "retention_days":
			var v int
			_, err := fmt.Sscanf(value, "%d", &v)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", value)
			}
			c.Logging.RetentionDays = v
			return nil
		}
	}

	return fmt.Errorf("unknown config key: %s", path)
}

// FilePath returns the path to the config file
func (c *Config) FilePath() string {
	return c.filePath
}
