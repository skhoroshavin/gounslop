## Why

The `nofalsesharing` analyzer currently counts only direct references to exported symbols in shared packages. When a type is used as a field in an exported struct or as a parameter/return type in an exported function, consumers of that struct or function never directly name the type—but they still depend on it. This causes the analyzer to falsely report such types as under-utilized and forbids their export. The workaround forces awkward API design, such as keeping the type unexported while exporting its constants (e.g., `selectorKind` in `pkg/analyzer`), which prevents external packages from declaring variables of that type.

## What Changes

- Update `nofalsesharing` to detect **indirect** usage of shared-package types through exported public APIs (exported struct fields, function parameters, and return types).
- When a type from a shared package appears in the public API of an exported symbol in another package, count consumers of that exported symbol as additional consumers of the type.
- Add E2E test coverage for indirect type usage scenarios.

## Capabilities

### New Capabilities
- *(none)*

### Modified Capabilities
- `no-false-sharing`: Requirement changes to account for indirect type usage via exported struct fields and function signatures. Types referenced only through public API of consuming packages must not be flagged as under-utilized.

## Impact

- `pkg/nofalsesharing/analyzer.go`: core counting logic extended to traverse exported symbol signatures and attribute their consumers back to shared-package types.
- `tests/nofalsesharing_test.go`: new E2E test cases for indirect usage.
- Potential follow-up: `selectorKind` in `pkg/analyzer` can be exported once the fix is in place.
