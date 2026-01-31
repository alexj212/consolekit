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
	"tailscale.com/tsnet"
)

//go:embed scripts/*.run
var Data embed.FS

var (
	// BuildDate date string of when build was performed filled in by -X compile flag
	BuildDate string

	// LatestCommit date string of when build was performed filled in by -X compile flag
	LatestCommit string

	// Version string of build filled in by -X compile flag
	Version string
)

func main() {
	// Create command executor (shared by all transports)
	customizer := func(exec *consolekit.CommandExecutor) error {
		exec.Scripts = Data
		exec.AddBuiltinCommands()
		exec.AddCommands(consolekit.AddRun(exec, Data))

		// Add custom version command
		var verCmd = &cobra.Command{
			Use:     "version",
			Aliases: []string{"v", "ver"},
			Short:   "Show version info",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Printf("BuildDate    : %s\n", BuildDate)
				cmd.Printf("LatestCommit : %s\n", LatestCommit)
				cmd.Printf("Version      : %s\n", Version)
			},
		}
		exec.AddCommands(func(rootCmd *cobra.Command) {
			rootCmd.AddCommand(verCmd)
		})

		return nil
	}

	executor, err := consolekit.NewCommandExecutor("multi-transport", customizer)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Launch Tailscale if configured
	tsServer, err := launchTailscaleNode()
	if err != nil {
		log.Fatalf("Tailscale start error: %v", err)
	}

	fmt.Println("\n=== Multi-Transport ConsoleKit Server ===")

	// Start SSH server
	sshHandler := startSSHServer(executor, tsServer)

	// Start HTTP/WebSocket server
	httpHandler := startHTTPServer(executor, tsServer)

	// Start local REPL (optional - comment out if not needed)
	replHandler := consolekit.NewREPLHandler(executor)
	replHandler.SetPrompt(func() string {
		return "\nmulti-transport > "
	})

	// Run all transports in goroutines
	go func() {
		fmt.Println("Starting SSH server...")
		if err := sshHandler.Start(); err != nil {
			log.Printf("SSH server error: %v", err)
		}
	}()

	go func() {
		fmt.Println("Starting HTTP server...")
		if err := httpHandler.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	fmt.Println("\nAll transports running. Press Ctrl+C to exit.")

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal or run REPL
	select {
	case <-sigChan:
		fmt.Println("\nShutting down...")
		sshHandler.Stop()
		httpHandler.Stop()
		os.Exit(0)
	default:
		// Run REPL in main goroutine
		if err := replHandler.Run(); err != nil {
			log.Fatalf("REPL error: %v", err)
		}
	}
}

// startSSHServer creates and configures SSH server
func startSSHServer(executor *consolekit.CommandExecutor, tsServer *tsnet.Server) *consolekit.SSHHandler {
	// Generate host key
	hostKey, err := generateHostKey()
	if err != nil {
		log.Fatalf("Failed to generate host key: %v", err)
	}

	// Create SSH handler
	sshHandler := consolekit.NewSSHHandler(executor, ":2222", hostKey)

	// Configure authentication
	sshHandler.SetAuthConfig(&consolekit.SSHAuthConfig{
		PasswordAuth: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			// Simple password check (use proper auth in production)
			if conn.User() == "admin" && string(password) == "secret123" {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"user": conn.User(),
					},
				}, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	})

	// Use Tailscale listener if available
	if tsServer != nil {
		listener, err := tsServer.Listen("tcp", ":2222")
		if err != nil {
			log.Fatalf("Failed to create Tailscale SSH listener: %v", err)
		}
		sshHandler.SetCustomListener(listener)

		ipv4, _ := tsServer.TailscaleIPs()
		fmt.Printf("SSH server (Tailscale):\n")
		fmt.Printf("  ssh admin@%s -p 2222\n", ipv4)
		fmt.Printf("  Password: secret123\n\n")

		// Inject custom listener (would need to add this method to SSHHandler)
		// For now, this is a placeholder showing the pattern
	} else {
		fmt.Printf("SSH server (localhost):\n")
		fmt.Printf("  ssh admin@localhost -p 2222\n")
		fmt.Printf("  Password: secret123\n\n")
	}

	return sshHandler
}

// startHTTPServer creates and configures HTTP server
func startHTTPServer(executor *consolekit.CommandExecutor, tsServer *tsnet.Server) *consolekit.HTTPHandler {
	// Create HTTP handler
	httpHandler := consolekit.NewHTTPHandler(
		executor,
		":8080",
		"admin",
		"secret123",
	)

	// Use Tailscale listener if available
	if tsServer != nil {
		listener, err := tsServer.Listen("tcp", ":8080")
		if err != nil {
			log.Fatalf("Failed to create Tailscale HTTP listener: %v", err)
		}

		ipv4, _ := tsServer.TailscaleIPs()
		fmt.Printf("HTTP server (Tailscale):\n")
		fmt.Printf("  http://%s:8080/admin\n", ipv4)
		fmt.Printf("  Username: admin\n")
		fmt.Printf("  Password: secret123\n\n")

		httpHandler.SetCustomListener(listener)
	} else {
		fmt.Printf("HTTP server (localhost):\n")
		fmt.Printf("  http://localhost:8080/admin\n")
		fmt.Printf("  Username: admin\n")
		fmt.Printf("  Password: secret123\n\n")
	}

	return httpHandler
}

// launchTailscaleNode initializes Tailscale if TS_AUTH_KEY is set
func launchTailscaleNode() (*tsnet.Server, error) {
	authKey := os.Getenv("TS_AUTH_KEY")
	if authKey == "" {
		fmt.Println("TS_AUTH_KEY not set, running without Tailscale")
		return nil, nil
	}

	fmt.Println("Initializing Tailscale node...")

	srv := &tsnet.Server{
		Hostname: "consolekit-multi",
		AuthKey:  authKey,
	}

	err := srv.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start Tailscale: %w", err)
	}

	fmt.Println("Tailscale node started successfully")
	return srv, nil
}

// generateHostKey generates an RSA host key for SSH
func generateHostKey() (ssh.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return signer, nil
}
