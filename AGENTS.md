# AGENTS.md

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
- Test a specific analyzer: `go test ./pkg/readfriendlyorder/`
- Single test by name: `go test ./pkg/boundarycontrol/ -run TestSharedPackageWithSingleConsumerFails`

## Command Notes

- `make lint` builds `custom-gcl` if needed (tracks source file changes), then runs `./custom-gcl run ./...`
- `make test` maps to `go test ./...`
- The `custom-gcl` binary is gitignored and cached — it rebuilds only when `.go` files, `.custom-gcl.yml`, `go.mod`, or `go.sum` change

## Repository Layout

- `plugin/module.go`: plugin entrypoint — registers all analyzers with golangci-lint
- `pkg/<analyzer>/analyzer.go`: each analyzer exports a single `Analyzer` variable of type `*analysis.Analyzer`
- `pkg/<analyzer>/testdata/src/`: test packages with `// want "..."` annotations
- `pkg/<analyzer>/*_test.go`: tests using `analysistest`
- `.custom-gcl.yml`: golangci-lint custom binary build config
- `.golangci.yml`: linter config for self-linting
- `Makefile`: build and test targets

## Important Current Structure Notes

- Each analyzer lives in its own package under `pkg/`
- `boundarycontrol` owns both import-boundary enforcement and shared-package false-sharing checks
- `readfriendlyorder` is split across `analyzer.go`, `method_order.go`, and `test_order.go`
- `nospecialunicode` and `nounicodeescape` are not enabled for self-linting because they flag their own test data

## Local Lint Guardrails

- Self-linting uses `boundarycontrol` and `readfriendlyorder` from this repo via `.custom-gcl.yml`
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

- Export a single `Analyzer` variable of type `*analysis.Analyzer` from each package
- Include `Name`, `Doc`, `Requires`, and `Run` in the analyzer definition
- Use `Flags` for configuration (e.g. `Analyzer.Flags.StringVar(...)`)
- Register the analyzer in `plugin/module.go` with appropriate load mode (`LoadModeSyntax` for AST-only, `LoadModeTypesInfo` for type-aware)
- For configurable analyzers, add a settings struct in `plugin/module.go`

## Testing Conventions

- Use `analysistest` from `golang.org/x/tools/go/analysis/analysistest`
- Test data lives in `testdata/src/` under each analyzer package
- Annotate expected diagnostics with `// want "..."` comments in test data files
- Use `analysistest.Run(t, analysistest.TestData(), Analyzer, "package/path")` pattern
- Test both valid code (no diagnostics) and invalid code (expected diagnostics)

## Error Handling and Resilience

- Return `(nil, nil)` from `Run` when prerequisites are intentionally absent; fail clearly when required module context cannot be discovered
- Guard nullable values before dereferencing
- Use `sync.Once` for expensive one-time computations across multiple package passes when analyzer-wide caches are needed

## Change Workflow for Agents

- Read the analyzer, nearby helpers, and its test file before changing behavior
- Keep edits scoped and avoid opportunistic refactors
- Run the most targeted test command first, then `make lint && make test` for full validation
- When adding a new analyzer: create package under `pkg/`, register in `plugin/module.go`, add to `.golangci.yml` if appropriate for self-linting
