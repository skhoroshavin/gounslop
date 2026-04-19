## MODIFIED Requirements

### Requirement: Plugin E2E scenarios run against temporary Go workspaces
The repository SHALL provide a reusable E2E test harness that can materialize a temporary Go workspace from scenario-defined files, including one or more `go.mod` files, and execute the repository's `custom-gcl` binary against that workspace.

#### Scenario: Multi-package scenario is executed end to end
- **WHEN** a test defines a temporary module with multiple Go packages and enables a `gounslop` plugin through generated linter configuration
- **THEN** the harness runs `custom-gcl` against that temporary workspace and returns the resulting diagnostics to the test

#### Scenario: Multi-module scenario is executed end to end
- **WHEN** a test defines a temporary workspace containing a parent module and a nested module with separate `go.mod` files
- **THEN** the harness materializes that workspace layout and runs the selected plugin tests against it without requiring external fixture directories

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
