# Script Piping for Automated Testing

## Overview

ConsoleKit now supports **stdin piping** for automated testing and CI/CD integration. You can pipe scripts directly to your ConsoleKit application for non-interactive execution.

## Quick Start

```bash
# Build your app
make simple

# Pipe a script file
cat test.run | ./build/simple

# Or use input redirection
./build/simple < test.run

# Pipe commands directly
echo "print Hello World" | ./build/simple

# Use here documents
./build/simple << EOF
print "Test 1"
print "Test 2"
EOF
```

## How It Works

1. **Automatic Detection**: ConsoleKit detects if stdin is a pipe (not a TTY) using `isatty.IsTerminal()`
2. **Batch Mode**: Automatically switches to `RunBatch()` mode
3. **Line-by-Line Execution**: Reads and executes commands one per line
4. **Error Handling**: Continues execution even if commands fail, reports errors to stderr
5. **Exit Code**: Returns error if any commands failed (useful for CI/CD)

## Features

✅ **Comments**: Lines starting with `#` are ignored  
✅ **Empty Lines**: Blank lines are skipped  
✅ **Error Reporting**: Shows line numbers for errors  
✅ **Continue on Error**: Doesn't stop on first error  
✅ **Exit Status**: Non-zero exit if any command fails  

## Script Format

```bash
# test.run - Example test script

# This is a comment
print Starting tests...

# Define variables
def myvar testvalue

# Execute commands
env | wc

# Use aliases
alias ll "env"
ll | wc

# Empty lines are OK

print Tests complete!
```

## Automated Testing Example

**Create test script** (`my_tests.run`):
```bash
# Functional tests
print === Running Tests ===
print Test 1: Basic functionality
env | wc
print Test 2: Aliases
alias test "print working"
test
print === Tests Complete ===
```

**Run in CI/CD**:
```bash
#!/bin/bash
cat my_tests.run | ./myapp
if [ $? -ne 0 ]; then
    echo "Tests failed!"
    exit 1
fi
echo "Tests passed!"
```

## Testing Multiple Scripts

```bash
# Run all test scripts
cat test1.run test2.run test3.run | ./myapp

# Or with a loop
for test in tests/*.run; do
    echo "Running $test..."
    cat "$test" | ./myapp || exit 1
done
```

## Dynamic Test Generation

```bash
# Generate tests programmatically
./generate_tests.sh | ./myapp

# Example generator:
#!/bin/bash
echo "print Test generated at $(date)"
for i in {1..5}; do
    echo "print Test case $i"
done
```

## Differences from REPL Mode

| Feature | REPL Mode | Batch Mode (Piped) |
|---------|-----------|-------------------|
| **Prompt** | Interactive | No prompt |
| **Input** | Terminal | Stdin pipe |
| **Comments** | Supported | Supported |
| **Empty Lines** | Ignored | Ignored |
| **Errors** | Stops execution | Continues |
| **History** | Saved | Not saved |
| **Completion** | Available | N/A |
| **Exit Code** | 0 | Non-zero if errors |

## Integration with Existing Features

All ConsoleKit features work in batch mode:

- ✅ **Aliases**: Define and use aliases
- ✅ **Variables**: Use `def` command
- ✅ **Background Jobs**: Start with `osexec -b`
- ✅ **Scheduled Tasks**: Schedule commands
- ✅ **Templates**: Execute templates
- ✅ **Logging**: Audit log captures all commands
- ✅ **Piping**: Commands can pipe within the script

## Example: Complete Test Workflow

```bash
#!/bin/bash
# complete_test.sh

set -e

APP="./build/myapp"

echo "Setting up test environment..."
{
  echo "# Setup"
  echo "def test_env staging"
  echo "alias check 'print Checking...'"
} | $APP

echo "Running functional tests..."
cat tests/functional.run | $APP

echo "Running integration tests..."
cat tests/integration.run | $APP

echo "Cleanup..."
{
  echo "print Cleaning up..."
  echo "jobs killall"
} | $APP

echo "✓ All tests passed!"
```

## Debugging Tips

### 1. See what's being executed:
```bash
# Add -v flag (if your app supports it) or use stderr
cat test.run | ./myapp 2>&1 | tee test.log
```

### 2. Run specific lines:
```bash
# Test line 5 only
sed -n '5p' test.run | ./myapp
```

### 3. Check syntax:
```bash
# Validate script before running
cat test.run | grep -v '^#' | grep -v '^$'
```

## CI/CD Integration

### GitHub Actions Example:
```yaml
name: ConsoleKit Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Build app
        run: make simple
      - name: Run tests
        run: |
          cat tests/smoke.run | ./build/simple
          if [ $? -ne 0 ]; then
            echo "Tests failed"
            exit 1
          fi
```

### GitLab CI Example:
```yaml
test:
  script:
    - make simple
    - cat tests/*.run | ./build/simple
  artifacts:
    when: on_failure
    paths:
      - test-output/
```

## Benefits

1. **Reproducible**: Scripts are version-controlled
2. **Fast**: No human interaction needed
3. **Automated**: CI/CD friendly
4. **Debuggable**: Line numbers on errors
5. **Flexible**: Combine with shell features
6. **Reliable**: Consistent execution every time

## Example Test Scripts Included

- `test.run` - Basic functionality test
- `test1.run` - Simple exit test
- `test_fail.run` - Error handling test
- `test_comprehensive.run` - Full feature test

Run them with:
```bash
cat examples/simple/test.run | ./build/simple
```

---

**You now have a fully testable REPL framework!** 🚀
