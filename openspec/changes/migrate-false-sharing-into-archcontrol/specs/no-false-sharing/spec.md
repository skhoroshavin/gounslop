## ADDED Requirements

### Requirement: Shared packages must have at least two consuming packages
When `boundarycontrol` configuration declares a selector policy with `shared: true`, the system SHALL treat every package owned by that selector as a shared package and SHALL report a diagnostic when that package has fewer than two consuming package paths.

#### Scenario: Shared package has a single consuming package
- **WHEN** a shared package is imported by exactly one consuming package path
- **THEN** the analyzer reports that the package is only used by that package and must be used by two or more entities

#### Scenario: Shared package has no consuming packages
- **WHEN** a shared package is not imported by any consuming package path
- **THEN** the analyzer reports that the package is not imported by any entity and must be used by two or more entities

### Requirement: Consumers are counted by importing package path from non-test code
For migrated no-false-sharing evaluation, the system SHALL count consumers by importing package path rather than by individual files. `_test.go` files SHALL not increase the consumer count.

#### Scenario: Two files in the same package count as one consumer
- **WHEN** multiple non-test files in the same importing package reference the same shared package
- **THEN** the analyzer counts that importing package path as a single consumer entity

#### Scenario: Different importing packages satisfy the shared-package threshold
- **WHEN** two different importing package paths reference the same shared package
- **THEN** the analyzer does not report a false-sharing violation for that shared package

#### Scenario: Test files do not increase the consumer count
- **WHEN** a shared package is imported by one non-test file and one `_test.go` file from the same importing package
- **THEN** the analyzer still treats that importing package path as a single consumer entity

### Requirement: Invalid migrated false-sharing settings fail clearly
The system SHALL return an actionable configuration error when `boundarycontrol` architecture settings for migrated false-sharing cannot be decoded into the expected configuration shape or include removed consumer-grouping options.

#### Scenario: Shared flag has the wrong type
- **WHEN** plugin configuration provides `shared` as a non-boolean value under an `architecture` selector policy
- **THEN** the plugin setup fails with an error that identifies the `boundarycontrol` architecture settings problem

#### Scenario: Removed mode option is supplied
- **WHEN** plugin configuration provides `mode` under an `architecture` selector policy for migrated false-sharing settings
- **THEN** the plugin setup fails with an error that identifies `mode` as unsupported for migrated false-sharing configuration
