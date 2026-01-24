# Prompt Display Issue in consolekit

## Problem Statement

When running applications built with consolekit (including the simple example), the initial prompt does not display until the user presses a key. The cursor appears but the prompt text is missing.

**Reproduction:**
```bash
cd /c/projects/consolekit
go run ./examples/simple/
# Cursor appears but no prompt is shown
# Press any key (e.g., up arrow) -> prompt appears correctly
```

## What We've Tried

1. **Recent Changes (CONFIRMED to make it worse - REVERTED):**
   - Added `c.app.NewlineBefore = true` in cli.go:AppBlock()
   - Added PreReadlineHook to sync stdout/stderr
   - These changes are in commit: 580d560 "fix: ensure terminal is in clean state before readline starts"
   - **REVERTED** - These changes made even the simple example fail
   - Simple example had the issue BEFORE these changes too

2. **Previous attempts:**
   - Running cli.Run() in goroutine vs synchronously
   - Printing newlines before cli.Run()
   - Using stdout vs stderr for initialization messages
   - Flushing stdout/stderr explicitly
   - Simplifying the prompt function to avoid slow lookups

## Environment

- **OS**: Linux 6.6.87.2-microsoft-standard-WSL2 (WSL2 on Windows)
- **Go**: 1.25.3
- **consolekit**: v0.5.68 (current working version)
- **Dependencies**:
  - github.com/reeflective/console v0.1.25
  - github.com/reeflective/readline v1.1.4

## Key Files

- `/c/projects/consolekit/cli.go` - Main CLI implementation, AppBlock() method around line 635
- `/c/projects/consolekit/examples/simple/main.go` - Minimal example that reproduces the issue
- Recent git commits:
  - 580d560 - The problematic changes
  - 0ddeeb1 - Previous working version (maybe?)

## What Works

- The simple example USED to work correctly (showed prompt immediately)
- After pressing any key, the prompt displays correctly
- All functionality works fine, just the initial display is broken

## Technical Details

The AppBlock() method:
1. Creates console.New()
2. Sets up shell config
3. Configures menu and prompt
4. Adds PreReadlineHooks (recently added)
5. Calls c.app.Start()

The underlying library (reeflective/console) has:
- PreReadlineHooks - executed before readline starts
- NewlineBefore/NewlineAfter - control newlines around prompts
- Uses github.com/reeflective/readline for terminal handling

## Screenshot Evidence

User has screenshot showing:
```
Configuration: qa
              █  (cursor here, but no prompt visible)
```

After keypress, prompt appears correctly as "qa >"

## Your Task

1. **Investigate** why the prompt doesn't display on first readline cycle
2. **Test** by reverting commit 580d560 to see if that's the culprit
3. **Fix** the underlying issue - possibly related to:
   - Terminal state after initialization
   - Readline library not triggering initial refresh
   - Buffering/flushing issues
   - The NewlineBefore setting interfering with initial display
4. **Verify** the fix works with both:
   - Simple example: `go run ./examples/simple/`
   - Heavy initialization (genrmi2): network calls, goroutines, etc.

## Suggested Approach

1. First revert commit 580d560 and test if simple example works
2. If that fixes it, understand what went wrong with those changes
3. Look at reeflective/console and reeflective/readline documentation/source
4. Check if there's a method to force initial prompt display/refresh
5. Consider if NewlineBefore should be false by default
6. Test any fix with both simple and complex applications

## Success Criteria

- Running `go run ./examples/simple/` shows prompt immediately
- No keypress required to see the prompt
- Works on WSL2 Linux environment
- Doesn't break existing functionality

## Additional Context

The consolekit library is used by genrmi2, a complex application with:
- Heavy initialization (NATS, Redis, Registry connections)
- Background goroutines
- Network I/O during startup

Both simple and complex apps should show prompt immediately after cli.Run() is called.
