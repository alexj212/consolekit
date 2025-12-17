# Using the Run() Method

ConsoleKit provides a unified `Run()` entry point that automatically handles both command-line and REPL execution modes.

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/alexj212/consolekit"
)

func main() {
    cli, err := consolekit.NewCLI("myapp", func(cli *consolekit.CLI) error {
        cli.AddAll()  // Add all standard commands including MCP
        return nil
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Run() automatically handles both modes
    if err := cli.Run(); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## How It Works

The `Run()` method checks for command-line arguments:

- **No arguments** → Starts the REPL (interactive mode)
- **With arguments** → Executes the command directly

## Usage Examples

```bash
# REPL Mode (no arguments)
./myapp
myapp > print "Hello"
myapp > exit

# Command-Line Mode (with arguments)
./myapp print "Hello"
./myapp mcp info
./myapp mcp start

# Perfect for MCP Integration
./myapp mcp start  # Runs as stdio server for Claude Desktop
```

## Benefits

1. **Standard CLI Behavior**: Matches how most CLI tools work
2. **MCP Integration**: Essential for running `mcp start` from command line
3. **Flexible**: Support both interactive and scripted usage
4. **Simple**: One method handles both modes

## Migration from AppBlock()

If your code currently uses `cli.AppBlock()`, simply replace it with `cli.Run()`:

```go
// Old
err = cli.AppBlock()

// New
err = cli.Run()
```

See [MIGRATION_RUN.md](./MIGRATION_RUN.md) for detailed migration guide.
