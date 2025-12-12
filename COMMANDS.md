# ConsoleKit - Command Reference

Complete reference guide for all ConsoleKit commands with usage examples.

---

## Table of Contents

- [Core Commands](#core-commands)
- [Job Management](#job-management)
- [Variable Management](#variable-management)
- [Configuration Management](#configuration-management)
- [Logging & Audit](#logging--audit)
- [Interactive Prompts](#interactive-prompts)
- [Templates](#templates)
- [Data Manipulation](#data-manipulation)
- [Utility Commands](#utility-commands)
- [Scripting](#scripting)
- [OS Execution](#os-execution)
- [History](#history)
- [Aliases](#aliases)

---

## Core Commands

### print
Print text to the console.

```bash
print "Hello, World!"
print "Line 1\nLine 2"
```

### cls / clear
Clear the console screen.

```bash
cls
clear  # alias
```

### exit
Exit the application.

```bash
exit
exit 0     # with exit code
exit 1     # non-zero exit code
```

### date
Display the current date and time.

```bash
date
```

### sleep
Pause execution for specified duration.

```bash
sleep 2s       # 2 seconds
sleep 500ms    # 500 milliseconds
sleep 1m       # 1 minute
```

### wait
Wait and display a spinner.

```bash
wait 3s "Processing..."
```

### repeat
Execute a command multiple times.

```bash
repeat 5 "print 'Hello'"
```

### waitfor
Wait for a condition to be true (with timeout).

```bash
waitfor --timeout 30s --interval 1s "http example.com"
```

### http
Make HTTP requests.

```bash
http https://api.example.com/users
http https://api.example.com/data --method POST --body '{"key":"value"}'
```

### if
Conditional execution.

```bash
if "condition" "command if true" "command if false"
```

### set
Set a default variable.

```bash
set myvar "Hello"
print "@myvar"   # Outputs: Hello
```

### watch
Execute a command repeatedly at a specified interval.

```bash
watch "date"                        # Run every 2 seconds (default)
watch --interval 5s "jobs"          # Run every 5 seconds
watch --count 10 "date"             # Run 10 times then stop
watch --clear "date"                # Clear screen before each run
watch -n 1s -c 60 --clear "http https://api.example.com/status"
```

**Example Output:**
```
Every 2s: date

Iteration 1 at 10:30:00:
------------------------------------------------------------
Thu Dec 12 10:30:00 EST 2025

Iteration 2 at 10:30:02:
------------------------------------------------------------
Thu Dec 12 10:30:02 EST 2025
```

---

## Job Management

Background job tracking and management.

### jobs
List all background jobs.

```bash
jobs              # List active jobs
jobs -v           # Verbose output with full commands
jobs -a           # Show all jobs (including completed)
```

**Example Output:**
```
ID   PID    STATUS    COMMAND           STARTED
1    12345  running   sleep 60          2025-12-12 10:30:00
2    12346  completed make build        2025-12-12 10:28:15
```

### job
Manage individual jobs.

```bash
job 1             # Show job details
job 1 logs        # View job output
job 1 kill        # Terminate job
job 1 wait        # Wait for completion
```

**Example:**
```bash
# Start a background job
osexec --background "sleep 30 && echo 'Done'"

# Check job status
job 1

# View logs
job 1 logs

# Kill if needed
job 1 kill
```

### killall
Kill all running jobs.

```bash
killall           # Kills all background jobs
```

### jobclean
Remove completed jobs from the list.

```bash
jobclean          # Clean up completed jobs
```

---

## Variable Management

Enhanced variable system with command substitution and arithmetic.

### let
Set variables with advanced features.

```bash
# Simple assignment
let name="John"

# Command substitution
let result=$(date)
let files=$(ls | wc -l)

# Numeric values for arithmetic
let counter=10
```

### unset
Remove variables.

```bash
unset name
unset counter
```

### vars
List and export variables.

```bash
vars                    # List all variables (pretty print)
vars --json             # Export as JSON
vars --export           # Export as shell script
```

**Example Output:**
```
Variables:
  @counter = 10
  @name = John
  @result = 2025-12-12 10:30:00
```

### inc
Increment numeric variables.

```bash
let counter=0
inc counter           # counter = 1
inc counter 5         # counter = 6
```

### dec
Decrement numeric variables.

```bash
let counter=10
dec counter           # counter = 9
dec counter 3         # counter = 6
```

**Complete Example:**
```bash
# Initialize counter
let counter=0

# Process items
repeat 10 "inc counter; print 'Item @counter'"

# Show final count
print "Total: @counter"
```

---

## Configuration Management

TOML-based configuration system.

### config get
Retrieve configuration values.

```bash
config get settings.history_size
config get logging.enabled
```

### config set
Set configuration values.

```bash
config set settings.history_size 5000
config set logging.enabled true
config set settings.prompt "mycli > "
```

### config show
Display all configuration.

```bash
config show
```

**Example Output:**
```toml
[settings]
history_size = 10000
prompt = "%s > "
color = true
pager = "less -R"

[logging]
enabled = false
log_file = "~/.myapp/audit.log"
log_success = true
log_failures = true
max_size_mb = 100
retention_days = 90
```

### config edit
Open configuration in $EDITOR.

```bash
config edit
```

### config reload
Reload configuration from file.

```bash
config reload
```

### config save
Save current state to file.

```bash
config save
```

### config path
Show configuration file location.

```bash
config path
# Output: /home/user/.myapp/config.toml
```

---

## Logging & Audit

Command execution logging for debugging and compliance.

### log enable
Enable command logging.

```bash
log enable
```

### log disable
Disable command logging.

```bash
log disable
```

### log status
Show logging status and configuration.

```bash
log status
```

**Example Output:**
```
Logging: ENABLED
Log file: /home/user/.myapp/audit.log
```

### log show
Display command logs with filtering.

```bash
log show                           # Show all logs
log show --last 20                 # Last 20 logs
log show --failed                  # Only failed commands
log show --search "deploy"         # Search by keyword
log show --since "2025-12-01"      # Since date
log show --json                    # JSON format
```

**Example Output:**
```
✓ 2025-12-12 10:30:15 [245ms] print "Hello"
✗ 2025-12-12 10:30:20 [12ms] http invalid-url - Get "invalid-url": unsupported protocol
✓ 2025-12-12 10:30:25 [1.2s] sleep 1s

Total: 3 logs
```

### log clear
Clear in-memory logs.

```bash
log clear
```

### log export
Export logs to JSON.

```bash
log export > audit.json
log export --format json
```

### log load
Load logs from file.

```bash
log load
```

### log config
Configure logging settings.

```bash
log config max_size 200            # Max file size in MB
log config retention 180           # Retention days
log config log_success true        # Log successful commands
log config log_failures true       # Log failed commands
```

---

## Interactive Prompts

User input and confirmation prompts.

### confirm
Simple yes/no confirmation.

```bash
confirm "Deploy to production?"
# Output: yes or no
```

**Usage in scripts:**
```bash
let response=$(confirm "Continue?")
if "@response" "print 'Proceeding...'" "print 'Cancelled'"
```

### input
Prompt for text input.

```bash
input "Enter your name"
input "Enter port" --default 8080
```

**Example:**
```bash
let username=$(input "Username")
let port=$(input "Port" --default 8080)
print "Connecting to @username at port @port"
```

### select
Single selection from options.

```bash
select "Choose environment" dev staging prod
select "Choose color" red green blue --default 1
```

**Example:**
```bash
let env=$(select "Environment" dev staging prod)
print "Deploying to @env"
```

### multiselect
Multiple selections from options.

```bash
multiselect "Enable features" auth logging cache monitoring
```

**Example:**
```bash
# Returns multiple lines, one per selection
let features=$(multiselect "Features" auth logging cache)
print "Enabled: @features"
```

### prompt-demo
Interactive demonstration of all prompt types.

```bash
prompt-demo
```

---

## Templates

Script templates with variable substitution.

### template list
List available templates.

```bash
template list
```

**Example Output:**
```
Available templates:
  - deployment.tmpl
  - backup.tmpl
  - setup.tmpl
```

### template show
Display template content.

```bash
template show deployment.tmpl
```

**Example Output:**
```
# Deployment Template
print "Deploying to {{.Env}}"
let region="{{.Region}}"
http {{.ApiEndpoint}}/deploy
```

### template exec
Execute template with variables.

```bash
template exec deployment.tmpl Env=prod Region=us-east-1 ApiEndpoint=https://api.example.com
```

**Example:**
```bash
# Create template
cat > deployment.tmpl << 'EOF'
print "Deploying {{.Service}} to {{.Env}}"
let timestamp=$(date)
http {{.ApiUrl}}/deploy/{{.Service}}
print "Deployment completed at @timestamp"
EOF

# Execute with variables
template exec deployment.tmpl Service=web Env=production ApiUrl=https://api.prod.com
```

### template render
Render template without executing.

```bash
template render deployment.tmpl Env=staging Region=eu-west-1 ApiEndpoint=https://staging.api.com
```

### template create
Create a new template interactively.

```bash
template create mytemplate.tmpl
# Enter content, then Ctrl+D to save
```

### template delete
Delete a template.

```bash
template delete mytemplate.tmpl
```

### template clear-cache
Clear template cache.

```bash
template clear-cache
```

---

## Data Manipulation

JSON, YAML, and CSV parsing and conversion.

### JSON Commands

#### json parse
Parse and format JSON.

```bash
json parse data.json
json parse data.json --pretty
cat data.json | json parse
```

**Example:**
```bash
# Pretty-print JSON
echo '{"name":"John","age":30}' | json parse --pretty
```

**Output:**
```json
{
  "name": "John",
  "age": 30
}
```

#### json get
Extract values using dot notation.

```bash
json get data.json users.0.name
cat data.json | json get users.0.email
```

**Example:**
```bash
# Extract nested value
echo '{"users":[{"name":"Alice","email":"alice@example.com"}]}' | json get users.0.email
# Output: "alice@example.com"
```

#### json validate
Validate JSON syntax.

```bash
json validate data.json
cat data.json | json validate
```

### YAML Commands

#### yaml parse
Parse and format YAML.

```bash
yaml parse config.yaml
cat config.yaml | yaml parse
```

#### yaml to-json
Convert YAML to JSON.

```bash
yaml to-json config.yaml
cat config.yaml | yaml to-json
```

**Example:**
```bash
echo "name: John
age: 30" | yaml to-json
```

**Output:**
```json
{
  "age": 30,
  "name": "John"
}
```

#### yaml from-json
Convert JSON to YAML.

```bash
yaml from-json data.json
cat data.json | yaml from-json
```

### CSV Commands

#### csv parse
Parse and display CSV as table.

```bash
csv parse data.csv
csv parse data.csv --header
cat data.csv | csv parse
```

**Example:**
```bash
echo "Name,Age,City
Alice,30,NYC
Bob,25,LA" | csv parse --header
```

**Output:**
```
Name | Age | City
-------------------
Alice | 30 | NYC
Bob | 25 | LA
```

#### csv to-json
Convert CSV to JSON (first row as header).

```bash
csv to-json data.csv
cat data.csv | csv to-json
```

**Example Output:**
```json
[
  {
    "Age": "30",
    "City": "NYC",
    "Name": "Alice"
  },
  {
    "Age": "25",
    "City": "LA",
    "Name": "Bob"
  }
]
```

---

## Utility Commands

### which
Check if name is a command, alias, or variable.

```bash
which print        # Output: built-in command
which ll           # Output: alias for "ls -la"
which myvar        # Output: variable with value "..."
which unknown      # Output: not found
```

### time
Measure command execution time.

```bash
time sleep 2s
time http https://api.example.com
```

**Example Output:**
```
Command completed in 2.001s
```

### . (dot)
Source a script in the current context (like bash source).

```bash
. myscript.sh
. /path/to/config.sh
```

**Difference from `run`:**
- `run` executes in isolated scope
- `.` executes in current scope (variables persist)

---

## Scripting

### run
Execute scripts with arguments.

```bash
run myscript.sh
run myscript.sh arg1 arg2
run @embedded-script.sh      # From embedded FS
run --spawn background.sh     # Run in background
```

**Script arguments:**
```bash
# In script: access as @arg0, @arg1, etc.
# myscript.sh
print "Argument 1: @arg0"
print "Argument 2: @arg1"
```

**Multi-line scripts:**
```bash
# Use backslash for continuation
print "Line 1" \
  "Line 2" \
  "Line 3"
```

---

## OS Execution

### osexec
Execute OS commands.

```bash
osexec "ls -la"
osexec "git status"
osexec --background "make build"
```

**Background execution:**
```bash
# Start in background
osexec --background "sleep 60 && echo 'Done'"

# Check status
jobs

# View output
job 1 logs
```

**Example - Build pipeline:**
```bash
print "Starting build..."
osexec "go build"
osexec "go test ./..."
osexec "go vet ./..."
print "Build complete"
```

---

## History

### history list
List command history.

```bash
history list
history list --last 20
```

### history search
Search command history.

```bash
history search "deploy"
history search "git"
```

### history clear
Clear command history.

```bash
history clear
```

---

## Aliases

### alias list
List all aliases.

```bash
alias list
```

**Example Output:**
```
Aliases:
  ll = ls -la
  gs = git status
  deploy = osexec './deploy.sh'
```

### alias set
Create or update an alias.

```bash
alias set ll "ls -la"
alias set gs "git status"
alias set deploy "osexec './deploy.sh'"
```

### alias delete
Remove an alias.

```bash
alias delete ll
alias delete gs
```

### alias save
Save aliases to file.

```bash
alias save
# Aliases saved to ~/.myapp.aliases
```

### alias load
Load aliases from file.

```bash
alias load
# Loaded N aliases from ~/.myapp.aliases
```

---

## Complete Examples

### Deployment Workflow

```bash
# Set environment
let env=$(select "Environment" dev staging prod)
let region=$(select "Region" us-east-1 eu-west-1 ap-southeast-1)

# Confirm
if ! $(confirm "Deploy to @env in @region?"); then
    print "Cancelled"
    exit 1
fi

# Deploy
print "Deploying to @env..."
template exec deployment.tmpl Env=@env Region=@region

# Check status
http https://api.@env.example.com/health
print "Deployment complete!"
```

### Automated Testing

```bash
# Enable logging
log enable

# Run tests with timing
print "Running test suite..."
time osexec "go test ./..."

# Check results
if [ $? -eq 0 ]; then
    print "✓ All tests passed"
else
    print "✗ Tests failed"
    log show --failed
    exit 1
fi
```

### Data Processing Pipeline

```bash
# Fetch data
http https://api.example.com/users > users.json

# Parse and extract
json get users.json results.0.name
let count=$(json get users.json total)

# Convert formats
json parse users.json | yaml from-json > users.yaml

# Process with template
template exec process-users.tmpl Count=@count
```

### Batch Operations

```bash
# Read server list
csv parse servers.csv --header

# Deploy to each
repeat 5 "
  let server=@arg0
  print 'Deploying to ' @server
  template exec deploy.tmpl Host=@server
"
```

---

## Tips & Best Practices

### 1. Use Templates for Repetitive Tasks
Create templates for common workflows and parameterize with variables.

### 2. Enable Logging for Auditing
Use `log enable` for production systems to track command execution.

### 3. Leverage Command Substitution
Use `$(command)` syntax in `let` for dynamic values.

### 4. Background Jobs for Long Operations
Use `--background` for long-running commands and monitor with `jobs`.

### 5. Interactive Prompts for Safety
Use `confirm` before destructive operations.

### 6. Aliases for Shortcuts
Create aliases for frequently used command sequences.

### 7. Variables for Reusable Values
Store configuration in variables at the start of scripts.

### 8. --dry-run for Testing
Use `--dry-run` flags (where available) to preview actions.

---

## Error Handling

### Check Command Success

```bash
osexec "git pull"
if [ $? -ne 0 ]; then
    print "Git pull failed"
    exit 1
fi
```

### Logging Errors

```bash
log enable
# Commands are automatically logged with success/failure
log show --failed    # View only failures
```

### Try-Catch Pattern

```bash
let result=$(osexec "risky-command" 2>&1)
if [ $? -ne 0 ]; then
    print "Error: @result"
    # Handle error
fi
```

---

## Environment Variables

Access environment variables with `@env:NAME` syntax:

```bash
print "@env:HOME"
print "@env:PATH"
let user="@env:USER"
```

---

## Command Chaining

```bash
# Sequential execution (;)
print "Step 1" ; sleep 1s ; print "Step 2"

# Piping (|)
env | grep PATH

# File redirection (>)
history list > history.txt

# Combined
env | grep PATH > path.txt ; cat path.txt
```

---

For more information, see:
- **CLAUDE.md** - Architecture and implementation details
- **README.md** - Getting started guide
- **SECURITY.md** - Security considerations
