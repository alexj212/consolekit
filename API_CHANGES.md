# ConsoleKit API Changes

This document tracks all breaking changes to the ConsoleKit API.

---

## v0.8.0 - Modular Commands & embed.FS Pointer (2026-02-05)

**Status:** Current Release
**Breaking Changes:** Yes

### Modular Command System

ConsoleKit now provides fine-grained control over which command groups to include. See [COMMAND_GROUPS.md](COMMAND_GROUPS.md) for details.

**New Convenience Bundles:**
- `AddAllCmds(exec)` - All commands
- `AddStandardCmds(exec)` - Recommended defaults
- `AddMinimalCmds(exec)` - Core + variables + scripting
- `AddDeveloperCmds(exec)` - Standard + developer features
- `AddAutomationCmds(exec)` - Automation-optimized

**Individual Command Groups:**
Applications can now selectively include specific command groups (see COMMAND_GROUPS.md).

**Backward Compatibility:**
`AddBuiltinCommands()` still works and is equivalent to `AddAllCmds()`.

### embed.FS Pointer Change

`AddRun` and `LoadScript` now accept `*embed.FS` (pointer) instead of `embed.FS` (by value), allowing `nil` to be passed when no embedded scripts are needed.

**Migration:**

Before (v0.7.0):
```go
//go:embed *.run
var Scripts embed.FS

exec.AddCommands(consolekit.AddRun(exec, Scripts))
exec.Scripts = Scripts
```

After (v0.8.0):
```go
//go:embed *.run
var Scripts embed.FS

exec.AddCommands(consolekit.AddRun(exec, &Scripts))  // Add &
exec.Scripts = &Scripts                               // Add &

// Or for no embedded scripts:
exec.AddCommands(consolekit.AddRun(exec, nil))        // Pass nil
exec.Scripts = nil                                    // Set nil
```

**Affected Functions:**
- `AddRun(exec *CommandExecutor, scripts *embed.FS)` - was `scripts embed.FS`
- `LoadScript(scripts *embed.FS, cmd, filename)` - was `scripts embed.FS`
- `CommandExecutor.Scripts` field - now `*embed.FS` instead of `embed.FS`

**Rationale:** Allows applications to use external-only scripts without requiring an empty embed.FS. More explicit and flexible.

---

## v0.7.0 - API Naming Refactor (2026-01-31)

**Status:** Previous Release
**Breaking Changes:** Yes

This comprehensive API naming refactor improved clarity, consistency, and professionalism.

## Summary

All naming changes have been implemented to create a world-class, self-documenting API.

---

## 🔄 Type Renames

### Handlers

| Old Name | New Name | Rationale |
|----------|----------|-----------|
| `LocalREPLHandler` | `REPLHandler` | "Local" is redundant - REPL implies local interaction |

### Managers

| Old Name | New Name | Rationale |
|----------|----------|-----------|
| `NotifyManager` | `NotificationManager` | Full term for professionalism, consistency with other managers |

---

## 🔄 Field Renames (CommandExecutor)

| Old Name | New Name | Rationale |
|----------|----------|-----------|
| `Defaults` | `Variables` | Stores variables (like `@myvar`), not just defaults. Clear and accurate. |
| `TokenReplacers` | `VariableExpanders` | Describes function more accurately - these expand variables |

---

## 🔄 Method Renames

### CommandExecutor Methods

| Old Name | New Name | Rationale |
|----------|----------|-----------|
| `ExecuteLine(line, defs)` | `Execute(line, scope)` | Shorter, cleaner. "Line" is implicit. Parameter renamed for clarity. |
| `ExecuteLineWithContext(ctx, line, defs)` | `ExecuteWithContext(ctx, line, scope)` | Consistency with Execute. |
| `ReplaceDefaults(cmd, defs, input)` | `ExpandCommand(cmd, scope, input)` | Accurate - expands aliases, variables, and tokens (not just defaults). Standard terminology. |
| `ReplaceTokens(cmd, defs, input)` | `ExpandVariables(cmd, scope, input)` | Consistent terminology - focuses on variable expansion. |
| `AddAll()` | `AddBuiltinCommands()` | Self-documenting - makes it clear these are built-in commands. |
| `BuildRootCmd()` | `RootCmd()` | Simpler name, now returns `*cobra.Command` directly instead of function. |

---

## 🔄 Parameter Renames

| Old Name | New Name | Context | Rationale |
|----------|----------|---------|-----------|
| `defs` | `scope` | All execution methods | Clear semantic meaning - represents scoped variables for command execution |

**Affected signatures:**
```go
// Before
func Execute(line string, defs *safemap.SafeMap[string, string]) (string, error)
func ExecuteWithContext(ctx context.Context, line string, defs *safemap.SafeMap[string, string]) (string, error)
func ExpandCommand(cmd *cobra.Command, defs *safemap.SafeMap[string, string], input string) string
func ExpandVariables(cmd *cobra.Command, defs *safemap.SafeMap[string, string], input string) string

// After
func Execute(line string, scope *safemap.SafeMap[string, string]) (string, error)
func ExecuteWithContext(ctx context.Context, line string, scope *safemap.SafeMap[string, string]) (string, error)
func ExpandCommand(cmd *cobra.Command, scope *safemap.SafeMap[string, string], input string) string
func ExpandVariables(cmd *cobra.Command, scope *safemap.SafeMap[string, string], input string) string
```

---

## 📋 Complete New API

### Core Types

```go
type CommandExecutor struct {
    AppName             string
    Variables           *safemap.SafeMap[string, string]  // Was: Defaults
    VariableExpanders   []func(string) (string, bool)     // Was: TokenReplacers
    aliases             *safemap.SafeMap[string, string]

    JobManager          *JobManager
    Config              *Config
    LogManager          *LogManager
    TemplateManager     *TemplateManager
    NotificationManager *NotificationManager               // Was: NotifyManager
    HistoryManager      *HistoryManager

    FileHandler FileHandler
    Scripts     *embed.FS  // Pointer to allow nil (v0.8.0+)
    NoColor     bool
}
```

### Main Execution API

```go
// Command execution
func (e *CommandExecutor) Execute(line string, scope *safemap.SafeMap[string, string]) (string, error)
func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, line string, scope *safemap.SafeMap[string, string]) (string, error)

// Variable/command expansion
func (e *CommandExecutor) ExpandCommand(cmd *cobra.Command, scope *safemap.SafeMap[string, string], input string) string
func (e *CommandExecutor) ExpandVariables(cmd *cobra.Command, scope *safemap.SafeMap[string, string], input string) string

// Command registration
func (e *CommandExecutor) AddBuiltinCommands()                // Was: AddAll
func (e *CommandExecutor) AddCommands(func(*cobra.Command))
func (e *CommandExecutor) RootCmd() *cobra.Command            // Was: BuildRootCmd() returning function
```

### Transport Handlers

```go
// REPL handler (was LocalREPLHandler)
type REPLHandler struct { ... }
func NewREPLHandler(executor *CommandExecutor) *REPLHandler

// SSH handler
type SSHHandler struct { ... }
func NewSSHHandler(executor *CommandExecutor, addr string, hostKey ssh.Signer) *SSHHandler

// HTTP handler
type HTTPHandler struct { ... }
func NewHTTPHandler(executor *CommandExecutor, addr, user, pass string) *HTTPHandler
```

### Managers

```go
type NotificationManager struct { ... }  // Was: NotifyManager
func NewNotificationManager() *NotificationManager
```

---

## 🔧 Migration Guide

### For Application Code

**Before:**
```go
executor, _ := consolekit.NewCommandExecutor("myapp", func(exec *consolekit.CommandExecutor) error {
    exec.AddAll()
    exec.Defaults.Set("@myvar", "value")
    return nil
})

handler := consolekit.NewLocalREPLHandler(executor)
handler.Run()

// Executing commands
defs := safemap.New[string, string]()
defs.Set("@arg", "value")
output, _ := executor.ExecuteLine("print @arg", defs)
```

**After:**
```go
executor, _ := consolekit.NewCommandExecutor("myapp", func(exec *consolekit.CommandExecutor) error {
    exec.AddBuiltinCommands()                    // Was: AddAll
    exec.Variables.Set("@myvar", "value")         // Was: Defaults
    return nil
})

handler := consolekit.NewREPLHandler(executor)   // Was: NewLocalREPLHandler
handler.Run()

// Executing commands
scope := safemap.New[string, string]()           // Was: defs
scope.Set("@arg", "value")
output, _ := executor.Execute("print @arg", scope) // Was: ExecuteLine
```

### For Command Implementations

**Before:**
```go
func AddMyCommand(exec *CommandExecutor) func(*cobra.Command) {
    return func(rootCmd *cobra.Command) {
        cmd := &cobra.Command{
            Use: "mycommand",
            Run: func(cmd *cobra.Command, args []string) {
                value := exec.Defaults.Get("@myvar")
                expanded := exec.ReplaceTokens(cmd, nil, args[0])
                exec.ExecuteLine("print result", nil)
            },
        }
        rootCmd.AddCommand(cmd)
    }
}
```

**After:**
```go
func AddMyCommand(exec *CommandExecutor) func(*cobra.Command) {
    return func(rootCmd *cobra.Command) {
        cmd := &cobra.Command{
            Use: "mycommand",
            Run: func(cmd *cobra.Command, args []string) {
                value := exec.Variables.Get("@myvar")         // Was: Defaults
                expanded := exec.ExpandVariables(cmd, nil, args[0])  // Was: ReplaceTokens
                exec.Execute("print result", nil)              // Was: ExecuteLine
            },
        }
        rootCmd.AddCommand(cmd)
    }
}
```

### For Custom Expanders

**Before:**
```go
executor.TokenReplacers = append(executor.TokenReplacers, func(input string) (string, bool) {
    // Custom replacement logic
    return input, false
})
```

**After:**
```go
executor.VariableExpanders = append(executor.VariableExpanders, func(input string) (string, bool) {
    // Custom replacement logic
    return input, false
})
```

---

## 📊 Impact Analysis

### Files Modified

- **Core library:** 15 files
- **Command modules:** 20+ files
- **Handlers:** 3 files
- **Examples:** 6 files
- **Tests:** 3 files

### Breaking Changes

All changes are breaking changes to the public API. This is a major version bump (0.6.x → 0.7.0 or 1.0.0).

### Benefits

✅ **Clarity:** Method names accurately describe their function
✅ **Consistency:** Uniform naming patterns across the API
✅ **Self-documenting:** Names explain purpose without documentation
✅ **Professional:** Industry-standard terminology
✅ **Searchable:** Better IDE autocomplete and documentation search
✅ **Maintainable:** Easier for new contributors to understand

---

## ✅ Verification

All tests pass:
- ✅ Parser tests: 26/26 passing
- ✅ Executor tests: 9/9 passing
- ✅ Integration tests: All passing
- ✅ All examples build successfully

---

## 🎯 Terminology Standardization

| Concept | Standard Term | Usage |
|---------|---------------|-------|
| Variable storage | `Variables` | Global variables in executor |
| Variable scope | `scope` | Scoped/local variables for execution |
| Command text | `command` or `line` | Raw command string |
| Variable substitution | `expand` | Replacing `@var` with values |
| Built-in commands | `builtin` | Standard library commands |
| User-defined shortcuts | `alias` | Command aliases |
| Session context | `scope` | Per-execution variable bindings |

---

**Version:** 0.7.0
**Status:** Complete
**Compatibility:** Breaking changes - requires code updates
