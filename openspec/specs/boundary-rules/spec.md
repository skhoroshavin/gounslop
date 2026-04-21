# Purpose

TBD — Boundary rules parsing, matching, and compilation for module boundary control.

## Requirements

### Requirement: Boundary rules parse selector DSL
The system SHALL parse a selector string into a typed `Selector` value. Supported selector forms SHALL be:
- `.` — matches the root package (empty relative path)
- `pkg` — exact match for `pkg` and all descendants (policy-key semantics)
- `pkg/*` — matches direct children of `pkg` only
- `pkg/+` — matches `pkg` itself and its direct children (import selectors only)

#### Scenario: Parse root selector
- **WHEN** the raw selector is `.`
- **THEN** the parsed selector has kind `KindRoot`

#### Scenario: Parse exact selector
- **WHEN** the raw selector is `feature/api`
- **THEN** the parsed selector has kind `KindExact`, base `feature/api`, and depth `2`

#### Scenario: Parse child wildcard selector
- **WHEN** the raw selector is `feature/*`
- **THEN** the parsed selector has kind `KindChildren`, base `feature`, and depth `1`

#### Scenario: Parse self-or-child selector
- **WHEN** the raw selector is `feature/+` and self-or-child is allowed
- **THEN** the parsed selector has kind `KindSelfOrChildren`, base `feature`, and depth `1`

#### Scenario: Reject invalid selector
- **WHEN** the raw selector is `feature/+` and self-or-child is not allowed
- **THEN** parsing returns an error

### Requirement: Boundary rules match selectors with appropriate semantics
A `Selector` SHALL support two matching operations:
- `Covers(relPath)` — for policy-key semantics: exact selectors cover self and all descendants; child wildcards cover direct children and their descendants; self-or-child covers self and direct children; root covers only empty path.
- `MatchesImport(importRel)` — for import-rule semantics: exact selectors match only the exact path; child wildcards match only direct children; self-or-child matches self and direct children; root matches only empty path.

#### Scenario: Exact selector covers subtree
- **WHEN** selector is `feature/api` (kind `KindExact`) and the path is `feature/api/internal`
- **THEN** `Covers` returns true

#### Scenario: Exact selector does not match import child
- **WHEN** selector is `feature/api` (kind `KindExact`) and the import path is `feature/api/internal`
- **THEN** `MatchesImport` returns false

#### Scenario: Child wildcard covers direct child subtree
- **WHEN** selector is `feature/*` (kind `KindChildren`) and the path is `feature/payments/internal`
- **THEN** `Covers` returns true

#### Scenario: Child wildcard does not cover parent
- **WHEN** selector is `feature/*` (kind `KindChildren`) and the path is `feature`
- **THEN** `Covers` returns false

### Requirement: Boundary rules find the best matching policy
Given a list of `Policy` values and a relative package path, the system SHALL find the most specific policy whose selector covers that path. Specificity ordering SHALL be: deeper match wins; at equal depth, non-wildcard wins; then longer selector wins.

#### Scenario: Exact child overrides wildcard
- **WHEN** policies include `feature/*` and `feature/api`, and the path is `feature/api/internal`
- **THEN** the best policy is `feature/api`

#### Scenario: Wildcard overrides parent exact for sibling subtree
- **WHEN** policies include `feature` and `feature/*`, and the path is `feature/payments`
- **THEN** the best policy is `feature/*`

### Requirement: Boundary rules compile into a rule set
The system SHALL compile an `Architecture` configuration into a `Rules` value containing a list of `Policy` values. Each policy SHALL contain a parsed selector, import selectors, compiled export regexes, and a shared flag. The rule set SHALL also track whether any policy has `shared: true` for cache optimization.

#### Scenario: Compile architecture with imports and exports
- **WHEN** the architecture contains `feature/api` with imports `["shared/contracts"]` and exports `["^New.*$"]`
- **THEN** the compiled rules contain a policy for `feature/api` with parsed import selectors and compiled regex patterns

#### Scenario: Compile architecture with shared flag
- **WHEN** the architecture contains `shared/lib` with `shared: true`
- **THEN** the compiled rules have `HasSharedSelectors` set to true
