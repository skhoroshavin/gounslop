## Why

`boundarycontrol` currently requires a verbose selector-list config, depends on a manually supplied `module-root`, and overlaps with `nodeepimports` for same-scope deep-import enforcement. This makes the rule harder to adopt, awkward in multi-module repositories, and leaves import-architecture policy split across two analyzers.

## What Changes

- Replace the current ordered `selectors` list with an `architecture` mapping keyed by package selector.
- Make `boundarycontrol` automatically detect its module root from the nearest relevant `go.mod` instead of requiring `module-root` in normal usage.
- Extend `boundarycontrol` so it can evaluate packages correctly in repositories that contain multiple Go modules.
- Fold the remaining useful `nodeepimports` behavior fully into `boundarycontrol`.
- **BREAKING** Remove the standalone `nodeepimports` rule from the plugin, configuration, documentation, and test matrix.
- **BREAKING** Replace the current `boundarycontrol` settings shape with the new nested `architecture` form.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `architecture-control`: Change `boundarycontrol` configuration to the new `architecture` map form, make module-root resolution automatic and multi-module aware, and make `boundarycontrol` the sole home of the deep-import restriction currently associated with `nodeepimports`.
- `plugin-e2e-harness`: Update representative analyzer coverage expectations to remove `nodeepimports` and cover the new `boundarycontrol` configuration and module-discovery behavior.

## Impact

- Affects `pkg/boundarycontrol`, `pkg/nodeepimports`, `plugin/module.go`, plugin registration tests, E2E coverage, and repository docs/config examples such as `README.md` and `.golangci.yml`.
- Requires a migration from `selectors` to `architecture` for existing `boundarycontrol` users.
- Simplifies the public architecture-policy story by consolidating import-boundary and deep-import enforcement into one rule.
