## MODIFIED Requirements

### Requirement: Shared packages must have at least two consuming packages
When `boundarycontrol` configuration declares a selector policy with `shared: true`, the system SHALL treat every package owned by that selector as a shared package and SHALL evaluate each exported symbol declared by that shared package independently. The system SHALL report a diagnostic for an exported symbol when that symbol is used by fewer than two consuming entities. A consuming entity SHALL be either a direct non-test importing package path that references the exported symbol, another declaration inside the same shared package that references that exported symbol, or a non-test importing package path that references an exported symbol in another package whose public API includes that shared type. A shared package with no exported symbols SHALL not produce a false-sharing diagnostic.

#### Scenario: Exported symbol has a single external consumer
- **WHEN** an exported symbol from a shared package is referenced by exactly one direct non-test importing package path and is not referenced by any other declaration inside the shared package
- **THEN** the analyzer reports a diagnostic at that exported symbol declaration that names the direct consuming package path so the symbol can be moved closer to that consumer

#### Scenario: Exported symbol has no consumers
- **WHEN** an exported symbol from a shared package is not referenced by any direct non-test importing package path and is not referenced by any other declaration inside the same shared package
- **THEN** the analyzer reports a diagnostic at that exported symbol declaration that the symbol is not used by any entity and must be shared by two or more entities to remain in the shared package

#### Scenario: Shared package has no exported symbols
- **WHEN** a shared package declares no exported symbols
- **THEN** the analyzer reports no false-sharing diagnostics for that package

#### Scenario: Internal shared-package use satisfies the threshold
- **WHEN** an exported symbol from a shared package is referenced by exactly one direct non-test importing package path and by another declaration inside the same shared package
- **THEN** the analyzer does not report a false-sharing violation for that exported symbol

#### Scenario: Shared type used indirectly through exported struct field
- **WHEN** a shared package exports a type that is used as an exported field in an exported struct declared by another package, and that struct is referenced by exactly one non-test importing package path
- **THEN** the analyzer counts the importing package path of the struct's consumer as an additional consumer of the shared type

#### Scenario: Shared type used indirectly through exported function signature
- **WHEN** a shared package exports a type that appears in the parameter list or return type of an exported function declared by another package, and that function is referenced by exactly one non-test importing package path
- **THEN** the analyzer counts the importing package path of the function's consumer as an additional consumer of the shared type

#### Scenario: Shared type used indirectly through exported interface method
- **WHEN** a shared package exports a type that appears in the signature of an exported method declared in an exported interface by another package, and that interface is referenced by exactly one non-test importing package path
- **THEN** the analyzer counts the importing package path of the interface's consumer as an additional consumer of the shared type

#### Scenario: Indirect usage reaches two consumers
- **WHEN** a shared package exports a type that is used in the public API of an exported symbol in another package, and that exported symbol is consumed by one non-test importing package path while the shared type itself is consumed by one other non-test importing package path
- **THEN** the analyzer does not report a false-sharing violation for that shared type
