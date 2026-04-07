# CLAUDE.md

## Project Overview

ConsoleKit is a modular CLI library for Go with REPL support. Built on `spf13/cobra` (commands) and `reeflective/console` (REPL).

## Build & Test

```bash
go build                          # Build library
go test ./...                     # Run tests
cd examples/simple_console && go build   # Build example
```

Entry point: `handler.Run()` — runs CLI args if provided, otherwise starts REPL.

## Key Architecture

- **Three-layer design**: Command Execution → Transport Handlers (REPL/SSH/HTTP) → Display Adapters. See [ARCHITECTURE.md](ARCHITECTURE.md).
- **DisplayAdapter interface**: Default is `ReflectiveAdapter` (reeflective/console). Bubbletea adapter is in `examples/simple_bubbletea/` as a separate Go module.
- **Modular commands**: Apps pick command groups via `AddCoreCmds`, `AddStandardCmds`, etc. See [COMMAND_GROUPS.md](COMMAND_GROUPS.md).
- **Token replacement** (`ExpandCommand`): Aliases → Defaults → Custom VariableExpanders → Built-in patterns (`@env:`, `@exec:`, `@varname`).
- **Parser**: Uses `github.com/alexj212/console/parser` + `go-shellquote` for pipes (`|`), redirections (`>`), chaining (`;`).

## Critical Implementation Details

- **Flag reset pattern**: Commands MUST use `PostRun` hooks calling `ResetAllFlags` / `ResetHelpFlagRecursively` (utils.go). Required for REPL reuse.
- **Recursion protection**: `execDepth` counter, max 10 levels. Prevents `@exec:` / alias infinite loops.
- **`DisableFlagParsing: true`** on root command — parser handles parsing before Cobra.
- **Per-instance state**: Aliases in `CLI.aliases` SafeMap, not global. Each CLI instance is isolated.
- **`AddRun` takes `*embed.FS`** (pointer), not `embed.FS` (v0.8.0 breaking change).
- **REPL migration history**: `alexj212/console` → `c-bata/go-prompt` → `reeflective/console`.

## Adding New Commands

```go
func AddMyFeature(cli *CLI) func(cmd *cobra.Command) {
    return func(rootCmd *cobra.Command) {
        // Add commands to rootCmd
        // Use PostRun for flag reset if command has flags
    }
}
```

Register via `exec.AddCommands(AddMyFeature(exec))` in the customizer function.

## Security

Designed for trusted environments only. See [SECURITY.md](SECURITY.md).

## Reference Docs

- [ARCHITECTURE.md](ARCHITECTURE.md) — Three-layer architecture details
- [COMMAND_GROUPS.md](COMMAND_GROUPS.md) — All command groups and bundles
- [COMMANDS.md](COMMANDS.md) — Command reference
- [MCP_INTEGRATION.md](MCP_INTEGRATION.md) — MCP server integration
- [SECURITY.md](SECURITY.md) — Security model and threat analysis
- [SOCKET_INTEGRATION.md](SOCKET_INTEGRATION.md) — Socket transport for programmatic access
- [CLAUDE_REFERENCE.md](CLAUDE_REFERENCE.md) — Detailed feature documentation (transport handlers, subsystems, APIs)
