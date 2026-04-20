## 1. Foundation — Shared Infrastructure

- [ ] 1.1 Create `pkg/analyzer/` package with module context discovery (`module_context.go`) extracted from `pkg/boundarycontrol/`
- [ ] 1.2 Move config compilation and selector parsing from `pkg/boundarycontrol/analyzer.go` into `pkg/analyzer/`
- [ ] 1.3 Move generic fixer utilities from `pkg/readfriendlyorder/fixer.go` into `pkg/analyzer/`
- [ ] 1.4 Implement `ModuleContextCache` struct with `NewModuleContextCache()` constructor in `pkg/analyzer/`
- [ ] 1.5 Ensure `pkg/analyzer/` has no imports of other `pkg/*` packages

## 2. Core Analyzer Packages

- [ ] 2.1 Create `pkg/importcontrol/` with AST traversal, deep-import logic, and diagnostic reporting; import `pkg/analyzer` for shared infrastructure
- [ ] 2.2 Create `pkg/exportcontrol/` with exported symbol enumeration, regex matching, and diagnostic reporting; import `pkg/analyzer`
- [ ] 2.3 Create `pkg/nofalsesharing/` with symbol consumer counting, cross-package reference analysis, and diagnostic reporting; import `pkg/analyzer`
- [ ] 2.4 Implement `Cache` struct with `NewCache()` constructor in `pkg/nofalsesharing/`
- [ ] 2.5 Refactor `pkg/readfriendlyorder/` to import generic fixers from `pkg/analyzer/`; keep rule-specific ordering logic
- [ ] 2.6 Verify `pkg/nospecialunicode/` and `pkg/nounicodeescape/` remain self-contained with no `pkg/analyzer` imports

## 3. Plugin Root and Wiring

- [ ] 3.1 Create `pkg/gounslop/` with `Config`, `PolicyConfig`, and `BuildAnalyzers` returning multiple `*analysis.Analyzer` values
- [ ] 3.2 Implement cache injection in `BuildAnalyzers`: instantiate `ModuleContextCache` and `nofalsesharing.Cache`, pass into analyzer closures
- [ ] 3.3 Implement `disable` list filtering using new analyzer names (`importcontrol`, `exportcontrol`, etc.)
- [ ] 3.4 Validate unknown entries in `disable` list and return descriptive errors
- [ ] 3.5 Decode and apply `architecture` settings to `importcontrol`, `exportcontrol`, and `nofalsesharing` flags
- [ ] 3.6 Update `plugin/` to import only `pkg/gounslop` and register the unified plugin

## 4. E2E Test Harness

- [ ] 4.1 Create `tests/rule/` package from `internal/ruletest/`
- [ ] 4.2 Remove `EnableOnly` field and associated `disableComplement`/`validateEnableOnly` logic from harness
- [ ] 4.3 Update `GivenConfig` to accept typed config struct from `pkg/gounslop`
- [ ] 4.4 Update `renderConfig` to generate config enabling single `gounslop` linter with `type: "module"`
- [ ] 4.5 Ensure harness hardcodes linter name as `gounslop` and never generates a `disable` list unless test supplies one via `GivenConfig`

## 5. E2E Test Migration

- [ ] 5.1 Create `tests/importcontrol_test.go` from `pkg/boundarycontrol/plugin_test.go` (import-boundary scenarios)
- [ ] 5.2 Create `tests/exportcontrol_test.go` from `pkg/boundarycontrol/plugin_test.go` (export contract scenarios)
- [ ] 5.3 Create `tests/nofalsesharing_test.go` from `pkg/boundarycontrol/false_sharing_plugin_test.go`
- [ ] 5.4 Create `tests/readfriendlyorder_test.go` from `pkg/readfriendlyorder/plugin_test.go`
- [ ] 5.5 Create `tests/nospecialunicode_test.go` from `pkg/nospecialunicode/plugin_test.go`
- [ ] 5.6 Create `tests/nounicodeescape_test.go` from `pkg/nounicodeescape/plugin_test.go`
- [ ] 5.7 Rewrite `nounicodeescape` test data to use Unicode characters not on `nospecialunicode` banned list (e.g., `é`, `ñ`, `中`)
- [ ] 5.8 Audit all test data to eliminate cross-analyzer conflicts

## 6. Cleanup and Documentation

- [ ] 6.1 Remove old `pkg/boundarycontrol/` package entirely
- [ ] 6.2 Remove all `pkg/*/plugin_test.go` files from analyzer packages
- [ ] 6.3 Remove `internal/ruletest/` directory entirely
- [ ] 6.4 Update `Makefile` to reflect new test paths and commands
- [ ] 6.5 Update `AGENTS.md` with new package layout, analyzer names, and testing conventions
- [ ] 6.6 Update `README.md` with new structure and analyzer names
- [ ] 6.7 Update `.golangci.yml` to use new analyzer names in `disable` list
- [ ] 6.8 Update `openspec/specs/unified-plugin/spec.md` to reflect multiple analyzers and new package layout
- [ ] 6.9 Update `openspec/specs/plugin-e2e-harness/spec.md` to reflect `EnableOnly` removal and `tests/` directory

## 7. Validation

- [ ] 7.1 Run `go mod tidy` to validate module graph after all import changes
- [ ] 7.2 Run `make lint` and fix any self-linting failures
- [ ] 7.3 Run `make test` and ensure all E2E tests pass with all analyzers enabled
- [ ] 7.4 Verify `custom-gcl` binary builds correctly with updated plugin wiring
- [ ] 7.5 Verify no `_test.go` files remain under `pkg/`
- [ ] 7.6 Verify `tests/` directory contains E2E coverage for all six analyzers
