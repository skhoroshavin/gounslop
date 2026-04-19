## Context

`boundarycontrol` currently accepts a `module-root` string plus an ordered `selectors` list. The analyzer then treats any import whose path starts with that module prefix as in-scope. That works for a single module with explicit configuration, but it has three gaps for the next version:

- the config is more verbose than the intended architecture-policy surface
- the rule cannot infer module scope from the package being analyzed
- prefix-only matching misclassifies nested modules as part of the parent module

The repository also still ships `nodeepimports` as a separate analyzer even though `boundarycontrol` already contains the same-scope deep-import check. This leaves the import-architecture story split across two rules and two test surfaces.

## Goals / Non-Goals

**Goals:**

- Replace the public `boundarycontrol` settings shape with an `architecture` map keyed by package selector.
- Resolve module scope automatically from the nearest relevant `go.mod` for each analyzed package.
- Correctly handle repositories with multiple Go modules, including nested modules that share an import-path prefix.
- Keep the existing `boundarycontrol` selector semantics and deep-import behavior, but make `boundarycontrol` the only analyzer that enforces them.
- Remove `nodeepimports` from plugin registration, docs, self-lint config, and E2E coverage.

**Non-Goals:**

- Preserving backward compatibility for the old `selectors` config shape.
- Adding new selector syntax beyond the current `boundarycontrol` model.
- Enforcing architecture rules across module boundaries.
- Reworking unrelated analyzers or the broader plugin architecture.

## Decisions

### 1. Make `architecture` the only public config shape

`boundarycontrol` should accept this public structure:

```yaml
settings:
  architecture:
    "plugin":
      imports: ["pkg/*"]
    "pkg/*":
      imports: ["internal/*"]
```

The plugin layer should decode that map into an internal normalized policy list or compiled rule set used by the analyzer. The old `module-root` and `selectors` fields should be removed from the documented and supported `boundarycontrol` configuration.

Rationale:

- This matches the requested user-facing format.
- It removes configuration duplication between the selector key and its policy object.
- The repository is small and can absorb a clean breaking change without carrying dual parsing paths.

Alternatives considered:

- Support both `selectors` and `architecture` temporarily: rejected because it adds migration code, more tests, and a longer-lived ambiguous API.
- Keep `module-root` as an explicit required setting: rejected because the goal is to infer scope from `go.mod` automatically.

### 2. Resolve ownership from selector specificity, not declaration order

The current implementation uses declaration order as the final tie-break because `selectors` is an ordered list. The new `architecture` form is keyed by selector, so duplicate selector entries are impossible and source order is no longer a reliable or necessary part of the contract.

Owner resolution should therefore be defined entirely by selector specificity:

- nearest owner first
- exact selector over wildcard selector at the same depth
- longer selector path when depth comparison still matters

Rationale:

- A keyed configuration naturally removes same-key duplicates.
- This gives deterministic behavior without depending on YAML map ordering through the plugin decode path.

Alternatives considered:

- Preserve declaration order by trying to recover map insertion order from decoded plugin settings: rejected because it is brittle and unnecessary once selectors are unique keys.

### 3. Discover module context per analyzed package from the filesystem

`boundarycontrol` should stop taking `module-root` from analyzer flags. Instead, for each analysis pass it should:

1. take the first file in `pass.Files`
2. walk upward to the nearest `go.mod`
3. parse the `module` directive from that file
4. cache the resulting module context by module directory

The module context should contain at least:

- module directory on disk
- module import path from `go.mod`
- nested module import paths discovered beneath that directory

This is the same general discovery model already used in `nofalsesharing`, but `boundarycontrol` needs a richer cached context because it must classify each imported package relative to the current module.

`go.mod` discovery should be the only supported way to establish module scope for `boundarycontrol`. The rule should not expose a documented or undocumented `module-root` override.

Rationale:

- It removes the need for manual `module-root` plumbing in config and tests.
- It makes behavior match the actual package being analyzed instead of assuming one global module prefix.

Alternatives considered:

- Infer the module root from `pass.Pkg.Path()` alone: rejected because package path alone cannot distinguish nested modules from packages inside the parent module.
- Use one repository-level `go.mod`: rejected because the change explicitly needs to support multiple modules in one project.

### 4. Treat nested modules as out-of-scope for the current module

Simple prefix matching is not enough in a multi-module repository. If the current module is `example.com/root`, an import like `example.com/root/tools/pkg` must be treated as outside the parent module when `tools/go.mod` declares `module example.com/root/tools`.

To support that, in-module matching should work like this:

- an import is a candidate in-module import only if it matches the current module path or its path prefix
- before accepting it as in-module, the analyzer checks whether a more specific nested module path also owns that import path
- if a nested module path matches, the import is treated as cross-module and skipped by `boundarycontrol`

Nested module paths should be discovered by scanning descendant directories under the current module root for additional `go.mod` files and parsing their module directives once per cached module context.

Rationale:

- This is the minimum logic needed to make module auto-discovery correct in monorepos and nested-module layouts.
- It preserves the rule's existing intent: enforce architecture only within one module at a time.

Alternatives considered:

- Continue using plain prefix matching: rejected because it will produce false positives against nested modules.
- Try to enforce boundaries across all local modules at once: rejected because the rule is module-scoped, not workspace-scoped.

### 5. Keep deep-import enforcement inside `boundarycontrol` and remove `nodeepimports`

The same-scope deep-import restriction already exists inside `boundarycontrol`. This change should make that the only supported implementation by removing:

- plugin registration for `nodeepimports`
- `nodeepimports` docs and examples
- `nodeepimports` plugin tests and coverage expectations
- self-lint configuration that still enables `nodeepimports`

The deep-import rule itself should stay behaviorally the same unless the updated specs say otherwise: immediate child imports remain allowed, and deeper same-scope imports remain violations.

Rationale:

- It removes overlapping analyzers with the same architectural purpose.
- It gives users one place to configure import policy.

Alternatives considered:

- Keep `nodeepimports` as a deprecated wrapper around `boundarycontrol`: rejected because the user requested removal, and a wrapper would prolong duplicate behavior and compatibility code.

### 6. Normalize and validate configuration once before import traversal

The current analyzer reparses selector strings during validation and owner lookup. This change should normalize the public `architecture` map into a compiled internal form before visiting imports. The compiled form should hold parsed key selectors and parsed import selectors so traversal only performs matching.

Rationale:

- The new config shape already requires a normalization step.
- Precompiled selectors keep the runtime path straightforward once module context is resolved.

Alternatives considered:

- Continue reparsing selectors on each lookup: rejected because it keeps validation and evaluation coupled to raw strings and makes the new module-aware path harder to reason about.

## Risks / Trade-offs

- [Breaking existing `boundarycontrol` configs] -> Update README, `.golangci.yml`, and E2E scenarios in the same change so the repository demonstrates only the new `architecture` form.
- [Nested-module discovery adds filesystem work] -> Cache parsed module contexts by nearest `go.mod` directory and scan for descendant `go.mod` files once per module.
- [Analyzer behavior now depends on on-disk `go.mod` files] -> Fail clearly when the package has files but no enclosing `go.mod` can be found, rather than silently disabling checks.
- [Removing `nodeepimports` shrinks migration flexibility] -> Accept the clean break because `boundarycontrol` already contains the behavior and the repository controls its own plugin/docs surface.

## Migration Plan

1. Update the architecture-control and plugin E2E specs to describe the new config shape, module auto-discovery, multi-module behavior, and `nodeepimports` removal.
2. Change plugin settings decoding and analyzer config normalization to consume `architecture` instead of `selectors`.
3. Add per-pass module discovery, nested-module exclusion, and compiled selector matching inside `pkg/boundarycontrol`.
4. Remove `nodeepimports` registration and its test/documentation references.
5. Update repository-facing examples and self-lint config to use only `boundarycontrol`.

Rollback strategy:

- Restore `nodeepimports` registration and revert `boundarycontrol` to the old `module-root` plus `selectors` configuration if the new module-discovery path proves unstable.

## Open Questions

- None.
