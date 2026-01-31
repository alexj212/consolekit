#!/bin/bash

echo "Testing simple_bufio with history support"
echo "=========================================="
echo ""

# Test 1: Command-line mode (should still work)
echo "Test 1: Command-line execution"
./simple-bufio print "Hello from command line"
echo ""

# Test 2: Piped input (should use simple mode)
echo "Test 2: Piped input (falls back to simple mode)"
echo -e "print 'From pipe'\ndate\nexit" | ./simple-bufio
echo ""

# Test 3: Interactive mode instructions
echo "Test 3: Interactive mode with history"
echo "Run: ./simple-bufio"
echo ""
echo "Then try:"
echo "  1. Type: print \"Hello\""
echo "  2. Press Enter"
echo "  3. Type: date"
echo "  4. Press Enter"
echo "  5. Press ↑ arrow (should show 'date')"
echo "  6. Press ↑ arrow again (should show 'print \"Hello\"')"
echo "  7. Press ↓ arrow (should show 'date')"
echo "  8. Try editing: press ← → to move cursor, type to insert"
echo "  9. Type: exit"
echo ""
echo "Keyboard shortcuts:"
echo "  ↑/↓       - Navigate history"
echo "  ←/→       - Move cursor"
echo "  Backspace - Delete before cursor"
echo "  Delete    - Delete at cursor"
echo "  Ctrl+C    - Cancel line"
echo "  Ctrl+D    - Exit (empty line)"
echo "  Ctrl+L    - Clear screen"
