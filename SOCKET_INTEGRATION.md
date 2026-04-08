# Socket Transport Integration

ConsoleKit includes a socket transport handler that exposes CLI commands via a lightweight JSON-line protocol over Unix domain sockets or TCP. This enables programmatic access from external tools, scripts, and Claude Code skills.

## Overview

The socket transport uses **Newline-Delimited JSON (NDJSON)** -- each message is a JSON object followed by a newline. This makes it trivially callable from any language, including plain bash.

```bash
# One-shot command execution (Linux/macOS)
echo '{"command":"help"}' | nc -U /tmp/myapp.sock
```

### Platform Defaults

| Platform | Default Network | Default Address | Auth |
|----------|----------------|-----------------|------|
| Linux/macOS | `unix` | `/tmp/{appname}.sock` | Filesystem permissions |
| Windows | `tcp` | `127.0.0.1:0` (auto-assigned port) | Token-based |

On all platforms, connection details are written to a **discovery file** (`{tempdir}/{appname}.sockinfo.json`) so that tools and scripts can auto-discover the server.

## Quick Start

### Starting the Socket Server

From within a ConsoleKit application:

```bash
# Default (Unix socket on Linux/macOS, TCP on Windows)
./myapp socket start

# Custom socket path (Linux/macOS)
./myapp socket start --addr /tmp/custom.sock

# TCP socket (auto-generates auth token, printed to stderr)
./myapp socket start --network tcp --addr 127.0.0.1:9999
```

### REPL vs CLI Behavior

When `socket start` is run from the **REPL**, the server starts in the background so the REPL remains interactive. Use `socket stop` to stop it. When run from the **command line**, it blocks until interrupted with Ctrl+C.

### Managing the Server

```bash
# Check if a server is running
./myapp socket status

# Stop a running server
./myapp socket stop

# Generate a client script for the current OS
./myapp socket script
./myapp socket script --shell bash
./myapp socket script --shell powershell

# View server configuration info
./myapp socket info
```

### Programmatic Setup

```go
executor, _ := consolekit.NewCommandExecutor("myapp", customizer)

// Unix socket (recommended for local tools)
handler := consolekit.NewSocketHandler(executor, "unix", "/tmp/myapp.sock")
go handler.Start()

// TCP socket (for remote access)
tcpHandler := consolekit.NewSocketHandler(executor, "tcp", "127.0.0.1:9999")
tcpHandler.SetAuthToken("my-secret-token")
go tcpHandler.Start()
```

## Protocol Specification

### Request

```json
{"id": "optional-correlation-id", "command": "print hello", "token": "for-tcp-only", "timeout": 30}
```

| Field     | Type   | Required | Description                                    |
|-----------|--------|----------|------------------------------------------------|
| `command` | string | yes      | The command line to execute                    |
| `id`      | string | no       | Correlation ID, echoed back in response        |
| `token`   | string | no       | Auth token (required for TCP on first message) |
| `timeout` | int    | no       | Per-request timeout in seconds (0 = no timeout) |

### Response

```json
{"id": "optional-correlation-id", "output": "hello\n", "success": true}
```

| Field     | Type   | Description                          |
|-----------|--------|--------------------------------------|
| `output`  | string | Command output (stdout capture)      |
| `error`   | string | Error message, empty on success      |
| `success` | bool   | `true` if command executed without error |
| `id`      | string | Echoed from request if provided      |

### Built-in Protocol Commands

These commands are handled directly by the socket handler without going through the command executor. They bypass command allow/deny filtering.

| Command    | Description                                          |
|------------|------------------------------------------------------|
| `ping`     | Health check. Returns `{"output":"pong","success":true}` |
| `conninfo` | Returns connection ID, remote address, uptime, idle time, auth status |

### Connection Model

Connections are persistent -- a single connection can send multiple requests sequentially. Each request receives exactly one response. Commands are processed one at a time per connection. For concurrency, open multiple connections.

TCP connections use keepalive (30s interval) to detect half-open connections. The `IdleTimeout` option actively disconnects idle connections when set.

## Connecting from the Command Line

### One-Shot Commands

```bash
# Using nc (netcat)
echo '{"command":"help"}' | nc -U /tmp/myapp.sock

# Using socat
echo '{"command":"vars"}' | socat - UNIX-CONNECT:/tmp/myapp.sock

# With correlation ID
echo '{"id":"1","command":"print hello world"}' | nc -U /tmp/myapp.sock

# Parse response with jq
echo '{"command":"date"}' | nc -U /tmp/myapp.sock | jq -r '.output'
```

### Interactive REPL over Socket

For a REPL-like experience where you type commands interactively:

**Using `socat` (recommended):**

```bash
socat READLINE UNIX-CONNECT:/tmp/myapp.sock
```

This gives you readline support (arrow keys, history). You type raw JSON lines:

```
{"command":"help"}
{"command":"print hello"}
{"command":"vars"}
```

**Using `rlwrap` + `nc` for a friendlier experience:**

```bash
rlwrap nc -U /tmp/myapp.sock
```

**Using the built-in script generator (recommended):**

ConsoleKit can generate platform-appropriate client scripts that auto-discover the server:

```bash
# Generate and save a client script
./myapp socket script > myapp-client.sh
chmod +x myapp-client.sh

# REPL mode
./myapp-client.sh

# Execute a single command
./myapp-client.sh help
./myapp-client.sh "print Hello from socket!"
```

On Windows:
```powershell
./myapp socket script --shell powershell > myapp-client.ps1

# REPL mode
.\myapp-client.ps1

# Execute a single command
.\myapp-client.ps1 help
```

The generated scripts:
- Auto-discover the server via `sockinfo.json` (no hardcoded paths)
- Validate the server PID is alive before connecting
- Handle token auth for TCP mode automatically
- Support both REPL (interactive) and single-command execution
- Use `jq` when available, fall back to `grep`/`sed` for portability (bash)

**Using a manual wrapper script:**

For custom integrations, here's a minimal example:

```bash
#!/usr/bin/env bash
# socket-repl.sh - REPL-like interface to a ConsoleKit socket server
SOCK="${1:-/tmp/myapp.sock}"

coproc SOCK { socat - UNIX-CONNECT:"$SOCK"; }
echo "Connected to $SOCK (type 'quit' to exit)"

while true; do
    read -r -e -p "> " cmd
    [ "$cmd" = "quit" ] && break
    [ -z "$cmd" ] && continue
    echo "{\"command\":$(printf '%s' "$cmd" | jq -Rs .)}" >&"${SOCK[1]}"
    read -r -t 5 response <&"${SOCK[0]}"
    echo "$response" | jq -r '.output // .error // "(no response)"'
done
kill "$SOCK_PID" 2>/dev/null
```

### TCP Connection

```bash
# One-shot with auth token
echo '{"command":"help","token":"YOUR_TOKEN"}' | nc 127.0.0.1 9999

# Interactive
rlwrap nc 127.0.0.1 9999
```

For TCP, the `token` field must be present on the first request. Once authenticated, subsequent requests on the same connection do not require it.

## Security

### Unix Sockets

Unix domain sockets are secured by file system permissions:

- Default permissions: `0600` (owner read/write only)
- No network exposure -- local access only
- No authentication layer needed -- OS enforces access control

To customize permissions:

```go
handler := consolekit.NewSocketHandler(executor, "unix", "/tmp/myapp.sock")
handler.SocketMode = 0660 // Allow group access
```

### TCP Sockets

TCP mode requires token-based authentication:

- A random 256-bit token is generated at startup and printed to stderr
- The token must be included in the first request's `token` field
- Bind to `127.0.0.1` to prevent external access
- For remote access, use SSH tunneling rather than exposing TCP directly

### Instance Conflict Detection

If a socket server is already running on the same path, `Start()` returns an error instead of silently replacing it. This prevents accidental conflicts between multiple instances. A stale socket file (from a crashed process) is automatically cleaned up.

### Command Filtering

Restrict which commands are available over the socket:

```go
config := &consolekit.TransportConfig{
    Executor:        executor,
    AllowedCommands: []string{"print", "date", "vars", "help"},
}
handler.SetTransportConfig(config)
```

## Session Variables

Commands executed over the socket have access to these session-scoped variables:

| Variable              | Description              |
|-----------------------|--------------------------|
| `@socket:conn_id`    | Unique connection ID     |
| `@socket:remote_addr`| Remote address string    |
| `@socket:network`    | `"unix"` or `"tcp"`     |

Example:

```bash
echo '{"command":"print Connection: @socket:conn_id"}' | nc -U /tmp/myapp.sock
```

## Discovery File

When started via the `socket start` command, the server writes a JSON info file for automatic discovery by tools and scripts:

```json
{
  "network": "tcp",
  "addr": "127.0.0.1:54321",
  "token": "abc123...",
  "pid": 12345,
  "app": "myapp"
}
```

Default path: `{tempdir}/{appname}.sockinfo.json`

### Programmatic Discovery

```go
// Check if a server is running
info, running := consolekit.IsServerRunning(consolekit.DefaultSocketInfoPath("myapp"))
if running {
    fmt.Printf("Server at %s:%s (PID %d)\n", info.Network, info.Addr, info.PID)
}

// Read info file directly
info, err := consolekit.ReadSocketInfo("/tmp/myapp.sockinfo.json")

// Stop a remote server
err := consolekit.StopServer(consolekit.DefaultSocketInfoPath("myapp"))
```

### Programmatic Info File Setup

When using `NewSocketHandler` directly, set the `InfoFile` field to enable discovery:

```go
handler := consolekit.NewSocketHandler(executor, "tcp", "127.0.0.1:0")
handler.InfoFile = consolekit.DefaultSocketInfoPath("myapp")
go handler.Start() // Info file written after listener binds
```

## Configuration Options

| Option           | Type            | Default | Description                            |
|------------------|-----------------|---------|----------------------------------------|
| `MaxConnections` | `int`           | 0       | Max concurrent connections (0 = unlimited) |
| `IdleTimeout`    | `time.Duration` | 0       | Disconnect idle connections (0 = disabled, actively enforced) |
| `SocketMode`     | `os.FileMode`   | `0600`  | Unix socket file permissions           |
| `InfoFile`       | `string`        | `""`    | Path to write discovery JSON (auto-cleaned on stop) |

```go
handler := consolekit.NewSocketHandler(executor, "unix", "/tmp/myapp.sock")
handler.MaxConnections = 10
handler.IdleTimeout = 5 * time.Minute
handler.SocketMode = 0660
handler.InfoFile = consolekit.DefaultSocketInfoPath("myapp")
```

### Idle Timeout

When `IdleTimeout` is set, each connection is monitored by a background goroutine. If no requests are received within the timeout period, the connection is automatically closed. Activity is tracked per-request.

### TCP Keepalive

TCP connections automatically have keepalive enabled (30-second interval) to detect half-open connections caused by network issues or ungraceful client disconnects.

## Integration with Claude Code Skills

A Claude Code skill can auto-discover and connect to the socket server using the info file:

```bash
#!/usr/bin/env bash
# Example skill: run a ConsoleKit command via auto-discovery
INFO_FILE="/tmp/myapp.sockinfo.json"

# Read connection details
NETWORK=$(jq -r '.network' "$INFO_FILE")
ADDR=$(jq -r '.addr' "$INFO_FILE")
TOKEN=$(jq -r '.token // empty' "$INFO_FILE")

# Build request
CMD="$1"
if [ -n "$TOKEN" ]; then
    REQ="{\"command\":\"$CMD\",\"token\":\"$TOKEN\"}"
else
    REQ="{\"command\":\"$CMD\"}"
fi

# Send and parse response
if [ "$NETWORK" = "unix" ]; then
    response=$(echo "$REQ" | nc -U "$ADDR")
else
    host="${ADDR%%:*}"; port="${ADDR##*:}"
    response=$(echo "$REQ" | nc -w 5 "$host" "$port")
fi

echo "$response" | jq -r '.output // .error'
```

### Health Check from Skills

```bash
# Quick health check before running commands
echo '{"command":"ping"}' | nc -U /tmp/myapp.sock
# Returns: {"output":"pong","success":true}
```

### Command with Timeout

```bash
# Execute with a 30-second timeout
echo '{"command":"slow-operation","timeout":30}' | nc -U /tmp/myapp.sock
```

## Multi-Transport Example

Run socket alongside other transports:

```go
executor, _ := consolekit.NewCommandExecutor("myapp", customizer)

// All transports share the same executor (shared state)
replHandler := consolekit.NewREPLHandler(executor)
sshHandler := consolekit.NewSSHHandler(executor, ":2222", hostKey)
httpHandler := consolekit.NewHTTPHandler(executor, ":8080", "admin", "pass")
socketHandler := consolekit.NewSocketHandler(executor, "unix", "/tmp/myapp.sock")

go sshHandler.Start()
go httpHandler.Start()
go socketHandler.Start()
replHandler.Run()
```

## Command Registration

The socket commands are included in these bundles:

- `AddAllCmds(exec)` -- includes socket commands
- `AddDeveloperCmds(exec)` -- includes socket commands

Or add individually:

```go
exec.AddCommands(consolekit.AddSocketCmds(exec))
```

## See Also

- [ARCHITECTURE.md](ARCHITECTURE.md) -- Three-layer architecture
- [MCP_INTEGRATION.md](MCP_INTEGRATION.md) -- MCP protocol integration (for Claude Desktop)
- [SECURITY.md](SECURITY.md) -- Security model
