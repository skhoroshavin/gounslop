## ADDED Requirements

### Requirement: CI runs on pushes to main
The repository SHALL execute a GitHub Actions workflow whenever a commit is pushed to the `main` branch.

#### Scenario: Push to main triggers workflow
- **WHEN** a commit is pushed to `main`
- **THEN** GitHub Actions starts the CI workflow for that commit

### Requirement: Workflow is named Test
The GitHub Actions workflow MUST be named `Test` so repository health is clearly visible under a consistent label.

#### Scenario: Workflow appears with requested name
- **WHEN** the workflow is displayed in GitHub Actions
- **THEN** its workflow name is `Test`

### Requirement: CI runs lint checks
The CI workflow MUST run repository linting using the project-defined lint command.

#### Scenario: Lint job executes make target
- **WHEN** the CI workflow runs
- **THEN** a lint job runs `make lint` and reports success or failure

### Requirement: CI runs tests
The CI workflow MUST run repository tests using the project-defined test command.

#### Scenario: Test job executes make target
- **WHEN** the CI workflow runs
- **THEN** a test job runs `make test` and reports success or failure

### Requirement: CI uses project Go version
Each CI job MUST use a Go version derived from the repository configuration to keep toolchain behavior aligned with local development.

#### Scenario: Setup Go from repository version source
- **WHEN** lint or test jobs initialize the build environment
- **THEN** the workflow configures Go using the version declared in the repository (for example `go.mod`)
