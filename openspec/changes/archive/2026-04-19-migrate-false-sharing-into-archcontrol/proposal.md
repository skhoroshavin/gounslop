## Why

`nofalsesharing` still exists as a separate plugin even though its configuration and analysis now belong with the repository's broader package-selector architecture model. Removing that standalone plugin now avoids carrying two parallel entry points for one concern and gives this change a clean point to also untangle the current spec layout before more architecture work lands.

## What Changes

- Fully remove the standalone `nofalsesharing` plugin entry point and move its package-level false-sharing functionality under `boundarycontrol` configuration and execution.
- Preserve the current package-level false-sharing threshold and package-level consumer grouping during the migration, while removing `file` mode and standardizing on `dir` semantics only.
- Rename the current `architecture-control` capability to `boundarycontrol` and narrow it so it contains only general architecture-model and selector-resolution requirements.
- Move all import-policy requirements out of the current architecture spec into a separate `import-control` capability.
- Rename the current false-sharing capability to `no-false-sharing` and keep it focused on the false-sharing behavior that is now hosted by `boundarycontrol`.
- Add spec and test coverage for plugin removal, new configuration ownership, updated capability boundaries, and preserved package-level false-sharing diagnostics.
- Keep symbol-level false-sharing analysis out of scope for this change; that remains a follow-up enhancement.

## Capabilities

### New Capabilities
- `boundarycontrol`: general architecture and selector-ownership capability that becomes the home for shared package declarations used by boundary-aware analyzers.
- `import-control`: import-policy requirements split out from the current architecture capability.
- `no-false-sharing`: package-level false-sharing requirements after the standalone `nofalsesharing` plugin is removed and its behavior is hosted by `boundarycontrol`.

### Modified Capabilities
- `architecture-control`: retire this capability in favor of `boundarycontrol` for general selector semantics and `import-control` for import-policy behavior.
- `false-sharing`: retire this capability in favor of `no-false-sharing`, with requirements rewritten around the migrated behavior that now runs through `boundarycontrol`.

## Impact

- User-facing `golangci-lint` plugin names and configuration for false-sharing and import-boundary behavior.
- Removal of standalone `nofalsesharing` plugin wiring from `plugin/module.go` and related settings handling.
- `pkg/nofalsesharing` analyzer ownership, configuration flow, and any code moved or absorbed into `boundarycontrol`.
- Shared architecture config parsing and validation used by `boundarycontrol`, plus new separation between selector semantics and import-policy rules.
- OpenSpec capability layout under `openspec/specs/`, including capability renames and the split of import requirements into a separate spec.
- Analyzer, plugin, and spec tests that cover plugin removal, spec reorganization, configuration migration, and preserved package-level diagnostics.
