## Context

`boundarycontrol` already provides the shared architecture model for selector ownership, import allowlists, and shared-package false-sharing. The remaining gap is export-surface policy: users can decide which packages may import each other, but they still cannot declare what exported API shape is allowed for selected package groups.

This change should add export contracts without creating another analyzer or another top-level configuration namespace. The repository is small, pure Go, and already has the right extension points in `plugin/module.go`, `boundarycontrol.Config`, selector compilation, and package-local analyzer execution. Unlike false-sharing, export-contract enforcement does not need a cross-package cached pass because the names to validate are declared in the current package.

## Goals / Non-Goals

**Goals:**

- Extend selector-owned `boundarycontrol` policy so a package group can declare export-name contracts.
- Enforce contracts on exported top-level package declarations using deterministic regex matching.
- Reuse existing selector ownership and precedence rules rather than introducing a second targeting model.
- Fail clearly when export-contract configuration is malformed.
- Keep import-control and false-sharing behavior unchanged.

**Non-Goals:**

- Adding a new plugin name, analyzer package, or top-level config section.
- Enforcing contracts on methods, struct fields, interface methods, or other non-package-scope members.
- Inferring public packages from Go visibility rules or directory naming conventions.
- Changing selector syntax, import policy semantics, or false-sharing thresholds.

## Decisions

### 1. Host export contracts inside each `boundarycontrol` selector policy

The existing `architecture` mapping should gain an `exports` field on each selector policy. `exports` is a list of regex patterns, and an exported top-level declaration passes when its name matches at least one configured pattern for the package's owning selector.

Example direction:

```yaml
linters-settings:
  custom:
    boundarycontrol:
      architecture:
        pkg/api:
          exports:
            - "^(New|MustNew)[A-Z].*$"
            - "^[A-Z][A-Za-z0-9]*Error$"
```

Rationale:

- It keeps export policy in the same selector-owned model that already drives import control and shared-package detection.
- It avoids creating another rule surface that would need separate selector parsing, precedence, and docs.
- It gives one predictable place for future architecture-aware policy extensions.

Alternatives considered:

- Add a standalone `exportcontrol` plugin with separate config: rejected because it would duplicate selector ownership logic that `boundarycontrol` already owns.
- Add a nested `public` object with its own schema first: rejected because the immediate need is export-name contracts, not a broader package-surface model.

### 2. Treat `exports` presence as the opt-in for export-surface enforcement

This change should not introduce a separate `public: true` flag. If a selector policy declares one or more `exports` patterns, every package owned by that selector participates in export-contract evaluation. Selectors without `exports` remain unaffected.

Rationale:

- It is the smallest change that still lets users mark selected package groups as contract-governed API surfaces.
- It keeps targeting aligned with the current owner-resolution model instead of layering another package classification concept on top.

Trade-off:

- A broad selector such as `pkg/api` will apply to that package and its descendants, so users need to choose selectors carefully when only part of a subtree should have an export contract.

Alternatives considered:

- Add a separate `public` or `surface` flag and require both fields: rejected because it adds configuration ceremony without improving the first delivery.

### 3. Enforce contracts only for exported top-level package-scope declarations

The analyzer should validate exported package-scope names only: exported functions, types, vars, and consts in the package scope. Methods should be excluded in this version, even when exported, because the proposal is explicitly about exported top-level declarations and package-scope analysis is both simpler and more predictable.

Implementation direction:

- inspect the current package scope through type information
- collect exported objects from `pass.Pkg.Scope()`
- report violations at the declaration position of each offending object

Rationale:

- Package-scope objects map directly to Go's exported package API surface.
- The current pass already has the data needed, so the rule stays cheap and local.
- Excluding methods avoids ambiguous questions about receiver ownership, embedded types, aliases, and method promotion in the first version.

Alternatives considered:

- Include exported methods too: rejected because it expands scope beyond the proposal and introduces edge cases that are not required for the first contract surface.

### 4. Compile regexes during `boundarycontrol` config validation and use full-name matching

Export patterns should be compiled during the existing `compileConfig(normalizeConfig(cfg))` path, alongside selector parsing. Invalid regexes should fail configuration before analysis begins. Matching should be full-name matching, not substring matching, so a declaration name must satisfy the entire pattern set contract rather than accidentally matching a fragment.

Implementation direction:

- extend `Policy` and plugin settings with `Exports []string`
- compile each pattern once into the compiled policy
- evaluate names with anchored full-string semantics while preserving the user's original pattern text in error messages

Rationale:

- Early compilation keeps configuration failures consistent with the rest of `boundarycontrol` validation.
- Full-name matching aligns with how naming contracts are usually understood by users.
- Compiling once avoids repeated regex work across declarations.

Alternatives considered:

- Lazy-compile during each package pass: rejected because it pushes config errors later and repeats work unnecessarily.
- Use raw substring-style regex matching: rejected because it makes contracts too permissive by default.

### 5. Run export-contract checks as a package-local pass inside `boundarycontrol`

Export-contract enforcement should run during the normal `analysis.Pass` for the current package, after config and module discovery succeed and before false-sharing reporting. The analyzer only needs the current package's relative path, resolved owner policy, and package-scope exported objects.

Implementation direction:

- resolve the current package owner using existing nearest-owner precedence
- skip evaluation when the owner has no compiled export patterns
- iterate exported package-scope objects and report a diagnostic for each non-matching name
- keep diagnostics deterministic by walking names in sorted order when needed

Rationale:

- The rule is inherently local to the declaring package, unlike false-sharing.
- Reusing the current pass keeps implementation small and avoids another cache or module scan.
- It fits naturally beside the existing import-control and false-sharing checks under one analyzer.

Alternatives considered:

- Add a separate cached module-wide export scan: rejected because there is no cross-package data dependency that justifies the extra complexity.

### 6. Use one clear diagnostic per offending exported declaration

Each exported declaration that fails the owner's contract should produce its own diagnostic at the declaration site. The message should identify the exported name and make it clear that the package's configured export contract did not allow that name.

Rationale:

- Declaration-local diagnostics are consistent with the rest of the repository's analyzer style.
- They make fixes obvious without forcing users to decode a package summary.

Trade-off:

- A package with many offending exports will produce several diagnostics, but each one is directly actionable.

## Risks / Trade-offs

- [Broad selectors may apply contracts more widely than intended] -> Reuse the documented owner-resolution rules and add spec/test examples that show subtree ownership clearly.
- [Regex-based contracts can be hard to read or too permissive] -> Fail invalid regexes early and document full-name matching so expectations stay predictable.
- [Excluding methods leaves part of a package API unchecked] -> Keep the first version limited to top-level declarations and treat method coverage as a follow-up only if users need it.
- [Per-declaration diagnostics may feel noisy in legacy packages] -> Keep messages short and deterministic, and let users widen patterns or narrow selectors as needed.

## Migration Plan

1. Extend `boundarycontrol` settings and config compilation with `exports` pattern support and validation.
2. Update `boundarycontrol` spec coverage so selector-owned policies can declare export contracts.
3. Add a new `export-control` capability spec covering top-level export-name enforcement and config-error cases.
4. Implement package-local export-contract reporting inside `boundarycontrol` without changing import-control or false-sharing behavior.
5. Add analyzer tests and plugin E2E scenarios for matching exports, violations, and invalid regex configuration.

Rollback strategy:

- Remove `exports` handling from `boundarycontrol` config and reporting, and revert the new spec coverage if the rule proves too noisy or the contract shape needs redesign.

## Open Questions

- None for this artifact. The first version intentionally resolves the scope to top-level package exports only and uses selector ownership alone to decide which packages are contract-governed.
