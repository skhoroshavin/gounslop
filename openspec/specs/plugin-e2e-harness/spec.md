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
The repository SHALL provide a `ruletest.Suite` base suite that embeds `testify/suite.Suite` and exposes `GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `FixFile`, `FixCode`, `ShouldPass`, `ShouldFailWith`, and `ShouldProduce` as suite methods. The suite SHALL hardcode the linter name as `gounslop` internally. The suite SHALL expose an `EnableOnly` field (`[]string`) that controls which analyzers are active for a given test run. When `EnableOnly` is set, the harness SHALL generate a `disable` list in the rendered config containing all known analyzer names except those in `EnableOnly`. `GivenConfig` SHALL accept a typed config struct (the same struct used by the plugin's settings decoder) instead of `map[string]any`.

#### Scenario: Analyzer suite embeds ruletest suite
- **WHEN** an analyzer test suite embeds `ruletest.Suite` and configures its linter name
- **THEN** its test methods can define files, execute one lint or fix operation, and assert the result without private `runScenario` or `runFixScenario` helpers

#### Scenario: Per-test state resets automatically
- **WHEN** testify invokes `SetupTest` before a suite test method
- **THEN** any files, config, temporary workspace state, and previous execution result from a prior test are cleared

#### Scenario: GivenConfig accepts typed struct
- **WHEN** a test method calls `GivenConfig` with a typed settings struct
- **THEN** the harness stores the struct and serializes it into the generated `.golangci.yml` without requiring the test author to construct `map[string]any`

#### Scenario: GivenConfig with nil or zero-value config
- **WHEN** a test method calls `GivenConfig` with the zero value of the settings struct
- **THEN** the harness treats it as no custom settings, equivalent to the previous `nil` `map[string]any` behavior

### Requirement: Plugin E2E scenarios are defined inline
The repository SHALL allow E2E scenarios to define their file set, plugin settings, execution target, and expected outcome inline through `ruletest.Suite` methods without requiring fixture-directory inputs or raw `Scenario` structs. Plugin settings SHALL be expressed as typed struct literals rather than `map[string]any` literals.

#### Scenario: Single-file lint case stays compact
- **WHEN** a contributor defines inline code with `LintCode`
- **THEN** the contributor can express the file content as variadic lines and assert pass or fail in the same test method without constructing a file map or expectation struct

#### Scenario: Multi-file project is built inline
- **WHEN** a contributor uses multiple `GivenFile` calls and then executes one `LintFile` call
- **THEN** the harness materializes the full temporary project structure and lints only the requested target file

#### Scenario: Inline fix case stays compact
- **WHEN** a contributor defines inline code with `FixCode` or builds a project with `GivenFile` before `FixFile`
- **THEN** the contributor can assert the fixed output with `ShouldProduce` without constructing expected fixed-file maps

#### Scenario: Typed config struct is used inline
- **WHEN** a test author writes a `GivenConfig` call
- **THEN** they use a typed struct literal (e.g., `GounslopSettings{Architecture: map[string]PolicySettings{...}}`) instead of `map[string]any{"architecture": map[string]any{...}}`

### Requirement: Harness config generation uses the unified gounslop plugin
The harness `renderConfig` function SHALL generate `.golangci.yml` content that enables the single `gounslop` linter under `linters.settings.custom` with `type: "module"`. The generated settings SHALL merge the `disable` list derived from `EnableOnly` with any test-supplied settings from `GivenConfig`.

#### Scenario: Config renders with disable list and architecture
- **WHEN** `EnableOnly` is `["boundarycontrol"]` and `GivenConfig` provides `{"architecture": {...}}`
- **THEN** the rendered config enables `gounslop` with settings containing both the `disable` list and the `architecture` map

#### Scenario: Config renders with no test settings
- **WHEN** `EnableOnly` is `["nospecialunicode"]` and no `GivenConfig` is called
- **THEN** the rendered config enables `gounslop` with settings containing only the `disable` list

#### Scenario: Test-supplied disable overrides harness-generated disable
- **WHEN** `GivenConfig` provides a `disable` key
- **THEN** the test-supplied `disable` list takes precedence over the harness-generated one from `EnableOnly`

### Requirement: Harness maintains a known analyzer name list
The harness SHALL maintain an internal list of all known analyzer names for computing the `disable` complement from `EnableOnly`. This list SHALL be updated when new analyzers are added to the plugin.

#### Scenario: EnableOnly with unknown analyzer name
- **WHEN** `EnableOnly` contains a name that is not in the known analyzer list
- **THEN** the harness fails the test with a clear error identifying the unknown analyzer name

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
- **THEN** they cover: `architecture` map configuration, allowed imports, undeclared import violations, same-scope deep-import violations, shared-package configuration failures, shared-package consumer threshold checks, auto-discovery of module scope from `go.mod`, and nested-module imports being treated as out of scope for the parent module

#### Scenario: nospecialunicode E2E coverage
- **WHEN** E2E tests for `nospecialunicode` are run
- **THEN** they cover: ASCII string passes, special Unicode punctuation flagged, raw string flagged, multiple banned characters reported

#### Scenario: nounicodeescape E2E coverage
- **WHEN** E2E tests for `nounicodeescape` are run
- **THEN** they cover: literal Unicode characters pass, `\uXXXX`/`\UXXXXXXXX` escapes flagged, raw strings not flagged

#### Scenario: readfriendlyorder E2E coverage
- **WHEN** E2E tests for `readfriendlyorder` are run
- **THEN** they cover: correct order passes, incorrect top-level order flagged, method ordering enforced, init ordering, TestMain ordering, cyclic dependencies exempt

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
