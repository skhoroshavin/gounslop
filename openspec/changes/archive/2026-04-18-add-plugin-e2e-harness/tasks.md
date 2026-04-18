## 1. Shared Harness

- [x] 1.1 Add a shared internal E2E test helper package for temporary workspace creation and file materialization.
- [x] 1.2 Implement inline scenario configuration for files, optional module setup, plugin settings, and expected outcomes.
- [x] 1.3 Implement `custom-gcl` command execution against temporary workspaces and capture exit status, stdout, and stderr.
- [x] 1.4 Normalize temp-path-specific output so tests can assert on stable, actionable diagnostics and failure fragments.

## 2. Seed Analyzer Coverage

- [x] 2.1 Add a failing multi-package `nofalsesharing` E2E scenario that reports a shared-package violation through the plugin binary.
- [x] 2.2 Add a passing multi-package `nofalsesharing` E2E scenario that produces no error when multiple consumers exist.
- [x] 2.3 Add a `nofalsesharing` configuration-error E2E scenario that verifies invalid plugin settings surface an actionable failure message.

## 3. Repository Wiring

- [x] 3.1 Add repository command wiring for E2E execution so `custom-gcl` is prepared outside the Go harness before E2E tests run.
- [x] 3.2 Ensure the chosen E2E command integrates cleanly with the existing local and CI-oriented test workflow.

## 4. Documentation And Validation

- [x] 4.1 Document when to use the shared plugin E2E harness versus `analysistest`, including the preference for compact inline scenarios.
- [x] 4.2 Run the targeted E2E tests, then run the repository validation commands needed to confirm the new harness and seed scenarios pass.
