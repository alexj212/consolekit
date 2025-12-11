# ConsoleKit - Feature Recommendations & Refactoring

**Target Audience:** Technical users, internal tools, management commands, system connections

---

## 📋 Implementation Status Summary

### ✅ Completed (Phase 1 + Phase 4 + Quick Wins)

**Phase 1 - Core Infrastructure:**
- ✅ Job Management System (jobs.go, jobcmds.go)
- ✅ Enhanced Variables (varcmds.go with let, unset, vars, inc, dec)
- ✅ Configuration System (config.go, configcmds.go with TOML support)

**Phase 4 - Quality & Performance:**
- ✅ Error Handling (errors.go with structured CLIError type)
- ✅ Testing Infrastructure (cli_test.go with 8 tests, jobs_test.go with 6 tests)

**Quick Wins:**
- ✅ `clear` alias for cls command
- ✅ `which` command (check if command is alias/variable/built-in)
- ✅ `time` command (measure execution time)
- ✅ `.` command (source scripts in current context)

### 🔴 Next Priority (Phase 2)
- Session Management (save/restore REPL sessions)
- Connection Management (manage remote system connections)
- Logging & Audit Trail (command logging for compliance)

**Files Added:**
- jobs.go, jobcmds.go (job management)
- config.go, configcmds.go (configuration)
- varcmds.go (enhanced variables)
- utilcmds.go (which, time, dot commands)
- errors.go (structured errors)
- cli_test.go, jobs_test.go (comprehensive testing)

---

## 🏗️ High Priority Refactorings

### 1. **Job/Process Management System**
**Status:** Critical missing feature for internal tooling

Currently spawned processes (`spawn`, `osexec --background`, `run --spawn`) are not tracked.

**Recommended Implementation:**
```go
// jobs.go
package consolekit

type Job struct {
    ID          int
    Command     string
    StartTime   time.Time
    Status      string // "running", "completed", "failed"
    PID         int
    Output      *bytes.Buffer
    Error       error
    Cancel      context.CancelFunc
}

type JobManager struct {
    jobs    map[int]*Job
    nextID  int
    mu      sync.RWMutex
}

func (jm *JobManager) Add(cmd string, cancel context.CancelFunc) int
func (jm *JobManager) Get(id int) (*Job, bool)
func (jm *JobManager) List() []*Job
func (jm *JobManager) Kill(id int) error
func (jm *JobManager) Wait(id int) error
func (jm *JobManager) Logs(id int) string
```

**Commands to add:**
```bash
jobs                 # List all background jobs
jobs -v              # Verbose: show full command, output preview
job 5                # Show details for job 5
job 5 logs           # Show full output for job 5
job 5 kill           # Kill job 5
job 5 wait           # Wait for job 5 to complete
killall              # Kill all background jobs
```

**Benefits:**
- Track spawned processes
- View output from background jobs
- Kill hung processes
- Wait for completion
- See what's running

---

### 2. **Variables System Enhancement**
**Current:** Basic token replacement with `@varname`

**Recommended:** Full variable system with scoping

```go
// variables.go
type VariableScope struct {
    parent    *VariableScope
    vars      *safemap.SafeMap[string, string]
    functions map[string]func([]string) (string, error)
}

// Commands:
let name=value           # Set variable (more intuitive than "set")
unset name               # Remove variable
vars                     # List all variables
vars --export           # Export as shell script
vars --json             # Export as JSON

# Advanced features:
let counter=0
let counter=$((counter + 1))    # Arithmetic
let result=$(print "hello")     # Command substitution
let path="$HOME/data"           # Environment variable expansion
```

**Functions:**
```bash
# Define custom functions
function deploy() {
    print "Deploying to $1"
    osexec "deploy.sh $1"
}

# Call functions
deploy production
```

---

### 3. **Session Management**
**Purpose:** Save/restore REPL sessions for reproducibility

```go
// session.go
type Session struct {
    Name      string
    Timestamp time.Time
    Variables map[string]string
    Aliases   map[string]string
    History   []string
}

// Commands:
session save mywork          # Save current session
session load mywork          # Restore session
session list                 # List saved sessions
session delete mywork        # Delete session
session export mywork.json   # Export to JSON
session import mywork.json   # Import from JSON
```

**Benefits:**
- Resume work after restart
- Share configurations between team members
- Snapshot before risky operations
- Reproducible debugging sessions

---

### 4. **Connection Management**
**Purpose:** Manage multiple connections to remote systems

```go
// connections.go
type Connection struct {
    Name     string
    Type     string // "ssh", "http", "grpc", "nats", etc.
    Endpoint string
    Config   map[string]interface{}
    Client   interface{}
    Active   bool
}

type ConnectionManager struct {
    connections map[string]*Connection
    active      string
    mu          sync.RWMutex
}

// Commands:
conn add prod ssh://user@server:22          # Add connection
conn add api http://api.example.com          # Add HTTP endpoint
conn list                                     # List all connections
conn use prod                                 # Switch active connection
conn test prod                                # Test connection
conn remove prod                              # Remove connection
conn info                                     # Show active connection info

# Use connections in commands:
@conn:prod exec "systemctl status nginx"     # Execute on connection
http @conn:api/users                          # HTTP request to endpoint
```

**Integration with existing genrmi2:**
```bash
# Connect to NATS endpoint
conn add qa.user nats://qa.svc.user.1
conn use qa.user
remote 'groovy exec p.getByUid(116869280)'   # Uses active connection
```

---

### 5. **Enhanced Pipeline Features**

#### Output Capture Variable
```bash
# Capture output to variable
let result=$(env | grep PATH)
print "Found: $result"

# Multi-line capture
let config=$(cat config.json | grep "port")
```

#### Tee Command (Write AND Display)
```bash
# Currently: redirect only writes
env | grep PATH > output.txt   # Writes to file, also displays

# Add explicit tee:
env | grep PATH | tee output.txt  # More explicit
env | tee all.txt | grep PATH | tee filtered.txt
```

#### Background Pipeline
```bash
# Run entire pipeline in background
spawn "http api.com/data | grep error | tee errors.log"
job 1 wait  # Wait for completion
```

---

### 6. **Conditional Execution & Control Flow**

**Current `if` command is basic. Enhance with:**

```bash
# Inline conditionals
command && success_cmd || failure_cmd

# Multi-line if blocks (via scripts)
if env | grep -q "PROD"; then
    print "Production environment"
    osexec "production-deploy.sh"
else
    print "Development environment"
    osexec "dev-deploy.sh"
fi

# Case statement
case $ENV in
    prod) print "Production" ;;
    qa)   print "QA" ;;
    *)    print "Unknown" ;;
esac

# While loops
let i=0
while [ $i -lt 5 ]; do
    print "Count: $i"
    let i=$((i + 1))
done
```

---

### 7. **Configuration File System**

**Current:** Aliases and history persisted separately

**Recommended:** Unified config system

```toml
# ~/.myapp/config.toml

[settings]
history_size = 10000
prompt = "%s > "
color = true
pager = "less -R"

[aliases]
ll = "print 'listing files'"
deploy = "osexec 'deploy.sh'"

[variables]
env = "qa"
region = "us-east-1"

[connections]
[connections.prod]
type = "ssh"
host = "server.example.com"
user = "admin"

[hooks]
on_startup = "print 'Welcome!'; date"
on_exit = "print 'Goodbye!'"
before_command = ""
after_command = ""
```

**Commands:**
```bash
config set settings.history_size 5000
config get settings.history_size
config edit                              # Open in $EDITOR
config reload                            # Reload from file
```

---

### 8. **Logging & Audit Trail**

**Purpose:** Debug issues, compliance, security auditing

```go
// logging.go
type AuditLog struct {
    Timestamp time.Time
    User      string
    Command   string
    Output    string
    Duration  time.Duration
    ExitCode  int
    Success   bool
}

// Commands:
log enable                    # Enable command logging
log disable                   # Disable command logging
log show                      # Show recent logs
log show --last 100           # Show last 100 commands
log show --failed             # Show only failed commands
log show --search "deploy"    # Search logs
log export --since "2025-12-10" --format json > audit.json
log clear                     # Clear logs
```

**Auto-logging options:**
```bash
# In config
[logging]
enabled = true
log_file = "~/.myapp/audit.log"
log_success = true
log_failures = true
max_size_mb = 100
retention_days = 90
```

---

### 9. **Interactive Prompts & Confirmations**

**For destructive operations:**

```go
// prompt.go
func (c *CLI) Confirm(message string) bool
func (c *CLI) Prompt(message string) string
func (c *CLI) Select(message string, options []string) string
func (c *CLI) MultiSelect(message string, options []string) []string

// Example usage:
var deployCmd = &cobra.Command{
    Use: "deploy",
    Run: func(cmd *cobra.Command, args []string) {
        if !cli.Confirm("Deploy to production?") {
            cmd.Println("Cancelled")
            return
        }
        // ... deploy logic
    },
}
```

**Commands with auto-confirm:**
```bash
deploy --yes           # Skip confirmation
deploy --dry-run       # Show what would happen
```

---

### 10. **Tab Completion Enhancements**

**Current:** Cobra provides basic completion

**Recommendations:**
```go
// Custom completers for dynamic data
func (c *CLI) RegisterCompleter(command string, completer func() []string)

// Example:
cli.RegisterCompleter("conn", func() []string {
    return connectionManager.ListNames()
})

// Smart completions:
deploy <TAB>       # Shows: production, staging, qa
conn use <TAB>     # Shows available connections
job <TAB>          # Shows running job IDs
```

---

## 🚀 New Feature Modules

### 11. **Template System**

**Purpose:** Generate scripts/configs from templates

```bash
# template.go
template list                           # List available templates
template show deployment                # Show template content
template exec deployment --env=prod     # Execute with variables
template create mytemplate              # Create new template

# Templates use Go text/template syntax
# deployment.tmpl:
print "Deploying to {{.Env}}"
let region="{{.Region}}"
http {{.ApiEndpoint}}/deploy
```

---

### 12. **Plugin System**

**Allow external Go plugins:**

```go
// plugin.go
type Plugin interface {
    Name() string
    Commands() []*cobra.Command
    Initialize(*CLI) error
}

// Commands:
plugin list                    # List installed plugins
plugin load myplug.so          # Load plugin
plugin unload myplug           # Unload plugin
plugin info myplug             # Show plugin info
```

---

### 13. **Data Manipulation Commands**

**For working with structured data:**

```bash
# JSON manipulation
json get users.json '.users[0].name'
json set users.json '.users[0].active' true
json query users.json '[].name'          # JQ-style queries

# CSV manipulation
csv read data.csv --header
csv filter data.csv 'age > 30'
csv sort data.csv --by age --desc

# YAML manipulation
yaml get config.yaml 'database.host'
yaml set config.yaml 'database.port' 5432

# Table formatting
http api.com/users | json | table --columns "id,name,email"
```

---

### 14. **Watch Command**

**Monitor command output:**

```bash
watch "http api.com/status" --interval 5     # Refresh every 5 seconds
watch "job 5" --interval 1                   # Monitor job progress
watch "conn test prod" --until success       # Wait for success
```

---

### 15. **Notification System**

**Alert on completion:**

```bash
# Long running command
spawn "deploy.sh"
job 1 notify --email admin@example.com       # Email on completion
job 1 notify --webhook https://hooks.slack.com/...

# Built-in commands
notify "Deployment complete" --level success
notify "Error occurred" --level error
```

---

### 16. **Metrics & Performance Tracking**

```bash
stats show                       # Show session statistics
stats show --commands            # Most used commands
stats show --timing              # Slowest commands
stats reset                      # Reset statistics

# Auto-timing
time command args                # Time single command
profile enable                   # Enable profiling
profile report                   # Show performance data
```

---

### 17. **Batch Operations**

```bash
# Run command for multiple inputs
batch "conn use {}; remote status" --inputs "qa.user.1,qa.user.2,qa.user.3"

# From file
batch "deploy.sh {}" --file servers.txt

# Parallel execution
batch "http {}/health" --inputs "api1.com,api2.com" --parallel 5
```

---

### 18. **Clipboard Integration**

```bash
# Copy output to clipboard
env | grep PATH | clip
http api.com/data | json | clip

# Paste from clipboard
let url=$(paste)
http $url
```

---

## 🔧 Code Quality Refactorings

### ✅ 19. **Error Handling Improvements** (COMPLETE)

**Implemented:** Structured error type with error wrapping

```go
// errors.go
type CLIError struct {
    Command   string
    Message   string
    Cause     error
    Timestamp time.Time
}

func (e *CLIError) Error() string
func (e *CLIError) Unwrap() error

// Use structured errors
return &CLIError{
    Command: "deploy",
    Message: "deployment failed",
    Cause:   err,
}
```

**Implemented in:** errors.go

---

### ✅ 20. **Testing Infrastructure** (PHASE 1 COMPLETE)

**Implemented:**
- ✅ Integration tests for piping (TestExecuteLinePiping)
- ✅ Command execution tests (TestExecuteLinePrint)
- ✅ Token replacement tests (TestTokenReplacement)
- ✅ Alias tests (TestAliasReplacement)
- ✅ Command chaining tests (TestCommandChaining)
- ✅ Recursion protection tests (TestRecursionProtection)
- ✅ Environment variable tests (TestEnvTokenReplacement)
- ✅ Job management tests (jobs_test.go - 6 tests)

```go
// cli_test.go - 8 comprehensive tests
func TestExecuteLinePiping(t *testing.T) {
    cli, _ := NewCLI("test", func(c *CLI) error {
        AddBaseCmds(c)
        AddMisc()
        return nil
    })
    output, err := cli.ExecuteLine("print \"line1\\nline2\\nline3\" | grep line2", nil)
    // ... assertions
}
```

**Implemented in:** cli_test.go, jobs_test.go

**TODO:** Benchmark tests for performance

---

### 21. **Context Support**

**Add context.Context throughout:**

```go
func (c *CLI) ExecuteLineWithContext(ctx context.Context, line string) (string, error)

// Allows:
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
output, err := cli.ExecuteLineWithContext(ctx, "long-running-command")

// Graceful cancellation on Ctrl+C
```

---

### 22. **Command Middleware**

**Hook system for all commands:**

```go
// middleware.go
type Middleware func(next CommandFunc) CommandFunc

func (c *CLI) Use(middleware Middleware)

// Example middlewares:
- LoggingMiddleware
- TimingMiddleware
- AuthenticationMiddleware
- RateLimitMiddleware
```

---

## 📊 Priority Matrix

### Must Have (Next Release)
1. ✅ Parser quote handling (DONE)
2. ✅ File redirection (DONE)
3. ✅ Alias first-word matching (DONE)
4. ✅ Job Management System (PHASE 1 COMPLETE)
5. ✅ Enhanced Variables System (PHASE 1 COMPLETE)
6. ✅ Configuration System (PHASE 1 COMPLETE)
7. 🔴 Session Management

### Should Have (Near Term)
7. Connection Management
8. Logging & Audit Trail
9. Interactive Prompts
10. Configuration System
11. Tab Completion Enhancements

### Nice to Have (Future)
12. Template System
13. Plugin System
14. Data Manipulation
15. Watch Command
16. Notification System
17. Metrics Tracking
18. Batch Operations

### Quality Improvements (Ongoing)
19. ✅ Error Handling (COMPLETE - errors.go with CLIError type)
20. ✅ Testing Coverage (PHASE 1 COMPLETE - cli_test.go with 8 tests, jobs_test.go with 6 tests)
21. Context Support (Future - ExecuteLineWithContext)
22. Middleware System (Future - Command hooks)

---

## 🎯 Recommended Implementation Order

### Phase 1: Core Infrastructure (COMPLETE ✅)
1. ✅ Job Management - Critical for internal tooling (IMPLEMENTED)
2. ✅ Enhanced Variables - Needed for everything else (IMPLEMENTED)
3. ✅ Configuration System - Unified settings (IMPLEMENTED)

**Phase 1 Status**: All features implemented and documented. Includes:
- jobs.go: JobManager with full job tracking
- cmds/jobcmds.go: jobs, job, killall, jobclean commands
- cmds/varcmds.go: let, unset, vars, inc, dec commands
- config.go: TOML-based configuration system
- cmds/configcmds.go: config management commands
- Integration with osexec --background for automatic job tracking

### Phase 2: User Experience (Week 3-4)
4. Session Management - Save/restore work
5. Connection Management - Key for your use case
6. Logging & Audit - Compliance, debugging

### Phase 3: Advanced Features (Week 5-6)
7. Interactive Prompts - Better UX
8. Template System - Code generation
9. Data Manipulation - JSON/CSV/YAML

### Phase 4: Quality & Performance (PARTIAL COMPLETE ✅)
10. ✅ Comprehensive Testing (cli_test.go with 8 tests, jobs_test.go with 6 tests)
11. ✅ Error Handling (errors.go with CLIError type)
12. Context Support (Future)
13. Middleware System (Future)
14. Performance Optimization (Future)

---

## 💡 Quick Wins

### ✅ 1. Add `clear` alias (COMPLETE)
```go
// cls command now has clear alias
var clsCmd = &cobra.Command{
    Use:     "cls",
    Aliases: []string{"clear"},
    ...
}
```
**Implemented in:** base.go:24-25

### ✅ 2. Add `which` command (COMPLETE)
```bash
which print  # Shows: built-in command
which myvar  # Shows: variable with value "..."
which myalias # Shows: alias for "..."
which ls     # Shows: not found
```
**Implemented in:** utilcmds.go (AddUtilityCommands)

### ✅ 3. Add `time` command (COMPLETE)
```bash
time http api.com/slow-endpoint
# Output: Command completed in 2.345s
```
**Implemented in:** utilcmds.go (AddUtilityCommands)

### ✅ 4. Add `.` script execution (COMPLETE)
```bash
. myscript.sh    # Execute script in current context (like bash source)
```
**Implemented in:** utilcmds.go (AddUtilityCommands)

### 5. Add command output to history
```bash
history show --with-output
```
**Status:** Not yet implemented (future enhancement)

---

## 🔐 Security Enhancements

### 1. Command Whitelisting Mode
```toml
[security]
whitelist_enabled = true
allowed_commands = ["print", "date", "history"]
```

### 2. Sensitive Data Filtering
```go
// Filter patterns from history/logs
[security.filters]
patterns = ["password=.*", "token=.*", "secret=.*"]
```

### 3. Permission System
```go
// Require permission for dangerous commands
func (c *CLI) RequirePermission(cmd string, perm Permission) error
```

---

## 📝 Documentation Needs

1. **User Guide** - Command reference, examples
2. **Developer Guide** - Creating custom commands
3. **Architecture Guide** - How it all fits together
4. **Migration Guide** - Upgrading between versions
5. **Best Practices** - Patterns for internal tools

---

## Summary

**Top 3 Priorities for Internal Tooling:**
1. **Job Management** - Track background processes
2. **Connection Management** - Manage remote systems
3. **Session Management** - Save/restore work

These will make ConsoleKit a truly powerful internal tool for technical users managing complex systems.
