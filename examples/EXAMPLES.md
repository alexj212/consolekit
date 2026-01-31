# ConsoleKit Examples

This directory contains examples demonstrating different ConsoleKit transport handlers and use cases.

## Examples Overview

### 1. simple/ - Basic REPL Example
The simplest example showing local REPL usage.

```bash
cd simple
go build
./simple
```

**Features:**
- Local interactive REPL
- Command-line argument execution
- Batch mode (stdin piping)
- Custom commands

### 2. ssh_server/ - SSH Server Example
Demonstrates SSH server transport for remote access.

```bash
cd ssh_server
go build
./ssh_server
```

Then connect via SSH:
```bash
ssh admin@localhost -p 2222
# Password: secret123
```

**Features:**
- Password authentication
- Public key authentication
- Multi-session support
- PTY support for interactive commands
- Session isolation

### 3. tailscale_http/ - Tailscale HTTP Example
HTTP/WebSocket server with optional Tailscale integration.

```bash
cd tailscale_http
go build

# Without Tailscale (local only)
./tailscale_http

# With Tailscale (requires auth key)
TS_AUTH_KEY="tskey-auth-..." ./tailscale_http
```

Then access:
- Web Terminal: http://localhost:8080/admin
- Username: admin
- Password: secret123

**Features:**
- Web-based xterm.js terminal
- WebSocket REPL connection
- Session authentication
- Tailscale integration for secure remote access
- Embedded web UI

**Tailscale Setup:**
1. Create auth key at https://login.tailscale.com/admin/settings/keys
2. Set TS_AUTH_KEY environment variable
3. Access server via Tailscale IP from anywhere

### 4. multi_transport/ - All Transports Example
Runs SSH, HTTP, and local REPL simultaneously.

```bash
cd multi_transport
go build

# Without Tailscale
./multi_transport

# With Tailscale
TS_AUTH_KEY="tskey-auth-..." ./multi_transport
```

**Access Methods:**
- Local REPL: Interactive prompt in terminal
- SSH: `ssh admin@localhost -p 2222`
- Web: http://localhost:8080/admin

**Features:**
- All transports share the same CommandExecutor
- Commands executed from any transport are logged
- Jobs created in one transport visible from all others
- Graceful shutdown of all transports

## Building Examples

Each example can be built with version info:

```bash
go build -ldflags "\
  -X main.BuildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  -X main.LatestCommit=$(git rev-parse HEAD) \
  -X main.Version=v0.6.0"
```

## Common Features

All examples include:
- Version command (`version` or `v`)
- All built-in ConsoleKit commands
- Job management
- Configuration system
- Logging support
- Template system
- Variable system

## Transport Comparison

| Feature | REPL | SSH | HTTP/WebSocket |
|---------|------|-----|----------------|
| Interactive | ✓ | ✓ | ✓ |
| Remote Access | ✗ | ✓ | ✓ |
| Multi-Session | ✗ | ✓ | ✓ |
| Authentication | ✗ | ✓ | ✓ |
| Web Browser | ✗ | ✗ | ✓ |
| Tailscale | ✗ | ✓ | ✓ |

## Security Considerations

### SSH Server
- Use strong passwords or public key authentication
- Limit allowed commands via TransportConfig
- Consider firewall rules for port 2222
- Use Tailscale for secure remote access

### HTTP Server
- Always use HTTPS in production (reverse proxy)
- Secure session cookies (HttpOnly, Secure, SameSite)
- Implement rate limiting for login endpoint
- Use strong passwords
- Consider 2FA for production
- Use Tailscale for secure remote access

### Tailscale
- Provides encrypted WireGuard tunnels
- Built-in authentication via Tailscale account
- No port forwarding required
- Audit logging via Tailscale admin console
- Access controls via Tailscale ACLs

## Development Tips

### Custom Transport
To create a custom transport:
1. Implement `TransportHandler` interface
2. Use `CommandExecutor.Execute()` to run commands
3. Handle session context and logging
4. See existing handlers for patterns

### Command Filtering
Restrict available commands:

```go
config := &TransportConfig{
    Executor: executor,
    DeniedCommands: []string{"osexec", "clip"},
}
handler.SetTransportConfig(config)
```

### Custom File Handler
Chroot-like file access for SSH:

```go
type RestrictedFileHandler struct {
    basePath string
}

func (h *RestrictedFileHandler) WriteFile(path, content string) error {
    fullPath := filepath.Join(h.basePath, path)
    if !strings.HasPrefix(fullPath, h.basePath) {
        return fmt.Errorf("access denied")
    }
    return os.WriteFile(fullPath, []byte(content), 0644)
}

executor.FileHandler = &RestrictedFileHandler{basePath: "/var/app"}
```

## Troubleshooting

### SSH Connection Refused
- Check port 2222 is not in use: `lsof -i :2222`
- Verify firewall allows port 2222
- Check server logs for errors

### WebSocket Connection Failed
- Verify HTTP server is running on port 8080
- Check browser console for errors
- Try `ws://` instead of `wss://` for local testing
- Ensure session cookie is set after login

### Tailscale Not Working
- Verify TS_AUTH_KEY is valid
- Check Tailscale status: `tailscale status`
- Ensure node appears in Tailscale admin console
- Check firewall allows Tailscale daemon

## Further Reading

- [ConsoleKit Documentation](../CLAUDE.md)
- [MCP Integration](../MCP_INTEGRATION.md)
- [Security Guide](../SECURITY.md)
- [Tailscale Documentation](https://tailscale.com/kb/)
