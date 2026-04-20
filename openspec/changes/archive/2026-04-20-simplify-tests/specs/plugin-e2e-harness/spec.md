## MODIFIED Requirements

### Requirement: Plugin E2E harness integrates with testify suites
The repository SHALL provide a `ruletest.Suite` base suite that embeds `testify/suite.Suite` and exposes `GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `FixFile`, `FixCode`, `ShouldPass`, `ShouldFailWith`, and `ShouldProduce` as suite methods. `GivenConfig` SHALL accept a typed config struct (the same struct used by the plugin's settings decoder) instead of `map[string]any`.

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
