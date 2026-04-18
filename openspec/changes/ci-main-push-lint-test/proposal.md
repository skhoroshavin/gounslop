## Why

The repository currently relies on local execution for linting and tests, which can allow regressions to reach `main` when checks are skipped or environments differ. Adding automated CI on every push to `main` creates a consistent quality gate and faster feedback for maintainers.

## What Changes

- Add a GitHub Actions workflow that triggers on pushes to `main`.
- Name the workflow/pipeline `Test` for clear status visibility.
- Run repository linting via `make lint` in CI.
- Run repository tests via `make test` in CI.
- Ensure the workflow uses a pinned Go version compatible with the project (`go.mod`).

## Capabilities

### New Capabilities
- `github-actions-ci`: Define and run continuous integration checks (lint + test) for repository changes on push events.

### Modified Capabilities
- None.

## Impact

- Adds workflow configuration under `.github/workflows/`.
- Uses existing project Makefile targets (`make lint`, `make test`) without changing analyzer behavior.
- Introduces GitHub Actions runtime dependency for Go toolchain setup and CI execution.
- Keeps governance scope minimal: no branch-protection or required-status-check configuration changes.
