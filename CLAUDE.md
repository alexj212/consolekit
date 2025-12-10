# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ConsoleKit is a modular CLI library for building powerful console applications in Go with REPL (Read-Eval-Print Loop) support. It provides advanced features like command chaining (`;`), piping (`|`), file redirection (`>`), and modular command registration.

The library is built on top of `spf13/cobra` for command management and `c-bata/go-prompt` for REPL functionality with colorization and arrow key support.

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

1. **CLI Creation** (`NewCLI` in cli.go:41): Creates the CLI instance with:
   - History file management (stored in user home directory as `.{appname}.history`)
   - Color support based on TTY detection
   - go-prompt executor and completer setup
   - Token replacers and defaults initialization

2. **Command Registration** (`AddCommands` in cli.go:80): Uses customizer functions to add modular commands
   - Each module (alias, exec, history, run, base) provides a function that accepts `*cobra.Command`
   - Commands are registered during CLI initialization via `rootInit` callbacks

3. **Command Execution** (`ExecuteLine` in cli.go:140):
   - Performs token replacement (`ReplaceDefaults`)
   - Parses commands using `github.com/alexj212/console/parser`
   - Executes through `executeCommands` with pipe support

### go-prompt Integration

The REPL is powered by `c-bata/go-prompt` which provides:
- **Executor Callback** (cli.go:207): Handles command execution when user submits input
- **Completer Callback** (cli.go:241): Provides auto-completion suggestions from cobra commands
- **History Management** (cli.go:268-307): Simple file-based history storage
- **AppBlock** (cli.go:310): Creates and configures the prompt with colors, key bindings, and options

### Token Replacement System

The token replacement system (cli.go:84-138) supports:
- **Aliases**: Replaced first from the global `aliases` SafeMap
- **Environment variables**: `@env:VAR_NAME`
- **Command execution**: `@exec:command` - executes command and uses output
- **Default variables**: `@varname` - from CLI.Defaults SafeMap
- **Custom replacers**: Via `CLI.TokenReplacers` slice

Execution order: Aliases → Defaults → Custom TokenReplacers → Built-in token patterns

### Command Module System

Commands are organized into modular functions that return command registration functions:

- **base.go**: Core commands (`cls`, `exit`, `print`, `date`, `http`, `sleep`, `wait`, `repeat`, `check`, `waitfor`, `set`, `if`)
- **alias.go**: Alias management system with file persistence (`~/.{appname}.aliases`)
- **history.go**: History commands (`list`, `search`, `clear`)
- **run.go**: Script execution system supporting embedded and external scripts
- **exec.go**: OS command execution with background support
- **misc.go**: Utility commands (`cat`, `grep`, `env`)

### Script Execution System (run.go)

Scripts can be:
- **Embedded**: Stored in `embed.FS`, referenced with `@filename`
- **External**: Read from filesystem with full path
- **Parameterized**: Arguments passed as `@arg0`, `@arg1`, etc. in script

Multi-line commands supported with backslash continuation (`\`).

### SafeMap Utility (safemap/safemap.go)

Thread-safe generic map used for:
- Global aliases storage
- Per-CLI default variables
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

### go-prompt vs console
The library was migrated from `github.com/alexj212/console` to `c-bata/go-prompt`. The parser from console is still used for pipe/redirection support, but the REPL interface is now go-prompt.

### Cobra Flag Parsing
`DisableFlagParsing: true` is set on the root command (cli.go:192) because the parser handles command-line parsing independently before Cobra execution.

### History Persistence
Command history is automatically saved to `~/.{appname}.history` in the user's home directory. History is loaded on startup (cli.go:269) and saved after each command (cli.go:291).

### Pipe and Redirection
The `github.com/alexj212/console/parser` package handles parsing of pipes (`|`) and redirections (`>`). The `executeCommands` function (cli.go:153-184) chains command execution through buffers.

### Color Support
Color output is disabled when `NO_COLOR` environment variable is set or when not running in a TTY (cli.go:49-56). Use `CLI.InfoString` and `CLI.ErrorString` for colored output.

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
- Call `cli.ExecuteLine` to execute individual commands
- Handle multi-line scripts with `ReadLines` which processes backslash continuations
