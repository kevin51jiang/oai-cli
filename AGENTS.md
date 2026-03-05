# Repository Guidelines

## Project Structure & Module Organization
This repository is a small Go CLI (`oaicheck`) for diagnosing OpenAI API configuration.

- `main.go`: process entrypoint and exit-code handling.
- `cmd/`: Cobra command wiring (`root`, `doctor`, `ping`, `models`, `probe`).
- `internal/config/`: resolves flags/environment into runtime config.
- `internal/checks/`: HTTP checks and result envelope construction.
- `internal/output/`: human-readable and JSON rendering.
- `*_test.go`: unit tests colocated with source.
- `.github/`: CI/workflow definitions.

Keep new packages under `internal/` unless they must be externally imported.

## Build, Test, and Development Commands
Use `make` targets where possible:

- `make fmt`: run `go fmt ./...`.
- `make test`: run all tests (`go test ./...`).
- `make build`: build binary (`oaicheck`).
- `make tidy`: clean up module dependencies.
- `make clean`: remove built binary.

Direct equivalents are fine, e.g. `go test ./...`.

## Coding Style & Naming Conventions
- Follow standard Go formatting (`gofmt`); do not hand-format alignment.
- Use short, descriptive package names (`config`, `checks`, `output`).
- Exported identifiers: `CamelCase`; unexported: `camelCase`.
- Keep functions focused and return structured errors/messages useful for CLI output.
- Avoid logging or printing sensitive values (API keys). Use redacted/safe views only.

## Testing Guidelines
- Framework: Go `testing` package.
- Place tests next to code with `*_test.go`.
- Name tests as `Test<FunctionOrBehavior>` (e.g., `TestRunProbeFailureIncludesEndpointErrors`).
- Prefer table-driven tests for multiple cases.
- For HTTP behavior, use `httptest.Server` instead of real network calls.

Run locally before pushing:

```bash
make fmt
make test
```

## Commit & Pull Request Guidelines
Use only these commit prefixes:

- `feat: ...` for user-visible functionality
- `fix: ...` for bug fixes
- `chore: ...` for maintenance/refactors

PRs should include:
- clear summary of behavioral changes,
- linked issue/ticket when applicable,
- test updates for new behavior,
- example CLI output when user-facing messages change.
