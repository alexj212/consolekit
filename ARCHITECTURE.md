# ConsoleKit Architecture

This document describes the three-layer architecture of ConsoleKit v0.6.0+.

## Overview

ConsoleKit is built on a three-layer architecture that separates:
1. **Command Execution** - Pure command logic
2. **Transport Handlers** - How commands are delivered (SSH, HTTP, REPL)
3. **Display Adapters** - UI presentation (only for REPL transport)

This design enables serving the same commands over multiple protocols simultaneously.

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────┐
│                    Application Layer                      │
│  (Your custom commands, business logic, scripts)         │
└───────────────────┬──────────────────────────────────────┘
                    │
┌───────────────────┴──────────────────────────────────────┐
│              Layer 3: Transport Handlers                  │
│                                                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │LocalREPL    │  │ SSHHandler  │  │ HTTPHandler │      │
│  │Handler      │  │             │  │             │      │
│  │             │  │             │  │             │      │
│  │ • Terminal  │  │ • Auth      │  │ • WebSocket │      │
│  │ • History   │  │ • PTY       │  │ • REST API  │      │
│  │ • Prompts   │  │ • Multi-    │  │ • Sessions  │      │
│  │ • Colors    │  │   Session   │  │ • Embedded  │      │
│  │             │  │             │  │   Web UI    │      │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘      │
│         │                │                 │             │
└─────────┼────────────────┼─────────────────┼─────────────┘
          │                │                 │
          └────────────────┼─────────────────┘
                          │
┌──────────────────────────┴───────────────────────────────┐
│           Layer 2: TransportHandler Interface             │
│                                                           │
│  • Start() error                - Begin serving          │
│  • Stop() error                 - Graceful shutdown      │
│  • Name() string                - Transport type         │
│  • TransportConfig              - Command filtering      │
│                                                           │
└───────────────────┬───────────────────────────────────────┘
                    │
┌───────────────────┴───────────────────────────────────────┐
│              Layer 1: CommandExecutor                      │
│                                                            │
│  Core Execution Engine (Transport-Agnostic)               │
│  ┌──────────────────────────────────────────────────┐    │
│  │ • Execute()           - Execute commands      │    │
│  │ • ExpandCommand()       - Token replacement     │    │
│  │ • RootCmd()          - Cobra command tree    │    │
│  │ • AddCommands()           - Register commands     │    │
│  └──────────────────────────────────────────────────┘    │
│                                                            │
│  State Management                                         │
│  ┌──────────────────────────────────────────────────┐    │
│  │ • Defaults (SafeMap)      - Variables (@vars)     │    │
│  │ • aliases (SafeMap)       - Command aliases       │    │
│  │ • VariableExpanders          - Custom replacers      │    │
│  └──────────────────────────────────────────────────┘    │
│                                                            │
│  Managers (Thread-Safe)                                   │
│  ┌──────────────────────────────────────────────────┐    │
│  │ • JobManager              - Background jobs       │    │
│  │ • Config                  - Configuration         │    │
│  │ • LogManager              - Audit logging         │    │
│  │ • TemplateManager         - Script templates      │    │
│  │ • NotificationManager           - Notifications         │    │
│  └──────────────────────────────────────────────────┘    │
│                                                            │
│  File I/O Abstraction                                     │
│  ┌──────────────────────────────────────────────────┐    │
│  │ • FileHandler interface   - Pluggable file access│    │
│  │   - LocalFileHandler      - Direct filesystem    │    │
│  │   - RestrictedHandler     - Chroot-like          │    │
│  │   - CustomHandler         - Your implementation  │    │
│  └──────────────────────────────────────────────────┘    │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

## Layer 1: CommandExecutor

The `CommandExecutor` is the core execution engine. It is completely transport-agnostic and handles:

### Responsibilities

1. **Command Execution**
   - Parse command lines with pipes, redirects, token replacement
   - Execute Cobra commands
   - Handle context cancellation and timeouts
   - Prevent infinite recursion

2. **State Management**
   - Variables (`@varname`) in thread-safe SafeMap
   - Aliases (command shortcuts)
   - Token replacers (custom `@token` handlers)

3. **Manager Coordination**
   - Background jobs
   - Configuration persistence
   - Command audit logging
   - Script templates
   - Notifications

4. **File I/O Abstraction**
   - Pluggable `FileHandler` interface
   - Allows chroot-like restrictions per transport
   - Supports custom file backends

### Key Methods

```go
type CommandExecutor struct {
    AppName         string
    Defaults        *safemap.SafeMap[string, string]
    aliases         *safemap.SafeMap[string, string]
    VariableExpanders  []func(string) (string, bool)
    JobManager      *JobManager
    Config          *Config
    LogManager      *LogManager
    TemplateManager *TemplateManager
    NotificationManager   *NotificationManager
    FileHandler     FileHandler
    Scripts         embed.FS
}

func (e *CommandExecutor) Execute(line string, scope *SafeMap) (string, error)
func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, line string, scope *SafeMap) (string, error)
func (e *CommandExecutor) ExpandCommand(cmd *cobra.Command, scope *SafeMap, input string) string
func (e *CommandExecutor) RootCmd() func() *cobra.Command
func (e *CommandExecutor) AddCommands(cmds func(*cobra.Command))
func (e *CommandExecutor) AddBuiltinCommands()
```

### Thread Safety

- All state maps are `SafeMap` (thread-safe)
- Managers are thread-safe
- Supports concurrent command execution from multiple transports
- Recursion protection via atomic counter

## Layer 2: TransportHandler Interface

The `TransportHandler` interface defines how commands are delivered to the executor.

### Interface Definition

```go
type TransportHandler interface {
    Start() error  // Begin serving (blocking)
    Stop() error   // Graceful shutdown
    Name() string  // Transport type ("repl", "ssh", "http")
}
```

### TransportConfig

Optional configuration for command filtering:

```go
type TransportConfig struct {
    Executor        *CommandExecutor
    SessionLogger   *LogManager
    AllowedCommands []string  // Whitelist
    DeniedCommands  []string  // Blacklist (takes precedence)
}
```

### Available Transports

#### 1. REPLHandler
- **Purpose**: Interactive terminal REPL
- **Features**:
  - Display adapter integration (reeflective, bubbletea)
  - History file persistence
  - Color output
  - TTY detection
  - Batch mode (stdin piping)
  - Command-line argument execution

```go
executor, _ := NewCommandExecutor("app", customizer)
handler := NewREPLHandler(executor)
handler.SetPrompt(func() string { return "app > " })
handler.Run()
```

#### 2. SSHHandler
- **Purpose**: Remote SSH access
- **Features**:
  - Multi-session support
  - Password authentication
  - Public key authentication
  - PTY support
  - Interactive shell
  - Single command execution (`ssh user@host cmd`)
  - Per-session environment

```go
hostKey, _ := generateHostKey()
sshHandler := NewSSHHandler(executor, ":2222", hostKey)
sshHandler.SetAuthConfig(&SSHAuthConfig{
    PasswordAuth: passwordAuthFunc,
    PublicKeyAuth: pubkeyAuthFunc,
})
sshHandler.Start()
```

#### 3. HTTPHandler
- **Purpose**: Web-based access
- **Features**:
  - HTTP REST API
  - WebSocket REPL
  - Session authentication
  - Embedded xterm.js terminal
  - Serve from embedded files or local directory
  - Security headers (CSP, HSTS, etc.)

```go
httpHandler := NewHTTPHandler(executor, ":8080", "user", "pass")
httpHandler.Start()
```

**Endpoints:**
- `/login` - POST authentication
- `/logout` - POST session termination
- `/repl` - WebSocket REPL
- `/admin` - Web terminal UI
- `/` - Landing page

## Layer 3: Display Adapters (REPL Only)

Display adapters are **only used by REPLHandler**. They handle the terminal UI.

### DisplayAdapter Interface

```go
type DisplayAdapter interface {
    Configure(config DisplayConfig)
    SetCommands(cmdFactory func() *cobra.Command)
    SetHistoryFile(path string)
    SetPrompt(promptFunc func() string)
    AddPreCommandHook(hook func([]string) ([]string, error))
    Start() error
}
```

### Available Adapters

1. **ReflectiveAdapter** (default)
   - Uses `reeflective/console`
   - Automatic Cobra completion
   - Full readline support
   - Production-ready

2. **BubbletteaAdapter** (stub)
   - Future: bubble tea TUI
   - Custom UI components
   - Rich terminal apps

## Multi-Transport Architecture

Multiple transports can run simultaneously sharing the same executor:

```go
executor, _ := NewCommandExecutor("app", customizer)

// Local REPL
replHandler := NewREPLHandler(executor)
go replHandler.Run()

// SSH Server
sshHandler := NewSSHHandler(executor, ":2222", hostKey)
go sshHandler.Start()

// HTTP Server
httpHandler := NewHTTPHandler(executor, ":8080", "user", "pass")
go httpHandler.Start()

// All transports share:
// - Same job manager (jobs visible from all transports)
// - Same variables
// - Same configuration
// - Same command audit log
```

## Data Flow Examples

### SSH Command Execution

```
User (SSH Client)
    │
    ├─ ssh admin@host -p 2222 "print hello"
    │
    ▼
SSHHandler
    │
    ├─ Authenticate user
    ├─ Create session context
    ├─ Parse command request
    │
    ▼
CommandExecutor
    │
    ├─ Token replacement
    ├─ Alias expansion
    ├─ Execute via Cobra
    │
    ▼
Cobra Command (print)
    │
    ├─ cmd.Printf("hello\n")
    │
    ▼
CommandExecutor
    │
    ├─ Capture output
    ├─ Log to audit trail
    │
    ▼
SSHHandler
    │
    ├─ Send output to SSH channel
    │
    ▼
User (SSH Client)
    │
    └─ Receives: "hello\n"
```

### WebSocket REPL

```
User (Web Browser)
    │
    ├─ WebSocket message: {"type": "input", "message": "jobs"}
    │
    ▼
HTTPHandler
    │
    ├─ Check session cookie
    ├─ Parse JSON message
    │
    ▼
CommandExecutor
    │
    ├─ Execute("jobs", nil)
    │
    ▼
Cobra Command (jobs)
    │
    ├─ Query JobManager
    ├─ Format output
    │
    ▼
CommandExecutor
    │
    ├─ Return output string
    │
    ▼
HTTPHandler
    │
    ├─ Create JSON response: {"type": "output", "message": "..."}
    ├─ Send via WebSocket
    │
    ▼
User (Web Browser - xterm.js)
    │
    └─ Displays output in terminal
```

## Command Filtering

Restrict commands per transport:

```go
// SSH: Deny dangerous commands
sshConfig := &TransportConfig{
    Executor: executor,
    DeniedCommands: []string{"osexec", "clip", "paste"},
}
sshHandler.SetTransportConfig(sshConfig)

// HTTP: Allow only safe commands
httpConfig := &TransportConfig{
    Executor: executor,
    AllowedCommands: []string{"print", "date", "jobs", "vars"},
}
httpHandler.SetTransportConfig(httpConfig)

// REPL: Full access (no restrictions)
```

## File Access Control

Different transports can have different file access:

```go
// SSH: Chroot to /var/app
type RestrictedFileHandler struct {
    basePath string
}

func (h *RestrictedFileHandler) WriteFile(path, content string) error {
    fullPath := filepath.Join(h.basePath, path)
    if !strings.HasPrefix(fullPath, h.basePath) {
        return fmt.Errorf("access denied: outside chroot")
    }
    return os.WriteFile(fullPath, []byte(content), 0644)
}

executor.FileHandler = &RestrictedFileHandler{basePath: "/var/app"}
```

## Session Context

Each transport can provide session-specific context:

```go
// SSH: Add session variables
defs := safemap.New[string, string]()
defs.Set("@ssh:user", session.Username)
defs.Set("@ssh:remote_ip", session.RemoteIP)
defs.Set("@ssh:session_id", session.ID)

executor.Execute(command, scope)

// HTTP: Add HTTP context
defs.Set("@http:user", session.Username)
defs.Set("@http:session_id", session.SessionID)
```

## Extensibility

### Adding a New Transport

1. Implement `TransportHandler` interface:
```go
type MyTransport struct {
    executor *CommandExecutor
}

func (t *MyTransport) Start() error {
    // Listen and serve
}

func (t *MyTransport) Stop() error {
    // Graceful shutdown
}

func (t *MyTransport) Name() string {
    return "mytransport"
}
```

2. Use the executor:
```go
output, err := t.executor.Execute(command, sessionDefaults)
```

3. Handle session lifecycle:
```go
// Track sessions
// Apply command filtering
// Log execution
// Return results to client
```

### Custom Display Adapter

```go
type MyAdapter struct {
    // ... your fields
}

func (a *MyAdapter) Configure(config DisplayConfig) { ... }
func (a *MyAdapter) SetCommands(cmdFactory func() *cobra.Command) { ... }
func (a *MyAdapter) SetHistoryFile(path string) { ... }
func (a *MyAdapter) SetPrompt(promptFunc func() string) { ... }
func (a *MyAdapter) AddPreCommandHook(hook func([]string) ([]string, error)) { ... }
func (a *MyAdapter) Start() error { ... }

// Use it
handler := NewREPLHandler(executor)
handler.SetDisplayAdapter(NewMyAdapter())
```

## Security Considerations

### Transport Layer

- **SSH**: Standard SSH security (keys, passwords, encryption)
- **HTTP**: HTTPS recommended (use reverse proxy)
- **REPL**: Local access only (no network exposure)

### Command Filtering

- Use `DeniedCommands` to blacklist dangerous commands
- Use `AllowedCommands` to whitelist safe commands
- Apply different policies per transport

### File Access

- Use custom `FileHandler` for chroot-like restrictions
- Validate file paths before access
- Consider transport-specific file handlers

### Audit Logging

- Enable `LogManager` for all command executions
- Log includes: timestamp, user, command, output, duration, success
- Per-transport session tracking

## Performance

### Concurrency

- Thread-safe design allows concurrent command execution
- Multiple transports can serve simultaneously
- SafeMap for all shared state

### Resource Management

- Context support for command timeouts
- Graceful shutdown for all transports
- Job cancellation support

### Scalability

- Stateless command execution
- Session state isolated per transport
- Horizontal scaling possible (with shared config/logs)

## Migration from v0.5.x

### Before (Monolithic CLI)

```go
cli, _ := consolekit.NewCLI("app", customizer)
cli.Run()
```

### After (Three-Layer Architecture)

```go
// Create executor
executor, _ := consolekit.NewCommandExecutor("app", customizer)

// Choose transport
handler := consolekit.NewREPLHandler(executor)
handler.Run()
```

### Key Changes

- `CLI` struct removed
- `NewCLI()` → `NewCommandExecutor()` + transport handler
- Commands take `*CommandExecutor` instead of `*CLI`
- Color functions removed from commands (use `cmd.Printf()`)
- History methods moved to REPL handler
- Interactive prompts moved to REPL handler

## See Also

- [EXAMPLES.md](examples/EXAMPLES.md) - Example applications
- [SECURITY.md](SECURITY.md) - Security guide
- [MCP_INTEGRATION.md](MCP_INTEGRATION.md) - MCP protocol integration
