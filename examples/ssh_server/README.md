# SSH Server Example

This example demonstrates a production-ready SSH server using ConsoleKit.

## Features

- **Password Authentication**: Multiple users with different passwords
- **Public Key Authentication**: Support for SSH keys via `~/.ssh/authorized_keys`
- **Persistent Host Key**: Automatically generates and saves host key
- **Command Filtering**: Restrict dangerous commands for certain users
- **Multi-Session**: Multiple simultaneous SSH connections
- **Interactive Shell**: Full REPL experience over SSH
- **Single Command Execution**: Execute commands without interactive session

## Building

```bash
cd examples/ssh_server
go build -ldflags "\
  -X main.BuildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  -X main.LatestCommit=$(git rev-parse HEAD) \
  -X main.Version=v1.0.0"
```

## Running

```bash
./ssh_server
```

The server will start on port 2222 and display connection information.

## Connecting

### Interactive Session

```bash
# Connect as admin
ssh admin@localhost -p 2222
# Password: secret123

# Connect as developer
ssh developer@localhost -p 2222
# Password: dev456

# Connect as guest (restricted access)
ssh guest@localhost -p 2222
# Password: guest
```

### Single Command Execution

Execute a command without entering interactive mode:

```bash
ssh admin@localhost -p 2222 'print "Hello from SSH"'
ssh admin@localhost -p 2222 'date'
ssh admin@localhost -p 2222 'jobs'
```

### Public Key Authentication

1. Generate SSH key (if you don't have one):
```bash
ssh-keygen -t rsa -b 2048 -f ~/.ssh/id_rsa
```

2. Add your public key to `~/.ssh/authorized_keys`:
```bash
cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

3. Connect without password:
```bash
ssh admin@localhost -p 2222
```

## User Accounts

| Username  | Password  | Access Level |
|-----------|-----------|--------------|
| admin     | secret123 | Full access  |
| developer | dev456    | Full access  |
| guest     | guest     | Restricted   |

## Restricted Commands

For security, certain commands are disabled:
- `osexec` - OS command execution
- `clip` - Clipboard access
- `paste` - Clipboard access

Attempting to run these will result in an error message.

## Example Session

```bash
$ ssh admin@localhost -p 2222
admin@localhost's password: ***

Welcome to ssh-server SSH console
User: admin, Session: ssh-123456789

admin@ssh-server > print "Hello, SSH!"
Hello, SSH!

admin@ssh-server > date
2026-01-31 03:00:00

admin@ssh-server > jobs
No jobs running

admin@ssh-server > exit
Goodbye!
Connection to localhost closed.
```

## Configuration

### Change Port

Edit `main.go` and change the port in:
```go
sshHandler := consolekit.NewSSHHandler(executor, ":2222", hostKey)
```

### Add Users

Modify the `validUsers` map in `PasswordAuth`:
```go
validUsers := map[string]string{
    "admin":     "secret123",
    "developer": "dev456",
    "guest":     "guest",
    "newuser":   "newpass",  // Add new user
}
```

### Customize Command Filtering

Modify the `DeniedCommands` list:
```go
config := &consolekit.TransportConfig{
    Executor: executor,
    DeniedCommands: []string{
        "osexec",
        "clip",
        "paste",
        "config",  // Deny config changes
    },
}
```

Or use `AllowedCommands` to whitelist:
```go
config := &consolekit.TransportConfig{
    Executor: executor,
    AllowedCommands: []string{
        "print",
        "date",
        "jobs",
        "version",
    },
}
```

## Security Considerations

### Production Deployment

1. **Use Strong Passwords**: Change default passwords
2. **Prefer Public Keys**: Disable password auth in production
3. **Firewall Rules**: Restrict SSH port to known IPs
4. **Logging**: Enable audit logging for all commands
5. **Tailscale**: Use Tailscale for secure remote access

### Public Key Only

To disable password authentication:
```go
sshHandler.SetAuthConfig(&consolekit.SSHAuthConfig{
    PublicKeyAuth: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
        // ... your public key validation logic
    },
    // Don't set PasswordAuth
})
```

### Command Auditing

Enable logging in the executor:
```go
customizer := func(exec *consolekit.CommandExecutor) error {
    // Enable command logging
    exec.LogManager.Enable()

    // Configure logging
    exec.LogManager.SetLogFile("/var/log/ssh-commands.log")
    exec.LogManager.SetLogSuccess(true)
    exec.LogManager.SetLogFailures(true)

    // ... rest of setup
}
```

## Troubleshooting

### Connection Refused

**Problem**: `ssh: connect to host localhost port 2222: Connection refused`

**Solutions**:
1. Check if server is running
2. Verify port 2222 is not used: `lsof -i :2222`
3. Check firewall rules
4. Try using IP address instead of hostname

### Authentication Failed

**Problem**: `Permission denied (password,publickey)`

**Solutions**:
1. Verify username and password are correct
2. Check server logs for authentication details
3. For public key auth, verify `~/.ssh/authorized_keys` permissions (should be 600)
4. Ensure public key is in correct format

### Host Key Changed Warning

**Problem**: `WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!`

**Solution**: This happens when the host key changes. Remove the old key:
```bash
ssh-keygen -R "[localhost]:2222"
```

Then reconnect to accept the new key.

## Advanced Usage

### Custom File Handler (Chroot)

Restrict file access to a specific directory:

```go
type RestrictedFileHandler struct {
    basePath string
}

func (h *RestrictedFileHandler) WriteFile(path string, content string) error {
    fullPath := filepath.Join(h.basePath, path)
    if !strings.HasPrefix(fullPath, h.basePath) {
        return fmt.Errorf("access denied: outside chroot")
    }
    return os.WriteFile(fullPath, []byte(content), 0644)
}

func (h *RestrictedFileHandler) ReadFile(path string) (string, error) {
    fullPath := filepath.Join(h.basePath, path)
    if !strings.HasPrefix(fullPath, h.basePath) {
        return "", fmt.Errorf("access denied: outside chroot")
    }
    data, err := os.ReadFile(fullPath)
    return string(data), err
}

// Set in executor
executor.FileHandler = &RestrictedFileHandler{basePath: "/var/app"}
```

### Per-User Command Filtering

Different restrictions for different users:

```go
PasswordAuth: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
    // ... authentication logic ...

    permissions := &ssh.Permissions{
        Extensions: map[string]string{
            "user": conn.User(),
        },
    }

    // Add role-based permissions
    if conn.User() == "guest" {
        permissions.Extensions["denied_commands"] = "osexec,clip,paste,config"
    }

    return permissions, nil
}
```

## Files Generated

- `ssh_host_key` - RSA private key (keep secure!)
- `~/.ssh-server/` - Configuration and data directory
- `~/.ssh-server/audit.log` - Command audit log (if logging enabled)

## See Also

- [Multi-Transport Example](../multi_transport/) - SSH + HTTP + REPL
- [Tailscale HTTP Example](../tailscale_http/) - Secure remote access
- [Main Documentation](../../CLAUDE.md) - Full ConsoleKit documentation
