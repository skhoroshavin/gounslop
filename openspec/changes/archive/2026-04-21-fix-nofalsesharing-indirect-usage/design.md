## Context

`nofalsesharing` evaluates exported symbols in shared packages by counting direct references found via `go/types` `Uses` and `Selections`. Each reference from a distinct non-test package path counts as one consumer. A symbol must reach at least two consumers (external packages or internal declarations) to avoid a diagnostic.

This model breaks for types that are only exposed through the *signatures* of exported symbols in other packages. For example, if package `feature/api` exports a struct with a field of type `shared.Widget`, consumers of that struct never directly name `Widget` in their code; they only reference the struct. `Widget` is then flagged as having a single consumer (`feature/api`), forcing the awkward workaround of keeping the type unexported while exporting its constants (`selectorKind` in `pkg/analyzer`).

The existing spec for `no-false-sharing` defines a consumer as a "direct non-test importing package path that references the exported symbol." We need to extend this definition so that a reference to a carrier symbol (an exported symbol whose public API contains a shared-package type) also counts as a reference to the shared type itself.

## Goals / Non-Goals

**Goals:**
- Shared-package types used in the public API of exported symbols in other packages must inherit the consumer count of those exported symbols.
- Existing direct-reference counting must remain unchanged; indirect counting is additive.
- No changes to the configuration schema, analyzer registration, or diagnostic message format.
- Add E2E test coverage for indirect type usage through struct fields and function signatures.

**Non-Goals:**
- Tracking usage through unexported symbols or implementation details (e.g., unexported struct fields, unexported method bodies).
- Tracking usage through reflection, generic constraints, or interface satisfaction.
- Changing the threshold (two consumers) or the definition of a shared package.
- Re-exporting `selectorKind` as part of this change (follow-up only).

## Decisions

### Decision 1: Two-pass consumer counting
**Rationale:** Carrier symbols must be fully identified before their consumers can be propagated to shared types. A single pass would require forward knowledge of all carriers.

**Approach:**
1. **Pass 1 — Identify carriers:** Walk all packages and declarations. For each exported symbol owned by a non-shared package, inspect its public API. If any named type from a shared package appears in the API, record the symbol as a carrier mapping to the shared type keys.
2. **Pass 2 — Propagate consumers:** Walk all packages again (or extend the existing walk). For each direct reference to a carrier symbol, add the referencing package as a consumer of every shared type the carrier carries.

This keeps the direct-counting logic intact and isolated.

### Decision 2: Carrier scope — exported public API only
**Rationale:** External packages only depend on shared types they can actually observe and use. Unexported fields or methods are invisible outside the owning package.

**Carriers are:**
- Exported variables and constants (their declared type).
- Exported struct types (exported fields and embedded fields only).
- Exported functions and exported methods (parameter and result types).
- Exported interface types (method signatures: parameters and results).

**Not carriers:**
- Unexported struct fields, unexported methods, local variables, or type aliases that are unexported.

### Decision 3: Recursive type traversal for shared-type detection
**Rationale:** A shared type may appear nested inside pointers, slices, maps, channels, function types, or structs.

**Approach:** A recursive helper `collectSharedTypeKeys(t types.Type, sharedPackages) []string` walks the type tree:
- `*types.Named`: if from a shared package, record its key; then walk `Underlying()`.
- `*types.Pointer`, `*types.Slice`, `*types.Array`, `*types.Chan`: walk the element type.
- `*types.Map`: walk key and element types.
- `*types.Signature`: walk parameters and results.
- `*types.Struct`: walk exported fields only.
- `*types.Interface`: walk method signatures.
- `*types.TypeParam`: walk the constraint.
- Basic types and predeclared types are terminal.

This ensures completeness without over-complicating the logic.

### Decision 4: Carrier consumer propagation uses the same package-path consumer model
**Rationale:** Consistency with existing counting rules.

When a package references a carrier symbol, the carrier's package path is *not* added as a consumer. Instead, the *referencing* package path is added to each shared type the carrier carries. The carrier's owning package is already counted as a direct consumer if it directly references the shared type; if not, it does not need to be counted separately.

Example:
- `shared` exports `type Widget struct{}`.
- `feature/api` exports `type Response struct { Data shared.Widget }`.
- `feature/web` references `feature/api.Response`.
- Pass 1: `feature/api.Response` is a carrier for `shared.Widget`.
- Pass 2: `feature/web` references `feature/api.Response`, so `feature/web` is added as a consumer of `shared.Widget`.
- `shared.Widget` now has consumers: `feature/api` (direct, via field type) and `feature/web` (indirect, via carrier).

### Decision 5: No changes to cache key or shared-package map shape
**Rationale:** The carrier map is local to `analyzeFalseSharing` and does not need to be cached. Only the final diagnostics are cached.

The `sharedPackageEntry` and `sharedSymbolEntry` structs stay the same. The carrier map is built in-memory and discarded after propagation.

## Risks / Trade-offs

- **[Risk]** Recursive type traversal over large modules with deeply nested generics could increase analysis time.  
  **→ Mitigation:** Only exported symbols are inspected; the traversal short-circuits on basic types and avoids walking into unexported struct fields.

- **[Risk]** A carrier symbol in a shared package could cause a shared type to be double-counted (direct + indirect).  
  **→ Mitigation:** Consumer sets are `map[string]struct{}`; duplicate package paths are naturally deduplicated. The direct consumer and the indirect consumer are the same package, so the set size remains correct.

- **[Risk]** Over-counting when a carrier symbol is never actually used by external packages (e.g., an exported function that is dead code but references a shared type).  
  **→ Mitigation:** The carrier symbol must itself be referenced by at least one external package for its consumers to be counted. If the carrier is unused, it contributes no indirect consumers, which is the correct behavior.

- **[Trade-off]** We do not traverse generic constraints for `TypeParam` beyond the immediate constraint interface. Complex generic bounds that indirectly reference shared types through multiple interface layers may be missed. This is acceptable because such patterns are rare and the fallback (direct reference counting) still applies.

## Migration Plan

No migration is required. This is a behavioral fix to an existing analyzer. Users who previously suppressed `nofalsesharing` diagnostics for indirectly-used types may see those diagnostics disappear.

## Open Questions

1. Should type aliases (`type T = shared.Widget`) in non-shared packages count as carriers? The alias itself is a direct reference to `Widget`, so it would already be counted in Pass 1 if the alias is exported. This should work naturally with the existing `info.Uses` walk.
2. Should we propagate consumers transitively? If `pkg/a` exports a carrier for `shared.T`, and `pkg/b` exports a carrier that uses `pkg/a`'s carrier, should consumers of `pkg/b` also count for `shared.T`? For now, no — only one level of indirection is supported. Transitive chains are rare and can be addressed if needed.
