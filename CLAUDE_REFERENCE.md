# ConsoleKit Detailed Reference

This file contains detailed feature documentation for ConsoleKit subsystems. For quick orientation, see [CLAUDE.md](CLAUDE.md).

## CLI Initialization Flow

1. **`NewCommandExecutor`** (cli.go): Creates CLI with history file (`~/.{appname}.history`), color/TTY detection, per-instance aliases SafeMap, token replacers, recursion protection (`execDepth`), eager console creation and configuration, pre-command hooks.
2. **`AddCommands`** (cli.go): Modules provide `func(cmd *cobra.Command)` customizer functions. Commands added to console menu after registration.
3. **`Execute`** (cli.go): Increments recursion depth → `ExpandCommand` for token/alias expansion → parse with `console/parser` + shellquote → `executeCommands` with pipe support.

## Token Replacement System

`ExpandCommand` processing order:
1. **Aliases** from `CLI.aliases` SafeMap
2. **Default variables** from `CLI.Variables` SafeMap or scoped scope parameter
3. **Custom VariableExpanders** slice
4. **Built-in patterns**: `@env:VAR_NAME`, `@exec:command` (recursion-limited), `@varname`

## Transport Handlers

### HTTP/WebSocket Server (handler_http.go)

- `NewHTTPHandler(executor, ":8080", "user", "pass")`
- xterm.js frontend with WebSocket REPL
- Session-based auth, configurable `IdleTimeout`, `MaxSessionTime`, `MaxConnections`
- `InitialHistory` for pre-populated command history
- Endpoints: `/` (status), `/admin` (terminal), `/login`, `/logout`, `/repl` (WebSocket)
- Session variables: `@http:user`, `@http:session_id`
- WebSocket JSON format: `{type: "input|output|error", message: "..."}`

### SSH Server (handler_ssh.go)

- `NewSSHHandler(executor, ":2222", hostKey)`
- Password and public key auth via `SetAuthConfig`
- Full PTY/readline support, arrow keys, tab completion
- `InitialHistory`, per-session history (in-memory)
- Session variables: `@ssh:user`, `@ssh:remote_ip`, `@ssh:session_id`
- Built-in prompt functions: `DefaultPrompt`, `DetailedPrompt`, `MinimalPrompt`, `ColorPrompt`
- Levenshtein-based command suggestions (max distance 3)

### MCP Server (mcp.go + mcpcmds.go)

- Exposes CLI commands as MCP tools via JSON-RPC 2.0 over stdio
- Cobra flags → JSON Schema parameters, positional args → `_args`
- Commands: `mcp start`, `mcp info`, `mcp list-tools`
- See [MCP_INTEGRATION.md](MCP_INTEGRATION.md)

## Command Subsystems

### Script Execution (run.go)

- Embedded scripts (`@filename` from `embed.FS`), external scripts (filesystem path)
- Arguments as `@arg0`, `@arg1` in scoped SafeMap (isolated per execution)
- Multi-line with backslash continuation
- Registration: `AddRun(exec, &Scripts)` or `AddRun(exec, nil)` for external only

### Job Management (jobs.go + jobcmds.go)

- `JobManager`: thread-safe tracking with auto-incrementing IDs, PID tracking, status monitoring, output capture, context cancellation
- Commands: `jobs`, `job [id]`, `job [id] logs/kill/wait`, `killall`, `jobclean`
- Background jobs from `osexec --background` and `run --spawn` are auto-tracked

### Variables (varcmds.go)

- `let name=value`, `let result=$(command)`, arithmetic with `inc`/`dec`
- `vars --export` (shell script), `vars --json`
- Stored with `@` prefix in `CLI.Variables` SafeMap

### Configuration (config.go + configcmds.go)

- TOML file at `~/.{appname}/config.toml`
- Sections: `[settings]`, `[aliases]`, `[variables]`, `[hooks]`, `[logging]`
- Commands: `config get/set/edit/reload/show/path/save`
- Auto-loaded on CLI init

### Logging & Audit (logging.go + logcmds.go)

- Log file: `~/.{appname}/audit.log` (configurable)
- Timestamps, duration, success/failure, user tracking, output capture, log rotation
- Commands: `log enable/disable/status/show/clear/export/load/config`
- Only logs top-level commands (not recursive)

### Interactive Prompts (prompt.go + promptcmds.go)

- `cli.Confirm()`, `cli.Prompt()`, `cli.PromptPassword()`, `cli.Select()`, `cli.MultiSelect()`, `cli.ConfirmDestructive()`, `cli.PromptInteger()`
- Helper flags: `AddYesFlag()`, `AddDryRunFlag()`, `ConfirmOrSkip()`, `ConfirmDestructiveOrSkip()`

### Templates (template.go + templatecmds.go)

- Go `text/template` syntax, embedded or filesystem templates
- Directory: `~/.{appname}/templates/`
- Commands: `template list/show/exec/render/create/delete/clear-cache`
- Variables via `key=value` args

### Data Manipulation (datamanipcmds.go)

- `json parse/get/validate` — dot notation path traversal
- `yaml parse/to-json/from-json`
- `csv parse/to-json` — auto header detection
- All commands support stdin (pipe-friendly)

### Control Flow (controlflowcmds.go)

- `case`, `while` (1000 iteration safety limit), `for` (scoped variables), `test` (numeric/string comparisons)

### Output Formatting (formatcmds.go)

- `table` (delimited → aligned), `highlight` (regex + color), `page` (pagination), `column` (columnize)

### Scheduling (schedulecmds.go)

- `schedule at/in/every` — one-time (`time.Timer`) or repeating (`time.Ticker`)
- `schedule list/cancel/pause/resume`

### Notifications (notify.go + notifycmds.go)

- Cross-platform desktop notifications (Linux/macOS/Windows)
- Webhook support with JSON payload
- `notify send`, `notify config`

### Clipboard (clipboardcmds.go)

- `clip` (copy) / `paste` — platform-aware (xclip/pbcopy/clip.exe)

## SafeMap (safemap/safemap.go)

Thread-safe generic map. Used for aliases, variables, scoped script args. Provides `ForEach`, `SortedForEach`, `Get`, `Set`, `Delete`.

## File Organization

| File | Purpose |
|------|---------|
| cli.go | Core CLI, Execute, ExpandCommand, NewCommandExecutor |
| base.go | Core commands (cls, exit, print, date), repeat, set, if |
| alias.go | Alias management with file persistence |
| history.go | History commands |
| run.go | Script execution |
| exec.go | OS command execution with background support |
| misc.go | Utility commands (cat, grep, env) |
| jobcmds.go | Job management commands |
| varcmds.go | Enhanced variable system |
| configcmds.go | Configuration management |
| logcmds.go | Logging/audit commands |
| promptcmds.go | Interactive prompt commands |
| templatecmds.go | Template commands |
| datamanipcmds.go | JSON/YAML/CSV commands |
| controlflowcmds.go | while, for, case, test |
| formatcmds.go | table, column, highlight, page |
| schedulecmds.go | Task scheduling commands |
| notifycmds.go | Notification commands |
| clipboardcmds.go | Clipboard commands |
| pipelinecmds.go | tee command |
| watchcmds.go | watch command |
| mcp.go + mcpcmds.go | MCP server |
| handler_http.go | HTTP/WebSocket handler |
| handler_ssh.go | SSH handler |
| display_reeflective.go | Default REPL adapter |
| utils.go | ResetAllFlags, ResetHelpFlagRecursively |
| safemap/safemap.go | Thread-safe generic map |
