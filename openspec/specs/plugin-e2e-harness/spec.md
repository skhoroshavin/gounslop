## Purpose

Define the repository's reusable plugin end-to-end testing harness, including the suite interface, inline scenario authoring model, assertion behavior, analyzer coverage expectations, and command wiring needed to execute plugin-level tests.

## Requirements

### Requirement: Plugin E2E scenarios run against temporary Go workspaces
The repository SHALL provide a reusable E2E test harness that can materialize a temporary Go workspace from scenario-defined files, including one or more `go.mod` files, and execute the repository's `custom-gcl` binary against that workspace.

#### Scenario: Multi-package scenario is executed end to end
- **WHEN** a test defines a temporary module with multiple Go packages and enables a `gounslop` plugin through generated linter configuration
- **THEN** the harness runs `custom-gcl` against that temporary workspace and returns the resulting diagnostics to the test

#### Scenario: Multi-module scenario is executed end to end
- **WHEN** a test defines a temporary workspace containing a parent module and a nested module with separate `go.mod` files
- **THEN** the harness materializes that workspace layout and runs the selected plugin tests against it without requiring external fixture directories

### Requirement: Plugin E2E harness integrates with testify suites
The repository SHALL provide a `ruletest.Suite` base suite that embeds `testify/suite.Suite` and exposes `GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `FixFile`, `FixCode`, `ShouldPass`, `ShouldFailWith`, and `ShouldProduce` as suite methods.

#### Scenario: Analyzer suite embeds ruletest suite
- **WHEN** an analyzer test suite embeds `ruletest.Suite` and configures its linter name
- **THEN** its test methods can define files, execute one lint or fix operation, and assert the result without private `runScenario` or `runFixScenario` helpers

#### Scenario: Per-test state resets automatically
- **WHEN** testify invokes `SetupTest` before a suite test method
- **THEN** any files, config, temporary workspace state, and previous execution result from a prior test are cleared

### Requirement: Plugin E2E scenarios are defined inline
The repository SHALL allow E2E scenarios to define their file set, plugin settings, execution target, and expected outcome inline through `ruletest.Suite` methods without requiring fixture-directory inputs or raw `Scenario` structs.

#### Scenario: Single-file lint case stays compact
- **WHEN** a contributor defines inline code with `LintCode`
- **THEN** the contributor can express the file content as variadic lines and assert pass or fail in the same test method without constructing a file map or expectation struct

#### Scenario: Multi-file project is built inline
- **WHEN** a contributor uses multiple `GivenFile` calls and then executes one `LintFile` call
- **THEN** the harness materializes the full temporary project structure and lints only the requested target file

#### Scenario: Inline fix case stays compact
- **WHEN** a contributor defines inline code with `FixCode` or builds a project with `GivenFile` before `FixFile`
- **THEN** the contributor can assert the fixed output with `ShouldProduce` without constructing expected fixed-file maps

### Requirement: Plugin E2E results are actionable
The harness SHALL store the most recent lint or fix result on `ruletest.Suite` and expose assertion helpers that report stable, human-readable failures. It SHALL normalize temporary-path details so tests can assert on stable diagnostics, failure fragments, and fixed output.

#### Scenario: Passing run is asserted from the suite
- **WHEN** a test method executes `LintFile` or `LintCode` and then calls `ShouldPass`
- **THEN** the suite asserts a successful exit with no diagnostics and reports normalized output if the assertion fails

#### Scenario: Failing run is asserted from the suite
- **WHEN** a test method executes `LintFile` or `LintCode` and then calls `ShouldFailWith` with expected fragments
- **THEN** the suite asserts a non-zero exit and that each fragment appears in normalized output

#### Scenario: Fixed output is asserted from the suite
- **WHEN** a test method executes `FixFile` or `FixCode` and then calls `ShouldProduce`
- **THEN** the suite asserts that the fixed content of the executed file matches the provided variadic lines

#### Scenario: Assertion before execution fails clearly
- **WHEN** a test method calls `ShouldPass`, `ShouldFailWith`, or `ShouldProduce` before executing `LintFile`, `LintCode`, `FixFile`, or `FixCode`
- **THEN** the harness fails the test with a clear message that no result is available yet

### Requirement: Repository coverage includes representative plugin E2E cases
The repository SHALL include representative plugin-level E2E coverage for all existing analyzers, including failing cases, passing cases, and configuration-error cases as appropriate per analyzer. E2E tests SHALL be the default and only test approach — no `analysistest`-based tests or `testdata/` fixture directories SHALL remain in analyzer packages.

#### Scenario: All analyzers have E2E test coverage
- **WHEN** the test suite is inspected for each analyzer under `pkg/`
- **THEN** each shipped analyzer has a `plugin_test.go` with E2E scenarios covering its major behaviors, and no `analyzer_test.go` using `analysistest` remains

#### Scenario: boundarycontrol E2E coverage
- **WHEN** E2E tests for `boundarycontrol` are run
- **THEN** they cover: `architecture` map configuration, allowed imports, undeclared import violations, same-scope deep-import violations, auto-discovery of module scope from `go.mod`, and nested-module imports being treated as out of scope for the parent module

#### Scenario: nospecialunicode E2E coverage
- **WHEN** E2E tests for `nospecialunicode` are run
- **THEN** they cover: ASCII string passes, special Unicode punctuation flagged, raw string flagged, multiple banned characters reported

#### Scenario: nounicodeescape E2E coverage
- **WHEN** E2E tests for `nounicodeescape` are run
- **THEN** they cover: literal Unicode characters pass, `\uXXXX`/`\UXXXXXXXX` escapes flagged, raw strings not flagged

#### Scenario: readfriendlyorder E2E coverage
- **WHEN** E2E tests for `readfriendlyorder` are run
- **THEN** they cover: correct order passes, incorrect top-level order flagged, method ordering enforced, init ordering, TestMain ordering, cyclic dependencies exempt

#### Scenario: nofalsesharing E2E coverage preserved
- **WHEN** existing E2E tests for `nofalsesharing` are run after migration
- **THEN** they continue to cover the same scenarios (shared package violation, multiple consumers pass, invalid settings)

### Requirement: Repository E2E command wiring prepares the plugin binary outside the harness
The repository SHALL provide a command entrypoint for E2E execution that ensures `custom-gcl` is available before plugin E2E tests run, while keeping build orchestration outside the Go harness itself. E2E tests SHALL run as part of the default `make test` target without requiring build tags.

#### Scenario: E2E tests run by default via make test
- **WHEN** `make test` is run
- **THEN** the `custom-gcl` binary is built (if stale) and all E2E tests execute as part of `go test ./...` without requiring `-tags=e2e`

#### Scenario: Running go test without custom-gcl fails clearly
- **WHEN** `go test ./...` is run without `custom-gcl` in the repo root
- **THEN** E2E tests fail with a clear message indicating `custom-gcl` must be built first

#### Scenario: No e2e build tag is used
- **WHEN** test files in `pkg/` are inspected for build tags
- **THEN** none use `//go:build e2e`
