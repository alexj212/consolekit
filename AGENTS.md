# Repository Guidelines

## Project Structure & Module Organization
- Core CLI, REPL, and command modules live in the repo root (`cli.go`, `run.go`, `alias.go`, `notify.go`, `prompt.go`, etc.); command helpers are grouped by feature in `*cmds.go` files.
- Parsing logic and related tests are in `parser/`.
- Concurrency helpers live in `jobs.go`/`jobcmds.go`; maps utilities in `safemap/`.
- Example application and scripts are under `examples/simple`; binaries built locally are written to `build/`.
- Version metadata is tracked in `version.txt`; tooling metadata is set via Makefile `COMPILE_LDFLAGS`.

## Build, Test, and Development Commands
- `go build ./...` — compile library packages.
- `make simple` — build the example binary to `build/simple` with version flags applied.
- `make test` or `go test -v ./...` — run unit tests.
- `make fmt` — format the entire tree with `gofmt -s -w` (removes local `./.history` first).
- Optional checks: `make vet`, `make lint`, `make staticcheck`, `make reportcard`, `make doc` (godoc on :6060).

## Coding Style & Naming Conventions
- Go 1.21+; always run `gofmt` before committing. Tabs for indentation, standard Go imports ordering.
- Exported identifiers use CamelCase; package-level helpers stay unexported unless required.
- Name new command modules `<feature>cmds.go` to match existing files (`clipboardcmds.go`, `configcmds.go`, etc.); keep related flags and help text nearby.
- Keep errors wrapped with context; prefer small, composable functions over monoliths.

## Testing Guidelines
- Place `_test.go` beside implementations (see `parser/parser_test.go`, `cli_test.go`, `jobs_test.go`).
- Use table-driven tests for command/parsing logic; favor deterministic input/output pairs over timing-based assertions.
- Run `go test -v ./...` before opening a PR; add coverage for new commands and parser branches.

## Commit & Pull Request Guidelines
- Commit messages are short and imperative; release bumps use `latest version: vX.Y.Z`. Follow that tone, e.g., `add history filtering` rather than status reports.
- PRs should include: summary of changes, key commands/tests executed (`make test`, `make fmt`), linked issues, and notes on user-facing behavior or sample REPL transcripts when adding commands.
- Keep PRs focused and small; update docs/examples (`examples/simple`) when adding user-visible flags or commands.

## Security & Configuration Tips
- Do not commit local history/alias files (`~/.{appname}.history`, `~/.{appname}.aliases`) or secrets. Prefer environment variables for tokens.
- `make publish` writes `version.txt` and tags a release; coordinate with maintainers before running it.
