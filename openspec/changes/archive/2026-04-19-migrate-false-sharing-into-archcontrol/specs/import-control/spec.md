## ADDED Requirements

### Requirement: Deep same-scope imports are limited within the discovered module scope
For each analyzed package, the system SHALL evaluate imports whose paths stay inside the discovered owning module. If the importing package and imported package share the same first path segment beneath the discovered module path, the imported package SHALL be at most one level deeper than the importing package, except that an immediate child import SHALL always remain allowed regardless of configured import policy.

#### Scenario: Import is too deep within the same top-level scope
- **WHEN** a package in module `example.com/mod` at `feature` imports `example.com/mod/feature/child/deep`
- **THEN** the system reports that the imported package is too deep for that importer within the same scope

#### Scenario: Immediate child import remains allowed without explicit rule
- **WHEN** a package in module `example.com/mod` at `feature` imports `example.com/mod/feature/child`
- **THEN** the system does not report a deep-import violation even if no selector-owned import policy explicitly allows that edge

#### Scenario: Deeper import from a deeper package remains allowed
- **WHEN** a package in module `example.com/mod` at `feature/child` imports `example.com/mod/feature/child/deep`
- **THEN** the system does not report a deep-import violation

### Requirement: Out-of-scope imports are ignored
The system SHALL ignore imports that are outside the discovered owning module for the importing package when applying import-control evaluation. External dependencies and packages owned by a different local module SHALL be outside import-control matching. Imports that stay inside the discovered owning module SHALL remain subject to import-control evaluation even when the importer and imported package are in different top-level scopes beneath that module path.

#### Scenario: Different top-level scope import remains subject to import-control
- **WHEN** a package in module `example.com/mod` at `featurea` imports `example.com/mod/featureb/other/deep` and no owning selector policy allows that import
- **THEN** the system reports an undeclared boundarycontrol import violation

#### Scenario: External dependency import is outside import-control matching
- **WHEN** a package in module `example.com/mod` at `feature` imports `github.com/external/lib`
- **THEN** the system does not apply import-control evaluation to that import

#### Scenario: Nested module import is outside import-control matching for the parent module
- **WHEN** a package in module `example.com/root` imports `example.com/root/tools/pkg` and `tools/go.mod` declares module `example.com/root/tools`
- **THEN** the system treats that import as outside the parent module's import-control evaluation

### Requirement: Import-control forbids cross-package imports unless explicitly allowed
For imports inside the discovered owning module, the system SHALL forbid an import unless the imported package matches one of the owning selector policy's configured `imports` selectors, or the import is an immediate child import that is always allowed by the deep-import rule integration. If no configured key owns the importing package path, the importing package SHALL be treated as having `imports: []`.

#### Scenario: Allowed import matches exact selector
- **WHEN** the owning policy for `feature/api` includes `imports: ["shared/contracts"]` and `feature/api` imports `shared/contracts`
- **THEN** the system allows that import

#### Scenario: Unmatched importer has empty import list
- **WHEN** `unknown/feature` matches no configured key and imports `shared/contracts`
- **THEN** the system reports an undeclared boundarycontrol import violation

#### Scenario: Immediate child import remains allowed for unmatched importer
- **WHEN** `feature` matches no configured key and imports `feature/api`
- **THEN** the system allows that import

### Requirement: Import selectors use non-recursive package matching
Import-control `imports` selectors SHALL use non-recursive package matching. An exact selector such as `shared/contracts` SHALL match only that package. A child wildcard selector such as `shared/*` SHALL match only a direct child package. A self-or-child selector such as `shared/+` SHALL match the parent package and its direct child packages, but SHALL not match deeper descendants.

#### Scenario: Exact import selector matches exact package only
- **WHEN** `imports` contains `shared/contracts` and the imported package is `shared/contracts`
- **THEN** the system allows that import

#### Scenario: Exact import selector does not match child package
- **WHEN** `imports` contains `shared/contracts` and the imported package is `shared/contracts/http`
- **THEN** the system reports an undeclared boundarycontrol import violation

#### Scenario: Child wildcard import selector matches direct child package only
- **WHEN** `imports` contains `shared/*` and the imported package is `shared/contracts`
- **THEN** the system allows that import

#### Scenario: Self-or-child import selector matches parent and direct child only
- **WHEN** `imports` contains `shared/+` and the imported package is `shared/contracts/http`
- **THEN** the system reports an undeclared boundarycontrol import violation
