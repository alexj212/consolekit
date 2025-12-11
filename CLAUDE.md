# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ConsoleKit is a modular CLI library for building powerful console applications in Go with REPL (Read-Eval-Print Loop) support. It provides advanced features like command chaining (`;`), piping (`|`), file redirection (`>`), and modular command registration.

The library is built on top of `spf13/cobra` for command management and `reeflective/console` for REPL functionality with automatic completion, colorization, and arrow key support.

## Building and Testing

```bash
# Build the library
go build

# Run the example application
cd examples/simple
go run main.go

# Run tests (if available)
go test ./...
```

## Core Architecture

### CLI Initialization Flow

1. **CLI Creation** (`NewCLI` in cli.go): Creates the CLI instance with:
   - History file management (stored in user home directory as `.{appname}.history`)
   - Color support based on TTY detection
   - **Per-instance aliases** stored in CLI.aliases SafeMap (not global)
   - Token replacers and defaults initialization
   - **Recursion protection** with execDepth counter (maxExecDepth = 10)

2. **Command Registration** (`AddCommands` in cli.go): Uses customizer functions to add modular commands
   - Each module (alias, exec, history, run, base, misc) provides a function that accepts `*cobra.Command`
   - Commands are registered during CLI initialization via `rootInit` callbacks

3. **Command Execution** (`ExecuteLine` in cli.go):
   - **Increments recursion depth** and checks against maxExecDepth to prevent infinite loops
   - Performs token replacement (`ReplaceDefaults`) with alias and variable expansion
   - Parses commands using `github.com/alexj212/console/parser` with shellquote support
   - Executes through `executeCommands` with pipe support

### reeflective/console Integration

The REPL is powered by `reeflective/console` which provides:
- **Automatic completion** for commands, subcommands, and flags via Cobra integration
- **Pre-command hooks** for token replacement and alias expansion before execution
- **Post-command hooks** for history management
- **History Management**: File-based history with proper loading/saving
- **AppBlock** (cli.go): Creates and configures the console application with menu setup

### Token Replacement System

The token replacement system (cli.go `ReplaceDefaults`) supports:
- **Aliases**: Replaced first from per-instance `CLI.aliases` SafeMap
- **Environment variables**: `@env:VAR_NAME`
- **Command execution**: `@exec:command` - executes command and uses output (with recursion protection)
- **Default variables**: `@varname` - from CLI.Defaults SafeMap or scoped defs parameter
- **Custom replacers**: Via `CLI.TokenReplacers` slice

Execution order: Aliases → Defaults → Custom TokenReplacers → Built-in token patterns

**Security Note**: The `@exec:` token allows arbitrary command execution. Recursion is limited to 10 levels to prevent stack overflow attacks.

### Command Module System

Commands are organized into modular functions that return command registration functions:

- **base.go**: Core commands (`cls`, `exit`, `print`, `date`, `http`, `sleep`, `wait`, `repeat`, `waitfor`, `set`, `if`)
  - Note: The `check` command was removed due to uninitialized data dependency
- **alias.go**: Alias management system with file persistence (`~/.{appname}.aliases`)
  - Aliases are now per-instance, not global
- **history.go**: History commands (`list`, `search`, `clear`)
- **run.go**: Script execution system supporting embedded and external scripts
  - Script arguments use scoped defaults to prevent leakage
- **exec.go**: OS command execution with background support
  - Output suppression uses `io.Discard` instead of `nil`
- **misc.go**: Utility commands (`cat`, `grep`, `env`)

### Script Execution System (run.go)

Scripts can be:
- **Embedded**: Stored in `embed.FS`, referenced with `@filename`
- **External**: Read from filesystem with full path
- **Parameterized**: Arguments passed as `@arg0`, `@arg1`, etc. in **scoped defaults**
  - Script arguments are now isolated in a scoped SafeMap to prevent leakage
  - Each script execution gets its own argument namespace

Multi-line commands supported with backslash continuation (`\`).

**Security Warning**: Scripts and the `cat` command can read any file accessible to the process. See SECURITY.md for deployment considerations.

### SafeMap Utility (safemap/safemap.go)

Thread-safe generic map used for:
- **Per-instance aliases storage** (CLI.aliases)
- **Per-CLI default variables** (CLI.Defaults)
- **Scoped script arguments** (created per script execution)
- Provides `ForEach`, `SortedForEach`, `Get`, `Set`, `Delete` operations

## Key Design Patterns

### Modular Command Registration

Each feature area provides a registration function:
```go
func AddFeature(cli *CLI) func(cmd *cobra.Command) {
    return func(rootCmd *cobra.Command) {
        // Add commands to rootCmd
    }
}
```

This allows applications to selectively include features.

### Flag Reset Pattern

Commands use `PostRun` hooks to reset flags via `ResetAllFlags` and `ResetHelpFlagRecursively` (utils.go:8-62). This is critical for REPL operation where commands are reused across multiple invocations.

## Important Implementation Notes

### REPL Library Migration
The library was migrated from `github.com/alexj212/console` → `c-bata/go-prompt` → **`reeflective/console`**. The parser from console is still used for pipe/redirection support, but the REPL interface is now reeflective/console which provides automatic Cobra integration and completion.

### Command Parser Quote Handling
The parser now uses `github.com/kballard/go-shellquote` for proper quote and escape handling. This fixes issues where special characters (`|`, `>`, `;`) inside quoted strings were incorrectly treated as operators.

### Recursion Protection
`ExecuteLine` tracks recursion depth with `CLI.execDepth` counter. Maximum depth is set to 10 (configurable via `CLI.maxExecDepth`). This prevents infinite loops from circular `@exec:` references or aliases.

### Cobra Flag Parsing
`DisableFlagParsing: true` is set on the root command because the parser handles command-line parsing independently before Cobra execution.

### History Persistence
Command history is automatically saved to `~/.{appname}.history` in the user's home directory. History is loaded on startup and saved via post-command hooks.

**Security Note**: History file contains all commands in plaintext, including any sensitive data typed.

### Pipe and Redirection
The `github.com/alexj212/console/parser` package handles parsing of pipes (`|`) and redirections (`>`). The `executeCommands` function chains command execution through buffers.

### Color Support
Color output is disabled when `NO_COLOR` environment variable is set or when not running in a TTY. Use `CLI.InfoString` and `CLI.ErrorString` for colored output.

### Per-Instance State
Aliases are stored per-CLI instance in `CLI.aliases` (not global). Each CLI instance has isolated state, allowing multiple instances in the same process.

## Common Development Patterns

When adding new commands:
1. Create a new function following the pattern `func AddMyFeature(cli *CLI) func(cmd *cobra.Command)`
2. Define cobra commands within the returned function
3. Add flag reset in `PostRun` if the command uses flags
4. Register in your application's customizer function

When working with token replacement:
- Use `cli.ReplaceDefaults(cmd, defs, input)` to process tokens
- Add custom token handlers via `cli.TokenReplacers` slice
- Token names starting with `@` are reserved for the system

When implementing script commands:
- Use `LoadScript` to read and parse script files
- Call `cli.ExecuteLine` to execute individual commands with scoped defaults
- Create scoped SafeMap for script arguments to prevent leakage
- Handle multi-line scripts with `ReadLines` which processes backslash continuations

## Security Considerations

**ConsoleKit is designed for trusted environments and trusted users only.** It provides extensive system access equivalent to shell access.

### Key Security Principles

1. **Not Sandboxed**: Commands can read arbitrary files, execute OS commands, and access network
2. **Intended Use**: Local development tools, internal automation, trusted administrator consoles
3. **Not Suitable For**: Web-facing apps, multi-tenant systems, untrusted users

### Built-in Protections

- ✅ **Recursion Protection**: Max depth of 10 prevents infinite loop attacks
- ✅ **HTTP Timeout**: 30 second timeout on HTTP requests
- ✅ **Quote Handling**: Proper parsing of quoted strings prevents some injection vectors
- ✅ **Scoped Script Args**: Script arguments don't leak between executions

### Known Security Considerations

See **SECURITY.md** for comprehensive documentation on:
- File system access (cat, LoadScript can read any file)
- OS command execution (osexec runs arbitrary commands)
- Token injection risks (@exec: allows command execution)
- Hardcoded credentials in source code
- HTTP SSRF potential
- Background process management
- History file plaintext storage

### Deployment Recommendations

For production/multi-user environments, implement additional controls:
- Run with minimal required permissions
- Use Docker/containers with resource limits
- Implement command allowlisting
- Add audit logging
- Block SSRF targets for HTTP command
- Use seccomp/AppArmor/SELinux profiles

**See SECURITY.md for detailed security documentation and threat model.**
