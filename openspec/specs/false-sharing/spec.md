## Purpose

Define the baseline false-sharing rules for shared package consumption thresholds, consumer-counting modes, and invalid configuration handling.

## Requirements

### Requirement: Shared packages must have at least two consumers
When `shared-dirs` is configured, the system SHALL treat every package under each configured shared directory as a shared package and SHALL report a diagnostic when that package has fewer than two consumer entities.

#### Scenario: Shared package has a single consumer
- **WHEN** a shared package is imported by exactly one consumer entity
- **THEN** the analyzer reports that the package is only used by that entity and must be used by two or more entities

#### Scenario: Shared package has no consumers
- **WHEN** a shared package is not imported by any consumer entity
- **THEN** the analyzer reports that the package is not imported by any entity and must be used by two or more entities

### Requirement: File mode counts non-test files as distinct consumers
In `file` mode, the system SHALL count each importing non-test Go file as a separate consumer entity. `_test.go` files SHALL not count as consumers.

#### Scenario: Two files in the same package count separately in file mode
- **WHEN** two different non-test files in the same importing package each import the same shared package while `mode` is `file`
- **THEN** the analyzer counts them as two consumers and does not report a false-sharing violation

#### Scenario: Test files do not increase the consumer count
- **WHEN** a shared package is imported by one non-test file and one `_test.go` file while `mode` is `file`
- **THEN** the analyzer still treats the shared package as having only one consumer

### Requirement: Directory mode groups consumers by importing package path
In `dir` mode, the system SHALL count consumers by importing package path rather than by individual files.

#### Scenario: Two files in the same package count as one consumer in dir mode
- **WHEN** multiple files in the same importing package reference the same shared package while `mode` is `dir`
- **THEN** the analyzer counts that package path as a single consumer entity

#### Scenario: Different importing packages satisfy the shared-package threshold in dir mode
- **WHEN** two different importing packages reference the same shared package while `mode` is `dir`
- **THEN** the analyzer does not report a false-sharing violation for that shared package

### Requirement: Invalid false-sharing settings fail clearly
The system SHALL return an actionable configuration error when false-sharing plugin settings cannot be decoded into the expected configuration shape.

#### Scenario: shared-dirs has the wrong type
- **WHEN** plugin configuration provides `shared-dirs` as a non-string value
- **THEN** the plugin setup fails with an error that identifies the `nofalsesharing` settings problem
