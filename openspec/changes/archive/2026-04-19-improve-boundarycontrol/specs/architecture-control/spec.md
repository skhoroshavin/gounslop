## MODIFIED Requirements

### Requirement: Deep same-scope imports are limited within a configured module root
For each analyzed package, the system SHALL discover the owning module from the nearest enclosing `go.mod` and evaluate imports whose paths stay inside that module. `boundarycontrol` SHALL subsume the same-scope deep-import behavior: if the importing package and imported package share the same first path segment beneath the discovered module path, the imported package SHALL be at most one level deeper than the importing package, except that an immediate child import SHALL always remain allowed regardless of configured boundary rules.

#### Scenario: Import is too deep within the same top-level scope
- **WHEN** a package in module `example.com/mod` at `feature` imports `example.com/mod/feature/child/deep`
- **THEN** the system reports that the imported package is too deep for that importer within the same scope

#### Scenario: Immediate child import remains allowed without explicit rule
- **WHEN** a package in module `example.com/mod` at `feature` imports `example.com/mod/feature/child`
- **THEN** the system does not report a deep-import violation even if no boundary policy explicitly allows that edge

#### Scenario: Deeper import from a deeper package remains allowed
- **WHEN** a package in module `example.com/mod` at `feature/child` imports `example.com/mod/feature/child/deep`
- **THEN** the system does not report a deep-import violation

### Requirement: Out-of-scope imports are ignored
The system SHALL ignore imports that are outside the discovered owning module for the importing package when applying boundary matching and deep-import evaluation. External dependencies and packages owned by a different local module SHALL be outside boundary matching. Imports that stay inside the discovered owning module SHALL remain subject to boundarycontrol evaluation even when the importer and imported package are in different top-level scopes beneath that module path.

#### Scenario: Different top-level scope import remains subject to boundarycontrol
- **WHEN** a package in module `example.com/mod` at `featurea` imports `example.com/mod/featureb/other/deep` and no owning boundarycontrol policy allows that import
- **THEN** the system reports an undeclared boundarycontrol import violation

#### Scenario: External dependency import is outside boundary matching
- **WHEN** a package in module `example.com/mod` at `feature` imports `github.com/external/lib`
- **THEN** the system does not apply architecture-control boundary matching to that import

#### Scenario: Nested module import is outside boundary matching for the parent module
- **WHEN** a package in module `example.com/root` imports `example.com/root/tools/pkg` and `tools/go.mod` declares module `example.com/root/tools`
- **THEN** the system treats that import as outside the parent module's architecture-control boundary matching

### Requirement: Boundarycontrol uses selector-owned package policy
The system SHALL support a `boundarycontrol` rule whose policy is defined under an `architecture` mapping keyed by package selectors. Each mapping entry SHALL be keyed by a selector and contain an `imports` list. Supported key selector forms SHALL be `.`, exact package paths such as `feature` or `feature/api`, and terminal child wildcard selectors such as `feature/*`. Exact keys SHALL own the named package and all of its descendants. Child wildcard keys SHALL own each direct child subtree beneath the parent path, including deeper descendants inside each owned child subtree, but SHALL not match the parent path itself.

#### Scenario: Exact key owns its subtree
- **WHEN** the importing package path is `feature/api/internal` and the configured `architecture` mapping contains key `feature/api`
- **THEN** the key `feature/api` owns that package path unless a nearer owner overrides it

#### Scenario: Child wildcard key owns direct child subtree
- **WHEN** the importing package path is `feature/payments/internal` and the configured `architecture` mapping contains key `feature/*`
- **THEN** the key `feature/*` owns that package path through child `payments`

#### Scenario: Child wildcard key does not own parent package
- **WHEN** the importing package path is `feature` and the only matching configured key in `architecture` is `feature/*`
- **THEN** the key `feature/*` does not match `feature`

### Requirement: Boundarycontrol resolves overlapping keys by nearest owner precedence
When multiple configured keys cover the same importing package path, the system SHALL choose the effective boundarycontrol policy by nearest owner first, then exact named path over wildcard path at the same depth, then longer selector path.

#### Scenario: Exact child key overrides wildcard key
- **WHEN** the importing package path is `feature/api/internal` and the configured `architecture` mapping contains both `feature/*` and `feature/api`
- **THEN** the system uses the policy from `feature/api`

#### Scenario: Child wildcard key overrides parent exact key for sibling subtree
- **WHEN** the importing package path is `feature/payments` and the configured `architecture` mapping contains both `feature` and `feature/*`
- **THEN** the system uses the policy from `feature/*`

### Requirement: Boundarycontrol forbids cross-package imports unless explicitly allowed
For imports inside the discovered owning module, boundarycontrol SHALL forbid an import unless the imported package matches one of the owning policy's configured `imports` selectors, or the import is an immediate child import that is always allowed by the deep-import rule integration. If no configured key owns the importing package path, the importing package SHALL be treated as having `imports: []`.

#### Scenario: Allowed import matches exact selector
- **WHEN** the owning policy for `feature/api` includes `imports: ["shared/contracts"]` and `feature/api` imports `shared/contracts`
- **THEN** the system allows that import

#### Scenario: Unmatched importer has empty import list
- **WHEN** `unknown/feature` matches no configured key and imports `shared/contracts`
- **THEN** the system reports an undeclared boundarycontrol import violation

#### Scenario: Immediate child import remains allowed for unmatched importer
- **WHEN** `feature` matches no configured key and imports `feature/api`
- **THEN** the system allows that import

## ADDED Requirements

### Requirement: Architecture-control discovers module scope from go.mod
The system SHALL derive module scope for each analyzed package from the nearest enclosing `go.mod`. It SHALL parse that file's `module` directive to determine the owning module path and SHALL use that discovered module path for boundarycontrol matching and deep-import evaluation.

#### Scenario: Nearest go.mod defines the owning module path
- **WHEN** a file under `tools/internal/checker` is analyzed and the nearest enclosing `go.mod` declares `module example.com/root/tools`
- **THEN** the system uses `example.com/root/tools` as the module scope for that package instead of any parent module path

#### Scenario: Missing go.mod fails clearly
- **WHEN** an analyzed package has no enclosing `go.mod`
- **THEN** the system reports a clear error that module scope could not be discovered from `go.mod`

## REMOVED Requirements

### Requirement: Architecture-control requires module-root
**Reason**: Module scope is now derived automatically from the nearest enclosing `go.mod` instead of being configured manually.
**Migration**: Remove `module-root` from `boundarycontrol` configuration and rely on `go.mod` discovery for module scoping.
