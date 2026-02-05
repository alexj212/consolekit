package main

import (
	"crypto/rand"
	"crypto/rsa"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexj212/consolekit"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

//go:embed scripts/*.run
var Scripts embed.FS

var (
	BuildDate    string
	LatestCommit string
	Version      string
)

// Configuration loaded from environment
type Config struct {
	// Server settings
	HTTPPort     string
	HTTPUser     string
	HTTPPassword string
	SSHPort      string
	SSHPassword  string

	// Feature flags
	EnableHTTP bool
	EnableSSH  bool
	EnableREPL bool

	// Security settings
	LogCommands   bool
	LogFile       string
	MaxRecursion  int
	CommandFilter []string
}

func loadConfig() *Config {
	cfg := &Config{
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		HTTPUser:     getEnv("HTTP_USER", "admin"),
		HTTPPassword: getEnv("HTTP_PASSWORD", "changeme"),
		SSHPort:      getEnv("SSH_PORT", "2222"),
		SSHPassword:  getEnv("SSH_PASSWORD", "changeme"),

		EnableHTTP: getEnv("ENABLE_HTTP", "true") == "true",
		EnableSSH:  getEnv("ENABLE_SSH", "true") == "true",
		EnableREPL: getEnv("ENABLE_REPL", "false") == "true",

		LogCommands:  getEnv("LOG_COMMANDS", "true") == "true",
		LogFile:      getEnv("LOG_FILE", "/var/log/consolekit-commands.log"),
		MaxRecursion: 10,
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Load configuration
	cfg := loadConfig()

	// Print banner
	printBanner(cfg)

	// Create command executor
	executor, err := createExecutor(cfg)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Start transports based on configuration
	transports := startTransports(executor, cfg)

	// Wait for shutdown signal
	waitForShutdown(transports)
}

func createExecutor(cfg *Config) (*consolekit.CommandExecutor, error) {
	customizer := func(exec *consolekit.CommandExecutor) error {
		exec.Scripts = &Scripts
		exec.AddBuiltinCommands()
		exec.AddCommands(consolekit.AddRun(exec, &Scripts))

		// Configure logging
		if cfg.LogCommands {
			exec.LogManager.Enable()
			exec.LogManager.SetLogFile(cfg.LogFile)
			exec.LogManager.SetLogSuccess(true)
			exec.LogManager.SetLogFailures(true)
			log.Printf("Command logging enabled: %s\n", cfg.LogFile)
		}

		// Add custom commands
		addCustomCommands(exec)

		return nil
	}

	return consolekit.NewCommandExecutor("production-server", customizer)
}

func addCustomCommands(exec *consolekit.CommandExecutor) {
	// Version command
	var verCmd = &cobra.Command{
		Use:     "version",
		Aliases: []string{"v", "ver"},
		Short:   "Show version info",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Production ConsoleKit Server\n")
			cmd.Printf("Version      : %s\n", Version)
			cmd.Printf("Build Date   : %s\n", BuildDate)
			cmd.Printf("Commit       : %s\n", LatestCommit)
		},
	}

	// Status command
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show server status",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Server Status:\n")
			cmd.Printf("  Application : %s\n", exec.AppName)
			cmd.Printf("  Version     : %s\n", Version)
			cmd.Printf("  Logging     : %v\n", exec.LogManager.IsEnabled())

			// Count jobs
			jobs := len(exec.JobManager.List())
			cmd.Printf("  Active Jobs : %d\n", jobs)

			// Count variables
			vars := 0
			exec.Variables.ForEach(func(k, v string) bool {
				vars++
				return false
			})
			cmd.Printf("  Variables   : %d\n", vars)
		},
	}

	exec.AddCommands(func(rootCmd *cobra.Command) {
		rootCmd.AddCommand(verCmd)
		rootCmd.AddCommand(statusCmd)
	})
}

func startTransports(executor *consolekit.CommandExecutor, cfg *Config) []consolekit.TransportHandler {
	var transports []consolekit.TransportHandler

	// Start HTTP server
	if cfg.EnableHTTP {
		httpHandler := consolekit.NewHTTPHandler(
			executor,
			":"+cfg.HTTPPort,
			cfg.HTTPUser,
			cfg.HTTPPassword,
		)

		go func() {
			log.Printf("Starting HTTP server on port %s...\n", cfg.HTTPPort)
			if err := httpHandler.Start(); err != nil {
				log.Printf("HTTP server error: %v\n", err)
			}
		}()

		transports = append(transports, httpHandler)
	}

	// Start SSH server
	if cfg.EnableSSH {
		// Generate host key
		hostKey, err := generateHostKey()
		if err != nil {
			log.Printf("Warning: Could not generate SSH host key: %v\n", err)
		} else {
			sshHandler := consolekit.NewSSHHandler(executor, ":"+cfg.SSHPort, hostKey)

			// Configure authentication
			sshHandler.SetAuthConfig(&consolekit.SSHAuthConfig{
				PasswordAuth: createPasswordAuth(cfg.SSHPassword),
			})

			go func() {
				log.Printf("Starting SSH server on port %s...\n", cfg.SSHPort)
				if err := sshHandler.Start(); err != nil {
					log.Printf("SSH server error: %v\n", err)
				}
			}()

			transports = append(transports, sshHandler)
		}
	}

	// Start local REPL
	if cfg.EnableREPL {
		replHandler := consolekit.NewREPLHandler(executor)
		replHandler.SetPrompt(func() string {
			return fmt.Sprintf("\n%s > ", executor.AppName)
		})

		go func() {
			log.Printf("Starting local REPL...\n")
			if err := replHandler.Start(); err != nil {
				log.Printf("REPL error: %v\n", err)
			}
		}()

		transports = append(transports, replHandler)
	}

	return transports
}

func printBanner(cfg *Config) {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ConsoleKit Production Server Example              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Version: %s (Built: %s)\n", Version, BuildDate)
	fmt.Println()
	fmt.Println("Configuration:")

	if cfg.EnableHTTP {
		fmt.Printf("  HTTP Server   : http://localhost:%s\n", cfg.HTTPPort)
		fmt.Printf("  Web Terminal  : http://localhost:%s/admin\n", cfg.HTTPPort)
		fmt.Printf("  HTTP User     : %s\n", cfg.HTTPUser)
		fmt.Printf("  HTTP Password : %s\n", maskPassword(cfg.HTTPPassword))
	}

	if cfg.EnableSSH {
		fmt.Printf("  SSH Server    : ssh://localhost:%s\n", cfg.SSHPort)
		fmt.Printf("  SSH Password  : %s\n", maskPassword(cfg.SSHPassword))
	}

	if cfg.EnableREPL {
		fmt.Printf("  Local REPL    : Enabled\n")
	}

	if cfg.LogCommands {
		fmt.Printf("  Logging       : Enabled (%s)\n", cfg.LogFile)
	}

	fmt.Println()
	fmt.Println("Ready to accept connections. Press Ctrl+C to stop.")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()
}

func maskPassword(password string) string {
	if len(password) <= 2 {
		return "***"
	}
	return password[:2] + "***"
}

func waitForShutdown(transports []consolekit.TransportHandler) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\nShutting down gracefully...")

	for _, transport := range transports {
		log.Printf("Stopping %s transport...\n", transport.Name())
		if err := transport.Stop(); err != nil {
			log.Printf("Error stopping %s: %v\n", transport.Name(), err)
		}
	}

	fmt.Println("Goodbye!")
}

// Helper functions
func createPasswordAuth(password string) func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		if string(pass) == password {
			return &ssh.Permissions{
				Extensions: map[string]string{
					"user": conn.User(),
				},
			}, nil
		}
		return nil, fmt.Errorf("invalid password")
	}
}

func generateHostKey() (ssh.Signer, error) {
	// Simplified - in production, load from file
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(privateKey)
}
