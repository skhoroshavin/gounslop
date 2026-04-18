## ADDED Requirements

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
The repository SHALL include representative plugin-level E2E coverage for at least one existing analyzer, including a failing multi-package case, a passing case, and a configuration-error case.

#### Scenario: Seed analyzer coverage exists
- **WHEN** the initial harness change is complete
- **THEN** the repository contains plugin E2E tests that cover a real analyzer with failing, passing, and configuration-error scenarios

### Requirement: Repository E2E command wiring prepares the plugin binary outside the harness
The repository SHALL provide a command entrypoint for E2E execution that ensures `custom-gcl` is available before plugin E2E tests run, while keeping build orchestration outside the Go harness itself.

#### Scenario: E2E suite is invoked from repository tooling
- **WHEN** a contributor runs the repository's E2E test command
- **THEN** the command ensures `custom-gcl` is prepared before the E2E tests execute and the Go harness only consumes the existing binary
