# Comparison: simple_bufio vs simple_console

## Overview

ConsoleKit provides two simple examples demonstrating different approaches to building CLI applications:

| Example | Directory | Input Method | Use Case |
|---------|-----------|--------------|----------|
| **simple_bufio** | `examples/simple_bufio` | `bufio.Scanner` (stdio) | Scripts, automation, pipes |
| **simple_console** | `examples/simple_console` | `reeflective/console` (REPL) | Interactive terminal use |

## Architecture Comparison

### simple_bufio (This Example)

```
User Input (stdin)
    ↓
Terminal Raw Mode (if TTY)
    ↓
Character-by-character reading
    ↓
History & Line Editing
    ↓
CommandExecutor.Execute()
    ↓
Output to stdout/stderr
```

**Characteristics:**
- ✅ Command history with ↑/↓ navigation
- ✅ Line editing with cursor movement
- ✅ Works great with pipes (auto-detects TTY)
- ✅ No external REPL dependencies
- ✅ Educational - see how terminals work
- ❌ No tab completion
- ❌ History not persisted to file

### simple_console

```
User Input (terminal)
    ↓
reeflective/console REPL
    ↓
REPLHandler
    ↓
CommandExecutor.Execute()
    ↓
Colored output to terminal
```

**Characteristics:**
- ✅ Full REPL experience (history, completion, colors)
- ✅ Arrow keys for history navigation
- ✅ File-based command history
- ✅ Tab completion for commands
- ❌ Heavier dependencies
- ❌ Requires PTY for full features

## Code Comparison

### simple_bufio (custom terminal approach)

```go
// Enable terminal raw mode for character-by-character input
oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
defer term.Restore(int(os.Stdin.Fd()), oldState)

history := []string{}
historyIndex := -1

for {
    // Read line with history and editing support
    line, exit := readLineWithHistory(&history, &historyIndex, prompt)
    if exit || line == "exit" {
        break
    }

    // Add to history (skip duplicates)
    if len(history) == 0 || history[len(history)-1] != line {
        history = append(history, line)
    }

    // Execute command
    output, err := executor.Execute(line, nil)

    // Print results (with \r\n for raw mode)
    if output != "" {
        fmt.Print(strings.ReplaceAll(output, "\n", "\r\n"))
    }
}
```

**Note:** `readLineWithHistory()` handles:
- Character-by-character reading
- Escape sequence detection for arrow keys
- Line editing (insert, delete, backspace)
- History navigation with ↑/↓
- Cursor movement with ←/→

### simple_console (REPL approach)

```go
// Create REPL handler with advanced features
handler := consolekit.NewREPLHandler(executor)

// Set custom prompt
handler.SetPrompt(func() string {
    return "\nsimple > "
})

// Run will execute command-line args if present,
// otherwise start the interactive REPL
err = handler.Run()
```

## Use Case Recommendations

### Use simple_bufio when:

1. **Automation & CI/CD**
   ```bash
   # Run commands from a file
   cat deploy.txt | simple-bufio

   # Process in scripts
   ./simple-bufio run @deploy.run
   ```

2. **Docker Containers**
   - Minimal image size
   - No PTY required
   - Works with `docker exec`

3. **Piped Input**
   ```bash
   echo "print 'Hello'" | simple-bufio
   find . -name "*.txt" | simple-bufio cat
   ```

4. **Embedded Systems**
   - Limited terminal capabilities
   - Serial console
   - No readline support

### Use simple_console when:

1. **Interactive Development**
   - Tab completion for discovery
   - History for repeated commands
   - Arrow keys for editing

2. **End-User CLI Tools**
   - Professional appearance
   - Color-coded output
   - Better UX

3. **Local Administration**
   - System management tools
   - Database CLIs
   - Infrastructure management

## Performance

| Metric | simple_bufio | simple_console |
|--------|--------------|----------------|
| Binary Size | ~18MB | ~18MB |
| Memory (idle) | ~12MB | ~15MB |
| Startup Time | ~10ms | ~20ms |
| Dependencies | Minimal | Medium |

## Migration Between Examples

Both examples use the same `CommandExecutor` core, so switching is easy:

```go
// From simple_bufio to simple_console
executor, _ := consolekit.NewCommandExecutor("myapp", customizer)

// OLD (bufio):
scanner := bufio.NewScanner(os.Stdin)
// ... scanner loop ...

// NEW (console):
handler := consolekit.NewREPLHandler(executor)
handler.Run()
```

## Hybrid Approach

You can combine both approaches:

```go
func main() {
    executor, _ := consolekit.NewCommandExecutor("myapp", customizer)

    // Check if stdin is a pipe or terminal
    stat, _ := os.Stdin.Stat()
    if (stat.Mode() & os.ModeCharDevice) == 0 {
        // Piped input - use bufio
        runBufioMode(executor)
    } else {
        // Interactive terminal - use REPL
        handler := consolekit.NewREPLHandler(executor)
        handler.Run()
    }
}
```

## Summary

- **simple_bufio**: Best for automation, scripts, and simple CLIs
- **simple_console**: Best for interactive terminal applications

Both share the same command execution engine, so you get consistent behavior regardless of the input method.
