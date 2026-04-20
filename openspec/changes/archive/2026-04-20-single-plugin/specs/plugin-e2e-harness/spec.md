## MODIFIED Requirements

### Requirement: Plugin E2E harness integrates with testify suites
The repository SHALL provide a `ruletest.Suite` base suite that embeds `testify/suite.Suite` and exposes `GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `FixFile`, `FixCode`, `ShouldPass`, `ShouldFailWith`, and `ShouldProduce` as suite methods. The suite SHALL hardcode the linter name as `gounslop` internally. The suite SHALL expose an `EnableOnly` field (`[]string`) that controls which analyzers are active for a given test run. When `EnableOnly` is set, the harness SHALL generate a `disable` list in the rendered config containing all known analyzer names except those in `EnableOnly`.

#### Scenario: Analyzer suite embeds ruletest suite with EnableOnly
- **WHEN** an analyzer test suite embeds `ruletest.Suite` and sets `EnableOnly` to `["boundarycontrol"]`
- **THEN** the generated `.golangci.yml` enables the `gounslop` linter with `disable` containing `nospecialunicode`, `nounicodeescape`, and `readfriendlyorder`

#### Scenario: Per-test state resets automatically
- **WHEN** testify invokes `SetupTest` before a suite test method
- **THEN** any files, config, `EnableOnly`, temporary workspace state, and previous execution result from a prior test are cleared

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
