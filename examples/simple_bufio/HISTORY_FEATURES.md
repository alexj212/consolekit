# History and Arrow Key Support

This document describes the history and line editing features added to `simple_bufio`.

## Overview

The `simple_bufio` example now includes a **custom terminal handler** that provides:
- Command history navigation
- Full line editing capabilities
- Automatic TTY detection

## Implementation Details

### Terminal Raw Mode

When running in a terminal (TTY), the program enables **raw mode** using `golang.org/x/term`:

```go
oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
defer term.Restore(int(os.Stdin.Fd()), oldState)
```

**Raw mode** means:
- Characters are read one at a time (not line-buffered)
- No automatic echo (program must echo back)
- Full control over terminal behavior
- Can detect special keys like arrows

### Automatic Fallback

The program detects if it's running in a terminal or receiving piped input:

```go
if !term.IsTerminal(int(os.Stdin.Fd())) {
    // Fallback to simple line-buffered mode
    runSimpleMode(executor)
}
```

This means:
- **Interactive terminal** → Full history and editing features
- **Piped input** → Simple line-by-line reading
- **Scripts/automation** → Works seamlessly

## Keyboard Handling

### Arrow Keys (ANSI Escape Sequences)

Arrow keys send **escape sequences** instead of regular characters:

| Key | Sequence | Hex |
|-----|----------|-----|
| ↑ Up | `ESC [ A` | `0x1b 0x5b 0x41` |
| ↓ Down | `ESC [ B` | `0x1b 0x5b 0x42` |
| → Right | `ESC [ C` | `0x1b 0x5b 0x43` |
| ← Left | `ESC [ D` | `0x1b 0x5b 0x44` |
| Delete | `ESC [ 3 ~` | `0x1b 0x5b 0x33 0x7e` |

The `readLineWithHistory()` function detects these sequences byte-by-byte:

```go
if len(escapeSeq) == 3 && escapeSeq[0] == 0x1b && escapeSeq[1] == '[' {
    switch escapeSeq[2] {
    case 'A': // Up arrow - navigate history backward
    case 'B': // Down arrow - navigate history forward
    case 'C': // Right arrow - move cursor right
    case 'D': // Left arrow - move cursor left
    }
}
```

### Control Characters

| Key | Byte | Action |
|-----|------|--------|
| Enter | `0x0d` or `0x0a` | Execute command |
| Backspace | `0x7f` or `0x08` | Delete before cursor |
| Ctrl+C | `0x03` | Cancel line |
| Ctrl+D | `0x04` | Exit (empty line) |
| Ctrl+L | `0x0c` | Clear screen |

## History Management

### Data Structure

```go
history := []string{}     // Command history (in-memory)
historyIndex := -1        // Current position in history
```

### Adding to History

```go
// Add command to history (skip duplicates)
if len(history) == 0 || history[len(history)-1] != line {
    history = append(history, line)
}
historyIndex = len(history)  // Reset to end
```

### Navigating History

**Up Arrow (↑):**
```go
if historyIndex > 0 {
    historyIndex--
    clearLine(prompt, line, cursor)
    line = []rune(history[historyIndex])
    cursor = len(line)
    fmt.Print(string(line))
}
```

**Down Arrow (↓):**
```go
if historyIndex < len(history)-1 {
    historyIndex++
    clearLine(prompt, line, cursor)
    line = []rune(history[historyIndex])
    cursor = len(line)
    fmt.Print(string(line))
} else {
    // At end, clear to new line
    historyIndex = len(history)
    clearLine(prompt, line, cursor)
    line = []rune{}
    cursor = 0
}
```

## Line Editing

### Cursor Management

The program maintains:
- `line []rune` - Current input buffer
- `cursor int` - Current cursor position (0 to len(line))

### Insert Character

```go
if cursor < len(line) {
    // Insert in middle
    line = append(line[:cursor], append([]rune{rune(b)}, line[cursor:]...)...)
    cursor++
    // Redraw from cursor to end
    fmt.Print(string(line[cursor-1:]))
    // Move cursor back
    fmt.Printf("\x1b[%dD", len(line)-cursor)
} else {
    // Append at end
    line = append(line, rune(b))
    cursor++
    fmt.Printf("%c", b)
}
```

### Delete Character (Backspace)

```go
if cursor > 0 {
    // Remove character before cursor
    line = append(line[:cursor-1], line[cursor:]...)
    cursor--
    // Redraw line
    fmt.Print("\b" + string(line[cursor:]) + " ")
    // Move cursor back
    fmt.Printf("\x1b[%dD", len(line)-cursor+1)
}
```

### Clear Line

```go
fmt.Print("\r")        // Move to start
fmt.Print("\x1b[K")    // Clear to end of line
fmt.Print(prompt)      // Reprint prompt
```

## ANSI Escape Codes Used

| Code | Description |
|------|-------------|
| `\r` | Carriage return (move to column 0) |
| `\n` | Line feed (move down one line) |
| `\r\n` | Both (required in raw mode) |
| `\b` | Backspace (move left one) |
| `\x1b[K` | Clear from cursor to end of line |
| `\x1b[2J` | Clear entire screen |
| `\x1b[H` | Move cursor to home (0,0) |
| `\x1b[C` | Move cursor right |
| `\x1b[D` | Move cursor left |
| `\x1b[%dD` | Move cursor left N positions |

## Differences from simple_console

| Feature | simple_bufio | simple_console |
|---------|--------------|----------------|
| Implementation | Custom (250 lines) | reeflective/console library |
| History Storage | In-memory only | File-based (~/.app.history) |
| Tab Completion | Not implemented | Full Cobra integration |
| Multi-line | Not supported | Supported (backslash continuation) |
| Syntax Highlighting | No | Yes |
| Colors | No | Yes |
| Home/End Keys | Not implemented | Supported |
| Kill/Yank | Not implemented | Supported |
| Learning Value | ⭐⭐⭐⭐⭐ (see how it works) | ⭐⭐ (hidden in library) |

## Performance

- **Memory**: History grows with each command (not limited)
- **CPU**: Minimal - character-by-character reading is fast
- **Latency**: Near-instant key response

## Security Considerations

- History is **not encrypted** (in-memory only)
- Terminal raw mode gives **full control** to the program
- **No command validation** before adding to history
- History contains commands **as typed** (may include secrets)

## Future Enhancements

Possible improvements:
- [ ] Persistent history (save to file like simple_console)
- [ ] History size limit
- [ ] History search (Ctrl+R)
- [ ] Tab completion
- [ ] Home/End key support
- [ ] Kill/Yank (Ctrl+K/Ctrl+Y)
- [ ] Multi-line editing
- [ ] Syntax highlighting
- [ ] Color output

## Educational Value

This implementation demonstrates:
- How terminals work (raw mode vs cooked mode)
- ANSI escape sequence handling
- Line editing algorithms
- History management
- Cross-platform terminal handling (golang.org/x/term)

**Great for learning** how REPL shells are built!

## References

- [ANSI Escape Codes](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [golang.org/x/term package](https://pkg.go.dev/golang.org/x/term)
- [VT100 Terminal Codes](https://vt100.net/docs/vt100-ug/chapter3.html)
