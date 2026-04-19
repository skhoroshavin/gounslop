## REMOVED Requirements

### Requirement: Shared packages must have at least two consumers
**Reason**: Package-level false-sharing behavior now lives in the new `no-false-sharing` capability and is configured through `boundarycontrol` shared selectors.
**Migration**: Use `no-false-sharing` for the migrated shared-package threshold requirements.

### Requirement: File mode counts non-test files as distinct consumers
**Reason**: `file` mode is intentionally removed from the migrated false-sharing design.
**Migration**: Configure shared selectors under `boundarycontrol` and use the dir-style package consumer model defined by `no-false-sharing`.

### Requirement: Directory mode groups consumers by importing package path
**Reason**: The surviving package-path consumer model now lives in the new `no-false-sharing` capability.
**Migration**: Use `no-false-sharing` for dir-style consumer counting requirements.

### Requirement: Invalid false-sharing settings fail clearly
**Reason**: Migrated configuration validation now applies to `boundarycontrol` architecture settings and lives in the new `no-false-sharing` capability.
**Migration**: Use `no-false-sharing` for migrated false-sharing configuration validation requirements.
