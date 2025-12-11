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
- 🔗 **Command Chaining** - Execute multiple commands sequentially using `;` separator
- 🚦 **Piping** - Chain command outputs using Unix-style `|` pipes
- 📁 **I/O Redirection** - Redirect command output to files with `>` while displaying on stdout
- 🎯 **Intelligent Completion** - Automatic command, subcommand, and flag completion via Cobra integration
- 📜 **Command History** - Persistent history with search and navigation (stored in `~/.{appname}.history`)

### Advanced Features

- 🔧 **Modular Architecture** - Plug-and-play command modules for easy feature integration
- 🏷️ **Aliases** - Create command shortcuts with persistent storage (`~/.{appname}.aliases`)
- 🔄 **Token Replacement** - Dynamic variable substitution with `@varname`, `@env:VAR`, and `@exec:command`
- 📝 **Script Execution** - Run embedded or external scripts with argument passing (`@arg0`, `@arg1`, etc.)
- ⚡ **Background Jobs** - Execute commands asynchronously with `--spawn` flag
- 💬 **Comment Support** - Use `#` for inline comments in commands and scripts
- 🎨 **Color Support** - Automatic color output with TTY detection and `NO_COLOR` support
- 🔒 **Recursion Protection** - Built-in safeguards against infinite execution loops (max depth: 10)

## 📦 Installation

```bash
go get github.com/alexj212/consolekit
```

**Requirements:** Go 1.21+

## 🚀 Quick Start

Create a simple CLI application in minutes:

```go
package main

import (
    "embed"
    "fmt"
    "github.com/alexj212/consolekit"
    "github.com/alexj212/consolekit/cmds"
)

//go:embed scripts/*
var scripts embed.FS

func main() {
    // Create CLI with standard command modules
    cli, err := consolekit.NewCLI("myapp", func(cli *consolekit.CLI) error {
        cmds.AddAlias(cli)      // Alias management
        cmds.AddExec(cli)       // OS command execution
        cmds.AddHistory(cli)    // History commands
        cmds.AddMisc(cli)       // Utility commands (cat, grep, env)
        cmds.AddBaseCmds(cli)   // Core commands (print, set, if, etc.)
        cmds.AddRun(cli, scripts) // Script execution
        return nil
    })
    if err != nil {
        fmt.Printf("Error initializing CLI: %v\n", err)
        return
    }

    // Start the interactive REPL
    if err := cli.AppBlock(); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

Build and run:

```bash
go build -o myapp
./myapp
```

## 💡 Usage Examples

### Interactive Commands

```bash
# Basic command execution
myapp> print "Hello, World!"
Hello, World!

# Command chaining with semicolons
myapp> set greeting "Hello" ; print @greeting
Hello

# Piping between commands
myapp> print "line1\nline2\nline3" | grep line2
line2

# File redirection (displays AND writes to file)
myapp> print "Logged data" > output.txt
Logged data
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

# Remove an alias
myapp> unalias ls
```

### Token Replacement

```bash
# Set and use variables
myapp> set name "Alice"
myapp> print "Hello, @name"
Hello, Alice

# Environment variables
myapp> print "User: @env:USER"
User: john

# Command execution in tokens
myapp> set timestamp "@exec:date"
myapp> print "Time: @timestamp"
Time: 2025-12-11 15:30:45
```

### Script Execution

Create a script file `tasks.sh`:

```bash
# tasks.sh
print "Starting task with arg: @arg0"
set counter "5"
repeat --count @counter --sleep 1 "print 'Working...'"
print "Task completed!"
```

Execute the script:

```bash
# Run from file system
myapp> run tasks.sh "my-task"
Starting task with arg: my-task
Working...
Working...
...

# Run embedded script (from embed.FS)
myapp> run @embedded-script
```

### Background Execution

```bash
# Run command in background
myapp> spawn "repeat --count 10 --sleep 2 'print tick'"
spawn cmd: myapp | repeat --count 10 --sleep 2 'print tick'

# Run script in background
myapp> run --spawn long-running-task.sh
```

## 🏗️ Architecture

### Command Modules

ConsoleKit uses a modular architecture for organizing functionality:

```go
// Custom command module
func AddMyFeature(cli *consolekit.CLI) func(cmd *cobra.Command) {
    return func(rootCmd *cobra.Command) {
        myCmd := &cobra.Command{
            Use:   "mycommand [args]",
            Short: "Description of my command",
            Run: func(cmd *cobra.Command, args []string) {
                cmd.Println("My custom command")
            },
        }
        rootCmd.AddCommand(myCmd)
    }
}

// Register the module
cli, err := consolekit.NewCLI("myapp", func(cli *consolekit.CLI) error {
    AddMyFeature(cli)
    return nil
})
```

### Built-in Modules

| Module | Commands | Description |
|--------|----------|-------------|
| **base** | `print`, `set`, `if`, `date`, `sleep`, `wait`, `repeat`, `http`, `cls`, `exit` | Core utility commands |
| **alias** | `alias`, `unalias`, `aliases` | Alias management with persistence |
| **history** | `history`, `history search`, `history clear` | Command history operations |
| **run** | `run`, `vs`, `spawn` | Script execution and background jobs |
| **exec** | `osexec` | Direct OS command execution |
| **misc** | `cat`, `grep`, `env` | File and environment utilities |

## 🔐 Security

> **⚠️ IMPORTANT SECURITY NOTICE**
>
> ConsoleKit is designed for **trusted environments only**. It provides extensive system access equivalent to shell access and should **never** be used in multi-tenant or untrusted environments.

### Suitable For ✅
- Local development tools
- Internal automation scripts
- Trusted administrator consoles
- Single-user applications
- Prototyping and testing

### Not Suitable For ❌
- Web-facing applications
- Multi-tenant systems
- Untrusted user environments
- Systems requiring command restrictions
- Compliance-restricted environments (without extensive hardening)

### Security Features

- ✅ **Recursion Protection** - Maximum execution depth limit (10 levels) prevents infinite loops
- ✅ **HTTP Timeouts** - 30-second timeout on HTTP requests prevents hanging
- ✅ **Proper Quote Handling** - Shellquote parsing prevents some injection vectors
- ✅ **Scoped Variables** - Script arguments are isolated per execution

### Security Considerations

- **File System Access** - Commands can read any file accessible to the process
- **OS Command Execution** - Full command execution with process permissions
- **Token Injection** - `@exec:` tokens allow arbitrary command execution
- **Background Processes** - Spawned processes may outlive the CLI session
- **History Storage** - Commands stored in plaintext in `~/.{appname}.history`

**See [SECURITY.md](SECURITY.md) for comprehensive security documentation, threat model, and deployment recommendations.**

## 📚 Documentation

| Document | Description |
|----------|-------------|
| **[CLAUDE.md](CLAUDE.md)** | Architecture guide and implementation details |
| **[SECURITY.md](SECURITY.md)** | Security considerations and deployment guidelines |
| **[REVIEW.md](REVIEW.md)** | Code review findings and fix status |
| **[GoDoc](https://pkg.go.dev/github.com/alexj212/consolekit)** | API reference and package documentation |

## 🛠️ Development

### Building

```bash
# Build the library
go build

# Run tests
go test ./...

# Run example application
cd examples/simple
go run main.go
```

### Project Structure

```
consolekit/
├── cli.go              # Core CLI implementation
├── alias.go            # Alias system
├── base.go             # Base commands
├── exec.go             # OS command execution
├── history.go          # History management
├── run.go              # Script execution
├── misc.go             # Utility commands
├── utils.go            # Helper functions
├── parser/             # Command parser
│   └── parser.go
├── safemap/            # Thread-safe map
│   └── safemap.go
└── examples/           # Example applications
    └── simple/
```

## 🤝 Contributing

Contributions are welcome! We appreciate:

- 🐛 **Bug reports** - Open an issue with reproduction steps
- 💡 **Feature requests** - Describe your use case and proposed solution
- 🔧 **Pull requests** - Fix bugs or add features (please discuss major changes first)
- 📖 **Documentation** - Improve docs, examples, or code comments
- 🧪 **Tests** - Add test coverage for existing or new functionality

### Contribution Guidelines

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with clear commit messages
4. Add or update tests as needed
5. Ensure `go test ./...` passes
6. Submit a pull request

## 📝 License

ConsoleKit is released under the [MIT License](LICENSE). See LICENSE file for details.

## 🙏 Acknowledgments

Built with excellent Go libraries:
- [spf13/cobra](https://github.com/spf13/cobra) - Command framework
- [reeflective/console](https://github.com/reeflective/console) - REPL interface with completion
- [kballard/go-shellquote](https://github.com/kballard/go-shellquote) - Shell-style quote parsing
- [fatih/color](https://github.com/fatih/color) - Colorized output
- [mattn/go-isatty](https://github.com/mattn/go-isatty) - TTY detection

---

<div align="center">

**[⬆ back to top](#consolekit)**

Made with ❤️ for building powerful CLI tools

</div>
