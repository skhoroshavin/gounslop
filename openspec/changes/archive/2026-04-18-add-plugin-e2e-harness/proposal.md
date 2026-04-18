## Why

The repository currently relies mainly on analyzer-level tests, which makes multi-package and plugin-integration scenarios harder to exercise and reuse. Adding a shared E2E harness now will make the upcoming architecture-oriented changes safer to implement and easier to validate end to end.

## What Changes

- Add a reusable plugin E2E test harness for creating temporary Go modules, writing fixture files, and running `custom-gcl` against them.
- Define a consistent way to assert diagnostics, command failures, and configuration errors in plugin-level scenarios.
- Add representative E2E coverage for an existing analyzer, including at least one multi-package case and one config-error case.
- Document the intended role of the harness alongside existing `analysistest` coverage so future changes reuse the same testing approach.

## Capabilities

### New Capabilities

- `plugin-e2e-harness`: Reusable test support for fixture-driven, temporary-repository plugin runs that validate diagnostics and configuration failures through the custom golangci-lint binary.

### Modified Capabilities

- None.

## Impact

- Affects test infrastructure and test documentation for this repository.
- Adds plugin-level test coverage that exercises `custom-gcl` in temporary workspaces.
- No analyzer behavior changes, runtime API changes, or new lint rules are introduced by this change.
