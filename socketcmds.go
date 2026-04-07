package consolekit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// AddSocketCmds adds socket server commands for programmatic access.
func AddSocketCmds(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {

		socketCmd := &cobra.Command{
			Use:   "socket [action]",
			Short: "Socket server for programmatic access",
			Long: `Start a socket server to expose CLI commands via a simple JSON-line protocol.

The socket server allows external tools and scripts to:
- Execute commands and receive structured JSON responses
- Integrate with Claude Code skills
- Automate workflows via Unix sockets or TCP

Protocol: Newline-delimited JSON (NDJSON)
  Request:  {"command":"help"}
  Response: {"output":"...","success":true}

Actions:
  start    - Start the socket server (default)
  info     - Show socket server information`,
		}

		startCmd := &cobra.Command{
			Use:   "start",
			Short: "Start the socket server",
			Long: `Start the socket server for programmatic command access.

Unix socket (default, no auth needed):
  ` + os.Args[0] + ` socket start
  ` + os.Args[0] + ` socket start --addr /tmp/custom.sock

TCP socket (auth token required):
  ` + os.Args[0] + ` socket start --network tcp --addr 127.0.0.1:9999`,
			Run: func(cmd *cobra.Command, args []string) {
				network, _ := cmd.Flags().GetString("network")
				addr, _ := cmd.Flags().GetString("addr")

				handler := NewSocketHandler(exec, network, addr)

				// For TCP, generate and display auth token
				if network == "tcp" {
					token := generateSecureToken()
					handler.SetAuthToken(token)
					fmt.Fprintf(os.Stderr, "Socket auth token: %s\n", token)
				}

				// Create context with cancellation
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				go func() {
					<-sigChan
					cancel()
					handler.Stop()
				}()

				fmt.Fprintf(os.Stderr, "Starting socket server on %s %s...\n", network, addr)
				if network == "unix" {
					fmt.Fprintf(os.Stderr, "Usage: echo '{\"command\":\"help\"}' | nc -U %s\n", addr)
				}

				_ = ctx // context used via signal handler

				if err := handler.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Socket server error: %v\n", err)
					os.Exit(1)
				}
			},
		}
		startCmd.Flags().String("network", "unix", "Network type: unix or tcp")
		startCmd.Flags().String("addr", DefaultSocketPath(exec.AppName), "Listen address (socket path or host:port)")

		infoCmd := &cobra.Command{
			Use:   "info",
			Short: "Show socket server information",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Println("Socket Server Information")
				cmd.Println(strings.Repeat("=", 60))
				cmd.Printf("Application: %s\n", exec.AppName)
				cmd.Printf("Protocol: Newline-delimited JSON (NDJSON)\n")
				cmd.Printf("Default socket: %s\n", DefaultSocketPath(exec.AppName))
				cmd.Println()

				cmd.Println("Request format:")
				cmd.Println(`  {"id":"optional","command":"help","token":"for-tcp-only"}`)
				cmd.Println()
				cmd.Println("Response format:")
				cmd.Println(`  {"id":"optional","output":"...","error":"","success":true}`)
				cmd.Println()

				cmd.Println("Usage examples:")
				cmd.Println()
				cmd.Println("  Unix socket (default):")
				cmd.Printf("    %s socket start\n", os.Args[0])
				cmd.Printf("    echo '{\"command\":\"help\"}' | nc -U %s\n", DefaultSocketPath(exec.AppName))
				cmd.Println()
				cmd.Println("  TCP socket:")
				cmd.Printf("    %s socket start --network tcp --addr 127.0.0.1:9999\n", os.Args[0])
				cmd.Printf("    echo '{\"command\":\"help\",\"token\":\"TOKEN\"}' | nc 127.0.0.1 9999\n")
			},
		}

		socketCmd.AddCommand(startCmd)
		socketCmd.AddCommand(infoCmd)

		// Make "start" the default action
		socketCmd.Run = startCmd.Run

		rootCmd.AddCommand(socketCmd)
	}
}

// DefaultSocketPath returns the default Unix socket path for an application.
func DefaultSocketPath(appName string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.sock", strings.ToLower(appName)))
}

// generateSecureToken creates a cryptographically random hex token.
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback_%d", 0)
	}
	return hex.EncodeToString(b)
}
