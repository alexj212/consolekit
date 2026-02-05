# ConsoleKit Migration Guide

This guide helps you migrate between major versions of ConsoleKit.

---

## Migrating to v0.8.0 (Current)

**Release Date:** 2026-02-05
**Breaking Changes:** Yes (embed.FS pointer change)

### What Changed

1. **embed.FS is now a pointer** - `AddRun` and related functions now accept `*embed.FS` instead of `embed.FS`
2. **Modular command system** - New fine-grained control over command inclusion
3. **nil support** - Can now pass `nil` to `AddRun` for external-only scripts

### Quick Migration Checklist

- [ ] Update all `AddRun(exec, Scripts)` to `AddRun(exec, &Scripts)`
- [ ] Update all `exec.Scripts = Scripts` to `exec.Scripts = &Scripts`
- [ ] Consider using new modular command groups (optional)
- [ ] Test script execution with both embedded and external scripts

### Detailed Migration

#### 1. Script Execution Changes

**Before (v0.7.x):**
```go
//go:embed *.run
var Scripts embed.FS

func main() {
    customizer := func(exec *consolekit.CommandExecutor) error {
        exec.Scripts = Scripts                           // By value
        exec.AddCommands(consolekit.AddRun(exec, Scripts)) // By value
        return nil
    }

    executor, _ := consolekit.NewCommandExecutor("myapp", customizer)
    handler := consolekit.NewREPLHandler(executor)
    handler.Run()
}
```

**After (v0.8.0):**
```go
//go:embed *.run
var Scripts embed.FS

func main() {
    customizer := func(exec *consolekit.CommandExecutor) error {
        exec.Scripts = &Scripts                            // Add & (pointer)
        exec.AddCommands(consolekit.AddRun(exec, &Scripts)) // Add & (pointer)
        return nil
    }

    executor, _ := consolekit.NewCommandExecutor("myapp", customizer)
    handler := consolekit.NewREPLHandler(executor)
    handler.Run()
}
```

#### 2. External-Only Scripts (New Capability)

**v0.8.0 adds the ability to use external scripts without embedded files:**

```go
func main() {
    customizer := func(exec *consolekit.CommandExecutor) error {
        // No embedded scripts needed
        exec.AddCommands(consolekit.AddRun(exec, nil))  // Pass nil
        return nil
    }

    executor, _ := consolekit.NewCommandExecutor("myapp", customizer)
    handler := consolekit.NewREPLHandler(executor)
    handler.Run()
}
```

**Usage:**
```bash
./myapp run /path/to/script.run  # ✅ Works
./myapp run @embedded.run        # ❌ Error: "No embedded scripts available"
```

#### 3. Modular Command Selection (Optional)

**v0.8.0 introduces fine-grained command group selection:**

**Before (v0.7.x) - All or nothing:**
```go
customizer := func(exec *consolekit.CommandExecutor) error {
    exec.AddBuiltinCommands()  // Adds ALL commands
    exec.AddCommands(consolekit.AddRun(exec, Scripts))
    return nil
}
```

**After (v0.8.0) - Selective inclusion:**
```go
customizer := func(exec *consolekit.CommandExecutor) error {
    // Option 1: Use convenience bundles
    exec.AddCommands(consolekit.AddStandardCmds(exec))  // Recommended defaults

    // Option 2: Pick specific groups
    exec.AddCommands(consolekit.AddCoreCmds(exec))
    exec.AddCommands(consolekit.AddVariableCmds(exec))
    exec.AddCommands(consolekit.AddNetworkCmds(exec))

    // Script execution (still separate)
    exec.AddCommands(consolekit.AddRun(exec, &Scripts))

    return nil
}
```

**Benefits:**
- Smaller binary size (only include what you need)
- Faster startup (fewer commands to register)
- Better security (exclude dangerous commands like `osexec`)
- Clearer intent (explicit about capabilities)

See [COMMAND_GROUPS.md](COMMAND_GROUPS.md) for complete documentation.

### LoadScript Function Changes

If you're using `LoadScript` directly:

**Before (v0.7.x):**
```go
commands, err := consolekit.LoadScript(scripts, cmd, filename)
```

**After (v0.8.0):**
```go
commands, err := consolekit.LoadScript(&scripts, cmd, filename)  // Add &
```

### Error Messages

**New error messages when using `@filename` without embedded scripts:**

```
Error: No embedded scripts available (scripts parameter was nil)
Use filesystem paths instead of @ prefix
```

This helps catch configuration errors early.

### Testing Your Migration

**1. Test embedded script execution:**
```bash
./myapp run @test.run  # Should work if you passed &Scripts
```

**2. Test external script execution:**
```bash
./myapp run /path/to/test.run  # Should always work
```

**3. Test error handling:**
```bash
# With nil scripts:
./myapp run @test.run  # Should show clear error message
```

**4. Run your test suite:**
```bash
go test ./...
```

### Common Migration Issues

**Issue 1: "cannot use Scripts (variable of type embed.FS) as *embed.FS"**

**Fix:** Add `&` before the variable:
```go
// Before
AddRun(exec, Scripts)

// After
AddRun(exec, &Scripts)  // Add &
```

**Issue 2: "No embedded scripts available" when trying to run `@script.run`**

**Cause:** You passed `nil` to `AddRun` but are trying to use embedded scripts.

**Fix:** Pass `&Scripts` instead of `nil`:
```go
// Wrong
exec.AddCommands(consolekit.AddRun(exec, nil))

// Correct
exec.AddCommands(consolekit.AddRun(exec, &Scripts))
```

**Issue 3: Application builds but scripts don't work**

**Check:**
1. Did you update `exec.Scripts = &Scripts`?
2. Did you update `AddRun(exec, &Scripts)`?
3. Are your script files actually embedded with `//go:embed`?

### Backwards Compatibility

**What still works:**
- `AddBuiltinCommands()` - Still works, equivalent to `AddAllCmds()`
- All command functionality - No behavior changes
- Script syntax - Scripts work exactly the same way
- All other APIs - Only script-related functions changed

**What doesn't work:**
- Passing `embed.FS` by value to `AddRun` - Must use pointer
- Assigning `embed.FS` to `CommandExecutor.Scripts` - Must use pointer

---

## Migrating to v0.7.0

**Release Date:** 2026-01-31
**Breaking Changes:** Yes (API naming refactor)

See [API_CHANGES.md](API_CHANGES.md) for complete v0.7.0 migration guide.

**Key changes in v0.7.0:**
- `LocalREPLHandler` → `REPLHandler`
- `ExecuteLine` → `Execute`
- `Defaults` → `Variables`
- `TokenReplacers` → `VariableExpanders`
- `AddAll()` → `AddBuiltinCommands()`

---

## Version Support

| Version | Status | Support End | Notes |
|---------|--------|-------------|-------|
| v0.8.x | Current | Active | Modular commands, pointer API |
| v0.7.x | Previous | 2026-06-01 | API naming refactor |
| v0.6.x | Deprecated | 2026-03-01 | Old naming conventions |

---

## Getting Help

- **Documentation:** See [CLAUDE.md](CLAUDE.md) for complete project documentation
- **Command Groups:** See [COMMAND_GROUPS.md](COMMAND_GROUPS.md) for modular commands
- **API Changes:** See [API_CHANGES.md](API_CHANGES.md) for detailed API history
- **Examples:** See `examples/` directory for working code samples
- **Issues:** Report problems at https://github.com/alexj212/consolekit/issues

---

## Migration Tools

### Find and Replace

Use these commands to help with migration:

```bash
# Find all AddRun calls that need updating
grep -r "AddRun.*Scripts\)" --include="*.go"

# Find all Scripts assignments
grep -r "exec.Scripts = " --include="*.go"

# Find all LoadScript calls
grep -r "LoadScript(" --include="*.go"
```

### Automated Migration Script

```bash
#!/bin/bash
# migrate-v0.8.sh - Migrate to v0.8.0

echo "Migrating to ConsoleKit v0.8.0..."

# Update AddRun calls
find . -name "*.go" -type f -exec sed -i \
    's/AddRun(exec, \([A-Za-z]*\))/AddRun(exec, \&\1)/g' {} \;

# Update Scripts assignments
find . -name "*.go" -type f -exec sed -i \
    's/exec.Scripts = \([A-Za-z]*\)$/exec.Scripts = \&\1/g' {} \;

echo "Migration complete. Please review changes and test thoroughly."
```

**Warning:** Always review automated changes before committing!

---

**Last Updated:** 2026-02-05
**Current Version:** v0.8.0
