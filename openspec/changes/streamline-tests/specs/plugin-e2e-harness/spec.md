## ADDED Requirements

### Requirement: Plugin E2E harness integrates with testify suites
The repository SHALL provide a `ruletest.Suite` base suite that embeds `testify/suite.Suite` and exposes `GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `FixFile`, `FixCode`, `ShouldPass`, `ShouldFailWith`, and `ShouldProduce` as suite methods.

#### Scenario: Analyzer suite embeds ruletest suite
- **WHEN** an analyzer test suite embeds `ruletest.Suite` and configures its linter name
- **THEN** its test methods can define files, execute one lint or fix operation, and assert the result without private `runScenario` or `runFixScenario` helpers

#### Scenario: Per-test state resets automatically
- **WHEN** testify invokes `SetupTest` before a suite test method
- **THEN** any files, config, temporary workspace state, and previous execution result from a prior test are cleared

## MODIFIED Requirements

### Requirement: Plugin E2E scenarios are defined inline
The repository SHALL allow E2E cases to define their file set, plugin settings, execution target, and expected outcome inline through `ruletest.Suite` methods without requiring fixture-directory inputs or raw `Scenario` structs.

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
