# Simple Bufio Example

This example demonstrates using ConsoleKit with **custom terminal handling** for an interactive REPL with history support.

## Features

- ✅ **Command history** - Navigate with ↑/↓ arrow keys
- ✅ **Line editing** - Move cursor with ←/→, insert/delete characters
- ✅ **Terminal raw mode** - Character-by-character input processing
- ✅ **Automatic fallback** - Detects TTY vs pipes and adjusts behavior
- ✅ **Keyboard shortcuts** - Ctrl+C, Ctrl+D, Ctrl+L
- ✅ **Standard input/output** - stdin/stdout
- ✅ **Command-line execution** - supports both REPL and direct command execution

## Comparison with simple_console

| Feature | simple_bufio (this) | simple_console |
|---------|---------------------|----------------|
| Input Method | Custom raw terminal | `reeflective/console` |
| History | ✅ Yes (in-memory) | ✅ Yes (file-based) |
| Completion | ❌ No | ✅ Yes (Cobra integration) |
| Arrow Keys | ✅ Yes (↑↓←→) | ✅ Yes (full readline) |
| Line Editing | ✅ Yes (insert/delete) | ✅ Yes (full featured) |
| Colors | ❌ No | ✅ Yes |
| Complexity | Medium | High |
| Use Case | Learning, automation, custom needs | Production interactive use |

## Building

```bash
go build -o simple-bufio
```

## Usage

### Interactive REPL Mode

```bash
./simple-bufio
```

You'll see an interactive prompt with history support:
```
ConsoleKit Simple Bufio Example (with history)
Type 'help' for available commands, 'exit' to quit
Use ↑/↓ arrows for history navigation

simple-bufio >
```

Type commands and press Enter to execute.

#### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ↑ / ↓ | Navigate command history |
| ← / → | Move cursor left/right |
| Backspace | Delete character before cursor |
| Delete | Delete character at cursor |
| Enter | Execute command |
| Ctrl+C | Cancel current line |
| Ctrl+D | Exit (when line is empty) |
| Ctrl+L | Clear screen |

**Features:**
- Commands are added to history automatically
- History is preserved during the session
- Duplicate consecutive commands are not added to history
- You can edit anywhere in the line (insert mode)
- Arrow keys work for both history and cursor movement

### Command-line Mode

```bash
# Execute a single command
./simple-bufio print "Hello World"

# Run a script
./simple-bufio run @test.run

# Chain commands (use quotes)
./simple-bufio "print Hello; date"
```

### Piped Input

```bash
# Pipe commands from a file
cat commands.txt | ./simple-bufio

# Pipe output from another command
echo "print 'Hello from pipe'" | ./simple-bufio
```

## Limitations

- **No persistent history** - history is lost when you exit (not saved to file)
- **No tab completion** - you must type full command names
- **No syntax highlighting** - plain text output only
- **No colors** - error messages and output are not color-coded
- **Basic line editing** - supports insert/delete but not advanced features like kill/yank

## When to Use This

Use `simple_bufio` when:
- Running in automation/CI scripts
- Piping input from files or other commands
- Building Docker containers (minimal dependencies)
- Embedding in systems with limited terminal capabilities
- You don't need interactive REPL features

Use `simple_console` when:
- Working interactively in a terminal
- You want history and completion
- You need a polished user experience
- You're building a CLI tool for end users

## Example Session

```bash
$ ./simple-bufio
ConsoleKit Simple Bufio Example
Type 'help' for available commands, 'exit' to quit

simple-bufio > help
Available Commands:
  alias       Manage command aliases
  cat         Display file contents
  cls         Clear the screen
  config      Manage configuration settings
  ...

simple-bufio > print "Hello World"
Hello World

simple-bufio > date
2026-01-31 08:30:00

simple-bufio > exit
Goodbye!
```

## Technical Details

- Uses `bufio.Scanner` for line reading
- Commands executed via `CommandExecutor.Execute()`
- Handles both REPL mode and command-line args
- Simple error handling to stderr
- Exit on Ctrl+D (EOF) or 'exit' command
