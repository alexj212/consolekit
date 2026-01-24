# Testing Consolekit Versions to Find Working Commit

## How to Test

For each version below, run:

```bash
cd /c/projects/consolekit
git checkout <commit-hash>
cd examples/simple
go run .
```

**Look for:** Does the prompt "> " appear immediately when the app starts, or do you need to press a key first?

## Versions to Test (newest to oldest)

### Recent (Known Broken)
- [ ] `c8e24e7` - v0.5.68 - BROKEN (my recent changes made it worse)
- [ ] `0ddeeb1` - v0.5.66 - TEST THIS

### Mid-December 2024
- [ ] `7154dd5` - "fix: critical reliability improvements and stdin piping support" - **SUSPECT**
- [ ] `15d0139` - v0.5.59
- [ ] `61f97c4` - "Fix Run() to distinguish flags from commands" - **SUSPECT**
- [ ] `a7c3b8c` - "added mcp support" - **SUSPECT**

### Late November 2024
- [ ] `286db9b` - v0.5.54
- [ ] `074208e` - v0.5.51
- [ ] `f5049a4` - "added new fetaures for data manipulation and aliases" - **SUSPECT**
- [ ] `1efb9be` - v0.5.50

## Once You Find a Working Version

1. Note the commit hash
2. Run: `git bisect start`
3. Run: `git bisect bad HEAD` (current main is broken)
4. Run: `git bisect good <working-commit-hash>`
5. Git will check out commits for you to test
6. For each commit, run `cd examples/simple && go run .` and test
7. Run `git bisect good` if it works, `git bisect bad` if broken
8. Repeat until git identifies the first bad commit

## Suspect Commits

These commits changed core functionality and are likely culprits:

1. **7154dd5** - "critical reliability improvements and stdin piping support"
   - Changed how stdin is handled
   - Added piping support

2. **61f97c4** - "Fix Run() to distinguish flags from commands"
   - Changed Run() method logic

3. **a7c3b8c** - "added mcp support"
   - Added MCP command handling

4. **f5049a4** - "added new features for data manipulation and aliases"
   - Significant feature additions

## Quick Test Script

You can also use this one-liner to quickly test if a version works:

```bash
(sleep 0.3; echo "exit") | go run ./examples/simple/
```

If you see "> " in the output before "exit", it's working. If not, it's broken.
