# ConsoleKit Examples

This directory contains examples demonstrating different ConsoleKit features, transport handlers, and use cases.

## Examples Overview

### 1. simple_console/ - Basic REPL Example
The simplest full-featured example showing local REPL usage with all commands.

```bash
cd simple_console
go build
./simple_console
```

**Features:**
- Local interactive REPL
- All built-in commands
- Command-line argument execution
- Batch mode (stdin piping)
- Custom commands

### 2. minimal_cli/ - Modular Command Selection (NEW in v0.8.0)
Demonstrates the new modular command selection system.

```bash
cd minimal_cli
go build
./minimal_cli
```

**Features:**
- **Selective command inclusion** - Only includes specific command groups
- Minimal footprint - Core + Variables + Scripting + Control Flow only
- Example of fine-grained command selection
- Shows how to build custom CLI with specific capabilities

**Use case:** Applications that need specific features without the full command set (e.g., script executor, automation tool, minimal REPL).

### 3. ssh_server/ - SSH Server Example
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

### 4. multi_transport/ - All Transports Example
Runs SSH, HTTP, and local REPL simultaneously.

```bash
cd multi_transport
go build
./multi_transport
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

Most examples include:
- Version command (`version` or `v`)
- Built-in ConsoleKit commands (varies by example)
- Job management
- Configuration system
- Logging support
- Template system
- Variable system

**Note:** `minimal_cli` demonstrates selective command inclusion and only includes a subset of commands.

## New in v0.8.0

### Modular Command System

ConsoleKit v0.8.0 introduces fine-grained control over which command groups to include. See `minimal_cli/` for an example.

**Convenience bundles:**
- `AddAllCmds()` - Everything (like old `AddBuiltinCommands()`)
- `AddStandardCmds()` - Recommended defaults
- `AddMinimalCmds()` - Core + variables + scripting
- `AddDeveloperCmds()` - Standard + developer tools
- `AddAutomationCmds()` - Optimized for automation

**Individual groups:** See [COMMAND_GROUPS.md](../COMMAND_GROUPS.md) for complete list.

### embed.FS Pointer Change

Script execution now requires pointers:

```go
// v0.8.0+
//go:embed *.run
var Scripts embed.FS
exec.AddCommands(consolekit.AddRun(exec, &Scripts))  // Note: &Scripts

// Or for external-only scripts:
exec.AddCommands(consolekit.AddRun(exec, nil))
```

See [MIGRATION.md](../MIGRATION.md) for migration guide.

## Transport Comparison

| Feature | REPL | SSH | HTTP/WebSocket |
|---------|------|-----|----------------|
| Interactive | ✓ | ✓ | ✓ |
| Remote Access | ✗ | ✓ | ✓ |
| Multi-Session | ✗ | ✓ | ✓ |
| Authentication | ✗ | ✓ | ✓ |
| Web Browser | ✗ | ✗ | ✓ |

## Security Considerations

### SSH Server
- Use strong passwords or public key authentication
- Limit allowed commands via TransportConfig
- Consider firewall rules for port 2222

### HTTP Server
- Always use HTTPS in production (reverse proxy)
- Secure session cookies (HttpOnly, Secure, SameSite)
- Implement rate limiting for login endpoint
- Use strong passwords
- Consider 2FA for production

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

## Further Reading

- [ConsoleKit Documentation](../CLAUDE.md)
- [MCP Integration](../MCP_INTEGRATION.md)
- [Security Guide](../SECURITY.md)
