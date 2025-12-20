# ConsoleKit Development Context

## Project Overview
ConsoleKit is a Go framework (v0.5.59+) for building reliable CLI/REPL tools. ~10K lines of production-ready code for internal tools.

## Recent Critical Improvements (2025-12-20)
**Commit:** 7154dd5 - "fix: critical reliability improvements and stdin piping support"

### Fixed Issues
1. ✅ SafeMap.Remove() race condition - Changed RLock to Lock for write operations
2. ✅ SafeMap.Random() deadlock - Inlined key collection to avoid nested locks
3. ✅ JobManager.Wait() busy-wait - Replaced polling with channel-based synchronization
4. ✅ ScheduledTask.Enabled races - Added mutex protection for concurrent access
5. ✅ Scheduled task goroutine leaks - Proper done channel cleanup
6. ✅ execDepth thread-safety - Changed to atomic.Int32
7. ✅ TemplateManager races - Added RWMutex for concurrent template loading

### New Features
- **Stdin Piping**: Auto-detects piped stdin for batch mode execution
- **Batch Mode**: RunBatch() method for automated testing
- **CI/CD Ready**: Non-zero exit codes on failures, line number error reporting
- **Documentation**: STDIN_PIPING.md with comprehensive testing guide

## Architecture Key Points

### Thread-Safe Components
- `SafeMap[K, V]`: Generic concurrent map with RWMutex
- `JobManager`: Manages background processes with proper synchronization
- `TemplateManager`: Concurrent template loading/caching
- `ScheduledTask`: Thread-safe task scheduling

### Execution Model
- Recursion depth tracking: `atomic.Int32` counter (max depth: 10)
- Context-aware cancellation throughout
- Channel-based job completion signaling (no polling)
- Pipeline support for command chaining

### Batch Mode
- Detects piped stdin automatically: `!isatty.IsTerminal(os.Stdin.Fd())`
- Line-by-line execution with error continuation
- Skips comments (#) and empty lines

## Build & Test Commands

```bash
# Build
make simple                           # Build example app with version info
go build ./...                        # Build all packages

# Test
go test ./...                         # Run all tests
go test -race ./...                   # Race detector (requires format fixes)
cat test.run | ./build/simple         # Test stdin piping
bash examples/simple/run_tests.sh     # Automated test suite

# Development
make fmt                              # Format code
make vet                              # Vet code
make lint                             # Lint (if golangci-lint installed)
```

## Code Standards

### Go Style
- Go 1.21+ required
- Always run `gofmt` before commit
- Use tabs for indentation
- Standard Go import ordering

### Concurrency
- Use `sync.RWMutex` for read-heavy shared state
- Use `atomic.*` for simple counters/flags
- Always use `defer` for unlock operations
- Channel-based synchronization over polling
- Context cancellation for long-running operations

### Error Handling
- Return errors, don't panic in production code
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Graceful degradation (logging failures are non-fatal)

### Command Modules
- Pattern: `Add<Feature>(cli *CLI) func(cmd *cobra.Command)`
- File naming: `*cmds.go` (e.g., `jobcmds.go`, `schedulecmds.go`)
- Reset flags in `PostRun` if command uses flags

## Known Patterns

### Adding Thread-Safe State
```go
type MyManager struct {
    data map[string]string
    mu   sync.RWMutex  // RWMutex for read-heavy, Mutex for write-heavy
}

func (m *MyManager) Get(key string) (string, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    val, ok := m.data[key]
    return val, ok
}

func (m *MyManager) Set(key, value string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.data[key] = value
}
```

### Background Job Pattern
```go
job := &Job{
    done: make(chan struct{}),
}

go func() {
    // Do work
    err := doWork()
    
    // Update state
    job.mu.Lock()
    job.Status = JobCompleted
    job.Error = err
    job.mu.Unlock()
    
    // Signal completion
    close(job.done)
}()

// Wait for completion
<-job.done
```

## Testing Approach

### Script-Based Testing
```bash
# Create test script
cat > my_test.run << 'EOF'
# Test comments work
print Starting tests

# Test variables
def myvar testvalue

# Test commands
env | wc
EOF

# Run test
cat my_test.run | ./build/simple
```

### CI/CD Integration
```yaml
# GitHub Actions example
- name: Run ConsoleKit tests
  run: |
    make simple
    cat tests/smoke.run | ./build/simple
    if [ $? -ne 0 ]; then
      echo "Tests failed"
      exit 1
    fi
```

## Project Structure
- Root: Core CLI, REPL, command modules (`*cmds.go`)
- `parser/`: Command parsing with pipe/redirect support
- `safemap/`: Thread-safe generic map
- `examples/simple/`: Example application
- `build/`: Compiled binaries (gitignored)

## Reliability Status
✅ Production-ready for internal tools  
✅ Thread-safe throughout  
✅ No race conditions or deadlocks  
✅ Proper resource cleanup  
✅ Automated testing support  
✅ CI/CD integration ready  

## Quick Reference

### Start REPL
```bash
./simple
```

### Execute Command
```bash
./simple print "Hello World"
```

### Pipe Script
```bash
cat script.run | ./simple
```

### Background Job
```bash
osexec -b "long-running-command"
jobs list
jobs wait <id>
```

### Scheduled Task
```bash
schedule in 5m "print Reminder"
schedule every 1h "print Check"
schedule list
schedule cancel <id>
```

## Future Considerations
- Consider metrics/telemetry if monitoring needed
- Add more integration tests for concurrent scenarios
- Optional job output size limits for long-running jobs
- Structured logging if log aggregation needed

## Contact & Contribution
When making changes:
1. Run `make fmt` before commit
2. Ensure `go build ./...` passes
3. Test with `cat test.run | ./build/simple`
4. Update relevant docs (README.md, STDIN_PIPING.md, etc.)
5. Use conventional commits: `fix:`, `feat:`, `docs:`, etc.

---
Last Updated: 2025-12-20  
Maintained by: alexj212  
Repository: github.com/alexj212/consolekit
