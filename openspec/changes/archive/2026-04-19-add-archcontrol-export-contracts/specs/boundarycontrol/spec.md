## ADDED Requirements

### Requirement: Boundarycontrol selectors can declare export-name contracts
A selector policy in the `boundarycontrol` `architecture` mapping SHALL be allowed to declare an `exports` list of regex patterns. When a package path is owned by that selector, the package SHALL use those patterns as its export-contract policy. A selector policy without `exports` SHALL not enable export-contract evaluation for its owned packages.

#### Scenario: Exact selector applies export contract to its owned subtree
- **WHEN** the configured `architecture` mapping contains `pkg/api` with `exports: ["^New[A-Z].*$"]` and the analyzed package path is `pkg/api/http`
- **THEN** the system treats `pkg/api/http` as governed by the export-contract policy declared on `pkg/api`

#### Scenario: Selector without exports does not enable export control
- **WHEN** the configured `architecture` mapping contains `pkg/internal` without an `exports` field and the analyzed package path is `pkg/internal/cache`
- **THEN** the system does not enable export-contract evaluation for `pkg/internal/cache`
