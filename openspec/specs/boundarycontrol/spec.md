## Purpose

Define boundarycontrol ownership, precedence, module-scope discovery, and shared subtree declaration behavior.

## Requirements

### Requirement: Boundarycontrol discovers module scope from go.mod
The system SHALL derive module scope for each analyzed package from the nearest enclosing `go.mod`. It SHALL parse that file's `module` directive to determine the owning module path and SHALL use that discovered module path for selector ownership and shared-package matching.

#### Scenario: Nearest go.mod defines the owning module path
- **WHEN** a file under `tools/internal/checker` is analyzed and the nearest enclosing `go.mod` declares `module example.com/root/tools`
- **THEN** the system uses `example.com/root/tools` as the module scope for that package instead of any parent module path

#### Scenario: Missing go.mod fails clearly
- **WHEN** an analyzed package has no enclosing `go.mod`
- **THEN** the system reports a clear error that module scope could not be discovered from `go.mod`

### Requirement: Boundarycontrol uses selector-owned package policy
The system SHALL support a `boundarycontrol` rule whose policy is defined under an `architecture` mapping keyed by package selectors. Each mapping entry SHALL be keyed by a selector. Supported key selector forms SHALL be `.`, exact package paths such as `feature` or `feature/api`, and terminal child wildcard selectors such as `feature/*`. Exact keys SHALL own the named package and all of its descendants. Child wildcard keys SHALL own each direct child subtree beneath the parent path, including deeper descendants inside each owned child subtree, but SHALL not match the parent path itself.

#### Scenario: Exact key owns its subtree
- **WHEN** the package path is `feature/api/internal` and the configured `architecture` mapping contains key `feature/api`
- **THEN** the key `feature/api` owns that package path unless a nearer owner overrides it

#### Scenario: Child wildcard key owns direct child subtree
- **WHEN** the package path is `feature/payments/internal` and the configured `architecture` mapping contains key `feature/*`
- **THEN** the key `feature/*` owns that package path through child `payments`

#### Scenario: Child wildcard key does not own parent package
- **WHEN** the package path is `feature` and the only matching configured key in `architecture` is `feature/*`
- **THEN** the key `feature/*` does not match `feature`

### Requirement: Boundarycontrol resolves overlapping keys by nearest owner precedence
When multiple configured keys cover the same package path, the system SHALL choose the effective boundarycontrol policy by nearest owner first, then exact named path over wildcard path at the same depth, then longer selector path.

#### Scenario: Exact child key overrides wildcard key
- **WHEN** the package path is `feature/api/internal` and the configured `architecture` mapping contains both `feature/*` and `feature/api`
- **THEN** the system uses the policy from `feature/api`

#### Scenario: Child wildcard key overrides parent exact key for sibling subtree
- **WHEN** the package path is `feature/payments` and the configured `architecture` mapping contains both `feature` and `feature/*`
- **THEN** the system uses the policy from `feature/*`

### Requirement: Boundarycontrol selectors can declare shared package subtrees
A selector policy with `shared: true` SHALL mark every package owned by that selector as a shared package for migrated false-sharing evaluation. A selector policy without `shared: true` SHALL not mark its owned packages as shared.

#### Scenario: Exact shared selector marks its subtree as shared
- **WHEN** the configured `architecture` mapping contains `shared/lib` with `shared: true` and the package path is `shared/lib/http`
- **THEN** the system treats `shared/lib/http` as a shared package owned by `shared/lib`

#### Scenario: Child wildcard shared selector marks direct child subtrees as shared
- **WHEN** the configured `architecture` mapping contains `shared/*` with `shared: true` and the package path is `shared/contracts/http`
- **THEN** the system treats `shared/contracts/http` as part of the shared subtree for the matched direct child selector owner

#### Scenario: Selector without shared flag is not a shared package declaration
- **WHEN** the configured `architecture` mapping contains `feature/api` without `shared: true`
- **THEN** the system does not treat packages owned by `feature/api` as shared packages for false-sharing evaluation
