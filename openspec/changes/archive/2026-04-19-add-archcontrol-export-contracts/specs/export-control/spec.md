## ADDED Requirements

### Requirement: Export-control enforces regex contracts on exported top-level declarations
When the owning `boundarycontrol` selector policy for a package declares one or more `exports` regex patterns, the system SHALL evaluate each exported top-level declaration in that package against those patterns. The system SHALL allow the declaration when its exported name matches at least one configured pattern and SHALL report a diagnostic at the declaration when its exported name matches none of them.

#### Scenario: Exported top-level declaration matches contract
- **WHEN** the owning selector policy declares `exports: ["^New[A-Z].*$"]` and package `pkg/api` declares exported function `NewClient`
- **THEN** the system reports no export-contract diagnostic for `NewClient`

#### Scenario: Exported top-level declaration violates contract
- **WHEN** the owning selector policy declares `exports: ["^New[A-Z].*$"]` and package `pkg/api` declares exported function `BuildClient`
- **THEN** the system reports an export-contract diagnostic at `BuildClient`

### Requirement: Export-control evaluates only exported package-scope declarations
The system SHALL evaluate only exported top-level package-scope declarations for export-control. Unexported declarations SHALL be ignored, and exported methods or other non-package-scope members SHALL not be evaluated by this version of export-control.

#### Scenario: Unexported declaration is ignored
- **WHEN** the owning selector policy declares `exports: ["^New[A-Z].*$"]` and package `pkg/api` declares unexported function `buildClient`
- **THEN** the system reports no export-contract diagnostic for `buildClient`

#### Scenario: Exported method is not evaluated
- **WHEN** the owning selector policy declares `exports: ["^New[A-Z].*$"]` and package `pkg/api` declares exported method `Client.Build`
- **THEN** the system does not evaluate `Build` for export-contract compliance

### Requirement: Export-control uses full-name regex matching
Export-control SHALL match each configured `exports` pattern against the full exported declaration name. A pattern SHALL not pass a declaration merely because it matches a substring of that name.

#### Scenario: Substring-only match does not satisfy contract
- **WHEN** the owning selector policy declares `exports: ["Error"]` and package `pkg/api` declares exported type `ClientError`
- **THEN** the system reports an export-contract diagnostic at `ClientError`

#### Scenario: Exact full-name match satisfies contract
- **WHEN** the owning selector policy declares `exports: ["ClientError"]` and package `pkg/api` declares exported type `ClientError`
- **THEN** the system reports no export-contract diagnostic for `ClientError`

### Requirement: Invalid export-control settings fail clearly
The system SHALL return an actionable configuration error when `boundarycontrol` export-contract settings cannot be decoded into the expected shape or when an `exports` regex pattern is invalid.

#### Scenario: Exports field has the wrong type
- **WHEN** plugin configuration provides `exports` as a non-list value under an `architecture` selector policy
- **THEN** plugin setup fails with an error that identifies the `boundarycontrol` export-contract settings problem

#### Scenario: Export regex is invalid
- **WHEN** plugin configuration provides an invalid regex pattern in `exports` under an `architecture` selector policy
- **THEN** plugin setup fails with an error that identifies the invalid `exports` regex pattern
