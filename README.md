# ConsoleKit

<div align="center">

**A powerful, modular CLI framework for building sophisticated console applications in Go**

[![Go Reference](https://pkg.go.dev/badge/github.com/alexj212/consolekit.svg)](https://pkg.go.dev/github.com/alexj212/consolekit)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexj212/consolekit)](https://goreportcard.com/report/github.com/alexj212/consolekit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

---

## Overview

ConsoleKit is a comprehensive CLI library that brings shell-like power to your Go applications. Built on top of [spf13/cobra](https://github.com/spf13/cobra) and [reeflective/console](https://github.com/reeflective/console), it provides a rich REPL (Read-Eval-Print Loop) environment with advanced features like command chaining, piping, token replacement, and scriptable automation.

Perfect for building internal tools, administrative consoles, and developer utilities that require a powerful interactive interface.

## ✨ Features

### Core Capabilities

- 🖥️ **Interactive REPL** - Full-featured shell-like environment with history, completion, and line editing
- 🌐 **Multi-Transport** - Serve commands over REPL, SSH, HTTP/WebSocket, Unix/TCP socket, or all simultaneously
- 🔗 **Command Chaining** - Execute multiple commands sequentially using `;` separator
- 🚦 **Piping** - Chain command outputs using Unix-style `|` pipes
- 📁 **I/O Redirection** - Redirect command output to files with `>` while displaying on stdout
- 🎯 **Intelligent Completion** - Automatic command, subcommand, and flag completion via Cobra integration
- 📜 **Command History** - Persistent history with search, bookmarks, and replay

### Advanced Features

- 🏗️ **Three-Layer Architecture** - CommandExecutor (core) + TransportHandlers (SSH/HTTP/REPL) + DisplayAdapters (UI)
- 🏷️ **Aliases** - Create command shortcuts with persistent storage
- 🔄 **Variable Expansion** - Dynamic variable substitution with `@varname`, `@env:VAR`, and `@exec:command`
- 📝 **Script Execution** - Run embedded or external scripts with argument passing
- ⚡ **Background Jobs** - Execute commands asynchronously with full job management
- 💬 **Comment Support** - Use `#` for inline comments in commands and scripts
- 🎨 **Color Support** - Automatic color output with TTY detection and `NO_COLOR` support
- 🔒 **Thread-Safe** - Concurrent command execution from multiple transports

## 📦 Installation

```bash
go get github.com/alexj212/consolekit
```

**Requirements:** Go 1.21+

## 🚀 Quick Start

### Simple REPL Application

```go
package main

import (
    "embed"
    "log"
    "github.com/alexj212/consolekit"
)

//go:embed scripts/*
var scripts embed.FS

func main() {
    // Create command executor with builtin commands
    executor, err := consolekit.NewCommandExecutor("myapp", func(exec *consolekit.CommandExecutor) error {
        exec.Scripts = &scripts  // v0.8.0+: pointer required
        exec.AddBuiltinCommands()  // Adds all standard commands
        return nil
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create REPL handler
    repl := consolekit.NewREPLHandler(executor)
    repl.SetPrompt(func() string {
        return "\nmyapp> "
    })

    // Start interactive REPL
    if err := repl.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Multi-Transport Server (REPL + SSH + HTTP)

```go
package main

import (
    "log"
    "github.com/alexj212/consolekit"
)

func main() {
    // Create shared command executor
    executor, _ := consolekit.NewCommandExecutor("myapp", func(exec *consolekit.CommandExecutor) error {
        exec.AddBuiltinCommands()
        return nil
    })

    // Start SSH server (port 2222)
    hostKey, _ := consolekit.GenerateHostKey()
    sshHandler := consolekit.NewSSHHandler(executor, ":2222", hostKey)
    go sshHandler.Start()

    // Start HTTP/WebSocket server (port 8080)
    httpHandler := consolekit.NewHTTPHandler(executor, ":8080", "admin", "password")
    go httpHandler.Start()

    // Start local REPL
    repl := consolekit.NewREPLHandler(executor)
    if err := repl.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Modular Command Selection (v0.8.0+)

Pick and choose which command groups to include:

```go
func main() {
    executor, _ := consolekit.NewCommandExecutor("myapp", func(exec *consolekit.CommandExecutor) error {
        // Option 1: Use convenience bundles
        exec.AddCommands(consolekit.AddMinimalCmds(exec))      // Core + variables + scripting
        exec.AddCommands(consolekit.AddStandardCmds(exec))     // Recommended defaults
        exec.AddCommands(consolekit.AddAllCmds(exec))          // Everything

        // Option 2: Pick specific groups
        exec.AddCommands(consolekit.AddCoreCmds(exec))         // cls, exit, print, date
        exec.AddCommands(consolekit.AddVariableCmds(exec))     // let, unset, vars
        exec.AddCommands(consolekit.AddNetworkCmds(exec))      // http
        exec.AddCommands(consolekit.AddDataManipulationCmds(exec)) // json, yaml, csv

        // Scripts (note: pointer required in v0.8.0+)
        exec.AddCommands(consolekit.AddRun(exec, &scripts))    // With embedded scripts
        exec.AddCommands(consolekit.AddRun(exec, nil))         // External scripts only

        return nil
    })

    repl := consolekit.NewREPLHandler(executor)
    repl.Run()
}
```

**Benefits:**
- ✅ Smaller binaries (only compile what you need)
- ✅ Faster startup (fewer commands to register)
- ✅ Better security (exclude dangerous commands)
- ✅ Clearer intent (explicit capabilities)

See [COMMAND_GROUPS.md](COMMAND_GROUPS.md) for complete documentation.

## 💡 Usage Examples

### Interactive Commands

```bash
# Basic command execution
myapp> print "Hello, World!"
Hello, World!

# Command chaining with semicolons
myapp> let greeting="Hello" ; print @greeting
Hello

# Piping between commands
myapp> print "line1\nline2\nline3" | grep line2
line2

# File redirection (displays AND writes to file)
myapp> print "Logged data" > output.txt
Logged data
```

### Variables & Expansion

```bash
# Set and use variables
myapp> let name="Alice"
myapp> print "Hello, @name"
Hello, Alice

# Environment variables
myapp> print "User: @env:USER"
User: john

# Command substitution
myapp> let timestamp=$(date)
myapp> print "Time: @timestamp"
Time: 2025-01-31 15:30:45

# Increment/decrement
myapp> let counter=0
myapp> inc counter
counter = 1
```

### Aliases

```bash
# Create an alias
myapp> alias ls="print 'Listing files...'"
Alias set: ls = print 'Listing files...'

# Use the alias
myapp> ls
Listing files...

# List all aliases
myapp> alias
Aliases:
----------------------------------------
ls=print 'Listing files...'
```

### Script Execution

Create a script file `tasks.sh`:

```bash
# tasks.sh
print "Starting task with arg: @arg0"
let counter=5
repeat --count @counter --sleep 1 "print 'Working...'"
print "Task completed!"
```

Execute the script:

```bash
# Run from file system
myapp> run tasks.sh "my-task"
Starting task with arg: my-task
Working...
...

# Run embedded script (from embed.FS)
myapp> run @embedded-script
```

### Background Jobs

```bash
# Run command in background
myapp> osexec --background "sleep 60"
Command started in background with PID 12345 (Job ID: 1)

# List all jobs
myapp> jobs
Background Jobs:
[1] [running] PID:12345 Duration:5s
    sleep 60

# View job details and logs
myapp> job 1
myapp> job 1 logs

# Kill a job or all jobs
myapp> job 1 kill
myapp> killall
```

### History & Bookmarks

```bash
# Search history
myapp> history search "print"
0: print "test"
15: print @timestamp

# Replay command by index
myapp> history replay 15

# Bookmark frequently used commands
myapp> history bookmark add deploy "run deploy.sh prod"
myapp> history bookmark run deploy

# View statistics
myapp> history stats
Total commands: 150
Unique commands: 45
Top 10 most used...
```

## 🏗️ Architecture

### Three-Layer Design

ConsoleKit uses a clean three-layer architecture:

```
┌─────────────────────────────────────────┐
│         Transport Handlers              │
│  (How commands are delivered)           │
│                                         │
│  • REPLHandler (local terminal)        │
│  • SSHHandler (SSH server)             │
│  • HTTPHandler (HTTP/WebSocket)        │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│         CommandExecutor                 │
│  (Pure command execution engine)        │
│                                         │
│  • Execute commands                     │
│  • Expand variables                     │
│  • Manage jobs, config, logs            │
│  • Thread-safe state                    │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│         Display Adapters                │
│  (UI abstraction - REPL only)           │
│                                         │
│  • ReflectiveAdapter (default)         │
│  • Custom adapters (implement iface)  │
└─────────────────────────────────────────┘
```

### Custom Commands

```go
// Custom command module
func AddMyCommand(exec *consolekit.CommandExecutor) func(*cobra.Command) {
    return func(rootCmd *cobra.Command) {
        cmd := &cobra.Command{
            Use:   "greet [name]",
            Short: "Greet someone",
            Args:  cobra.ExactArgs(1),
            Run: func(cmd *cobra.Command, args []string) {
                name := args[0]
                // Expand variables in arguments
                name = exec.ExpandVariables(cmd, nil, name)
                cmd.Printf("Hello, %s!\n", name)
            },
        }
        rootCmd.AddCommand(cmd)
    }
}

// Register the command
executor, _ := consolekit.NewCommandExecutor("myapp", func(exec *consolekit.CommandExecutor) error {
    AddMyCommand(exec)
    return nil
})
```

### Built-in Command Modules

| Module | Key Commands | Description |
|--------|--------------|-------------|
| **base** | `print`, `let`, `if`, `date`, `sleep`, `repeat`, `http` | Core utilities |
| **alias** | `alias`, `unalias` | Alias management with persistence |
| **history** | `history list/search/replay`, `bookmark add/run` | History and bookmarks |
| **run** | `run`, `vs`, `spawn` | Script execution |
| **exec** | `osexec` | OS command execution |
| **jobs** | `jobs`, `job`, `killall` | Background job management |
| **variables** | `let`, `vars`, `inc`, `dec` | Variable operations |
| **config** | `config get/set/edit` | Configuration management |
| **logging** | `log enable/show/export` | Command audit logging |
| **template** | `template exec/create` | Script templating |
| **notify** | `notify send` | Desktop notifications |
| **data** | `json`, `yaml`, `csv` | Data parsing/conversion |
| **format** | `table`, `highlight` | Output formatting |

## 🔐 Security

> **⚠️ IMPORTANT SECURITY NOTICE**
>
> ConsoleKit is designed for **trusted environments only**. It provides extensive system access equivalent to shell access and should **never** be used in multi-tenant or untrusted environments.

### Suitable For ✅
- Local development tools
- Internal automation scripts
- Trusted administrator consoles
- SSH access to internal systems
- Single-user applications

### Not Suitable For ❌
- Web-facing applications (without extensive hardening)
- Multi-tenant systems
- Untrusted user environments
- Public APIs

### Security Features

- ✅ **Recursion Protection** - Maximum execution depth limit prevents infinite loops
- ✅ **Thread-Safe** - Concurrent access from multiple transports
- ✅ **Proper Quote Handling** - Prevents some injection vectors
- ✅ **Scoped Variables** - Script arguments are isolated per execution
- ✅ **SSH Authentication** - Public key, password, or anonymous modes
- ✅ **HTTP Authentication** - Basic auth for HTTP transport

**See [SECURITY.md](SECURITY.md) for comprehensive security documentation.**

## 📚 Documentation

| Document | Description |
|----------|-------------|
| **[API_CHANGES.md](API_CHANGES.md)** | v0.7.0 API naming refactor details and migration guide |
| **[ARCHITECTURE.md](ARCHITECTURE.md)** | Three-layer architecture and design patterns |
| **[COMMANDS.md](COMMANDS.md)** | Complete command reference |
| **[MCP_INTEGRATION.md](MCP_INTEGRATION.md)** | Model Context Protocol integration guide |
| **[SECURITY.md](SECURITY.md)** | Security considerations and deployment guidelines |
| **[CLAUDE.md](CLAUDE.md)** | Development guide for Claude Code |
| **[examples/EXAMPLES.md](examples/EXAMPLES.md)** | Example applications and use cases |

## 🛠️ Development

### Building

```bash
# Build the library
go build

# Run tests
go test ./...

# Run parser tests
go test ./parser

# Run benchmarks
go test -bench . ./...
```

### Project Structure

```
consolekit/
├── executor.go         # CommandExecutor - core execution engine
├── handler_repl.go     # REPLHandler - local terminal
├── handler_ssh.go      # SSHHandler - SSH server
├── handler_http.go     # HTTPHandler - HTTP/WebSocket server
├── transport.go        # TransportHandler interface
├── display.go          # DisplayAdapter interface
├── history.go          # HistoryManager
├── jobs.go             # JobManager
├── logging.go          # LogManager
├── template.go         # TemplateManager
├── notify.go           # NotificationManager
├── config.go           # Configuration system
├── base.go             # Base commands
├── *cmds.go            # Command modules
├── parser/             # Command parser
├── safemap/            # Thread-safe map
└── examples/           # Example applications
    ├── simple/         # Basic REPL
    ├── ssh_server/     # SSH server
    ├── multi_transport/# All transports
    └── rest_api/       # REST API wrapper
```

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add or update tests
4. Ensure `go test ./...` passes
5. Submit a pull request

## 📝 License

ConsoleKit is released under the [MIT License](LICENSE).

## 🙏 Acknowledgments

Built with excellent Go libraries:
- [spf13/cobra](https://github.com/spf13/cobra) - Command framework
- [reeflective/console](https://github.com/reeflective/console) - REPL interface
- [kballard/go-shellquote](https://github.com/kballard/go-shellquote) - Shell-style quote parsing
- [fatih/color](https://github.com/fatih/color) - Colorized output

---

<div align="center">

**[⬆ back to top](#consolekit)**

Made with ❤️ for building powerful CLI tools

</div>

---

## 📖 Examples

ConsoleKit includes 6 comprehensive example applications. See **[EXAMPLES_REFERENCE.md](EXAMPLES_REFERENCE.md)** for complete documentation including:

- Command-line flags and options
- Environment variable configuration  
- Authentication setup
- API endpoints and usage
- Docker deployment
- Troubleshooting guides

**Quick Links:**
- [Simple REPL](EXAMPLES_REFERENCE.md#1-simple-example-examplessimple) - Getting started
- [SSH Server](EXAMPLES_REFERENCE.md#2-ssh-server-example-examplesssh_server) - Remote CLI access
- [Multi-Transport](EXAMPLES_REFERENCE.md#3-multi-transport-example-examplesmulti_transport) - All transports
- [Production Server](EXAMPLES_REFERENCE.md#4-production-server-example-examplesproduction_server) - Enterprise deployment
- [REST API](EXAMPLES_REFERENCE.md#5-rest-api-example-examplesrest_api) - HTTP API integration
