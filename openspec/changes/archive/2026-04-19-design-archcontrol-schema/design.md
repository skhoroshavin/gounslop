## Context

`architecture-control` currently defines only one narrow import-boundary behavior: the same-scope deep-import restriction implemented by `nodeepimports`. This change expands that capability by introducing `boundarycontrol`, a new rule whose first version should cover the package-boundary capabilities of the TypeScript `import-control` rule, adapted to Go package paths.

The scope is intentionally limited to import control. `boundarycontrol` should not define export-control or false-sharing behavior in this version. At the same time, it must absorb the useful deep-import protection from `nodeepimports` so that `nodeepimports` can be removed in a later version without losing coverage.

## Goals / Non-Goals

**Goals:**

- Expand `architecture-control` to cover selector-based import-boundary rules.
- Introduce the first version of `boundarycontrol`.
- Keep selector semantics for rule keys and `imports` entries aligned with the TypeScript `import-control` rule, with Go packages replacing TypeScript modules.
- Treat unmatched packages as having an empty import-control list.
- Resolve overlapping key matches using the same nearest-owner precedence as the TypeScript spec.
- Always allow one-level-deep imports.
- Integrate the current `nodeepimports` behavior into `boundarycontrol`.
- Leave `nodeepimports` itself unchanged in this version.

**Non-Goals:**

- Changing or deleting `nodeepimports` in this version.
- Adding export-control behavior.
- Adding false-sharing behavior.
- Inventing a new selector language that diverges from the TypeScript `import-control` rule.

## Decisions

### 1. Expand `architecture-control` by adding `boundarycontrol`

The capability should be broadened around a new rule named `boundarycontrol`. `boundarycontrol` becomes the new home for selector-driven import-boundary checks, while `nodeepimports` remains as-is until the follow-up removal change.

Rationale:

- It gives the capability a clear import-boundary rule rather than overloading the old deep-import-only behavior.
- It allows the spec and implementation to grow around the new rule without coupling this change to immediate removal of `nodeepimports`.

Alternatives considered:

- Extend `nodeepimports` directly: rejected because the requested behavior is broader than the existing rule's identity.
- Remove `nodeepimports` immediately: rejected because this version must not touch `nodeepimports` itself.

### 2. Restrict rule-key selector syntax to `.`, `module`, `module/a`, and `module/*`

The selector language used for `boundarycontrol` rule keys should support only these shapes:

- `.`
- `module`
- `module/a`
- `module/*`

For rule keys, the matching semantics are:

- `.` matches the module root package
- `module` matches that package and all of its subpackages
- `module/a` matches that package and all of its subpackages
- `module/*` matches all subpackages under `module`, but not `module` itself

Rationale:

- This gives rule keys the exact syntax the user requested.
- The matching semantics make broad package ownership rules possible without introducing additional key forms.

Alternatives considered:

- Reuse the full selector syntax for rule keys, including `+`: rejected because the requested key syntax is narrower.

### 3. Keep `imports` selector semantics as previously defined

The selector language used inside `imports` entries should remain aligned with the TypeScript `import-control` behavior adapted to Go package paths. This includes the broader selector forms already described for imported-package matching, including direct-child and self-or-direct-child matching where needed.

Rationale:

- The user explicitly wants narrower syntax for rule keys but to keep the current `imports` requirements.
- This preserves expressive imported-package matching without widening the key syntax.

Alternatives considered:

- Restrict `imports` selectors to the same narrow syntax as rule keys: rejected because that would remove requested `imports` behavior.

### 4. Match packages against a selector-keyed import allowlist

`boundarycontrol` should evaluate an importing package by finding the selectors whose keys match that package and then applying the configured `imports` selector list for that package. Imported packages are validated against those `imports` selectors using the same selector semantics.

This keeps the rule centered on one concept: package selectors control which other package selectors may be imported.

Rationale:

- It is the core model of selector-driven import control.
- It keeps key selectors and `imports` selectors structurally related even though the allowed syntax differs.

Alternatives considered:

- Use unrelated matching languages for keys and `imports`: rejected because the user still wants them to correspond to the TypeScript rule model.

### 5. Treat unmatched packages as having an empty import-control list

If an importing package does not match any configured selector key, it should behave as though it had `imports: []`.

Rationale:

- This matches the requested import-control behavior.
- It makes the policy explicit: packages are only allowed what their matching selector grants them.

Trade-off:

- This default is intentionally strict. Repositories that want broader access will need to define a broad selector such as `.` with an explicit `imports` policy.

### 6. Resolve overlapping key matches by nearest-owner precedence

If more than one rule-key selector matches an importing package, `boundarycontrol` should choose the owning key using the same precedence as the TypeScript architecture config:

- nearest owner first
- exact named path over wildcard path at the same depth
- longer selector path
- declaration order

This applies after the key-shape matching rules described above.

Rationale:

- It provides deterministic behavior for overlapping selectors such as `module`, `module/a`, and `module/*`.
- It matches the upstream ownership semantics rather than relying on a looser summary.

Implication:

- A narrower package selector can override a broader package-family selector without needing additional precedence settings.
- If two selectors are otherwise equivalent in ownership terms, the earlier declaration wins.

### 7. Always allow one-level-deep imports

An import from `module` to `module/a` should always be allowed, even when no configured rule grants it. This built-in allowance applies independently of the selector-based import-control list.

Rationale:

- This preserves the current one-level-deep allowance behavior.
- It keeps the first version from over-restricting normal parent-to-immediate-child package structure.

Trade-off:

- This creates one intentional exception to the otherwise strict "unmatched means empty imports" model.

### 8. Keep Go-specific behavior limited to package-path evaluation and `nodeepimports` integration

The first version should adapt the rule to Go in only the ways required by the language and the existing analyzer set:

- selectors operate on Go package paths rather than file-module paths
- evaluation is scoped by the configured Go module root, so standard-library and third-party imports outside that root are outside boundary matching
- the existing `nodeepimports` same-scope deep-import restriction becomes part of `boundarycontrol`
- immediate parent-to-child imports remain allowed regardless of rules

This last point is important: `boundarycontrol` must preserve the behavior currently provided by `nodeepimports`, but `nodeepimports` itself must remain unchanged in this version.

Rationale:

- These are the minimum Go-specific modifications needed to make TypeScript-style import control meaningful in this repository.
- Folding the deep-import rule into `boundarycontrol` now avoids a later feature gap when `nodeepimports` is removed.

Alternatives considered:

- Keep deep-import checks separate forever: rejected because the user explicitly wants that functionality integrated into `boundarycontrol`.
- Rewrite `nodeepimports` to delegate to `boundarycontrol` now: rejected because this change must not touch `nodeepimports` itself.

## Risks / Trade-offs

- [Duplicate diagnostics when both rules are enabled] -> Document `boundarycontrol` as the new import-boundary rule and keep `nodeepimports` removal as the next versioned change.
- [Exact selector parity may feel unfamiliar in Go-specific setups] -> Preserve the original semantics but explain them in package-path terms in the spec and user-facing docs.
- [Unmatched packages defaulting to empty imports may be stricter than existing repos expect] -> Encourage an explicit root selector such as `.` when a repository wants a broader baseline policy, while preserving the built-in one-level-deep allowance.
- [Integrating `nodeepimports` behavior without touching `nodeepimports` can temporarily duplicate capability] -> Accept the overlap for one version so migration stays low risk.

## Migration Plan

1. Expand the `architecture-control` spec so it defines `boundarycontrol` and its selector-based import semantics.
2. Implement `boundarycontrol` with package-path selector-key matching, `imports` allowlist evaluation, and nearest-owner precedence for overlapping key matches.
3. Include the current `nodeepimports` deep-import behavior inside `boundarycontrol`.
4. Preserve the unconditional one-level-deep import allowance.
5. Leave `nodeepimports` unchanged and available during this version.
6. Remove `nodeepimports` in the next version once `boundarycontrol` fully covers its use case.

Rollback strategy:

- Revert the `boundarycontrol` addition and keep `architecture-control` limited to its previous deep-import scope.
