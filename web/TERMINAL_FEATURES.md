# ConsoleKit Web Terminal - Feature Documentation

## Overview

The ConsoleKit Web Terminal provides a fully-featured terminal interface accessible through a web browser. It includes advanced text editing capabilities, keyboard shortcuts, and visual enhancements.

## Implemented Features

### 1. **Cursor Navigation** ✓

The terminal now supports full cursor movement within the input line:

- **Arrow Left/Right**: Move cursor character by character
- **Home / Ctrl+A**: Jump to beginning of line
- **End / Ctrl+E**: Jump to end of line

### 2. **Advanced Text Editing** ✓

Readline-style keyboard shortcuts for efficient text manipulation:

- **Backspace**: Delete character before cursor
- **Delete**: Delete character at cursor position
- **Ctrl+U**: Clear all text before cursor
- **Ctrl+K**: Clear all text after cursor (kill to end)
- **Ctrl+W**: Delete word before cursor
- **Tab**: Insert 4 spaces (configurable for auto-completion)

### 3. **Line Control** ✓

- **Ctrl+C**: Cancel current input and start fresh line (displays `^C`)
- **Ctrl+D**: Logout when line is empty, otherwise delete character ahead
- **Ctrl+L**: Clear screen while preserving current input

### 4. **Command History** ✓

Enhanced history management with persistence:

- **Up/Down Arrows**: Navigate through command history
- **Persistent Storage**: History saved to browser localStorage (up to 1000 commands)
- **Session Restoration**: Command history persists across browser sessions
- **Duplicate Prevention**: Each command added to history separately

### 5. **Paste Support** ✓

- **Multi-character paste detection**: Automatically handles clipboard paste
- **Character filtering**: Only printable characters (ASCII 32-126) are inserted
- **Cursor-aware**: Pasted text inserted at cursor position

### 6. **Visual Enhancements** ✓

#### Status Indicator
- **Real-time connection status**: Visual dot indicator (green = connected, red = disconnected)
- **Status text**: "Connected" / "Disconnected" label
- **Animated glow effect**: Subtle shadow on status dot

#### Terminal Styling
- **Professional theme**: Classic green-on-black terminal aesthetic
- **Modern fonts**: "Cascadia Code", "Fira Code", or fallback monospace
- **Better cursor**: Block-style blinking cursor
- **Selection highlighting**: Green-tinted selection background (#336633)
- **High scrollback**: 10,000 lines of scrollback buffer

#### UI Polish
- **Hover effects**: Button hover states for better UX
- **Glowing title**: Text shadow on login screen
- **Responsive layout**: Flexbox-based layout that adapts to screen size

### 7. **Welcome Screen** ✓

Informative welcome message with:
- Keyboard shortcut reference
- Command help pointers
- Professional branding

### 8. **Error Handling** ✓

Improved error feedback:
- **Connection errors**: Clear messages with automatic cleanup
- **Graceful degradation**: Status updates on disconnect
- **Delayed cleanup**: 1-second delay before showing login on disconnect (allows reading error messages)

## Keyboard Shortcuts Reference

### Navigation
| Shortcut | Action |
|----------|--------|
| Left Arrow | Move cursor left |
| Right Arrow | Move cursor right |
| Home | Beginning of line |
| End | End of line |
| Ctrl+A | Beginning of line |
| Ctrl+E | End of line |

### Editing
| Shortcut | Action |
|----------|--------|
| Backspace | Delete character before cursor |
| Delete | Delete character at cursor |
| Ctrl+U | Delete from start to cursor |
| Ctrl+K | Delete from cursor to end |
| Ctrl+W | Delete word before cursor |
| Tab | Insert 4 spaces |

### History
| Shortcut | Action |
|----------|--------|
| Up Arrow | Previous command |
| Down Arrow | Next command |

### Control
| Shortcut | Action |
|----------|--------|
| Enter | Execute command |
| Ctrl+C | Cancel current line |
| Ctrl+D | Logout (empty line) or Delete char |
| Ctrl+L | Clear screen |

## Technical Implementation

### Architecture

1. **xterm.js Integration**: Professional terminal emulation library
2. **Event Handling**:
   - `term.onKey()`: Keyboard event processing
   - `term.onData()`: Paste event detection
3. **State Management**:
   - `input`: Current input buffer
   - `cursorPos`: Cursor position within input
   - `history`: Command history array
   - `historyIndex`: Current position in history

### Key Functions

#### Line Management
- `redrawLine(term)`: Redraws entire line with cursor at correct position
- `clearLine(term)`: Clears line and shows prompt

#### Editing Functions
- `insertChar(char)`: Insert character at cursor position
- `deleteCharBefore()`: Backspace functionality
- `deleteCharAhead()`: Delete key functionality
- `deleteToEnd()`: Ctrl+K functionality
- `deleteToStart()`: Ctrl+U functionality
- `deleteWordBefore()`: Ctrl+W functionality

#### History Functions
- `loadHistory()`: Load from localStorage
- `saveHistory()`: Save to localStorage (debounced)
- `setInput(text)`: Set input and move cursor to end

#### Status Management
- `updateStatus(connected)`: Update connection indicator

### ANSI Escape Sequences Used

- `\x1b[2K` - Clear entire line
- `\x1b[D` - Move cursor left one position
- `\x1b[C` - Move cursor right one position
- `\x1b[<n>D` - Move cursor left n positions
- `\r` - Carriage return
- `\n` - Line feed

## Browser Compatibility

- **Modern Browsers**: Chrome, Firefox, Edge, Safari (latest versions)
- **localStorage Required**: For persistent history
- **WebSocket Support**: Required for terminal connection
- **ES6+ JavaScript**: Arrow functions, template literals, const/let

## Future Enhancement Possibilities

### Not Yet Implemented (Potential Additions)

1. **Tab Completion**: Server-side command/path completion
2. **Ctrl+R**: Reverse history search (like bash)
3. **Multi-line Editing**: Support for line continuation with `\`
4. **Syntax Highlighting**: Color-coded command syntax
5. **Search in Output**: Ctrl+F to search terminal output
6. **Terminal Resize**: Dynamic terminal resizing
7. **Themes**: Multiple color schemes
8. **Copy Mode**: Vim-style navigation in scrollback
9. **Bracketed Paste**: Distinguish pasted vs typed text
10. **History Search**: Incremental search through history

## Configuration

### Terminal Options (in `startTerminal()`)

```javascript
{
    theme: {
        background: '#000000',
        foreground: '#00ff00',
        cursor: '#00ff00',
        selection: '#336633',
    },
    cursorBlink: true,
    cursorStyle: 'block',
    fontFamily: '"Cascadia Code", "Fira Code", "Consolas", "Monaco", monospace',
    fontSize: 14,
    lineHeight: 1.2,
    scrollback: 10000,
    tabStopWidth: 4,
}
```

### Customizable Constants

- **History Limit**: 1000 commands (in `saveHistory()`)
- **Tab Width**: 4 spaces (in Tab key handler)
- **Disconnect Delay**: 1000ms (in socket close/error handlers)

## Security Considerations

- **localStorage**: History stored in browser (clear browser data to remove)
- **Session Management**: WebSocket authentication via cookies
- **Input Sanitization**: Only printable characters accepted
- **XSS Protection**: Terminal output is text-based (xterm.js handles escaping)

## Performance Notes

- **Debouncing**: History saved immediately on Enter (consider debouncing for larger histories)
- **Redraw Optimization**: Full line redraw on edits (acceptable for typical input lengths)
- **Memory**: 10,000 line scrollback may use significant memory on long sessions

## Testing Recommendations

### Manual Testing Checklist

- [ ] Arrow keys navigate correctly within input
- [ ] Ctrl+A/E jump to line boundaries
- [ ] Backspace/Delete work at cursor position
- [ ] Ctrl+U/K/W delete correct portions
- [ ] History navigation preserves cursor position
- [ ] Paste inserts at cursor position
- [ ] Ctrl+C clears line
- [ ] Ctrl+D logs out on empty line
- [ ] Ctrl+L clears screen
- [ ] Status indicator updates on connect/disconnect
- [ ] History persists across page reloads
- [ ] Long lines handle correctly
- [ ] Special characters display properly

## Troubleshooting

### Common Issues

1. **Cursor position desync**: Check that all edit operations update `cursorPos`
2. **History not persisting**: Verify localStorage is enabled in browser
3. **Status indicator not updating**: Check WebSocket event handlers
4. **Paste not working**: Verify `onData` handler is registered
5. **Keyboard shortcuts not working**: Check for browser extension conflicts

## Credits

- **xterm.js**: Terminal emulation library
- **xterm-addon-fit**: Terminal sizing addon
- **xterm-addon-web-links**: Clickable link detection

---

**Version**: 1.0
**Last Updated**: 2026-02-01
**Tested With**: Chrome 120+, Firefox 121+, Safari 17+
