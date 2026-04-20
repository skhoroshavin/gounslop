Guidance for coding agents working in `gounslop`.

## Purpose

- This repository publishes a golangci-lint v2 module plugin focused on detecting low-quality or AI-slop patterns in Go code.
- The codebase is small, pure Go, and optimized for strict linting and predictable analyzer behavior.
- Most changes involve analyzer logic, plugin registration, or analyzer tests.

## Project Snapshot

- Language: Go 1.25.6
- Analysis framework: `golang.org/x/tools/go/analysis`
- Plugin system: `github.com/golangci/plugin-module-register` (golangci-lint v2 module plugins)
- Build: `make` with a custom golangci-lint binary for self-linting

## Setup and Commands

- Build custom linter binary: `golangci-lint custom` (reads `.custom-gcl.yml`)
- Lint: `make lint`
- Test all: `make test`
- Lint and test (run after each serious change): `make lint && make test`
- Test a specific analyzer: `go test ./tests/ -run TestImportcontrolE2E`
- Single test by name: `go test ./tests/ -run TestSharedPackageWithSingleConsumerFails`

## Command Notes

- `make lint` builds `custom-gcl` if needed (tracks source file changes), then runs `./custom-gcl run ./...`
- `make test` maps to `go test ./...`
- The `custom-gcl` binary is gitignored and cached — it rebuilds only when `.go` files, `.custom-gcl.yml`, `go.mod`, or `go.sum` change

## Repository Layout

- `plugin/module.go`: plugin entrypoint — registers the unified plugin with golangci-lint
- `pkg/analyzer/`: shared analyzer infrastructure (module context discovery, config compilation, selector parsing, generic fixers)
- `pkg/importcontrol/`, `pkg/exportcontrol/`, `pkg/nofalsesharing/`: boundary-control analyzers importing `pkg/analyzer`
- `pkg/readfriendlyorder/`: code ordering rules (top-level, method, init, test ordering); imports `pkg/analyzer` for fixers
- `pkg/nospecialunicode/` and `pkg/nounicodeescape/`: self-contained analyzers with no `pkg/analyzer` imports
- `pkg/gounslop/`: root package with `Config`, `BuildAnalyzers`, and cache injection
- `tests/`: flat directory with E2E test files for each analyzer
- `tests/rule/`: reusable E2E harness (formerly `internal/ruletest/`)
- `.custom-gcl.yml`: golangci-lint custom binary build config
- `.golangci.yml`: linter config for self-linting
- `Makefile`: build and test targets

## Important Current Structure Notes

- `pkg/analyzer/` contains only generic infrastructure; it does not import any other `pkg/*` package
- `pkg/gounslop/` wires all analyzers together and injects shared caches
- `plugin/` imports only `pkg/gounslop`
- `tests/` imports only `pkg/gounslop` and `tests/rule`
- `tests/rule/` imports only `pkg/gounslop`
- `nospecialunicode` and `nounicodeescape` are not enabled for self-linting because they flag their own test data
- All E2E tests run with every analyzer enabled; there is no `EnableOnly` mechanism

## Local Lint Guardrails

- Self-linting uses `importcontrol`, `exportcontrol`, `nofalsesharing`, and `readfriendlyorder` from this repo via `.custom-gcl.yml`
- Standard linters enabled: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocritic`, `dupl` (threshold: 100)
- Formatters: `gofmt`, `goimports`

## Code Style: Imports

- Use standard Go import grouping: stdlib, then external packages, then internal packages
- Use the full module path for internal imports: `github.com/skhoroshavin/gounslop/pkg/...`

## Code Style: File Order

- Follow the dependency-direction ordering enforced by `readfriendlyorder`
- Exported symbols before unexported helpers that serve them
- `init()` before other functions
- Types and their methods grouped together

## Code Style: Formatting

- `gofmt` and `goimports` are enforced
- Prefer small focused functions and early returns
- Add comments only when the logic is not obvious from the code itself

## Code Style: Types

- Avoid `any` except where required by framework interfaces (`analysis.Analyzer.Run` returns `(any, error)`)
- Use concrete types and interfaces from `go/ast`, `go/types`, and `golang.org/x/tools/go/analysis`

## Naming Conventions

- Analyzer packages use lowercase concatenated names: `nospecialunicode`, `boundarycontrol`
- Analyzer names match package names
- Variables and functions use camelCase
- Types use PascalCase
- Constants with fixed symbolic meaning use camelCase (following existing style, e.g. `maxDirDepth`)
- Test names use `Test` prefix with PascalCase description

## Analyzer Authoring Conventions

- Self-contained analyzers (`nospecialunicode`, `nounicodeescape`, `readfriendlyorder`) export a single `Analyzer` variable of type `*analysis.Analyzer`
- Cache-dependent analyzers (`importcontrol`, `exportcontrol`, `nofalsesharing`) export a `Run` function and any cache types
- Include `Name`, `Doc`, `Requires`, and `Run` in dynamically created analyzers
- Use `Flags` for configuration on self-contained analyzers only
- Register analyzers in `pkg/gounslop/` with appropriate load mode (`LoadModeSyntax` for AST-only, `LoadModeTypesInfo` for type-aware)
- For configurable analyzers, settings are decoded in `pkg/gounslop/` and passed into analyzer closures

## Testing Conventions

- All test coverage is provided through the E2E framework in `tests/`
- No `_test.go` files remain in `pkg/`
- E2E tests live in `tests/<analyzer>_test.go` and import `tests/rule` and `pkg/gounslop`
- Every E2E test runs with all analyzers enabled; test data must not trigger cross-analyzer conflicts
- The harness hardcodes the linter name as `gounslop` and never generates a `disable` list unless the test supplies one via `GivenConfig`

## Error Handling and Resilience

- Return `(nil, nil)` from `Run` when prerequisites are intentionally absent; fail clearly when required module context cannot be discovered
- Guard nullable values before dereferencing
- Use `sync.Once` for expensive one-time computations across multiple package passes when analyzer-wide caches are needed

## Change Workflow for Agents

- Read the analyzer, nearby helpers, and its test file before changing behavior
- Keep edits scoped and avoid opportunistic refactors
- Run the most targeted test command first, then `make lint && make test` for full validation
- When adding a new analyzer: create package under `pkg/`, register in `pkg/gounslop/`, add to `.golangci.yml` if appropriate for self-linting
