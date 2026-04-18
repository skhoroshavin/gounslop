## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: (none removed)