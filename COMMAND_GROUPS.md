# ConsoleKit Command Groups

ConsoleKit provides a modular command system that allows you to selectively include only the features you need. This document describes the available command groups and how to use them.

## Overview

Instead of including all commands with `AddBuiltinCommands()`, you can now pick and choose specific command groups to build a custom CLI tailored to your needs.

## Quick Start

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    // Option 1: Use convenience bundles
    exec.AddCommands(consolekit.AddMinimalCmds(exec))  // Core + variables + scripting

    // Option 2: Pick individual groups
    exec.AddCommands(consolekit.AddCoreCmds(exec))
    exec.AddCommands(consolekit.AddVariableCmds(exec))
    exec.AddCommands(consolekit.AddNetworkCmds(exec))

    return nil
}
```

## Convenience Bundles

These pre-configured bundles provide common command combinations:

### `AddAllCmds(exec)`
Includes **all available commands**. Equivalent to the old `AddBuiltinCommands()` behavior.

**Use case:** Full-featured REPLs, comprehensive CLI tools

### `AddStandardCmds(exec)`
Includes **recommended default commands** (excludes advanced integrations like MCP, notifications).

**Includes:**
- Core, Variables, Aliases, History, Config
- Scripting, Control Flow
- OS Execution, Jobs
- File Utils, Data Manipulation
- Formatting, Pipelines
- Network, Time

**Use case:** General-purpose CLIs, development tools

### `AddMinimalCmds(exec)`
Includes only **essential commands** for basic CLI operation.

**Includes:**
- Core (cls, exit, print, date)
- Variables (let, unset, vars, inc, dec)
- Scripting (run - requires AddRun)
- Control Flow (if, repeat, while, for, case, test)

**Use case:** Lightweight CLIs, embedded scripts, minimal footprint applications

### `AddDeveloperCmds(exec)`
Includes **standard commands plus developer features**.

**Adds to Standard:**
- Schedule commands
- Template system
- Interactive prompts
- Logging & audit trails
- Clipboard integration

**Use case:** Developer tools, automation scripts, interactive debugging

### `AddAutomationCmds(exec)`
Includes **commands optimized for automation** (excludes interactive features).

**Includes:**
- Core, Variables, Config
- Scripting, Control Flow, Templates
- OS Execution, Jobs, Scheduling
- Data manipulation, Formatting, Pipelines
- Network, Time
- Logging (for audit trails)

**Use case:** Automation scripts, CI/CD tools, background services

## Individual Command Groups

### Core Commands

#### `AddCoreCmds(exec)`
Essential commands for basic CLI operation.

**Commands:** `cls`, `exit`, `print`, `date`

**Required for:** All CLIs (provides basic I/O and exit functionality)

---

### State Management

#### `AddVariableCmds(exec)`
Variable management and manipulation.

**Commands:** `let`, `unset`, `vars`, `inc`, `dec`

**Use case:** Scripts that need variable storage, counters, state tracking

#### `AddAliasCmds(exec)`
Command alias management.

**Commands:** `alias add`, `alias list`, `alias remove`, `alias clear`

**Use case:** Power users who want shortcuts, customization

#### `AddHistoryCmds(exec)`
Command history and bookmarks.

**Commands:** `history list`, `history search`, `history clear`, `history bookmark`, `history replay`, `history stats`

**Use case:** Interactive REPLs, debugging, command replay

#### `AddConfigCmds(exec)`
Configuration file management.

**Commands:** `config get`, `config set`, `config edit`, `config reload`, `config show`, `config path`, `config save`

**Use case:** Applications with persistent configuration

---

### Scripting & Control Flow

#### `AddScriptingCmds(exec)`
Script execution support.

**Note:** This is a placeholder. Use `AddRun(exec, scripts *embed.FS)` directly to enable script execution. Pass `&scripts` for embedded scripts or `nil` for external-only scripts.

**Commands:** `run`

**Examples:**
```go
// With embedded scripts
//go:embed *.run
var Scripts embed.FS
exec.AddCommands(consolekit.AddRun(exec, &Scripts))

// External scripts only
exec.AddCommands(consolekit.AddRun(exec, nil))
```

**Use case:** Applications with embedded or external scripts

#### `AddControlFlowCmds(exec)`
Control flow commands for scripting.

**Commands:** `if`, `repeat`, `while`, `for`, `case`, `test`

**Use case:** Complex scripts, automation, conditional logic

---

### OS Integration

#### `AddOSExecCmds(exec)`
Operating system command execution.

**Commands:** `osexec`

**Use case:** Wrappers around system commands, automation scripts

**Security note:** Allows arbitrary OS command execution - use with caution

#### `AddJobCmds(exec)`
Background job management.

**Commands:** `jobs`, `job`, `killall`, `jobclean`, `spawn`

**Use case:** Long-running operations, parallel execution

#### `AddScheduleCmds(exec)`
Task scheduling and timed execution.

**Commands:** `schedule at`, `schedule in`, `schedule every`, `schedule list`, `schedule cancel`, `schedule pause`, `schedule resume`

**Use case:** Cron-like scheduling, delayed execution, periodic tasks

---

### File & Data

#### `AddFileUtilCmds(exec)`
File utility commands.

**Commands:** `cat`, `grep`, `env`

**Use case:** File inspection, searching, environment access

#### `AddDataManipulationCmds(exec)`
JSON, YAML, and CSV parsing and conversion.

**Commands:** `json parse`, `json get`, `json validate`, `yaml parse`, `yaml to-json`, `yaml from-json`, `csv parse`, `csv to-json`

**Use case:** API clients, data transformation, config file processing

---

### Output & Formatting

#### `AddFormatCmds(exec)`
Output formatting and presentation.

**Commands:** `table`, `column`, `highlight`, `page`

**Use case:** Pretty-printing data, reports, log analysis

#### `AddPipelineCmds(exec)`
Pipeline utilities.

**Commands:** `tee`

**Use case:** Command output duplication, multi-destination output

#### `AddClipboardCmds(exec)`
System clipboard integration.

**Commands:** `clip`, `paste`

**Use case:** Copy command output, paste clipboard content

**Platform:** Linux (xclip/xsel), macOS (pbcopy/pbpaste), Windows (clip.exe/PowerShell)

---

### Advanced Features

#### `AddTemplateCmds(exec)`
Template system for parameterized scripts.

**Commands:** `template list`, `template show`, `template exec`, `template render`, `template create`, `template delete`, `template clear-cache`

**Use case:** Code generation, parameterized deployments, reusable scripts

#### `AddInteractiveCmds(exec)`
Interactive prompts for user input.

**Commands:** `prompt-demo`, `confirm`, `input`, `select`, `multiselect`

**Use case:** Interactive CLIs, user confirmations, menu systems

#### `AddLoggingCmds(exec)`
Command execution logging and audit trails.

**Commands:** `log enable`, `log disable`, `log status`, `log show`, `log clear`, `log export`, `log load`, `log config`

**Use case:** Compliance, debugging, command history analysis

**Security note:** Logs may contain sensitive command arguments

---

### Integrations

#### `AddNetworkCmds(exec)`
Network-related commands.

**Commands:** `http`

**Use case:** API testing, web scraping, health checks

#### `AddTimeCmds(exec)`
Time-related commands and delays.

**Commands:** `sleep`, `wait`, `waitfor`, `watch`

**Use case:** Timed operations, polling, monitoring

#### `AddNotificationCmds(exec)`
Desktop notifications and webhooks.

**Commands:** `notify send`, `notify config`

**Use case:** Alert on completion, external integrations

**Platform:** Linux (notify-send), macOS (osascript), Windows (PowerShell)

#### `AddMCPCmds(exec)`
Model Context Protocol server integration.

**Commands:** `mcp start`, `mcp info`, `mcp list-tools`

**Use case:** Claude Desktop integration, AI tool access

---

### Utilities

#### `AddUtilityCmds(exec)`
Miscellaneous utility commands.

**Commands:** Various utilities from `utilcmds.go`

**Use case:** Helper functions, utilities

---

## Usage Examples

### Minimal Automation Script

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    exec.AddCommands(consolekit.AddCoreCmds(exec))
    exec.AddCommands(consolekit.AddVariableCmds(exec))
    exec.AddCommands(consolekit.AddControlFlowCmds(exec))
    exec.AddCommands(consolekit.AddOSExecCmds(exec))
    return nil
}
```

### API Testing Tool

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    exec.AddCommands(consolekit.AddCoreCmds(exec))
    exec.AddCommands(consolekit.AddVariableCmds(exec))
    exec.AddCommands(consolekit.AddNetworkCmds(exec))
    exec.AddCommands(consolekit.AddDataManipulationCmds(exec))
    exec.AddCommands(consolekit.AddFormatCmds(exec))
    exec.AddCommands(consolekit.AddHistoryCmds(exec))
    return nil
}
```

### Full Developer REPL

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    exec.AddCommands(consolekit.AddDeveloperCmds(exec))
    exec.AddCommands(consolekit.AddRun(exec, scripts))
    return nil
}
```

### Security-Focused CLI (No OS Execution)

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    exec.AddCommands(consolekit.AddCoreCmds(exec))
    exec.AddCommands(consolekit.AddVariableCmds(exec))
    exec.AddCommands(consolekit.AddAliasCmds(exec))
    exec.AddCommands(consolekit.AddHistoryCmds(exec))
    exec.AddCommands(consolekit.AddConfigCmds(exec))
    exec.AddCommands(consolekit.AddDataManipulationCmds(exec))
    exec.AddCommands(consolekit.AddFormatCmds(exec))
    // Deliberately omit: AddOSExecCmds, AddJobCmds, AddScheduleCmds
    return nil
}
```

## Backward Compatibility

### v0.8.0: AddBuiltinCommands

The old `AddBuiltinCommands()` method still works and is equivalent to `AddAllCmds()`:

```go
// Old way (still supported)
exec.AddBuiltinCommands()

// New way (identical behavior)
exec.AddCommands(consolekit.AddAllCmds(exec))
```

### v0.8.0: embed.FS Pointer Change

`AddRun` now requires a pointer to `embed.FS`:

```go
// Before v0.8.0
//go:embed *.run
var Scripts embed.FS
exec.AddCommands(consolekit.AddRun(exec, Scripts))  // ❌ No longer works

// After v0.8.0
//go:embed *.run
var Scripts embed.FS
exec.AddCommands(consolekit.AddRun(exec, &Scripts))  // ✅ Add &

// New capability: No embedded scripts
exec.AddCommands(consolekit.AddRun(exec, nil))      // ✅ Pass nil
```

## See Also

- [CLAUDE.md](CLAUDE.md) - Complete project documentation
- [examples/minimal_cli](examples/minimal_cli/) - Example of selective command inclusion
- [examples/simple_console](examples/simple_console/) - Example using all commands
- [SECURITY.md](SECURITY.md) - Security considerations for command groups
