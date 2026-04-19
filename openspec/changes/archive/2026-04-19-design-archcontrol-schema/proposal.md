## Why

`architecture-control` currently covers only a narrow same-scope deep-import check. The project needs a first-class import-boundary rule that can express the package-level capabilities of the TypeScript `import-control` rule in Go terms, while also absorbing the useful behavior already provided by `nodeepimports`.

## What Changes

- Expand the `architecture-control` capability from a single deep-import rule into a broader import-boundary capability.
- Introduce the first version of a new rule named `boundarycontrol`.
- Define `boundarycontrol` so its selector keys and `imports` selectors match the TypeScript `import-control` rule semantics, adapted from modules to Go package paths, with the requested Go-specific key syntax and rule-resolution behavior.
- Treat unmatched packages as having an empty import-control list.
- Integrate the current `nodeepimports` behavior into `boundarycontrol` as part of the Go-specific rule behavior.
- Always allow one-level-deep imports, independent of configured rules.
- Resolve overlapping key matches using nearest-owner precedence: exact key over wildcard at the same depth, then longer selector path, then declaration order.
- Leave `nodeepimports` itself unchanged in this version; its removal is deferred to the next version.
- Keep this change import-control only. It does not add export-control or false-sharing behavior.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `architecture-control`: Expand the capability to include the new `boundarycontrol` rule and its package-based import-boundary semantics.

## Impact

- Affects `openspec/specs/architecture-control/spec.md` and the implementation work for package-boundary analysis.
- Adds a new rule surface, `boundarycontrol`, while intentionally leaving `nodeepimports` intact for one release.
- Establishes package-selector ownership, import-allowlist, precedence, and built-in deep-import semantics that must stay aligned with the TypeScript `import-control` model, except for the Go-specific package-path adaptation.
