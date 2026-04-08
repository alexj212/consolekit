package consolekit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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

		defaultNetwork := DefaultNetwork()
		defaultAddr := DefaultAddr(defaultNetwork, exec.AppName)

		startCmd := &cobra.Command{
			Use:   "start",
			Short: "Start the socket server",
			Long: `Start the socket server for programmatic command access.

Unix socket (default on Linux/macOS, no auth needed):
  ` + os.Args[0] + ` socket start
  ` + os.Args[0] + ` socket start --addr /tmp/custom.sock

TCP socket (default on Windows, auth token auto-generated):
  ` + os.Args[0] + ` socket start --network tcp --addr 127.0.0.1:9999

Connection details are written to a .sockinfo.json file in the temp directory
for automatic discovery by Claude Code skills and other tools.
`,
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

				// Set up info file for skill/tool discovery
				handler.InfoFile = DefaultSocketInfoPath(exec.AppName)

				fmt.Fprintf(os.Stderr, "Starting socket server on %s %s...\n", network, addr)
				printUsageHint(network, addr)
				fmt.Fprintf(os.Stderr, "Socket info file: %s\n", handler.InfoFile)

				// In REPL mode, run the socket server in the background
				// so the REPL remains interactive. In CLI mode, block.
				if exec.Interactive {
					go func() {
						if err := handler.Start(); err != nil {
							fmt.Fprintf(os.Stderr, "Socket server error: %v\n", err)
						}
					}()
					fmt.Fprintf(os.Stderr, "Socket server started in background (use 'socket stop' to stop)\n")
					return
				}

				// CLI mode: block until signal
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

				go func() {
					<-sigChan
					cancel()
					handler.Stop()
				}()

				_ = ctx // context used via signal handler

				if err := handler.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Socket server error: %v\n", err)
					os.Exit(1)
				}
			},
		}
		startCmd.Flags().String("network", defaultNetwork, "Network type: unix or tcp")
		startCmd.Flags().String("addr", defaultAddr, "Listen address (socket path or host:port)")

		infoCmd := &cobra.Command{
			Use:   "info",
			Short: "Show socket server information",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Println("Socket Server Information")
				cmd.Println(strings.Repeat("=", 60))
				cmd.Printf("Application: %s\n", exec.AppName)
				cmd.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
				cmd.Printf("Default network: %s\n", defaultNetwork)
				cmd.Printf("Default address: %s\n", defaultAddr)
				cmd.Printf("Protocol: Newline-delimited JSON (NDJSON)\n")
				cmd.Printf("Info file: %s\n", DefaultSocketInfoPath(exec.AppName))
				cmd.Println()

				cmd.Println("Request format:")
				cmd.Println(`  {"id":"optional","command":"help","token":"for-tcp-only"}`)
				cmd.Println()
				cmd.Println("Response format:")
				cmd.Println(`  {"id":"optional","output":"...","error":"","success":true}`)
				cmd.Println()

				cmd.Println("Usage examples:")
				cmd.Println()
				if runtime.GOOS == "windows" {
					cmd.Println("  TCP socket (default on Windows):")
					cmd.Printf("    %s socket start\n", os.Args[0])
					cmd.Println("    Connection details are written to the .sockinfo.json file")
					cmd.Printf("    Info file: %s\n", DefaultSocketInfoPath(exec.AppName))
				} else {
					cmd.Println("  Unix socket (default on Linux/macOS):")
					cmd.Printf("    %s socket start\n", os.Args[0])
					cmd.Printf("    echo '{\"command\":\"help\"}' | nc -U %s\n", DefaultSocketPath(exec.AppName))
				}
				cmd.Println()
				cmd.Println("  TCP socket:")
				cmd.Printf("    %s socket start --network tcp --addr 127.0.0.1:9999\n", os.Args[0])
				if runtime.GOOS == "windows" {
					cmd.Println("    Connection details and token are in the .sockinfo.json file")
				} else {
					cmd.Printf("    echo '{\"command\":\"help\",\"token\":\"TOKEN\"}' | nc 127.0.0.1 9999\n")
				}
				cmd.Println()
				cmd.Println("  Discover running server:")
				cmd.Printf("    cat %s\n", DefaultSocketInfoPath(exec.AppName))
			},
		}

		statusCmd := &cobra.Command{
			Use:   "status",
			Short: "Check if the socket server is running",
			Run: func(cmd *cobra.Command, args []string) {
				infoPath := DefaultSocketInfoPath(exec.AppName)
				info, running := IsServerRunning(infoPath)
				if !running {
					if info != nil {
						cmd.Printf("Server not running (stale info file for PID %d)\n", info.PID)
						cmd.Printf("Cleaning up stale info file: %s\n", infoPath)
						os.Remove(infoPath)
					} else {
						cmd.Println("Server not running")
					}
					cmd.Printf("Info file: %s\n", infoPath)
					return
				}
				cmd.Println("Socket Server Status: RUNNING")
				cmd.Println(strings.Repeat("=", 60))
				cmd.Printf("  Network:   %s\n", info.Network)
				cmd.Printf("  Address:   %s\n", info.Addr)
				cmd.Printf("  PID:       %d\n", info.PID)
				cmd.Printf("  App:       %s\n", info.App)
				cmd.Printf("  Info file: %s\n", infoPath)
				if info.Token != "" {
					cmd.Printf("  Auth:      token-based (TCP)\n")
				} else {
					cmd.Printf("  Auth:      filesystem (Unix socket)\n")
				}
			},
		}

		stopCmd := &cobra.Command{
			Use:   "stop",
			Short: "Stop a running socket server",
			Run: func(cmd *cobra.Command, args []string) {
				infoPath := DefaultSocketInfoPath(exec.AppName)
				info, running := IsServerRunning(infoPath)
				if !running {
					if info != nil {
						cmd.Printf("Server not running (stale info file for PID %d), cleaning up\n", info.PID)
						os.Remove(infoPath)
					} else {
						cmd.Println("No running server found")
					}
					return
				}
				cmd.Printf("Stopping socket server (PID %d) on %s %s...\n", info.PID, info.Network, info.Addr)
				if err := StopServer(infoPath); err != nil {
					cmd.Printf("Error stopping server: %v\n", err)
					return
				}
				cmd.Println("Stop signal sent successfully")
			},
		}

		scriptCmd := &cobra.Command{
			Use:   "script",
			Short: "Generate a client script for connecting to the socket server",
			Long: `Generate a shell script to connect to the socket server.

The script auto-discovers the server via the .sockinfo.json file.

Modes:
  REPL mode (default): Interactive prompt for sending commands
  Execute mode:        Run a single command and display the result

Examples:
  ` + os.Args[0] + ` socket script                  # Generate script for current OS
  ` + os.Args[0] + ` socket script --shell bash      # Force bash script
  ` + os.Args[0] + ` socket script --shell powershell # Force PowerShell script
  ` + os.Args[0] + ` socket script > client.sh && chmod +x client.sh
  ./client.sh                                         # REPL mode
  ./client.sh help                                    # Execute single command
`,
			Run: func(cmd *cobra.Command, args []string) {
				shell, _ := cmd.Flags().GetString("shell")
				if shell == "" {
					if runtime.GOOS == "windows" {
						shell = "powershell"
					} else {
						shell = "bash"
					}
				}
				infoPath := DefaultSocketInfoPath(exec.AppName)
				switch shell {
				case "bash", "sh":
					fmt.Print(generateBashScript(exec.AppName, infoPath))
				case "powershell", "ps1", "pwsh":
					fmt.Print(generatePowerShellScript(exec.AppName, infoPath))
				default:
					fmt.Fprintf(os.Stderr, "Unknown shell: %s (use bash or powershell)\n", shell)
				}
			},
		}
		scriptCmd.Flags().String("shell", "", "Script type: bash or powershell (auto-detected from OS)")

		socketCmd.AddCommand(startCmd)
		socketCmd.AddCommand(infoCmd)
		socketCmd.AddCommand(statusCmd)
		socketCmd.AddCommand(stopCmd)
		socketCmd.AddCommand(scriptCmd)

		// Make "start" the default action
		socketCmd.Run = startCmd.Run

		rootCmd.AddCommand(socketCmd)
	}
}

// DefaultSocketPath returns the default Unix socket path for an application.
func DefaultSocketPath(appName string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.sock", strings.ToLower(appName)))
}

// printUsageHint prints platform-appropriate connection examples to stderr.
func printUsageHint(network, addr string) {
	if network == "unix" {
		if runtime.GOOS == "windows" {
			fmt.Fprintf(os.Stderr, "Usage: see .sockinfo.json for connection details\n")
		} else {
			fmt.Fprintf(os.Stderr, "Usage: echo '{\"command\":\"help\"}' | nc -U %s\n", addr)
		}
	} else {
		if runtime.GOOS == "windows" {
			fmt.Fprintf(os.Stderr, "Usage (PowerShell): $c = [System.Net.Sockets.TcpClient]::new('%s'); ...\n", addr)
			fmt.Fprintf(os.Stderr, "  Or use: curl http://%s (if HTTP adapter is enabled)\n", addr)
		} else {
			fmt.Fprintf(os.Stderr, "Usage: echo '{\"command\":\"help\",\"token\":\"TOKEN\"}' | nc %s\n",
				strings.Replace(addr, ":", " ", 1))
		}
		fmt.Fprintf(os.Stderr, "  Connection details written to .sockinfo.json for automatic discovery\n")
	}
}

// generateSecureToken creates a cryptographically random hex token.
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback_%d", 0)
	}
	return hex.EncodeToString(b)
}

// generateBashScript returns a bash client script that reads sockinfo.json
// and supports both REPL and single-command execution.
func generateBashScript(appName, infoPath string) string {
	r := strings.NewReplacer("{{APP_NAME}}", appName, "{{APP_LOWER}}", strings.ToLower(appName), "{{INFO_FILE}}", infoPath)
	return r.Replace(`#!/usr/bin/env bash
# {{APP_NAME}} socket client - auto-generated
# Usage:
#   ./{{APP_LOWER}}-client.sh              # REPL mode (interactive)
#   ./{{APP_LOWER}}-client.sh help         # Execute single command
#   ./{{APP_LOWER}}-client.sh "set x 42"  # Execute command with args

set -euo pipefail

INFO_FILE="{{INFO_FILE}}"
APP_NAME="{{APP_NAME}}"

# --- Read connection info ---
load_info() {
    if [ ! -f "$INFO_FILE" ]; then
        echo "Error: Socket info file not found: $INFO_FILE" >&2
        echo "Is the server running? Start it with: $APP_NAME socket start" >&2
        exit 1
    fi

    # Parse JSON (use jq if available, fallback to grep/sed)
    if command -v jq &>/dev/null; then
        NETWORK=$(jq -r '.network' "$INFO_FILE")
        ADDR=$(jq -r '.addr' "$INFO_FILE")
        TOKEN=$(jq -r '.token // empty' "$INFO_FILE")
        PID=$(jq -r '.pid' "$INFO_FILE")
    else
        NETWORK=$(grep -o '"network"[[:space:]]*:[[:space:]]*"[^"]*"' "$INFO_FILE" | head -1 | sed 's/.*:.*"\([^"]*\)"/\1/')
        ADDR=$(grep -o '"addr"[[:space:]]*:[[:space:]]*"[^"]*"' "$INFO_FILE" | head -1 | sed 's/.*:.*"\([^"]*\)"/\1/')
        TOKEN=$(grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' "$INFO_FILE" | head -1 | sed 's/.*:.*"\([^"]*\)"/\1/')
        PID=$(grep -o '"pid"[[:space:]]*:[[:space:]]*[0-9]*' "$INFO_FILE" | head -1 | sed 's/.*:[[:space:]]*//')
    fi

    # Verify server is alive
    if ! kill -0 "$PID" 2>/dev/null; then
        echo "Error: Server (PID $PID) is not running" >&2
        rm -f "$INFO_FILE"
        exit 1
    fi
}

# --- Send a command and print the result ---
send_command() {
    local cmd="$1"
    local req

    # Build JSON request (escape quotes in command)
    local escaped_cmd
    escaped_cmd=$(echo "$cmd" | sed 's/\\/\\\\/g; s/"/\\"/g')

    if [ -n "$TOKEN" ]; then
        req="{\"command\":\"${escaped_cmd}\",\"token\":\"${TOKEN}\"}"
    else
        req="{\"command\":\"${escaped_cmd}\"}"
    fi

    local resp
    if [ "$NETWORK" = "unix" ]; then
        resp=$(echo "$req" | nc -U "$ADDR" 2>/dev/null)
    else
        local host="${ADDR%%:*}"
        local port="${ADDR##*:}"
        resp=$(echo "$req" | nc -w 5 "$host" "$port" 2>/dev/null)
    fi

    if [ -z "$resp" ]; then
        echo "Error: No response from server" >&2
        return 1
    fi

    # Parse response (use jq if available)
    if command -v jq &>/dev/null; then
        local success
        success=$(echo "$resp" | jq -r '.success')
        if [ "$success" = "true" ]; then
            echo "$resp" | jq -r '.output // empty'
        else
            local err
            err=$(echo "$resp" | jq -r '.error // empty')
            [ -n "$err" ] && echo "Error: $err" >&2
            echo "$resp" | jq -r '.output // empty'
            return 1
        fi
    else
        local success
        success=$(echo "$resp" | grep -o '"success"[[:space:]]*:[[:space:]]*[a-z]*' | head -1 | sed 's/.*:[[:space:]]*//')
        local output
        output=$(echo "$resp" | sed 's/.*"output"[[:space:]]*:[[:space:]]*"//' | sed 's/"[[:space:]]*,.*//' | sed 's/"[[:space:]]*}//')
        local error_msg
        error_msg=$(echo "$resp" | grep -o '"error"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*:.*"\([^"]*\)"/\1/')

        # Unescape common JSON escapes
        output=$(printf '%b' "$output")

        if [ "$success" = "true" ]; then
            [ -n "$output" ] && echo "$output"
        else
            [ -n "$error_msg" ] && echo "Error: $error_msg" >&2
            [ -n "$output" ] && echo "$output"
            return 1
        fi
    fi
}

# --- Main ---
load_info

echo "$APP_NAME socket client (${NETWORK}://${ADDR})" >&2

# Single command mode: pass all args as the command
if [ $# -gt 0 ]; then
    send_command "$*"
    exit $?
fi

# REPL mode
echo "Type commands, 'quit' to exit, 'ping' for health check" >&2
while true; do
    printf "%s> " "$APP_NAME"
    read -r line || break
    [ -z "$line" ] && continue
    case "$line" in
        quit|exit) break ;;
    esac
    send_command "$line"
done
echo ""
`)
}

// generatePowerShellScript returns a PowerShell client script that reads sockinfo.json
// and supports both REPL and single-command execution.
func generatePowerShellScript(appName, infoPath string) string {
	r := strings.NewReplacer("{{APP_NAME}}", appName, "{{APP_LOWER}}", strings.ToLower(appName), "{{INFO_FILE}}", infoPath)
	return r.Replace(`# {{APP_NAME}} socket client - auto-generated
# Usage:
#   .\{{APP_LOWER}}-client.ps1              # REPL mode (interactive)
#   .\{{APP_LOWER}}-client.ps1 help         # Execute single command
#   .\{{APP_LOWER}}-client.ps1 "set x 42"  # Execute command with args

$ErrorActionPreference = "Stop"

$InfoFile = "{{INFO_FILE}}"
$AppName = "{{APP_NAME}}"

# --- Read connection info ---
function Load-SocketInfo {
    if (-not (Test-Path $InfoFile)) {
        Write-Error "Socket info file not found: $InfoFile"
        Write-Error "Is the server running? Start it with: $AppName socket start"
        exit 1
    }
    $script:Info = Get-Content $InfoFile -Raw | ConvertFrom-Json

    # Verify server is alive
    try {
        Get-Process -Id $script:Info.pid -ErrorAction Stop | Out-Null
    } catch {
        Write-Error "Server (PID $($script:Info.pid)) is not running"
        Remove-Item $InfoFile -Force -ErrorAction SilentlyContinue
        exit 1
    }
}

# --- Send a command and print the result ---
function Send-SocketCommand {
    param([string]$Command)

    $req = @{ command = $Command }
    if ($script:Info.token) {
        $req.token = $script:Info.token
    }
    $jsonBytes = [System.Text.Encoding]::UTF8.GetBytes(($req | ConvertTo-Json -Compress) + [char]10)

    try {
        if ($script:Info.network -eq "unix") {
            $socket = New-Object System.Net.Sockets.Socket(
                [System.Net.Sockets.AddressFamily]::Unix,
                [System.Net.Sockets.SocketType]::Stream,
                [System.Net.Sockets.ProtocolType]::Unspecified
            )
            $endpoint = New-Object System.Net.Sockets.UnixDomainSocketEndPoint($script:Info.addr)
            $socket.Connect($endpoint)
            $stream = New-Object System.Net.Sockets.NetworkStream($socket, $true)
        } else {
            $parts = $script:Info.addr -split ":"
            $tcp = New-Object System.Net.Sockets.TcpClient($parts[0], [int]$parts[1])
            $stream = $tcp.GetStream()
        }

        $stream.Write($jsonBytes, 0, $jsonBytes.Length)
        $stream.Flush()

        # Half-close: signal no more data
        if ($script:Info.network -eq "unix") {
            $socket.Shutdown([System.Net.Sockets.SocketShutdown]::Send)
        } else {
            $tcp.Client.Shutdown([System.Net.Sockets.SocketShutdown]::Send)
        }

        $reader = New-Object System.IO.StreamReader($stream)
        $resp = $reader.ReadToEnd()
        $reader.Close()
        $stream.Close()

        if ($script:Info.network -eq "unix") {
            $socket.Dispose()
        } else {
            $tcp.Close()
        }
    } catch {
        Write-Error "Connection failed: $_"
        return
    }

    if (-not $resp) {
        Write-Error "No response from server"
        return
    }

    $result = $resp.Trim() | ConvertFrom-Json
    if ($result.success) {
        if ($result.output) { Write-Output $result.output }
    } else {
        if ($result.error) { Write-Error $result.error }
        if ($result.output) { Write-Output $result.output }
    }
}

# --- Main ---
Load-SocketInfo

Write-Host "$AppName socket client ($($script:Info.network)://$($script:Info.addr))" -ForegroundColor Cyan

# Single command mode
if ($args.Count -gt 0) {
    $cmd = $args -join " "
    Send-SocketCommand -Command $cmd
    exit
}

# REPL mode
Write-Host "Type commands, 'quit' to exit, 'ping' for health check" -ForegroundColor DarkGray
while ($true) {
    $line = Read-Host "$AppName>"
    if (-not $line) { continue }
    if ($line -eq "quit" -or $line -eq "exit") { break }
    Send-SocketCommand -Command $line
}
`)
}
