## Requirements

### Requirement: Plugin E2E scenarios run against temporary Go workspaces
The repository SHALL provide a reusable E2E test harness that can materialize a temporary Go workspace from scenario-defined files and execute the repository's `custom-gcl` binary against that workspace.

#### Scenario: Multi-package scenario is executed end to end
- **WHEN** a test defines a temporary module with multiple Go packages and enables a `gounslop` plugin through generated linter configuration
- **THEN** the harness runs `custom-gcl` against that temporary workspace and returns the resulting diagnostics to the test

### Requirement: Plugin E2E scenarios are defined inline
The repository SHALL allow E2E scenarios to define their file set, module contents, plugin settings, and expected outcome inline in Go tests without requiring fixture-directory inputs.

#### Scenario: Scenario stays compact in test code
- **WHEN** a contributor adds a new E2E case for an analyzer
- **THEN** the contributor can express the scenario as a compact inline Go definition without creating a dedicated fixture directory

### Requirement: Plugin E2E results are actionable
The harness SHALL expose whether a scenario completed without error or failed with a human-readable, directly actionable message. It SHALL normalize temporary-path details so tests can assert on stable diagnostics and failure fragments.

#### Scenario: Successful run produces stable diagnostics
- **WHEN** a scenario triggers a plugin diagnostic in a temporary workspace
- **THEN** the test can assert on normalized diagnostic output without depending on machine-specific temporary paths

#### Scenario: Failing run produces actionable error output
- **WHEN** a scenario provides invalid plugin configuration or otherwise causes `custom-gcl` to fail
- **THEN** the harness returns an error result whose message is readable enough for a human or LLM to act on directly

### Requirement: Repository coverage includes representative plugin E2E cases
The repository SHALL include representative plugin-level E2E coverage for all existing analyzers, including failing cases, passing cases, and configuration-error cases as appropriate per analyzer. E2E tests SHALL be the default and only test approach — no `analysistest`-based tests or `testdata/` fixture directories SHALL remain in analyzer packages.

#### Scenario: All analyzers have E2E test coverage
- **WHEN** the test suite is inspected for each analyzer under `pkg/`
- **THEN** each analyzer has a `plugin_test.go` with E2E scenarios covering its major behaviors, and no `analyzer_test.go` using `analysistest` remains

#### Scenario: nodeepimports E2E coverage
- **WHEN** E2E tests for `nodeepimports` are run
- **THEN** they cover: one-level deep import passes, deep import flagged, different top-level scope passes, test file import passes

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
