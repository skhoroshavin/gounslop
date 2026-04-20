## Why

The current repository structure grew organically and now shows several friction points: tests live alongside implementation code in `pkg/`, the `EnableOnly` mechanism in the test harness allows running E2E tests with subsets of analyzers enabled, and the five analyzers are scattered across separate packages without clear grouping. These issues slow down contributors and increase the risk of incompatibilities slipping through.

## What Changes

- Move all E2E tests into a dedicated `tests/` directory, separated from implementation
- Remove the `EnableOnly` mechanism so every E2E test always runs with all analyzers enabled
- Reorganize `pkg/` into functional groups: `importcontrol/`, `exportcontrol/`, `nofalsesharing/`, `readfriendlyorder/`
- Return all analyzers from a single unified plugin with injected shared caches

## Capabilities

### Modified Capabilities

- `unified-plugin`: The plugin will register multiple analyzers (`importcontrol`, `exportcontrol`, `nofalsesharing`, `readfriendlyorder`, `nospecialunicode`, `nounicodeescape`) returned from a single plugin. Caches are injected from the root package. The package structure under `pkg/` will be reorganized into functional groups mirroring these analyzers.
- `plugin-e2e-harness`: E2E tests will be moved to a dedicated `tests/` directory, separated from implementation. The `EnableOnly` mechanism will be removed so every E2E test always runs with all rules enabled. Self-linting configuration continues to disable `nospecialunicode` and `nounicodeescape` to avoid flagging their own test data.

## Impact

- `pkg/boundarycontrol/` will be removed (split into `importcontrol/`, `exportcontrol/`, `nofalsesharing/`)
- `pkg/nospecialunicode/`, `pkg/nounicodeescape/`, `pkg/readfriendlyorder/` remain as separate packages
- `pkg/analyzer/` will be created for shared infrastructure
- `tests/` directory will be created with E2E coverage for each analyzer
- `pkg/gounslop/` remains the root package with plugin wiring and config
