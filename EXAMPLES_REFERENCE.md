# ConsoleKit Examples Reference

Complete guide to all example applications included with ConsoleKit, including command-line flags, usage patterns, and configuration options.

---

## Overview

ConsoleKit includes 6 example applications demonstrating different use cases and deployment patterns:

| Example | Description | Transports | Best For |
|---------|-------------|------------|----------|
| **simple** | Basic REPL application | REPL only | Quick start, learning basics |
| **ssh_server** | SSH server with authentication | SSH only | Remote CLI access |
| **multi_transport** | All transports simultaneously | REPL + SSH + HTTP | Production multi-access |
| **production_server** | Production-ready server | SSH + HTTP | Enterprise deployment |
| **rest_api** | HTTP REST API wrapper | HTTP only | API integration |

---

## 1. Simple Example (`examples/simple`)

### Description
Basic REPL application demonstrating core ConsoleKit features. Perfect for getting started and learning the API.

### Features
- Interactive REPL with all builtin commands
- Command history and completion
- Embedded scripts support
- Color output with TTY detection

### Usage

```bash
# Build
cd examples/simple
go build

# Run interactive REPL
./simple

# Run single command and exit
./simple print "Hello World"

# Pipe commands via stdin
echo "print 'Automated test'" | ./simple

# Run multiple commands
./simple "let x=5; inc x; print @x"
```

### Command-Line Flags

**Standard Cobra Flags:**
- `-h, --help` - Show help message
- `--version` - Show version information (if configured)

**No custom flags** - Uses default REPL configuration

### Configuration

**History File:** `~/.simple.history`
**Aliases File:** `~/.simple/.simple.aliases`
**Config File:** `~/.simple/config.toml`

### Example Session

```bash
$ ./simple

simple> print "Hello, World!"
Hello, World!

simple> let name="Alice"
name = Alice

simple> print "Hello, @name"
Hello, Alice

simple> alias greet="print 'Hello, @name'"
Alias set: greet = print 'Hello, @name'

simple> greet
Hello, Alice

simple> exit
```

### Code Structure

```go
// Create executor with all builtin commands
executor, _ := consolekit.NewCommandExecutor("simple", func(exec *consolekit.CommandExecutor) error {
    exec.Scripts = Data  // Embed scripts
    exec.AddBuiltinCommands()  // All standard commands
    return nil
})

// Create REPL handler
handler := consolekit.NewREPLHandler(executor)
handler.SetPrompt(func() string {
    return "\nsimple > "
})

// Run with auto-detection (REPL if no args, execute if args provided)
handler.Run()
```

---

## 2. SSH Server Example (`examples/ssh_server`)

### Description
Standalone SSH server providing remote CLI access with authentication options.

### Features
- SSH server on configurable port (default: 2222)
- Multiple authentication methods
- Session isolation
- Per-session command history
- PTY support for interactive commands

### Usage

```bash
# Build
cd examples/ssh_server
go build

# Run with default settings (port 2222, anonymous auth)
./ssh_server

# Run on custom port
SSH_PORT=2200 ./ssh_server

# Run with password authentication
SSH_AUTH=password SSH_USER=admin SSH_PASS=secret ./ssh_server

# Run with public key authentication
SSH_AUTH=pubkey SSH_AUTHORIZED_KEYS=/path/to/authorized_keys ./ssh_server
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_PORT` | `2222` | SSH server listen port |
| `SSH_HOST` | `0.0.0.0` | SSH server bind address |
| `SSH_AUTH` | `anonymous` | Authentication mode: `anonymous`, `password`, `pubkey` |
| `SSH_USER` | `admin` | Username for password auth |
| `SSH_PASS` | `password` | Password for password auth |
| `SSH_AUTHORIZED_KEYS` | `~/.ssh/authorized_keys` | Path to authorized_keys file for pubkey auth |

### Authentication Modes

#### Anonymous (Default)
```bash
./ssh_server
# Connect from client:
ssh -p 2222 localhost
```

#### Password Authentication
```bash
SSH_AUTH=password SSH_USER=admin SSH_PASS=secret123 ./ssh_server
# Connect from client:
ssh -p 2222 admin@localhost
# Password: secret123
```

#### Public Key Authentication
```bash
SSH_AUTH=pubkey SSH_AUTHORIZED_KEYS=/home/user/.ssh/authorized_keys ./ssh_server
# Connect from client:
ssh -p 2222 -i ~/.ssh/id_rsa localhost
```

### Client Usage

```bash
# Interactive session
ssh -p 2222 localhost
> print "Hello from SSH"
Hello from SSH
> jobs
No jobs running
> exit

# Single command execution
ssh -p 2222 localhost "date"

# Pipe commands
echo "print 'Test'" | ssh -p 2222 localhost

# Command with arguments
ssh -p 2222 localhost 'let x=10; print @x'
```

### Session Isolation

Each SSH connection gets:
- ✅ Isolated session ID
- ✅ Per-session environment variables (`@ssh:user`, `@ssh:remote_ip`, `@ssh:session_id`)
- ✅ Independent command execution
- ✅ Shared global state (jobs, config, etc.)

### Security Considerations

- **Production Use:** Always use public key or password authentication
- **Firewall:** Restrict SSH port to trusted networks
- **Host Key:** Generated on startup (2048-bit RSA)
- **Command Filtering:** All commands available by default

---

## 3. Multi-Transport Example (`examples/multi_transport`)

### Description
Comprehensive example running all transports simultaneously: REPL, SSH, HTTP/WebSocket.

### Features
- All transports running concurrently
- Shared command executor (state synchronized)
- Graceful shutdown on SIGINT/SIGTERM
- Web UI for HTTP transport

### Usage

```bash
# Build
cd examples/multi_transport
go build

# Run all transports (REPL + SSH + HTTP)
./multi_transport

# Custom ports
SSH_PORT=2200 HTTP_PORT=8888 ./multi_transport
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_PORT` | `2222` | SSH server port |
| `HTTP_PORT` | `8080` | HTTP server port |
| `HTTP_USER` | `admin` | HTTP basic auth username |
| `HTTP_PASS` | `password` | HTTP basic auth password |

### Access Points

Once running, you can access the application through multiple channels:

#### 1. Local REPL
```bash
# Automatically starts in foreground
# Just type commands directly
```

#### 2. SSH (Port 2222)
```bash
ssh -p 2222 localhost
```

#### 3. HTTP Web UI (Port 8080)
```bash
# Open browser
http://localhost:8080

# Or use curl
curl -u admin:password http://localhost:8080/execute \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"command": "print \"Hello\""}'
```

#### 4. WebSocket (Port 8080)
```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8080/ws');
ws.send(JSON.stringify({command: 'print "Hello"'}));
```

### Shared State Example

Commands executed from any transport affect global state:

```bash
# In SSH session:
ssh -p 2222 localhost
> let shared_var="from SSH"

# In REPL:
> print @shared_var
from SSH

# Via HTTP:
curl -u admin:password http://localhost:8080/execute \
  -d '{"command": "print @shared_var"}'
# Output: from SSH
```

### Graceful Shutdown

Press `Ctrl+C` to gracefully shut down all transports:

```
^C
Shutting down multi-transport server...
Stopping SSH server...
Stopping HTTP server...
Stopping REPL...
All transports stopped.
```

---

## 4. Production Server Example (`examples/production_server`)

### Description
Production-ready server with environment-based configuration, logging, and best practices.

### Features
- Configuration via environment variables
- Structured logging
- Health check endpoints
- Metrics and monitoring ready
- Docker-friendly
- Graceful shutdown

### Usage

```bash
# Build
cd examples/production_server
go build

# Run with default config
./production_server

# Run with environment config
export SSH_PORT=2222
export HTTP_PORT=8080
export LOG_LEVEL=info
export ENABLE_METRICS=true
./production_server

# Run in Docker
docker build -t consolekit-prod .
docker run -p 2222:2222 -p 8080:8080 consolekit-prod
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_PORT` | `2222` | SSH server port |
| `SSH_ENABLED` | `true` | Enable SSH transport |
| `SSH_AUTH_MODE` | `pubkey` | SSH auth: `anonymous`, `password`, `pubkey` |
| `HTTP_PORT` | `8080` | HTTP server port |
| `HTTP_ENABLED` | `true` | Enable HTTP transport |
| `HTTP_USER` | `admin` | HTTP basic auth username |
| `HTTP_PASS` | _(required)_ | HTTP basic auth password |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | Log format: `json`, `text` |
| `ENABLE_METRICS` | `false` | Enable Prometheus metrics endpoint |
| `METRICS_PORT` | `9090` | Metrics server port |
| `ALLOWED_COMMANDS` | _(all)_ | Comma-separated allowed commands |
| `DENIED_COMMANDS` | _(none)_ | Comma-separated denied commands |

### Health Checks

```bash
# SSH health check
nc -zv localhost 2222

# HTTP health check
curl http://localhost:8080/healthz
# Returns: ok

# Metrics (if enabled)
curl http://localhost:9090/metrics
```

### Command Filtering

Restrict available commands for security:

```bash
# Allow only specific commands
ALLOWED_COMMANDS="print,let,vars,date" ./production_server

# Deny dangerous commands
DENIED_COMMANDS="osexec,spawn" ./production_server
```

### Logging

Structured JSON logging for production:

```bash
LOG_FORMAT=json LOG_LEVEL=debug ./production_server

# Output example:
{"level":"info","time":"2026-01-31T15:30:00Z","msg":"SSH server started","port":2222}
{"level":"info","time":"2026-01-31T15:30:01Z","msg":"HTTP server started","port":8080}
{"level":"debug","time":"2026-01-31T15:30:15Z","msg":"Command executed","user":"admin","command":"print test"}
```

### Docker Deployment

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o production_server

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/production_server /usr/local/bin/
EXPOSE 2222 8080
CMD ["production_server"]
```

```bash
# Build and run
docker build -t consolekit-prod .
docker run -d \
  -p 2222:2222 \
  -p 8080:8080 \
  -e HTTP_PASS=secretpassword \
  -e LOG_LEVEL=info \
  consolekit-prod
```

---

## 5. REST API Example (`examples/rest_api`)

### Description
HTTP REST API wrapper around ConsoleKit commands, perfect for integration with other systems.

### Features
- RESTful HTTP API
- JSON request/response
- Command execution via HTTP
- CORS support
- Rate limiting (optional)
- API key authentication

### Usage

```bash
# Build
cd examples/rest_api
go build

# Run on default port (8080)
./rest_api

# Run on custom port
PORT=9000 ./rest_api

# Run with API key authentication
API_KEY=your-secret-key ./rest_api
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `API_KEY` | _(none)_ | Required API key for authentication |
| `CORS_ENABLED` | `true` | Enable CORS headers |
| `CORS_ORIGIN` | `*` | CORS allowed origin |
| `RATE_LIMIT` | `100` | Requests per minute per IP |
| `TIMEOUT` | `30s` | Command execution timeout |

### API Endpoints

#### Execute Command
```bash
POST /execute
Content-Type: application/json
X-API-Key: your-secret-key (if API_KEY set)

{
  "command": "print 'Hello World'",
  "scope": {
    "@arg1": "value1",
    "@arg2": "value2"
  }
}

# Response:
{
  "output": "Hello World\n",
  "success": true,
  "duration_ms": 5,
  "timestamp": "2026-01-31T15:30:00Z"
}
```

#### Health Check
```bash
GET /health

# Response:
{
  "status": "ok",
  "timestamp": "2026-01-31T15:30:00Z"
}
```

#### Version Info
```bash
GET /version

# Response:
{
  "version": "0.7.0",
  "api_version": "v1"
}
```

### Usage Examples

#### cURL

```bash
# Simple command
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{"command": "date"}'

# With API key
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-secret-key" \
  -d '{"command": "print \"Hello\""}'

# With scoped variables
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "print \"Name: @name\"",
    "scope": {
      "@name": "Alice"
    }
  }'

# Background job
curl -X POST http://localhost:8080/execute \
  -d '{"command": "osexec --background \"sleep 30\""}'
```

#### Python

```python
import requests

url = "http://localhost:8080/execute"
headers = {
    "Content-Type": "application/json",
    "X-API-Key": "your-secret-key"
}

# Execute command
response = requests.post(url, headers=headers, json={
    "command": "let x=42; print @x"
})

result = response.json()
print(result["output"])  # Outputs: 42
```

#### JavaScript

```javascript
const execute = async (command, scope = {}) => {
  const response = await fetch('http://localhost:8080/execute', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': 'your-secret-key'
    },
    body: JSON.stringify({ command, scope })
  });

  return await response.json();
};

// Usage
const result = await execute('print "Hello"');
console.log(result.output);
```

### Error Handling

```bash
# Invalid command
curl -X POST http://localhost:8080/execute \
  -d '{"command": "invalid_command"}'

# Response:
{
  "output": "",
  "success": false,
  "error": "unknown command \"invalid_command\"",
  "duration_ms": 2
}
```

### Rate Limiting

When rate limit is exceeded:

```bash
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "Rate limit exceeded",
  "retry_after": 60
}
```

---

## Common Command-Line Patterns

### All Examples Support

```bash
# Show help
./<example> --help

# Run single command
./<example> "command here"

# Pipe commands
echo "command" | ./<example>

# Exit codes
./<example> "exit 1"
echo $?  # Returns: 1
```

### Environment Variable Priority

All examples follow this priority for configuration:

1. Environment variables (highest)
2. Configuration file (`~/.{appname}/config.toml`)
3. Default values (lowest)

---

## Comparison Matrix

| Feature | simple | ssh_server | multi_transport | production_server | rest_api |
|---------|--------|------------|-----------------|-------------------|----------|
| REPL | ✅ | ❌ | ✅ | ❌ | ❌ |
| SSH | ❌ | ✅ | ✅ | ✅ | ❌ |
| HTTP | ❌ | ❌ | ✅ | ✅ | ✅ |
| WebSocket | ❌ | ❌ | ✅ | ✅ | ❌ |
| Env Config | ❌ | ✅ | ✅ | ✅ | ✅ |
| Logging | Basic | Basic | Structured | Structured | Structured |
| Auth | N/A | Multiple | Multiple | Multiple | API Key |
| Production Ready | ❌ | ⚠️ | ⚠️ | ✅ | ✅ |

---

## Best Practices

### Development
- Start with **simple** to learn the basics
- Use **ssh_server** for remote testing
- Use **multi_transport** to explore all features

### Production
- Use **production_server** for internal tools
- Use **rest_api** for API integration

### Testing
```bash
# All examples support automated testing
echo "test command" | ./<example>

# Exit code testing
./<example> "exit 0" && echo "Success"
```

---

## Troubleshooting

### SSH Connection Refused
```bash
# Check if server is running
netstat -an | grep 2222

# Check firewall
sudo ufw allow 2222/tcp
```

### HTTP 401 Unauthorized
```bash
# Verify credentials
curl -v -u admin:password http://localhost:8080/health

# Check environment
echo $HTTP_USER $HTTP_PASS
```

---

## Next Steps

1. **Try the examples** - Start with `simple`, then explore others
2. **Customize** - Modify examples for your use case
3. **Deploy** - Use `production_server` as a template
4. **Integrate** - Use `rest_api` for system integration

For more information, see:
- [README.md](README.md) - Main documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Architecture details
- [API_CHANGES.md](API_CHANGES.md) - Migration guide
