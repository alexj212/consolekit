package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alexj212/consolekit"
	"github.com/spf13/cobra"
	"tailscale.com/tsnet"
)

var (
	// BuildDate date string of when build was performed filled in by -X compile flag
	BuildDate string

	// LatestCommit date string of when build was performed filled in by -X compile flag
	LatestCommit string

	// Version string of build filled in by -X compile flag
	Version string
)

func main() {
	// Create command executor
	customizer := func(exec *consolekit.CommandExecutor) error {
		exec.AddBuiltinCommands()

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

	executor, err := consolekit.NewCommandExecutor("tailscale-http", customizer)
	if err != nil {
		fmt.Printf("consolekit.NewCommandExecutor error, %v\n", err)
		return
	}

	// Launch Tailscale node if TS_AUTH_KEY is set
	tsServer, err := launchTailscaleNode()
	if err != nil {
		log.Fatalf("Tailscale start error: %v", err)
	}

	// Create HTTP handler
	httpHandler := consolekit.NewHTTPHandler(
		executor,
		":8080",
		"admin",     // HTTP username
		"secret123", // HTTP password
	)

	// If Tailscale is enabled, use Tailscale listener
	if tsServer != nil {
		listener, err := tsServer.Listen("tcp", ":8080")
		if err != nil {
			log.Fatalf("Failed to create Tailscale listener: %v", err)
		}

		// Get Tailscale IPs
		ipv4, ipv6 := tsServer.TailscaleIPs()
		fmt.Printf("Tailscale HTTP server starting:\n")
		fmt.Printf("  IPv4: http://%s:8080\n", ipv4)
		if ipv6.IsValid() {
			fmt.Printf("  IPv6: http://[%s]:8080\n", ipv6)
		}
		fmt.Printf("  Username: admin\n")
		fmt.Printf("  Password: secret123\n")
		fmt.Printf("\nAccess the web terminal at http://%s:8080/admin\n", ipv4)

		httpHandler.SetCustomListener(listener)
	} else {
		fmt.Printf("HTTP server starting on :8080\n")
		fmt.Printf("  Username: admin\n")
		fmt.Printf("  Password: secret123\n")
		fmt.Printf("\nAccess the web terminal at http://localhost:8080/admin\n")
	}

	// Start HTTP server (blocking)
	if err := httpHandler.Start(); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// launchTailscaleNode initializes Tailscale if TS_AUTH_KEY is set
func launchTailscaleNode() (*tsnet.Server, error) {
	authKey := os.Getenv("TS_AUTH_KEY")
	if authKey == "" {
		fmt.Println("TS_AUTH_KEY not set, running without Tailscale")
		return nil, nil
	}

	fmt.Println("Initializing Tailscale node...")

	// Initialize the embedded Tailscale node
	srv := &tsnet.Server{
		Hostname: "consolekit-http",
		AuthKey:  authKey,
	}

	err := srv.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start Tailscale: %w", err)
	}

	fmt.Println("Tailscale node started successfully")
	return srv, nil
}
