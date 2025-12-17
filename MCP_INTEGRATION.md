# MCP Integration Guide

This guide explains how to use ConsoleKit's built-in Model Context Protocol (MCP) server functionality.

## Overview

ConsoleKit includes MCP server support that exposes all CLI commands as MCP tools via stdio. This allows external applications (like Claude Desktop, IDEs, or other MCP clients) to discover and execute your CLI commands remotely.

## Features

- **Automatic Tool Discovery**: All registered CLI commands are automatically exposed as MCP tools
- **JSON-RPC 2.0**: Standards-compliant protocol over stdio
- **Flag Support**: Command flags are converted to tool input parameters
- **Resource Support**: Templates and scripts are exposed as MCP resources
- **Context Support**: Full support for command execution contexts

## Quick Start

### 1. Add MCP Commands to Your CLI

MCP commands are automatically included when you call `AddAll()`:

```go
cli, err := consolekit.NewCLI("myapp", func(cli *consolekit.CLI) error {
    cli.AddAll()  // Includes MCP commands
    return nil
})

// Use Run() instead of AppBlock() to support command-line execution
if err := cli.Run(); err != nil {
    fmt.Printf("Error: %v\n", err)
}
```

**Important**: Use `cli.Run()` instead of `cli.AppBlock()` as your entry point. The `Run()` method automatically detects command-line arguments and executes them directly, or starts the REPL if no arguments are provided. This is essential for MCP integration.

Or add MCP commands manually:

```go
cli.AddCommands(consolekit.AddMCPCommands(cli))
```

### 2. Test MCP Server Locally

Get information about the MCP server:

```bash
./myapp mcp info
```

List all available tools:

```bash
./myapp mcp list-tools
```

Start the MCP server:

```bash
./myapp mcp start
```

The server listens on stdin/stdout and communicates using JSON-RPC 2.0 messages.

### 3. Serve MCP Over HTTP (Optional)

ConsoleKit can also expose MCP over HTTP using an SSE + POST transport:

```bash
./myapp mcp start --http --http-addr 127.0.0.1:7331
```

Endpoints:

- `GET /sse` — SSE stream (server → client). Sends an `endpoint` event with the `POST /messages?sessionId=...` URL.
- `POST /messages?sessionId=...` — client → server JSON-RPC messages (responses are delivered over SSE).
- `POST /mcp` — fallback single-request HTTP JSON-RPC (non-SSE).

## Claude Desktop Integration

To integrate your CLI with Claude Desktop, add a configuration entry to Claude Desktop's MCP settings:

### Configuration Location

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

### Configuration Example

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

**Important**: Use the absolute path to your executable.

### Verification

After adding the configuration:

1. Restart Claude Desktop
2. In a new chat, Claude will have access to your CLI commands
3. Ask Claude to list available tools to verify the connection

Example prompt: "What tools do you have access to from myapp?"

## MCP Protocol Details

### Supported Methods

#### `initialize`
Establishes the MCP session and exchanges capabilities.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "claude-desktop",
      "version": "1.0"
    }
  }
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": { "listChanged": false },
      "resources": { "subscribe": false, "listChanged": false }
    },
    "serverInfo": {
      "name": "myapp",
      "version": "1.0.0"
    }
  }
}
```

#### `tools/list`
Returns all available CLI commands as MCP tools.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "print",
        "description": "Print a message to console",
        "inputSchema": {
          "type": "object",
          "properties": {
            "_args": {
              "type": "string",
              "description": "Positional arguments for the command"
            }
          }
        }
      }
    ]
  }
}
```

#### `tools/call`
Executes a CLI command and returns the output.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "print",
    "arguments": {
      "_args": "Hello, World!"
    }
  }
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Hello, World!\n"
      }
    ],
    "isError": false
  }
}
```

#### `resources/list`
Returns available resources (templates, scripts, etc.).

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "resources/list"
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "resources": [
      {
        "uri": "template://deploy.tmpl",
        "name": "deploy.tmpl",
        "description": "ConsoleKit template",
        "mimeType": "text/plain"
      }
    ]
  }
}
```

## Command Mapping

### CLI Commands → MCP Tools

CLI commands are automatically mapped to MCP tools:

- **Command Name**: Becomes the tool name
- **Short Description**: Becomes the tool description
- **Flags**: Become input schema properties
- **Arguments**: Become `_args` input property

### Example Mapping

**CLI Command**:
```go
&cobra.Command{
    Use:   "greet [name]",
    Short: "Greet someone",
    Run: func(cmd *cobra.Command, args []string) {
        name := "World"
        if len(args) > 0 {
            name = args[0]
        }
        cmd.Printf("Hello, %s!\n", name)
    },
}
```

**MCP Tool**:
```json
{
  "name": "greet",
  "description": "Greet someone",
  "inputSchema": {
    "type": "object",
    "properties": {
      "_args": {
        "type": "string",
        "description": "Positional arguments for the command"
      }
    }
  }
}
```

### Flag Mapping

Cobra flags are automatically converted to input schema properties:

```go
cmd.Flags().StringP("output", "o", "", "Output file")
cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
```

Becomes:

```json
{
  "properties": {
    "output": {
      "type": "string",
      "description": "Output file",
      "default": ""
    },
    "verbose": {
      "type": "string",
      "description": "Verbose output",
      "default": "false"
    }
  }
}
```

## Best Practices

### 1. Command Descriptions

Provide clear, concise descriptions for your commands:

```go
&cobra.Command{
    Use:   "deploy",
    Short: "Deploy application to production",  // Good: Clear and specific
    Long:  `Deploy the application to production environment...`,
}
```

### 2. Flag Documentation

Document flags thoroughly:

```go
cmd.Flags().StringP("env", "e", "production", "Target environment (dev, staging, production)")
```

### 3. Error Handling

Return meaningful error messages:

```go
if err != nil {
    return fmt.Errorf("deployment failed: %w", err)
}
```

### 4. Idempotency

Design commands to be safe when called multiple times:

```go
// Check if already deployed before deploying
if isDeployed() {
    cmd.Println("Already deployed")
    return nil
}
```

### 5. Confirmation Prompts

For destructive operations, use `--yes` flags to allow automation:

```go
yesFlag := cmd.Flags().Bool("yes", false, "Skip confirmation")
if !*yesFlag && !cli.Confirm("Delete all data?") {
    return nil
}
```

## Troubleshooting

### Server Won't Start

1. Check that the executable path in the config is absolute
2. Verify the executable has execute permissions
3. Test the server manually: `./myapp mcp start`

### Commands Not Appearing

1. Ensure `AddMCPCommands` is called or `AddAll()` is used
2. Check that commands have Run/RunE functions
3. Verify commands are not hidden

### Connection Issues

1. Restart Claude Desktop after config changes
2. Check Claude Desktop logs for errors
3. Test MCP protocol manually with a simple JSON-RPC client

### Debugging

Enable stderr logging in your CLI to see server messages:

```go
fmt.Fprintf(os.Stderr, "Debug: command executed\n")
```

MCP protocol uses stdout for communication, so stderr is safe for logging.

## Advanced Usage

### Custom Tool Names

Subcommands are automatically prefixed:

```
alias add    → "alias add"
alias delete → "alias delete"
config get   → "config get"
```

### Resource Exposure

Templates are automatically exposed as resources:

```go
cli.TemplateManager.ListTemplates()
// Returns templates accessible via resource URIs like "template://deploy.tmpl"
```

### Security Considerations

- **Access Control**: MCP exposes all CLI commands. Ensure your CLI has appropriate security measures.
- **Input Validation**: Validate all command inputs as you would for direct CLI usage.
- **Secrets**: Avoid exposing sensitive data in command outputs or descriptions.
- **Audit Logging**: Enable logging to track MCP command executions.

## Examples

### Example 1: Simple Integration

```go
package main

import (
    "fmt"
    "github.com/alexj212/consolekit"
    "github.com/spf13/cobra"
)

func main() {
    customizer := func(cli *consolekit.CLI) error {
        cli.AddAll()

        // Add custom command
        cli.AddCommands(func(rootCmd *cobra.Command) {
            rootCmd.AddCommand(&cobra.Command{
                Use:   "hello",
                Short: "Say hello",
                Run: func(cmd *cobra.Command, args []string) {
                    cmd.Println("Hello from MCP!")
                },
            })
        })

        return nil
    }

    cli, err := consolekit.NewCLI("myapp", customizer)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Use Run() to support both command-line and REPL modes
    if err := cli.Run(); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

**Key Points**:
- Use `cli.Run()` as the entry point (not `cli.AppBlock()`)
- With no arguments: `./myapp` starts the REPL
- With arguments: `./myapp mcp start` runs the MCP server
- Command-line execution: `./myapp hello` executes the hello command

### Example 2: Testing MCP Locally

```bash
# Start the server
./myapp mcp start

# In another terminal, send JSON-RPC request
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./myapp mcp start
```

## References

- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [Claude Desktop MCP Documentation](https://docs.anthropic.com/claude/docs/mcp)
- [ConsoleKit Documentation](./README.md)
