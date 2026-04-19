## MODIFIED Requirements

### Requirement: Shared packages must have at least two consuming packages
When `boundarycontrol` configuration declares a selector policy with `shared: true`, the system SHALL treat every package owned by that selector as a shared package and SHALL evaluate each exported symbol declared by that shared package independently. The system SHALL report a diagnostic for an exported symbol when that symbol is used by fewer than two consuming entities. A consuming entity SHALL be either a direct non-test importing package path that references the exported symbol or another declaration inside the same shared package that references that exported symbol. A shared package with no exported symbols SHALL not produce a false-sharing diagnostic.

#### Scenario: Exported symbol has a single external consumer
- **WHEN** an exported symbol from a shared package is referenced by exactly one direct non-test importing package path and is not referenced by any other declaration inside the shared package
- **THEN** the analyzer reports a diagnostic at that exported symbol declaration that names the direct consuming package path so the symbol can be moved closer to that consumer

#### Scenario: Exported symbol has no consumers
- **WHEN** an exported symbol from a shared package is not referenced by any direct non-test importing package path and is not referenced by any other declaration inside the shared package
- **THEN** the analyzer reports a diagnostic at that exported symbol declaration that the symbol is not used by any entity and must be shared by two or more entities to remain in the shared package

#### Scenario: Shared package has no exported symbols
- **WHEN** a shared package declares no exported symbols
- **THEN** the analyzer reports no false-sharing diagnostics for that package

#### Scenario: Internal shared-package use satisfies the threshold
- **WHEN** an exported symbol from a shared package is referenced by exactly one direct non-test importing package path and by another declaration inside the same shared package
- **THEN** the analyzer does not report a false-sharing violation for that exported symbol

### Requirement: Consumers are counted by importing package path from non-test code
For migrated no-false-sharing evaluation, the system SHALL count consumers per exported symbol by consuming package path rather than by individual files or by package import alone. `_test.go` files SHALL not increase the consumer count for an exported symbol, and multiple references to the same exported symbol from the same importing package path SHALL count as a single external consumer entity.

#### Scenario: Two files in the same package count as one consumer
- **WHEN** multiple non-test files in the same importing package reference the same exported symbol from a shared package
- **THEN** the analyzer counts that importing package path as a single external consumer entity for that exported symbol

#### Scenario: Different importing packages using different exports do not satisfy symbol-level sharing
- **WHEN** two different importing package paths reference the same shared package but each package path references a different exported symbol from that shared package
- **THEN** the analyzer reports a false-sharing violation for each exported symbol that is still used by fewer than two consuming entities

#### Scenario: Test files do not increase the consumer count
- **WHEN** an exported symbol from a shared package is referenced by one non-test file and one `_test.go` file from the same importing package path
- **THEN** the analyzer still treats that importing package path as a single external consumer entity for that exported symbol
