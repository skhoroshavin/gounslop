## 1. Boundarycontrol Analyzer Setup

- [x] 1.1 Add `boundarycontrol` as a new analyzer package and register it in `plugin/module.go`
- [x] 1.2 Define analyzer settings for `boundarycontrol`, including required `module-root` and selector-based boundary policy configuration
- [x] 1.3 Add configuration validation for supported key selector forms (`.`, exact package path, terminal `/*`) and fail loudly when `module-root` is unset

## 2. Selector Ownership And Precedence

- [x] 2.1 Implement key-selector ownership matching for `.`, exact package paths, and terminal child wildcards
- [x] 2.2 Implement nearest-owner precedence for overlapping key matches: nearest owner, exact over wildcard at the same depth, longer selector path, then declaration order
- [x] 2.3 Treat unmatched in-module importers as having `imports: []`

## 3. Import Allowlist Evaluation

- [x] 3.1 Implement `imports` selector matching for exact package paths, `/*`, and `/+` with non-recursive semantics
- [x] 3.2 Enforce boundarycontrol violations for in-module imports that are not allowed by the effective owning policy
- [x] 3.3 Exclude standard-library and third-party imports outside `module-root` from boundary matching

## 4. Nodeepimports Integration

- [x] 4.1 Integrate the existing same-scope deep-import restriction into `boundarycontrol` without modifying `nodeepimports`
- [x] 4.2 Preserve the unconditional one-level-deep parent-to-child import allowance even when no policy explicitly allows the edge
- [x] 4.3 Ensure different top-level in-module imports remain subject to boundarycontrol rather than being ignored as out of scope

## 5. Tests And Documentation

- [x] 5.1 Add analyzer tests covering selector ownership, wildcard behavior, precedence, unmatched importers, and missing `module-root` configuration errors
- [x] 5.2 Add tests covering import selector matching for exact, `/*`, and `/+`, including allowed and rejected cases
- [x] 5.3 Add tests covering integrated deep-import behavior, unconditional immediate-child allowance, external imports outside `module-root`, and different top-level in-module violations
- [x] 5.4 Update repository documentation and examples to describe `boundarycontrol` configuration and its relationship to the still-existing `nodeepimports` rule
- [x] 5.5 Run `make lint && make test` and fix any issues until the full suite passes
