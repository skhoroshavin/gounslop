## RENAMED Requirements

### Requirement: FROM: The deep-import rule is disabled without module-root
### TO: Architecture-control requires module-root

## MODIFIED Requirements

### Requirement: Deep same-scope imports are limited within a configured module root
When `module-root` is configured, the system SHALL evaluate imports whose paths stay under that module root. `boundarycontrol` SHALL subsume the existing same-scope deep-import behavior: if the importing package and imported package share the same first path segment beneath `module-root`, the imported package SHALL be at most one level deeper than the importing package, except that an immediate child import SHALL always remain allowed regardless of configured boundary rules.

#### Scenario: Import is too deep within the same top-level scope
- **WHEN** `example.com/mod/feature` imports `example.com/mod/feature/child/deep`
- **THEN** the system reports that the imported package is too deep for that importer within the same scope

#### Scenario: Immediate child import remains allowed without explicit rule
- **WHEN** `example.com/mod/feature` imports `example.com/mod/feature/child`
- **THEN** the system does not report a deep-import violation even if no boundary policy explicitly allows that edge

#### Scenario: Deeper import from a deeper package remains allowed
- **WHEN** `example.com/mod/feature/child` imports `example.com/mod/feature/child/deep`
- **THEN** the system does not report a deep-import violation

### Requirement: Out-of-scope imports are ignored
The system SHALL ignore imports that are outside the configured `module-root` for boundary matching and deep-import evaluation. Imports that stay inside the configured `module-root` SHALL remain subject to boundarycontrol evaluation even when the importer and imported package are in different top-level scopes beneath that root.

#### Scenario: Different top-level scope import remains subject to boundarycontrol
- **WHEN** `example.com/mod/featurea` imports `example.com/mod/featureb/other/deep` and no owning boundarycontrol policy allows that import
- **THEN** the system reports an undeclared boundarycontrol import violation

#### Scenario: External dependency import is outside boundary matching
- **WHEN** `example.com/mod/feature` imports `github.com/external/lib`
- **THEN** the system does not apply architecture-control boundary matching to that import

### Requirement: Architecture-control requires module-root
The system SHALL report a configuration error when architecture-control evaluation is requested without a configured `module-root`. Missing `module-root` SHALL fail loud instead of disabling deep-import or boundarycontrol checks.

#### Scenario: Module root is omitted
- **WHEN** the analyzer runs without a configured `module-root`
- **THEN** it reports a configuration error that `module-root` is required

## ADDED Requirements

### Requirement: Boundarycontrol uses selector-owned package policy
When `module-root` is configured, the system SHALL support a `boundarycontrol` rule whose policy is keyed by package selectors. Supported key selector forms SHALL be `.`, exact package paths such as `feature` or `feature/api`, and terminal child wildcard selectors such as `feature/*`. Exact keys SHALL own the named package and all of its descendants. Child wildcard keys SHALL own each direct child subtree beneath the parent path, including deeper descendants inside each owned child subtree, but SHALL not match the parent path itself.

#### Scenario: Exact key owns its subtree
- **WHEN** the importing package path is `feature/api/internal` and the configured key is `feature/api`
- **THEN** the key `feature/api` owns that package path unless a nearer owner overrides it

#### Scenario: Child wildcard key owns direct child subtree
- **WHEN** the importing package path is `feature/payments/internal` and the configured key is `feature/*`
- **THEN** the key `feature/*` owns that package path through child `payments`

#### Scenario: Child wildcard key does not own parent package
- **WHEN** the importing package path is `feature` and the only matching configured key is `feature/*`
- **THEN** the key `feature/*` does not match `feature`

### Requirement: Boundarycontrol resolves overlapping keys by nearest owner precedence
When multiple configured keys cover the same importing package path, the system SHALL choose the effective boundarycontrol policy by nearest owner first, then exact named path over wildcard path at the same depth, then longer selector path, then declaration order.

#### Scenario: Exact child key overrides wildcard key
- **WHEN** the importing package path is `feature/api/internal` and both `feature/*` and `feature/api` are configured
- **THEN** the system uses the policy from `feature/api`

#### Scenario: Child wildcard key overrides parent exact key for sibling subtree
- **WHEN** the importing package path is `feature/payments` and both `feature` and `feature/*` are configured
- **THEN** the system uses the policy from `feature/*`

#### Scenario: Declaration order breaks remaining tie
- **WHEN** two configured wildcard keys are otherwise equal in ownership precedence for the same importing package path
- **THEN** the system uses the policy from the key declared first

### Requirement: Boundarycontrol forbids cross-package imports unless explicitly allowed
For imports inside the configured `module-root`, boundarycontrol SHALL forbid an import unless the imported package matches one of the owning policy's configured `imports` selectors, or the import is an immediate child import that is always allowed by the deep-import rule integration. If no configured key owns the importing package path, the importing package SHALL be treated as having `imports: []`.

#### Scenario: Allowed import matches exact selector
- **WHEN** the owning policy for `feature/api` includes `imports: ["shared/contracts"]` and `feature/api` imports `shared/contracts`
- **THEN** the system allows that import

#### Scenario: Unmatched importer has empty import list
- **WHEN** `unknown/feature` matches no configured key and imports `shared/contracts`
- **THEN** the system reports an undeclared boundarycontrol import violation

#### Scenario: Immediate child import remains allowed for unmatched importer
- **WHEN** `feature` matches no configured key and imports `feature/api`
- **THEN** the system allows that import

### Requirement: Boundarycontrol import selectors use non-recursive package matching
Boundarycontrol `imports` selectors SHALL use non-recursive package matching. An exact selector such as `shared/contracts` SHALL match only that package. A child wildcard selector such as `shared/*` SHALL match only a direct child package. A self-or-child selector such as `shared/+` SHALL match the parent package and its direct child packages, but SHALL not match deeper descendants.

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
