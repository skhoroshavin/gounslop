## Purpose

Define the baseline architecture-control rules for limiting deep same-scope imports within a configured module root.

## Requirements

### Requirement: Deep same-scope imports are limited within a configured module root
When `module-root` is configured, the system SHALL evaluate imports whose paths stay under that module root. If the importing package and imported package share the same first path segment beneath `module-root`, the imported package SHALL be at most one level deeper than the importing package.

#### Scenario: Import is too deep within the same top-level scope
- **WHEN** `example.com/mod/feature` imports `example.com/mod/feature/child/deep`
- **THEN** the analyzer reports that the imported package is too deep for that importer within the same scope

#### Scenario: Immediate child import remains allowed
- **WHEN** `example.com/mod/feature` imports `example.com/mod/feature/child`
- **THEN** the analyzer does not report a deep-import violation

#### Scenario: Deeper import from a deeper package remains allowed
- **WHEN** `example.com/mod/feature/child` imports `example.com/mod/feature/child/deep`
- **THEN** the analyzer does not report a deep-import violation

### Requirement: Out-of-scope imports are ignored
The system SHALL ignore imports that are outside the configured `module-root` or that target a different top-level scope beneath that root.

#### Scenario: Different top-level scope import is allowed
- **WHEN** `example.com/mod/featurea` imports `example.com/mod/featureb/other/deep`
- **THEN** the analyzer does not report a deep-import violation for that import

### Requirement: The deep-import rule is disabled without module-root
The system SHALL not report deep-import violations when `module-root` is unset.

#### Scenario: Module root is omitted
- **WHEN** the analyzer runs without a configured `module-root`
- **THEN** it reports no deep-import diagnostics
