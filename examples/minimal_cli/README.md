# Minimal CLI Example

This example demonstrates ConsoleKit's **modular command selection system** (new in v0.8.0).

## What This Shows

Instead of including all built-in commands with `AddBuiltinCommands()`, this example selectively includes only the commands needed for a basic scripting environment:

- **Core commands:** `cls`, `exit`, `print`, `date`
- **Variable management:** `let`, `unset`, `vars`, `inc`, `dec`
- **Script execution:** `run` command
- **Control flow:** `if`, `repeat`, `while`, `for`, `case`, `test`

## Why Use Selective Inclusion?

1. **Smaller binary** - Only compile what you need
2. **Faster startup** - Fewer commands to register
3. **Better security** - Exclude dangerous commands like `osexec`
4. **Clearer intent** - Explicit about capabilities
5. **Custom builds** - Tailor CLI to specific use case

## Building & Running

```bash
go build
./minimal_cli
```

## Commands Available

In the REPL, try:

```bash
# Core commands
print "Hello World"
date
cls

# Variables
let name="ConsoleKit"
let count=5
vars

# Control flow
repeat --count @count "print Hello"
if @count 5 --if-true="print Count is 5"

# Scripts
run demo.run
run /path/to/external/script.run

# Exit
exit
```

## What's NOT Included

This minimal example intentionally excludes:
- OS execution (`osexec`)
- HTTP commands (`http`)
- Job management (`jobs`)
- Aliases, history, config commands
- Data manipulation (json, yaml, csv)
- Templates, notifications, MCP
- And more...

## Customizing Command Selection

Edit `main.go` to include different command groups:

```go
customizer := func(exec *consolekit.CommandExecutor) error {
    // Current: Minimal setup
    exec.AddCommands(consolekit.AddCoreCmds(exec))
    exec.AddCommands(consolekit.AddVariableCmds(exec))
    exec.AddCommands(consolekit.AddControlFlowCmds(exec))
    exec.AddCommands(consolekit.AddRun(exec, &Data))

    // Add more as needed:
    // exec.AddCommands(consolekit.AddNetworkCmds(exec))     // http
    // exec.AddCommands(consolekit.AddJobCmds(exec))         // jobs
    // exec.AddCommands(consolekit.AddAliasCmds(exec))       // aliases
    // exec.AddCommands(consolekit.AddDataManipulationCmds(exec)) // json, yaml

    return nil
}
```

## Use Cases

**This approach is ideal for:**
- Script runners (only need run + basic commands)
- Automation tools (core + control flow + specific integrations)
- Embedded CLIs (minimal footprint)
- Security-sensitive apps (exclude dangerous commands)
- Custom REPLs (only features you need)

## Comparison with simple_console

| Feature | minimal_cli | simple_console |
|---------|-------------|----------------|
| Commands | ~15 commands | 50+ commands |
| Binary Size | Smaller | Larger |
| Startup Time | Faster | Slower |
| Customization | Explicit selection | All included |
| Use Case | Specific needs | Full-featured REPL |

## More Information

- **Command Groups:** See [COMMAND_GROUPS.md](../../COMMAND_GROUPS.md)
- **Migration Guide:** See [MIGRATION.md](../../MIGRATION.md)
- **Full Documentation:** See [CLAUDE.md](../../CLAUDE.md)
