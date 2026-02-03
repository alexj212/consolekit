# Simple Bubbletea Example

This example demonstrates ConsoleKit with a [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI (Terminal User Interface) instead of the standard reeflective/console REPL.

**Note:** This example has its own `go.mod` file and is a separate Go module from the core ConsoleKit library. This keeps bubbletea dependencies isolated from the main library, reducing the dependency footprint for users who don't need TUI features.

## Features

- **Beautiful TUI**: Styled terminal interface using Bubbletea and Lipgloss
- **Command History**: Navigate history with ↑/↓ arrows
- **Color Output**: Distinct colors for prompt, input, output, and errors
- **Scrolling Output**: Shows last 20 lines of output
- **Keyboard Shortcuts**: Full keyboard navigation support
- **Alt Screen Mode**: Uses alternate screen buffer (clean exit)
- **Dual Mode**: Interactive TUI or command-line execution

## Building

```bash
cd examples/simple_bubbletea
go build
```

**Note:** This example uses a `replace` directive in its `go.mod` to reference the parent ConsoleKit module:
```
replace github.com/alexj212/consolekit => ../..
```

When `go.work` is present (workspace mode), the build system automatically includes both modules.

## Running

### Interactive Mode (Bubbletea TUI)

Run without arguments to start the interactive TUI:
```bash
./simple_bubbletea
```

### Command-Line Mode

Pass commands as arguments for non-interactive execution:
```bash
./simple_bubbletea version
./simple_bubbletea print "Hello from Bubbletea"
./simple_bubbletea date
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Execute command |
| `↑` | Previous command in history |
| `↓` | Next command in history |
| `Backspace` | Delete character |
| `Ctrl+U` | Clear input line |
| `Ctrl+C` or `Ctrl+D` | Quit |

## Example Session

```
╔═══════════════════════════════════════════════════════════╗
║         ConsoleKit Bubbletea REPL Example                ║
╚═══════════════════════════════════════════════════════════╝

simple > print "Hello from Bubbletea!"
Hello from Bubbletea!

simple > date
2026-01-31 04:30:00

simple > exit
Goodbye!
```

## Customization

### Colors

Colors are defined using Lipgloss styles in `main.go`:
```go
var (
    promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
    inputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
    outputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
    errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
    successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
    headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
)
```

### Output History

The TUI shows the last 20 lines of output. Adjust in `model.View()`:
```go
start := 0
if len(m.output) > 20 {  // Change 20 to your preferred value
    start = len(m.output) - 20
}
```

### Command History Limit

History is limited to last 100 commands. Adjust in `model.Update()`:
```go
if len(m.output) > 100 {  // Change 100 to your preferred value
    m.output = m.output[len(m.output)-100:]
}
```

## Using the BubbletteaAdapter

This example includes a `BubbletteaAdapter` that implements the `consolekit.DisplayAdapter` interface. You can use it in your own applications:

```go
// Create executor
executor, _ := consolekit.NewCommandExecutor("myapp", customizer)

// Create REPL handler with bubbletea adapter
handler := consolekit.NewREPLHandler(executor)

// Replace default adapter with bubbletea
adapter := NewBubbletteaAdapter("myapp")
handler.SetDisplayAdapter(adapter)

// Start REPL
handler.Start()
```

Alternatively, you can use Bubbletea directly without the adapter (as shown in `main.go`).

## Architecture

The example demonstrates the **Elm Architecture** pattern used by Bubbletea:

1. **Model**: Holds application state (input, output, history)
2. **Update**: Handles keyboard events and updates state
3. **View**: Renders the current state to the terminal

```go
type model struct {
    executor *consolekit.CommandExecutor  // ConsoleKit executor
    input    string                       // Current input line
    output   []string                     // Output history
    history  []string                     // Command history
    histIdx  int                          // History navigation index
    cursor   int                          // Cursor position
    quitting bool                         // Quit flag
}
```

## Comparison with Standard REPL

| Feature | Standard (reeflective) | Bubbletea |
|---------|----------------------|-----------|
| Tab Completion | ✅ Yes | ❌ No (TBD) |
| Syntax Highlighting | ✅ Yes | ❌ No (TBD) |
| Multi-line Input | ✅ Yes | ❌ No |
| Visual Polish | Basic | ✅ Excellent |
| Custom Styling | Limited | ✅ Full Control |
| Alt Screen Mode | No | ✅ Yes |
| Mouse Support | No | ✅ Possible |

## Future Enhancements

Potential improvements for this example:

- [ ] Tab completion support
- [ ] Multi-line command editing
- [ ] Mouse support for scrolling
- [ ] Syntax highlighting
- [ ] Split panes for output/input
- [ ] Command suggestions
- [ ] Search history (Ctrl+R)
- [ ] Vim/Emacs keybindings

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling library
- ConsoleKit - Command execution engine

## See Also

- [Simple Example](../simple/) - Standard REPL version
- [Bubbletea Examples](https://github.com/charmbracelet/bubbletea/tree/master/examples) - More Bubbletea patterns
- [ConsoleKit Documentation](../../CLAUDE.md) - Full feature documentation
