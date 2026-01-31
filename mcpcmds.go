package consolekit

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// AddMCPCommands adds MCP (Model Context Protocol) server commands
func AddMCPCommands(exec *CommandExecutor) func(cmd *cobra.Command) {
	return func(rootCmd *cobra.Command) {

		// mcp command - main MCP command
		mcpCmd := &cobra.Command{
			Use:   "mcp [action]",
			Short: "Model Context Protocol (MCP) server",
			Long: `Start an MCP stdio server to expose CLI commands as MCP tools.

The MCP server allows external applications (like Claude Desktop) to:
- Discover available CLI commands as MCP tools
- Execute commands with parameters
- Access templates and scripts as resources

MCP uses JSON-RPC 2.0 over stdio for communication.

Actions:
  start    - Start the MCP stdio server (default)
  info     - Show MCP server information`,
		}

		// mcp start - start the MCP server
		startCmd := &cobra.Command{
			Use:   "start",
			Short: "Start the MCP stdio server",
			Long: `Start the MCP stdio server.

The server listens on stdin/stdout using JSON-RPC 2.0 protocol.
It exposes all CLI commands as MCP tools that can be called remotely.

Example MCP client configuration (for Claude Desktop):
{
  "mcpServers": {
    "consolekit": {
      "command": "/path/to/your/app",
      "args": ["mcp", "start"]
    }
  }
}`,
			Run: func(cmd *cobra.Command, args []string) {
				useHTTP, _ := cmd.Flags().GetBool("http")
				httpAddr, _ := cmd.Flags().GetString("http-addr")

				// Create context with cancellation
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Handle interrupt signals
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				go func() {
					<-sigChan
					cancel()
				}()

				if useHTTP {
					server := NewMCPHTTPServer(exec, exec.AppName, "1.0.0")

					fmt.Fprintf(os.Stderr, "Starting MCP HTTP server for %s...\n", exec.AppName)
					fmt.Fprintf(os.Stderr, "Listening on %s\n", httpAddr)
					fmt.Fprintf(os.Stderr, "SSE endpoint: http://%s/sse\n", httpAddr)
					fmt.Fprintf(os.Stderr, "HTTP JSON-RPC endpoint: http://%s/mcp\n", httpAddr)

					if err := server.ListenAndServe(ctx, httpAddr); err != nil && err != context.Canceled && err != http.ErrServerClosed {
						fmt.Fprintf(os.Stderr, "MCP HTTP server error: %v\n", err)
						os.Exit(1)
					}

					fmt.Fprintf(os.Stderr, "MCP HTTP server stopped\n")
					return
				}

				// Create and start MCP stdio server
				server := NewMCPServer(exec, exec.AppName, "1.0.0")

				// Write initialization message to stderr (not stdout, which is for MCP protocol)
				fmt.Fprintf(os.Stderr, "Starting MCP server for %s...\n", exec.AppName)
				fmt.Fprintf(os.Stderr, "Listening on stdin/stdout using JSON-RPC 2.0\n")

				// Run the server
				if err := server.Run(ctx); err != nil && err != context.Canceled {
					fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
					os.Exit(1)
				}

				fmt.Fprintf(os.Stderr, "MCP server stopped\n")
			},
		}
		startCmd.Flags().Bool("http", false, "Serve MCP over HTTP (SSE + POST) instead of stdio")
		startCmd.Flags().String("http-addr", "127.0.0.1:7331", "HTTP listen address for MCP when --http is set")

		// mcp info - show MCP server information
		infoCmd := &cobra.Command{
			Use:   "info",
			Short: "Show MCP server information",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Println("MCP Server Information")
				cmd.Println(strings.Repeat("=", 60))
				cmd.Printf("Application: %s\n", exec.AppName)
				cmd.Printf("Protocol: JSON-RPC 2.0 over stdio (or HTTP with --http)\n")
				cmd.Printf("MCP Version: 2024-11-05\n")
				cmd.Println()

				cmd.Println("Capabilities:")
				cmd.Println("  - Tools: List and execute CLI commands")
				cmd.Println("  - Resources: List templates and scripts")
				cmd.Println()

				cmd.Println("Available Commands as Tools:")
				// Get root command and count tools
				rootCmd := exec.RootCmd()
				toolCount := 0
				countTools(rootCmd, &toolCount)
				cmd.Printf("  Total: %d commands\n", toolCount)
				cmd.Println()

				cmd.Println("Usage:")
				cmd.Printf("  Start server   : %s mcp start\n", os.Args[0])
				cmd.Printf("  Start over HTTP: %s mcp start --http --http-addr 127.0.0.1:7331\n", os.Args[0])
				cmd.Println()

				cmd.Println("Claude Desktop Configuration:")
				cmd.Printf(`  {
    "mcpServers": {
      "`+exec.AppName+`": {
        "command": "%s",
        "args": ["mcp", "start"]
      }
    }
  }\n\n`, os.Args[0])
			},
		}

		// mcp list-tools - list all available tools
		listToolsCmd := &cobra.Command{
			Use:   "list-tools",
			Short: "List all available MCP tools",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Println("Available MCP Tools:")
				cmd.Println(strings.Repeat("=", 60))

				// Get root command
				rootCmd := exec.RootCmd()

				// Collect and display tools
				displayTools(rootCmd, "", cmd)
			},
		}

		mcpCmd.AddCommand(startCmd)
		mcpCmd.AddCommand(infoCmd)
		mcpCmd.AddCommand(listToolsCmd)

		// Make "start" the default action
		mcpCmd.Run = startCmd.Run

		rootCmd.AddCommand(mcpCmd)
	}
}

// countTools recursively counts the number of executable commands
func countTools(cmd *cobra.Command, count *int) {
	if cmd.Hidden {
		return
	}

	// Handle root command (empty name) - just recurse
	if cmd.Name() == "" {
		for _, subCmd := range cmd.Commands() {
			countTools(subCmd, count)
		}
		return
	}

	if cmd.Run != nil || cmd.RunE != nil {
		*count++
	}

	for _, subCmd := range cmd.Commands() {
		countTools(subCmd, count)
	}
}

// displayTools recursively displays tools in a readable format
func displayTools(cmd *cobra.Command, prefix string, outCmd *cobra.Command) {
	if cmd.Hidden {
		return
	}

	// Handle root command (empty name) - just recurse
	if cmd.Name() == "" {
		for _, subCmd := range cmd.Commands() {
			displayTools(subCmd, prefix, outCmd)
		}
		return
	}

	fullName := cmd.Name()
	if prefix != "" {
		fullName = prefix + " " + cmd.Name()
	}

	// Display if this command is executable
	if cmd.Run != nil || cmd.RunE != nil {
		outCmd.Printf("  %-30s %s\n", fullName, cmd.Short)
	}

	// Recurse into subcommands
	for _, subCmd := range cmd.Commands() {
		displayTools(subCmd, fullName, outCmd)
	}
}
