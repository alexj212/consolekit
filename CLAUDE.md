# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ConsoleKit is a modular CLI library for building powerful console applications in Go with REPL (Read-Eval-Print Loop) support. It provides advanced features like command chaining (`;`), piping (`|`), file redirection (`>`), and modular command registration.

The library is built on top of `spf13/cobra` for command management and `reeflective/console` for REPL functionality with automatic completion, colorization, and arrow key support.

## Building and Testing

```bash
# Build the library
go build

# Run the example applications
cd examples/simple_console
go build
./simple_console

# Run minimal CLI example (showcases modular command selection)
cd examples/minimal_cli
go build
./minimal_cli

# Run in REPL mode (no arguments)
./simple_console

# Run a command directly (command-line mode)
./simple_console mcp info
./simple_console version
./simple_console print "Hello World"

# Start MCP server for external integration
./simple_console mcp start

# Run tests (if available)
go test ./...
```

**Entry Point**: Applications should use `handler.Run()` as the entry point, which automatically:
- Executes command-line arguments if provided (e.g., `./app mcp start`)
- Starts the REPL if no arguments are provided (e.g., `./app`)

## Core Architecture

### CLI Initialization Flow

1. **CLI Creation** (`NewCommandExecutor` in cli.go): Creates the CLI instance with:
   - History file management (stored in user home directory as `.{appname}.history`)
   - Color support based on TTY detection
   - **Per-instance aliases** stored in CLI.aliases SafeMap (not global)
   - Token replacers and defaults initialization
   - **Recursion protection** with execDepth counter (maxExecDepth = 10)
   - **Console creation** (eager initialization) - console.Console is created immediately
   - **Console configuration** - menus, history, prompts, and hooks are set up
   - **Pre-command hooks** for token replacement and alias expansion

2. **Command Registration** (`AddCommands` in cli.go): Uses customizer functions to add modular commands
   - Each module (alias, exec, history, run, base, misc) provides a function that accepts `*cobra.Command`
   - Commands are registered during CLI initialization via `rootInit` callbacks
   - Commands are added to console menu after customizer completes

3. **Command Execution** (`Execute` in cli.go):
   - **Increments recursion depth** and checks against maxExecDepth to prevent infinite loops
   - Performs token replacement (`ExpandCommand`) with alias and variable expansion
   - Parses commands using `github.com/alexj212/console/parser` with shellquote support
   - Executes through `executeCommands` with pipe support

### Display Adapter Architecture

ConsoleKit uses a **DisplayAdapter** interface to abstract the REPL/display layer from the command execution engine:

- **ReflectiveAdapter** (display_reeflective.go) - **Default adapter**, uses `reeflective/console`
  - Full readline support with tab completion
  - Command history with arrow key navigation
  - Cobra integration for automatic completion
  - Production-ready, recommended for most applications

- **BubbletteaAdapter** - **Optional adapter** (in `examples/simple_bubbletea/`)
  - Beautiful TUI using `charmbracelet/bubbletea` and `lipgloss`
  - Styled terminal interface with custom layouts
  - **Separate Go module** to avoid pulling bubbletea deps into core library
  - See `examples/simple_bubbletea/` for implementation and usage

**Key Points:**
- Core library only depends on `reeflective/console` (default adapter)
- Bubbletea is optional and lives in its own module (`examples/simple_bubbletea/go.mod`)
- Applications can implement custom adapters by implementing `DisplayAdapter` interface
- Use `handler.SetDisplayAdapter()` to switch adapters at runtime

### reeflective/console Integration (Default)

The default REPL is powered by `reeflective/console` which provides:
- **Automatic completion** for commands, subcommands, and flags via Cobra integration
- **Pre-command hooks** for token replacement and alias expansion before execution
- **Post-command hooks** for history management
- **History Management**: File-based history with proper loading/saving
- **Console creation** happens in `NewCommandExecutor()` with eager initialization
- **AppBlock** (cli.go): Starts the REPL (console is already created and configured)
- **Console() method**: Always returns a valid `*console.Console` instance (created during NewCommandExecutor)
- **SetPrompt() method**: Can be called anytime after NewCommandExecutor() to update the prompt function

### Token Replacement System

The token replacement system (cli.go `ExpandCommand`) supports:
- **Aliases**: Replaced first from per-instance `CLI.aliases` SafeMap
- **Environment variables**: `@env:VAR_NAME`
- **Command execution**: `@exec:command` - executes command and uses output (with recursion protection)
- **Default variables**: `@varname` - from CLI.Variables SafeMap or scoped scope parameter
- **Custom replacers**: Via `CLI.VariableExpanders` slice

Execution order: Aliases → Defaults → Custom VariableExpanders → Built-in token patterns

**Security Note**: The `@exec:` token allows arbitrary command execution. Recursion is limited to 10 levels to prevent stack overflow attacks.

### Modular Command System

**Phase 1 Feature**: ConsoleKit provides a modular command selection system, allowing applications to include only the commands they need.

**Documentation**: See [COMMAND_GROUPS.md](COMMAND_GROUPS.md) for complete documentation.

#### Convenience Bundles

- `AddAllCmds(exec)` - All commands (equivalent to `AddBuiltinCommands()`)
- `AddStandardCmds(exec)` - Recommended defaults (excludes advanced integrations)
- `AddMinimalCmds(exec)` - Core + variables + scripting only
- `AddDeveloperCmds(exec)` - Standard + developer features (templates, prompts, logging)
- `AddAutomationCmds(exec)` - Optimized for automation (no interactive features)

#### Individual Command Groups

Commands are organized into logical groups that can be included selectively:

- **Core**: `AddCoreCmds` - cls, exit, print, date
- **Variables**: `AddVariableCmds` - let, unset, vars, inc, dec
- **Aliases**: `AddAliasCmds` - alias management
- **History**: `AddHistoryCmds` - history and bookmarks
- **Config**: `AddConfigCmds` - configuration management
- **Scripting**: `AddScriptingCmds` / `AddRun` - script execution
- **Control Flow**: `AddControlFlowCmds` - if, repeat, while, for, case, test
- **OS Exec**: `AddOSExecCmds` - osexec
- **Jobs**: `AddJobCmds` - background job management
- **Schedule**: `AddScheduleCmds` - task scheduling
- **File Utils**: `AddFileUtilCmds` - cat, grep, env
- **Data**: `AddDataManipulationCmds` - json, yaml, csv
- **Format**: `AddFormatCmds` - table, column, highlight, page
- **Pipeline**: `AddPipelineCmds` - tee
- **Clipboard**: `AddClipboardCmds` - clip, paste
- **Templates**: `AddTemplateCmds` - template system
- **Interactive**: `AddInteractiveCmds` - prompts
- **Logging**: `AddLoggingCmds` - audit logging
- **Network**: `AddNetworkCmds` - http
- **Time**: `AddTimeCmds` - sleep, wait, waitfor, watch
- **Notifications**: `AddNotificationCmds` - notify
- **MCP**: `AddMCPCmds` - MCP integration

#### Usage Example

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    // Option 1: Use a convenience bundle
    exec.AddCommands(consolekit.AddStandardCmds(exec))

    // Option 2: Pick specific groups
    exec.AddCommands(consolekit.AddCoreCmds(exec))
    exec.AddCommands(consolekit.AddVariableCmds(exec))
    exec.AddCommands(consolekit.AddNetworkCmds(exec))

    return nil
}
```

See `examples/minimal_cli` for a complete example of selective command inclusion.

### Command Module Files

Commands are organized into modular files that provide registration functions:

- **base.go**: Core commands (`cls`, `exit`, `print`, `date`), network (`http`), time (`sleep`, `wait`, `waitfor`), and basic control flow (`repeat`, `set`, `if`)
  - Note: The `check` command was removed due to uninitialized data dependency
- **alias.go**: Alias management system with file persistence (`~/.{appname}.aliases`)
  - Aliases are now per-instance, not global
- **history.go**: History commands (`list`, `search`, `clear`)
- **run.go**: Script execution system supporting embedded and external scripts
  - Script arguments use scoped defaults to prevent leakage
- **exec.go**: OS command execution with background support
  - Output suppression uses `io.Discard` instead of `nil`
  - **Background jobs are tracked** in JobManager with PID and output capture
- **misc.go**: Utility commands (`cat`, `grep`, `env`)
- **jobcmds.go**: Job management commands (`jobs`, `job`, `killall`, `jobclean`)
  - Track background processes from `osexec --background` and `spawn` commands
  - View job status, logs, kill jobs, wait for completion
- **varcmds.go**: Enhanced variable system (`let`, `unset`, `vars`, `inc`, `dec`)
  - Command substitution with `$(...)` syntax
  - Arithmetic operations support
  - Export to shell script or JSON format
- **configcmds.go**: Configuration management (`config get/set/edit/reload/show/path/save`)
  - TOML-based configuration file
  - Settings, aliases, variables, hooks, and logging configuration
- **logcmds.go**: Logging and audit trail commands (`log enable/disable/status/show/clear/export/load/config`)
  - Command execution logging with timestamps, duration, and success/failure tracking
  - Search and filter logs by command text, date, or status
  - Export logs to JSON format
  - Configurable via TOML configuration file

### Script Execution System (run.go)

**v0.8.0 Breaking Change:** `AddRun` now accepts `*embed.FS` (pointer) instead of `embed.FS` (by value).

Scripts can be:
- **Embedded**: Stored in `embed.FS`, referenced with `@filename`
- **External**: Read from filesystem with full path
- **Parameterized**: Arguments passed as `@arg0`, `@arg1`, etc. in **scoped defaults**
  - Script arguments are now isolated in a scoped SafeMap to prevent leakage
  - Each script execution gets its own argument namespace

Multi-line commands supported with backslash continuation (`\`).

**Registration:**
```go
// With embedded scripts
//go:embed *.run
var Scripts embed.FS
exec.AddCommands(consolekit.AddRun(exec, &Scripts))  // Note: pointer

// External scripts only (v0.8.0+)
exec.AddCommands(consolekit.AddRun(exec, nil))
```

**Security Warning**: Scripts and the `cat` command can read any file accessible to the process. See SECURITY.md for deployment considerations.

### SafeMap Utility (safemap/safemap.go)

Thread-safe generic map used for:
- **Per-instance aliases storage** (CLI.aliases)
- **Per-CLI default variables** (CLI.Variables)
- **Scoped script arguments** (created per script execution)
- Provides `ForEach`, `SortedForEach`, `Get`, `Set`, `Delete` operations

### Job Management System (jobs.go)

**Phase 1 Feature**: Comprehensive background job tracking system

- **JobManager**: Thread-safe job tracking with:
  - Auto-incrementing job IDs
  - PID tracking
  - Status monitoring (running, completed, failed, killed)
  - Output capture in buffers
  - Start/end time tracking
  - Context-based cancellation

- **Job Commands**:
  - `jobs` - List all jobs (with `-v` for verbose, `-a` for all)
  - `job [id]` - Show job details
  - `job [id] logs` - View job output
  - `job [id] kill` - Terminate a job
  - `job [id] wait` - Wait for completion
  - `killall` - Kill all running jobs
  - `jobclean` - Remove completed jobs

- **Integration**: Background jobs from `osexec --background` and `run --spawn` are automatically tracked

### Enhanced Variables System (cmds/varcmds.go)

**Phase 1 Feature**: Advanced variable management beyond simple `set` command

- **let command**: Set variables with enhanced features
  - Simple assignment: `let name=value`
  - Command substitution: `let result=$(print hello)`
  - Numeric values for arithmetic operations

- **Variable Operations**:
  - `unset [name]` - Remove variables
  - `vars` - List all variables (pretty print)
  - `vars --export` - Export as shell script
  - `vars --json` - Export as JSON
  - `inc [name] [amount]` - Increment numeric variable (default +1)
  - `dec [name] [amount]` - Decrement numeric variable (default -1)

- **Storage**: Variables stored with `@` prefix in `CLI.Variables` SafeMap

### Configuration System (config.go + configcmds.go)

**Phase 1 Feature**: TOML-based configuration file management

- **Config Location**: `~/.{appname}/config.toml`

- **Config Sections**:
  - `[settings]` - History size, prompt, color, pager
  - `[aliases]` - Persistent command aliases
  - `[variables]` - Persistent variables
  - `[hooks]` - Lifecycle hooks (on_startup, on_exit, before_command, after_command)
  - `[logging]` - Audit logging configuration

- **Config Commands**:
  - `config get [key]` - Retrieve config value (e.g., `config get settings.history_size`)
  - `config set [key] [val]` - Set and save config value
  - `config edit` - Open config in $EDITOR
  - `config reload` - Reload from file
  - `config show` - Display all configuration
  - `config path` - Show config file location
  - `config save` - Save current state to file

- **Auto-loading**: Configuration automatically loaded on CLI initialization

### Logging & Audit Trail System (logging.go + logcmds.go)

**Phase 2 Feature**: Command execution logging for debugging and compliance

- **Log Location**: `~/.{appname}/audit.log` (configurable)

- **Features**:
  - **Automatic logging** of all command executions with timestamps
  - **Duration tracking** for performance analysis
  - **Success/failure tracking** with error messages
  - **User tracking** for multi-user environments
  - **Output capture** (optional, configurable)
  - **Log rotation** when file size exceeds configurable limit
  - **In-memory log storage** for quick queries
  - **JSON export** for external analysis

- **LogManager**:
  - Thread-safe log management with mutex protection
  - Configurable log success/failure filtering
  - Search logs by command text
  - Filter logs by date range or status
  - Automatic file rotation based on size limits
  - Retention policy support

- **Log Commands**:
  - `log enable` - Enable command logging
  - `log disable` - Disable command logging
  - `log status` - Show logging status and configuration
  - `log show` - Display command logs (with filtering options)
    - `--last N` - Show last N logs
    - `--failed` - Show only failed commands
    - `--search "text"` - Search logs by command text
    - `--since "YYYY-MM-DD"` - Show logs since date
    - `--json` - Output in JSON format
  - `log clear` - Clear all in-memory logs
  - `log export` - Export logs to JSON format
  - `log load` - Load logs from file
  - `log config [setting] [value]` - Configure logging settings
    - `max_size` - Maximum log file size in MB
    - `retention` - Log retention period in days
    - `log_success` - Enable/disable logging of successful commands
    - `log_failures` - Enable/disable logging of failed commands

- **Configuration** (in config.toml):
  ```toml
  [logging]
  enabled = false
  log_file = "~/.{appname}/audit.log"
  log_success = true
  log_failures = true
  max_size_mb = 100
  retention_days = 90
  ```

- **Integration**: Logging is automatically integrated into `Execute` and only logs top-level commands (not recursive calls) to avoid log spam

**Security Note**: Audit logs may contain sensitive command arguments and output. Secure the log file appropriately.

### Interactive Prompts System (prompt.go + promptcmds.go)

**Phase 2 Feature**: Interactive user prompts for confirmations and input

- **Features**:
  - **Confirmation prompts** for yes/no questions
  - **String input prompts** with optional defaults
  - **Password prompts** (simplified version)
  - **Single selection** from options list
  - **Multi-selection** from options list
  - **Integer input** with defaults
  - **Destructive operation confirmation** (requires typing "yes")

- **Prompt Methods** (available on CLI instance):
  - `cli.Confirm(message)` - Simple yes/no confirmation
  - `cli.Prompt(message)` - String input
  - `cli.PromptDefault(message, default)` - String input with default
  - `cli.PromptPassword(message)` - Password input (simplified)
  - `cli.Select(message, options)` - Single selection
  - `cli.SelectWithDefault(message, options, defaultIdx)` - Single selection with default
  - `cli.MultiSelect(message, options)` - Multiple selection
  - `cli.ConfirmDestructive(message)` - Destructive confirmation (requires "yes")
  - `cli.PromptInteger(message, default)` - Integer input with default

- **Interactive Commands**:
  - `prompt-demo` - Demonstrate all prompt types
  - `confirm [message]` - Generic confirmation command
  - `input [message] [--default value]` - Text input command
  - `select [message] [options...] [--default idx]` - Single selection command
  - `multiselect [message] [options...]` - Multiple selection command

- **Helper Functions for Command Developers**:
  - `AddYesFlag(cmd, &yesVar)` - Add --yes flag to skip confirmations
  - `AddDryRunFlag(cmd, &dryRunVar)` - Add --dry-run flag to simulate
  - `ConfirmOrSkip(cli, yesFlag, message)` - Confirm or skip if --yes
  - `ConfirmDestructiveOrSkip(cli, yesFlag, message)` - Destructive confirm or skip
  - `ShowDryRun(cmd, cli, dryRun, action)` - Show dry-run message

- **Usage Examples**:
  ```go
  // In a custom command
  if !cli.Confirm("Deploy to production?") {
      cmd.Println("Cancelled")
      return
  }

  // With --yes flag
  var yesFlag bool
  AddYesFlag(cmd, &yesFlag)
  if !ConfirmOrSkip(cli, yesFlag, "Delete all data?") {
      return
  }

  // Single selection
  env := cli.Select("Choose environment", []string{"dev", "staging", "prod"})

  // Multi-selection
  features := cli.MultiSelect("Enable features", []string{"auth", "logging", "cache"})
  ```

**Design Philosophy**: Interactive prompts improve user safety for destructive operations while maintaining automation capability via `--yes` and `--dry-run` flags.

### Template System (template.go + templatecmds.go)

**Phase 3 Feature**: Script template system with variable substitution

- **Templates Directory**: `~/.{appname}/templates/` (configurable)

- **Features**:
  - **Go text/template syntax** for variable substitution
  - **Embedded templates** support via embed.FS
  - **File system templates** for user-created templates
  - **Template caching** for performance
  - **Variable parsing** from command line (key=value format)
  - **Execute or render** templates

- **TemplateManager**:
  - Load templates from embedded FS or file system
  - Cache parsed templates for reuse
  - Execute templates with variable substitution
  - List, create, show, delete templates
  - Clear template cache

- **Template Commands**:
  - `template list` - List all available templates
  - `template show [name]` - Display template content
  - `template exec [name] [key=value...]` - Execute template with variables
  - `template render [name] [key=value...]` - Render template without executing
  - `template create [name]` - Create new template interactively
  - `template delete [name]` - Delete a template
  - `template clear-cache` - Clear template cache

- **Template Syntax** (Go text/template):
  ```
  # deployment.tmpl
  print "Deploying to {{.Env}}"
  let region="{{.Region}}"
  http {{.ApiEndpoint}}/deploy
  ```

- **Usage Examples**:
  ```bash
  # Execute template with variables
  template exec deployment.tmpl Env=prod Region=us-east-1 ApiEndpoint=https://api.example.com

  # Render template without executing
  template render deployment.tmpl Env=staging Region=eu-west-1 ApiEndpoint=https://staging.example.com

  # Create a new template
  template create mydeployment.tmpl
  # (enter template content, then Ctrl+D)

  # List templates
  template list

  # Show template
  template show deployment.tmpl

  # Delete template
  template delete mydeployment.tmpl
  ```

**Use Cases**:
- Parameterized deployment scripts
- Code generation
- Repetitive command sequences with variable parameters
- Environment-specific configurations

### Data Manipulation Commands (datamanipcmds.go)

**Phase 3 Feature**: JSON, CSV, and YAML parsing and conversion

- **JSON Commands**:
  - `json parse [file] [--pretty]` - Parse and format JSON
  - `json get [file] [path]` - Extract value using dot notation (e.g., `users.0.name`)
  - `json validate [file]` - Validate JSON syntax

- **YAML Commands**:
  - `yaml parse [file]` - Parse and format YAML
  - `yaml to-json [file]` - Convert YAML to JSON
  - `yaml from-json [file]` - Convert JSON to YAML

- **CSV Commands**:
  - `csv parse [file] [--header]` - Parse and display CSV as table
  - `csv to-json [file]` - Convert CSV to JSON (first row as header)

- **Features**:
  - Stdin support for all commands (pipe-friendly)
  - Pretty-printing for JSON output
  - Dot notation for JSON path traversal
  - Automatic header detection for CSV
  - Format conversion pipeline

- **Usage Examples**:
  ```bash
  # Parse and pretty-print JSON
  cat data.json | json parse --pretty

  # Extract nested value
  json get users.json users.0.email

  # Convert YAML to JSON
  yaml to-json config.yaml | json parse

  # Convert CSV to JSON
  csv to-json data.csv > output.json

  # Pipeline example
  http api.example.com/users | json get results | yaml from-json
  ```

**Use Cases**:
- API response parsing
- Configuration file format conversion
- Data extraction and transformation
- Log analysis

### Watch Command (watchcmds.go)

**Phase 3 Feature**: Repeatedly execute a command at a specified interval

- **Command**: `watch [command] [flags]`
  - `--interval, -n` - Interval between executions (default 2s)
  - `--count, -c` - Number of times to execute (0 = infinite)
  - `--clear` - Clear screen before each execution

- **Features**:
  - Configurable execution interval (e.g., 2s, 500ms, 1m)
  - Optional iteration limit
  - Screen clearing support
  - Iteration counter and timestamps
  - Automatic error display

- **Usage Examples**:
  ```bash
  # Monitor date every 2 seconds (default)
  watch "date"

  # Monitor jobs every 5 seconds
  watch --interval 5s "jobs"

  # Run 10 times with screen clearing
  watch --count 10 --clear "date"

  # Monitor API endpoint status
  watch -n 1s "http https://api.example.com/health | json get status"

  # Monitor system metrics
  watch -n 2s --clear "osexec 'ps aux | head -10'"
  ```

**Use Cases**:
- Monitoring system resources
- Tracking job status
- Polling API endpoints
- Observing log file changes
- Testing periodic scripts
- Report generation

### MCP Server Integration (mcp.go + mcpcmds.go)

**Phase 3 Feature**: Model Context Protocol server for external tool integration

- **MCP Server**: Exposes CLI commands as MCP tools via stdio
  - JSON-RPC 2.0 protocol over stdin/stdout
  - Automatic tool discovery from Cobra commands
  - Flag-to-parameter schema conversion
  - Resource exposure (templates, scripts)
  - Full context support for cancellation and timeouts

- **Commands**:
  - `mcp start` - Start the MCP stdio server
  - `mcp info` - Show server information and configuration
  - `mcp list-tools` - List all available MCP tools

- **Features**:
  - **Automatic Tool Generation**: All CLI commands exposed as MCP tools
  - **Schema Generation**: Command flags converted to JSON Schema input parameters
  - **Resource Discovery**: Templates and scripts accessible as MCP resources
  - **Standards Compliant**: Follows MCP specification 2024-11-05

- **Claude Desktop Integration**:
  ```json
  {
    "mcpServers": {
      "myapp": {
        "command": "/full/path/to/myapp",
        "args": ["mcp", "start"]
      }
    }
  }
  ```

- **Tool Mapping**:
  - Command name → Tool name (e.g., `alias add` → `"alias add"`)
  - Short description → Tool description
  - Cobra flags → Input schema properties
  - Positional arguments → `_args` parameter

- **Usage Examples**:
  ```bash
  # Get MCP server information
  mcp info

  # List all available tools
  mcp list-tools

  # Start MCP server (for external clients)
  mcp start
  ```

**Use Cases**:
- Claude Desktop integration for AI-assisted CLI operations
- IDE integration for command discovery and execution
- Automation frameworks requiring CLI tool discovery
- Remote CLI execution via MCP protocol
- Tool documentation generation

**Implementation Notes**:
- Server uses `RootCmd()` to access commands dynamically
- Works in both REPL and non-REPL modes
- Hidden commands are excluded from tool list
- Parent-only commands (no Run function) are recursively expanded
- Error messages are included in tool call responses

**Security Considerations**:
- MCP exposes all CLI commands to external clients
- Ensure proper access control at the MCP client level
- Consider limiting which commands are exposed for production use
- Audit logging recommended for MCP command executions

**Documentation**: See [MCP_INTEGRATION.md](./MCP_INTEGRATION.md) for detailed integration guide.

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

### REPL Library Migration & Display Adapters
The library was migrated from `github.com/alexj212/console` → `c-bata/go-prompt` → **`reeflective/console`**. The parser from console is still used for pipe/redirection support, but the REPL interface is now reeflective/console which provides automatic Cobra integration and completion.

**DisplayAdapter Refactoring (2026-02):**
- `BubbletteaAdapter` was moved to a separate module (`examples/simple_bubbletea/go.mod`)
- Core library only depends on `reeflective/console` to reduce dependency footprint
- Bubbletea dependencies (bubbletea, lipgloss) removed from root `go.mod`
- Applications requiring bubbletea can implement their own adapter or use the example as reference

### Command Parser Quote Handling
The parser now uses `github.com/kballard/go-shellquote` for proper quote and escape handling. This fixes issues where special characters (`|`, `>`, `;`) inside quoted strings were incorrectly treated as operators.

### Recursion Protection
`Execute` tracks recursion depth with `CLI.execDepth` counter. Maximum depth is set to 10 (configurable via `CLI.maxExecDepth`). This prevents infinite loops from circular `@exec:` references or aliases.

### Context Support (cli.go)

**Phase 3 Feature**: Context-aware command execution for cancellation and timeout

- **Methods**:
  - `Execute(line, scope)` - Standard execution (uses background context)
  - `ExecuteWithContext(ctx, line, scope)` - Context-aware execution

- **Features**:
  - Command cancellation via context
  - Timeout support via `context.WithTimeout`
  - Deadline support via `context.WithDeadline`
  - Cancellation checks at command boundaries and in pipelines
  - Proper error wrapping with `context.Err()`

- **Usage Examples**:
  ```go
  // Execute with timeout
  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()
  output, err := cli.ExecuteWithContext(ctx, "http slow-api.example.com", nil)

  // Execute with cancellation
  ctx, cancel := context.WithCancel(context.Background())
  go func() {
    time.Sleep(5 * time.Second)
    cancel() // Cancel after 5 seconds
  }()
  output, err := cli.ExecuteWithContext(ctx, "repeat 100 'sleep 1s'", nil)
  ```

**Design**: Maintains backward compatibility by wrapping `Execute` to call `ExecuteWithContext` with `context.Background()`. Cancellation is checked before each command and at each stage in a pipeline.

### Phase 2 Features (Advanced CLI Capabilities)

**Phase 2 Feature Set**: Additional power-user features implemented in December 2025

#### Pipeline Enhancements (pipelinecmds.go)
- **tee command**: Read from stdin and write to both stdout and multiple files
  - `tee file1.txt file2.txt` - Write to multiple files
  - `tee --append file.txt` - Append mode
  - Supports multiple file outputs simultaneously

#### Advanced Control Flow (controlflowcmds.go)
- **case command**: Pattern matching similar to switch/case
  - `case $env prod "print Production" dev "print Dev" "*" "print Other"`
  - Supports wildcard patterns

- **while command**: Loop while condition is true
  - `while "test @i -lt 5" "print @i; inc i"`
  - Safety limit of 1000 iterations to prevent infinite loops

- **for command**: Iterate over values
  - `for i in 1 2 3 do "print Item @i"`
  - Proper variable scoping (saves/restores old values)

- **test command**: Condition testing for loops
  - Numeric: `-eq`, `-ne`, `-lt`, `-le`, `-gt`, `-ge`
  - String: `=`, `!=`
  - Exits with 0 (true) or 1 (false)

#### Notification System (notify.go + notifycmds.go)
- **NotificationManager**: Cross-platform desktop notifications
  - Linux: `notify-send` with urgency levels (low, normal, critical)
  - macOS: `osascript` for system notifications
  - Windows: PowerShell toast notifications

- **Webhook Support**: HTTP POST notifications
  - JSON payload with title, message, timestamp
  - Configurable webhook URL stored in config file

- **Commands**:
  - `notify send "Title" "Message" --urgency critical`
  - `notify send "Alert" "Text" --webhook`
  - `notify config https://hooks.example.com/webhook`

#### Command Scheduling (schedulecmds.go + jobs.go)
- **ScheduledTask**: Background task execution with timing control
  - Tracked in JobManager.scheduledTasks map
  - Supports one-time and repeating tasks
  - Pause/resume for repeating tasks

- **Commands**:
  - `schedule at 14:30 "print Reminder"` - Run at specific time
  - `schedule in 5m "print Done"` - Run after delay
  - `schedule every 1h "print Tick"` - Repeat at interval
  - `schedule list` - Show all scheduled tasks
  - `schedule cancel [id]` - Stop a scheduled task
  - `schedule pause [id]` / `schedule resume [id]` - Control repeating tasks

- **Implementation**: Uses `time.Timer` for one-time tasks, `time.Ticker` for repeating

#### Enhanced History (history.go)
- **HistoryBookmark**: Persistent command bookmarks with metadata
  - Stored in `~/.{appname}.bookmarks` as JSON
  - Includes name, command, description, timestamp

- **New Commands**:
  - `history bookmark add name "command" -d "description"`
  - `history bookmark list` - Show all bookmarks
  - `history bookmark run name` - Execute bookmarked command
  - `history bookmark remove name` - Delete bookmark
  - `history replay 5` - Re-execute command at index 5
  - `history stats` - Show usage statistics (total, unique, top 10 most used)

- **Features**: Bookmark management separate from history, replay by index, detailed stats

#### Output Formatting (formatcmds.go)
- **table command**: Format delimited input as aligned table
  - `table --delim ","` - Custom delimiter (default: whitespace)
  - `table --headers` - First line as headers with separator

- **highlight command**: Regex-based text highlighting
  - `highlight "Error|Warning" --color red`
  - Colors: red, green, blue, yellow, magenta, cyan
  - Respects `CLI.NoColor` setting

- **page command**: Simple pagination
  - `page --size 20` - Lines per page
  - Basic implementation (shows first page)

- **column command**: Columnize output
  - `column --count 3` - Number of columns
  - Auto-calculates column widths

#### Clipboard Integration (clipboardcmds.go)
- **clip command**: Copy stdin to system clipboard
  - Linux: Uses `xclip` or `xsel`
  - macOS: Uses `pbcopy`
  - Windows: Uses `clip.exe`

- **paste command**: Output clipboard contents
  - Linux: Uses `xclip` or `xsel`
  - macOS: Uses `pbpaste`
  - Windows: Uses PowerShell `Get-Clipboard`

- **Usage**: `print "text" | clip`, `paste | grep pattern`

#### Config Extensions
- **NotificationConfig**: Added to config.go
  - `WebhookURL string` - Persistent webhook configuration
  - Automatically loaded via `applyNotificationConfig()`
  - Saved via `notify config` command

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
- Use `cli.ExpandCommand(cmd, defs, input)` to process tokens
- Add custom token handlers via `cli.VariableExpanders` slice
- Token names starting with `@` are reserved for the system

When implementing script commands:
- Use `LoadScript` to read and parse script files
- Call `cli.Execute` to execute individual commands with scoped defaults
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
