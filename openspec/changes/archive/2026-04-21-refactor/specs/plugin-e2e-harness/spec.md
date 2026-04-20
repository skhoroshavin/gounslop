## MODIFIED Requirements

### Requirement: Plugin E2E harness integrates with testify suites
The repository SHALL provide a `ruletest.Suite` base suite that embeds `testify/suite.Suite` and exposes `GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `FixFile`, `FixCode`, `ShouldPass`, `ShouldFailWith`, and `ShouldProduce` as suite methods. The suite SHALL hardcode the linter name as `gounslop` internally. `GivenConfig` SHALL accept a typed config struct (the same struct used by the plugin's settings decoder) instead of `map[string]any`.

#### Scenario: Analyzer suite embeds ruletest suite
- **WHEN** an analyzer test suite embeds `ruletest.Suite`
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

### Requirement: Harness config generation uses the unified gounslop plugin
The harness `renderConfig` function SHALL generate `.golangci.yml` content that enables the single `gounslop` linter under `linters.settings.custom` with `type: "module"`. The generated settings SHALL include any test-supplied settings from `GivenConfig` without adding a `disable` list.

#### Scenario: Config renders with test-supplied settings
- **WHEN** `GivenConfig` provides `{"architecture": {...}}`
- **THEN** the rendered config enables `gounslop` with settings containing the `architecture` map

#### Scenario: Config renders with no test settings
- **WHEN** no `GivenConfig` is called
- **THEN** the rendered config enables `gounslop` with empty settings

### Requirement: Repository coverage includes representative plugin E2E cases
The repository SHALL include representative plugin-level E2E coverage for all existing analyzers, including failing cases, passing cases, and configuration-error cases as appropriate per analyzer. E2E tests SHALL be the default and only test approach — no `analysistest`-based tests or `testdata/` fixture directories SHALL remain in analyzer packages. All E2E test files SHALL live in the top-level `tests/` directory.

#### Scenario: All analyzers have E2E test coverage
- **WHEN** the test suite is inspected
- **THEN** each shipped analyzer has E2E scenarios in `tests/<analyzer>_test.go` covering its major behaviors, and no `analysistest`-based tests remain in `pkg/`

#### Scenario: importcontrol E2E coverage
- **WHEN** E2E tests for `importcontrol` are run
- **THEN** they cover: `architecture` map configuration, allowed imports, undeclared import violations, same-scope deep-import violations, auto-discovery of module scope from `go.mod`, and nested-module imports being treated as out of scope for the parent module

#### Scenario: exportcontrol E2E coverage
- **WHEN** E2E tests for `exportcontrol` are run
- **THEN** they cover: allowed exports, export contract violations, regex pattern matching

#### Scenario: nofalsesharing E2E coverage
- **WHEN** E2E tests for `nofalsesharing` are run
- **THEN** they cover: shared-package configuration failures, shared-package consumer threshold checks, and multi-consumer passing cases

#### Scenario: nospecialunicode E2E coverage
- **WHEN** E2E tests for `nospecialunicode` are run
- **THEN** they cover: ASCII string passes, special Unicode punctuation flagged, raw string flagged, multiple banned characters reported

#### Scenario: nounicodeescape E2E coverage
- **WHEN** E2E tests for `nounicodeescape` are run
- **THEN** they cover: literal Unicode characters pass, `\uXXXX`/`\UXXXXXXXX` escapes flagged, raw strings not flagged

#### Scenario: readfriendlyorder E2E coverage
- **WHEN** E2E tests for `readfriendlyorder` are run
- **THEN** they cover: correct order passes, incorrect top-level order flagged, method ordering enforced, init ordering, TestMain ordering, cyclic dependencies exempt

## REMOVED Requirements

### Requirement: Harness maintains a known analyzer name list
**Reason:** The `EnableOnly` mechanism and its associated `disable` complement computation have been removed. Tests always run with all analyzers enabled.
**Migration:** No replacement needed. The harness no longer generates `disable` lists.

### Requirement: Harness config generation merges EnableOnly-derived disable list
**Reason:** The `EnableOnly` field has been removed from `ruletest.Suite`. All E2E tests run with all analyzers enabled by default.
**Migration:** Tests that need to verify specific rule behavior in isolation should use `GivenConfig` with a selective `disable` list.
