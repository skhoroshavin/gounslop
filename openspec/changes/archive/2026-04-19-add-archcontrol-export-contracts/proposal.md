## Why

`boundarycontrol` now owns the repository's shared architecture model for package selectors, import policy, and shared-package analysis, but it still cannot express export contracts for public package surfaces. Adding that capability closes the remaining planned parity gap around upstream `export-control` while keeping architecture policy in one Go-native configuration model instead of introducing another disconnected rule surface.

## What Changes

- Add export-contract enforcement driven by `boundarycontrol` architecture selectors.
- Allow selector policies to declare regex-based contracts for exported top-level declaration names in matched packages.
- Define how export contracts apply to package groups that are intended to act as public API surfaces, without changing import-control or false-sharing behavior.
- Validate export-contract configuration during plugin setup and analyzer startup, including clear failures for invalid regex patterns or unsupported settings shapes.
- Add analyzer, spec, and end-to-end coverage for compliant exports, contract violations, and configuration-error scenarios.

## Capabilities

### New Capabilities
- `export-control`: Enforce regex-based contracts for exported top-level declarations in package groups selected through the shared architecture model.

### Modified Capabilities
- `boundarycontrol`: Extend selector-owned architecture policy so matched package groups can carry export-contract settings and participate in export-surface targeting.

## Impact

- Affects `pkg/boundarycontrol` configuration parsing, validation, and analysis flow to evaluate exported declarations against configured contracts.
- Affects `plugin/module.go` settings decoding so `boundarycontrol` can accept and validate export-contract policy fields.
- Adds `openspec/specs/export-control/spec.md` and updates `openspec/specs/boundarycontrol/spec.md` to describe the new configuration surface.
- Adds analyzer tests and plugin E2E coverage for valid contracts, invalid contracts, and export naming violations.
