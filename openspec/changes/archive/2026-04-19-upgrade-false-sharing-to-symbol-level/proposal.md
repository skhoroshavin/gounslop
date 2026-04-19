## Why

`boundarycontrol` currently treats false sharing as a package-level problem: a shared package passes as soon as two package paths import it. That leaves the largest remaining parity gap with upstream `no-false-sharing`, where the real signal is whether exported symbols are actually shared across consumers rather than whether the package is merely imported.

## What Changes

- Upgrade `boundarycontrol` false-sharing evaluation from package-level consumer counting to exported-symbol-level consumer counting for packages matched by `shared: true`.
- Keep the existing `architecture` and `shared: true` configuration shape stable; this change deepens analysis rather than introducing a new configuration model.
- **BREAKING** Change false-sharing diagnostics so code that currently passes on package-level imports can fail when shared packages expose exported symbols that are used by fewer than two consuming packages.
- Add analyzer, plugin, and documentation coverage for symbol-level false-sharing cases, including cases where multiple packages import the same shared package but do not share the same exported symbols.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `no-false-sharing`: Replace package-path consumer counting with exported-symbol-level consumer counting for shared packages, and update diagnostics and scenarios to reflect symbol-level sharing expectations.

## Impact

- Affects `pkg/boundarycontrol/false_sharing.go` and related analyzer support needed to inspect symbol usage across importing packages.
- Affects `pkg/boundarycontrol/*_test.go`, plugin E2E coverage, and spec coverage for false-sharing behavior.
- Affects user-visible diagnostics and README/spec documentation that currently describe package-level false-sharing semantics.
