# Socket Transport Integration

ConsoleKit includes a socket transport handler that exposes CLI commands via a lightweight JSON-line protocol over Unix domain sockets or TCP. This enables programmatic access from external tools, scripts, and Claude Code skills.

## Overview

The socket transport uses **Newline-Delimited JSON (NDJSON)** -- each message is a JSON object followed by a newline. This makes it trivially callable from any language, including plain bash.

```bash
# One-shot command execution
echo '{"command":"help"}' | nc -U /tmp/myapp.sock
```

## Quick Start

### Starting the Socket Server

From within a ConsoleKit application:

```bash
# Unix socket (default, no authentication needed)
./myapp socket start

# Custom socket path
./myapp socket start --addr /tmp/custom.sock

# TCP socket (auto-generates auth token, printed to stderr)
./myapp socket start --network tcp --addr 127.0.0.1:9999
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
{"id": "optional-correlation-id", "command": "print hello", "token": "for-tcp-only"}
```

| Field     | Type   | Required | Description                                    |
|-----------|--------|----------|------------------------------------------------|
| `command` | string | yes      | The command line to execute                    |
| `id`      | string | no       | Correlation ID, echoed back in response        |
| `token`   | string | no       | Auth token (required for TCP on first message) |

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

### Connection Model

Connections are persistent -- a single connection can send multiple requests sequentially. Each request receives exactly one response. Commands are processed one at a time per connection. For concurrency, open multiple connections.

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

**Using a wrapper script for natural command entry:**

Save the following as `socket-repl.sh`:

```bash
#!/usr/bin/env bash
# socket-repl.sh - REPL-like interface to a ConsoleKit socket server
# Usage: ./socket-repl.sh [socket-path]

SOCK="${1:-/tmp/myapp.sock}"

if [ ! -S "$SOCK" ]; then
    echo "Socket not found: $SOCK"
    exit 1
fi

echo "Connected to $SOCK (type 'quit' to exit)"
echo ""

# Open a persistent connection using a coprocess
coproc SOCK { socat - UNIX-CONNECT:"$SOCK"; }

while true; do
    read -r -e -p "> " cmd
    [ "$cmd" = "quit" ] && break
    [ -z "$cmd" ] && continue

    # Send command as JSON
    echo "{\"command\":$(printf '%s' "$cmd" | jq -Rs .)}" >&"${SOCK[1]}"

    # Read response and display output
    read -r -t 5 response <&"${SOCK[0]}"
    if [ -n "$response" ]; then
        output=$(echo "$response" | jq -r '.output // empty')
        error=$(echo "$response" | jq -r '.error // empty')
        [ -n "$output" ] && printf '%s' "$output"
        [ -n "$error" ] && printf 'Error: %s\n' "$error"
    else
        echo "(no response)"
    fi
done

# Close the coprocess
kill "$SOCK_PID" 2>/dev/null
echo "Disconnected."
```

Usage:

```bash
chmod +x socket-repl.sh
./socket-repl.sh /tmp/myapp.sock
```

```
Connected to /tmp/myapp.sock (type 'quit' to exit)

> help
Available commands:
  cls        Clear screen
  date       Display current date/time
  ...

> print Hello from socket!
Hello from socket!

> quit
Disconnected.
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

## Configuration Options

| Option           | Type          | Default | Description                            |
|------------------|---------------|---------|----------------------------------------|
| `MaxConnections` | `int`         | 0       | Max concurrent connections (0 = unlimited) |
| `IdleTimeout`    | `time.Duration` | 0     | Disconnect after inactivity (0 = disabled) |
| `SocketMode`     | `os.FileMode` | `0600`  | Unix socket file permissions           |

```go
handler := consolekit.NewSocketHandler(executor, "unix", "/tmp/myapp.sock")
handler.MaxConnections = 10
handler.IdleTimeout = 5 * time.Minute
handler.SocketMode = 0660
```

## Integration with Claude Code Skills

A Claude Code skill can use the socket interface to execute commands:

```bash
#!/usr/bin/env bash
# Example skill: run a ConsoleKit command and return the output
SOCK="/tmp/myapp.sock"
CMD="$1"

response=$(echo "{\"command\":$(printf '%s' "$CMD" | jq -Rs .)}" | nc -U "$SOCK")
echo "$response" | jq -r '.output // .error'
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
