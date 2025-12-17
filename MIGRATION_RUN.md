# Migration Guide: AppBlock() → Run()

## Overview

ConsoleKit now provides a unified `Run()` method that automatically handles both command-line execution and REPL modes. This replaces the need to manually call `AppBlock()`.

## Why This Change?

The new `Run()` method enables:
- **Command-line execution**: Execute commands directly without entering the REPL
- **MCP server integration**: Start the MCP server from the command line
- **Better CLI patterns**: Match standard CLI tool behavior (args = execute, no args = REPL)

## Migration Steps

### Before (Old Pattern)

```go
func main() {
    cli, err := consolekit.NewCLI("myapp", customizer)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    err = cli.AppBlock()  // Old: Always enters REPL
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

**Problem**: This always enters the REPL, even when you want to run a command directly (e.g., `./myapp mcp start`).

### After (New Pattern)

```go
func main() {
    cli, err := consolekit.NewCLI("myapp", customizer)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    err = cli.Run()  // New: Smart execution
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

**Benefits**:
- No arguments: `./myapp` → Enters REPL
- With arguments: `./myapp mcp start` → Executes command directly
- MCP integration works perfectly

## New Capabilities

### Command-Line Execution

```bash
# Execute a command directly
./myapp print "Hello World"

# Start MCP server
./myapp mcp start

# Show version
./myapp version

# Get MCP info
./myapp mcp info
```

### REPL Mode

```bash
# Start REPL (no arguments)
./myapp

# Now you're in the REPL
myapp > print "Hello from REPL"
myapp > mcp info
myapp > exit
```

## Backward Compatibility

If you need to force REPL mode (ignoring command-line arguments), you can still use `AppBlock()` directly:

```go
// Force REPL mode regardless of arguments
err = cli.AppBlock()
```

However, for most applications, `Run()` is the recommended entry point.

## Advanced: ExecuteArgs()

If you need to execute specific arguments programmatically:

```go
// Execute specific arguments
err := cli.ExecuteArgs([]string{"print", "Hello"})

// Execute MCP server
err := cli.ExecuteArgs([]string{"mcp", "start"})
```

## Testing

### Test Command-Line Execution

```bash
./myapp print "test"
./myapp mcp info
./myapp version
```

### Test REPL Mode

```bash
# Start with no arguments
./myapp

# Should enter REPL with your prompt
```

### Test MCP Server

```bash
# Start MCP server
./myapp mcp start

# Should start listening on stdin/stdout
# Press Ctrl+C to stop
```

## Summary

| Mode | Command | Behavior |
|------|---------|----------|
| REPL | `./myapp` | Enters interactive REPL |
| Command-line | `./myapp <command>` | Executes command directly |
| MCP Server | `./myapp mcp start` | Starts MCP stdio server |
| Legacy | `cli.AppBlock()` | Forces REPL mode |

**Recommendation**: Use `cli.Run()` for all new applications.
