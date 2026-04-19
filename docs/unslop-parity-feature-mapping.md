# Unslop Parity Feature Mapping And Incremental Plan

## Purpose

This document captures the current parity findings between `gounslop` and `../eslint-plugin-unslop`, plus a high-level feature breakdown that can be turned into OpenSpec changes one feature at a time.

It is intentionally planning-oriented:

- no code changes
- no final schema lock-in
- explicit options where design is still open
- task slices sized for incremental OpenSpec proposals

## Current Baseline

### Upstream rule inventory

`eslint-plugin-unslop` currently ships these full-config rules:

- `no-special-unicode`
- `no-unicode-escape`
- `read-friendly-order`
- `import-control`
- `no-whitebox-testing`
- `export-control`
- `no-false-sharing`
- `no-single-use-constants`

### Current Go analyzer inventory

`gounslop` currently ships these analyzers:

- `boundarycontrol`
- `nospecialunicode`
- `nounicodeescape`
- `readfriendlyorder`

### High-level parity summary

| Upstream rule | Current Go status | Decision direction | Notes |
| --- | --- | --- | --- |
| `no-special-unicode` | Covered by `nospecialunicode` | Keep | Good parity, Go-native literal handling already exists |
| `no-unicode-escape` | Covered by `nounicodeescape` | Keep | Good parity, no strong `golangci-lint` equivalent |
| `read-friendly-order` | Covered by `readfriendlyorder` | Keep and spec | Go adaptation differs from TS by design |
| `import-control` | Partially covered by `boundarycontrol` | Add via unified architecture feature | Current Go rule covers in-module boundaries and same-scope deep-import restrictions |
| `no-whitebox-testing` | Missing | Drop as direct port | Go package model differs; closest ecosystem equivalent is `testpackage` |
| `export-control` | Missing | Add via unified architecture feature | Needs Go-specific public-package/export contract model |
| `no-false-sharing` | Covered by `boundarycontrol` | Keep expanding | Current Go rule now tracks exported symbols; continue closing edge-case parity gaps |
| `no-single-use-constants` | Missing | Add as separate analyzer | Existing Go linters only partially overlap |

### Existing Go-specific value to preserve

- Keep the Go-specific analyzers already implemented.
- Keep Go-native `readfriendlyorder` behavior such as `init()` placement and method ordering.
- Preserve the useful same-scope deep-import behavior now carried by `boundarycontrol`.

## Existing Ecosystem Overlap

These overlaps matter because parity should avoid re-implementing what `golangci-lint` already covers well.

| Concern | Existing linter overlap | Why it is not enough for full parity |
| --- | --- | --- |
| Import allow/deny rules | `depguard` | Good package allow/deny tool, but not a shared subtree-ownership architecture model |
| Declaration ordering | `decorder`, `funcorder` | Partial overlap only; they do not replace dependency-aware `readfriendlyorder` |
| Test white-box isolation | `testpackage` | Different mechanism, but closest Go equivalent; enough reason to drop direct port |
| Unused constants | `unused` | Catches zero-use, not single-use |
| Constant extraction | `goconst`, `revive` `add-constant` | Opposite direction from `no-single-use-constants` |
| Unicode identifier safety | `asciicheck` | Identifiers only, not string literal hygiene |
| Dangerous bidi characters | `bidichk` | Security-oriented bidi checks, not smart-quote / invisible-space cleanup |
| i18n string rules | `gosmopolitan` | Different intent from ASCII-equivalent literal cleanup |

## Recommended Product Direction

### Core recommendation

Introduce a unified architecture capability around a new concept tentatively named `archcontrol`.

That capability should eventually cover:

- same-domain deep import restrictions now handled by `boundarycontrol`
- broader import boundary control
- shared package declaration and false-sharing analysis
- exported symbol contract checks for selected package groups

Keep non-architecture analyzers separate:

- `nospecialunicode`
- `nounicodeescape`
- `readfriendlyorder`
- future `nosingleuseconstants`

### Why unify architecture config

The upstream plugin has one shared architecture-policy model used by multiple rules. The Go side currently has separate flags per analyzer.

Without unification, `gounslop` is likely to grow a pile of disconnected flags:

- `boundarycontrol`: selector-owned `imports`, `exports`, and `shared` policy
- future import control: unknown new flags
- future export control: unknown new flags

With unification, the architecture-aware features can share one package-subtree model.

### Recommended unit of architecture

Use package subtree ownership, not free-form regex-only rules.

Recommended mental model:

```text
module root
└── package selectors
    ├── import policy
    ├── public package policy
    ├── shared package policy
    └── export-name policy
```

This is easier to reason about than raw regex matching and maps better to several analyzers sharing one policy engine.

## Naming Options For Import-Control Evolution

### Recommended name

- `boundarycontrol`

Why:

- short enough for `golangci-lint`
- matches the implemented first-step scope: import boundaries only
- leaves room for a later broader architecture wrapper if needed

### Other plausible names

- `archpolicy`
- `archboundaries`
- `boundarycontrol`
- `pkgcontrol`
- `moduleboundaries`

### Naming recommendation

Use `boundarycontrol` for the current import-boundary rule.

### Rename strategy options

| Option | Description | Pros | Cons |
| --- | --- | --- | --- |
| A | Hard rename `boundarycontrol` to `archcontrol` | Cleanest long-term UX | Breaking change immediately |
| B | Add `archcontrol` while keeping `boundarycontrol` as the focused import rule | Easier migration | More naming overlap during transition |
| C | Keep `boundarycontrol` and add separate analyzers with shared config | Smallest short-term change | Weaker unified product story |

### Rename recommendation

Prefer option B.

That gives a clean destination name while preserving a low-risk migration path.

## Unified Architecture Config Direction

### Proposed scope for shared config

The shared architecture config should describe:

- module root
- package selectors
- allowed imports between selectors
- shared package selectors
- export naming contracts for selected packages

### Go-native concepts to use

- package paths, not file entrypoints
- subtree selectors, not TypeScript file modules
- public package surfaces, not `index.ts`
- exported Go declarations, not `export *`

### TypeScript-specific concepts to avoid porting literally

- `typeImports`
- namespace import restrictions
- `entrypoints` as filenames like `index.ts`
- `export *`
- same-directory sibling-file test imports

### Early schema sketch

This is not a final proposal, just a directionally useful shape:

```yaml
architecture:
  ".":
    imports: ["pkg/+", "internal/*", "pkg/shared"]
  "pkg/models":
    imports: ["pkg/utils"]
  "pkg/repository/*":
    imports: ["pkg/models/+", "pkg/utils"]
  "pkg/shared":
    shared: true
```

### Selector direction

Suggested selector semantics:

- `.` for module root
- exact package subtree selectors like `pkg/models`
- direct-child wildcard selectors like `pkg/repository/*`
- self-or-direct-child wildcard selectors like `pkg/models/+`

## E2E Testing Framework Direction

### Upstream pattern worth borrowing

The upstream ESLint plugin uses a single shared `scenario()` helper that supports:

- in-memory tests for simple rules
- temp-dir full-project scenarios for architecture-aware rules
- shared injection of architecture settings
- explicit fixture-driven configuration-error cases

### Go adaptation target

For `gounslop`, the E2E layer should complement `analysistest`, not replace it.

Recommended test pyramid:

- `analysistest` for analyzer internals and small AST/type cases
- feature-level E2E tests that create temp Go modules on disk
- plugin-level E2E tests that build or reuse `custom-gcl` and run `./custom-gcl run ./...`

### What the Go E2E harness should support

- temp workspace creation
- fixture file sets for multi-package test repos
- generated `.golangci.yml` and optional `.custom-gcl.yml`
- invocation of the plugin through `golangci-lint` custom binary
- stdout/stderr capture and golden-style assertions
- explicit configuration-error scenarios
- reuse by all architecture-aware features

### Suggested first testing milestone

Build a minimal shared harness that can:

- create a temp module
- write files
- run `./custom-gcl run ./...`
- assert diagnostics

Keep the first version narrow. Avoid overbuilding a framework before `archcontrol` exists.

## Proposed OpenSpec-Friendly Change Breakdown

The following slices are designed to be small enough for separate changesets while still fitting a coherent roadmap.

### Change 1: `add-plugin-e2e-harness`

Goal:

- introduce a reusable E2E test harness for plugin-level and architecture-level scenarios

Why first:

- later architecture changes will be safer with end-to-end coverage
- prevents repeated ad hoc temp-project test helpers

Scope:

- add fixture-driven temp-module test helper
- support running `custom-gcl` against generated test repositories
- document test conventions

Out of scope:

- no analyzer behavior changes
- no new lint rules

Acceptance ideas:

- one sample E2E test for an existing analyzer
- one sample config-error case
- one sample multi-package case

Dependencies:

- none

### Change 2: `specify-existing-analyzers`

Goal:

- backfill explicit specs for current shipped behavior before larger refactors

Capabilities to specify:

- unicode literal hygiene
- read-friendly ordering
- deep same-domain import restriction
- current false-sharing behavior

Why early:

- clarifies what is contract versus implementation accident
- reduces refactor ambiguity when boundary-related analyzers are reorganized

Scope:

- create specs for currently shipped analyzers
- mark known deliberate Go-specific deviations from upstream

Out of scope:

- no renames yet
- no new behavior

Dependencies:

- none

### Change 3: `design-archcontrol-schema`

Goal:

- define the shared architecture capability and config schema without yet implementing all subfeatures

Scope:

- choose selector model
- choose config placement and naming
- choose how package-level false-sharing fits the shared config
- decide whether public-package semantics are part of v1
- document migration path from current flags

Key open decisions:

- package selectors only, or selectors plus raw regex escape hatch
- single umbrella analyzer versus multiple analyzers sharing the same schema
- compatibility story for `boundarycontrol` and future `archcontrol` naming

Recommended output:

- one architecture capability spec
- one design doc section for config shape and migration

Dependencies:

- `specify-existing-analyzers` is helpful but not strictly required

### Change 4: `introduce-archcontrol-name`

Goal:

- establish the new architecture feature name and compatibility posture

Scope:

- add the `archcontrol` concept to docs/specs
- define whether `boundarycontrol` remains standalone or becomes a wrapper/alias under a broader architecture surface
- define how shared-package checks remain under `boundarycontrol` during migration

Why separate:

- keeps naming and migration concerns from getting tangled with feature behavior

Dependencies:

- `design-archcontrol-schema`

### Change 5: `add-archcontrol-import-boundaries`

Goal:

- add import boundary control on top of the new architecture model

Minimum target scope:

- preserve current same-domain deep-import restriction
- add cross-selector allowlist behavior
- define default deny/allow behavior for matched selectors

Possible sub-slices if needed:

- phase A: port `boundarycontrol` semantics into `archcontrol`
- phase B: add cross-selector import allowlists
- phase C: add public-package-only import rules if adopted in schema

Open questions:

- should unmatched packages be anonymous-and-denied like upstream, or ignored by default in v1
- how strongly should Go `internal/` semantics interact with rule logic

Dependencies:

- `design-archcontrol-schema`
- `introduce-archcontrol-name`
- strongly benefits from `add-plugin-e2e-harness`

### Change 6: `migrate-false-sharing-into-archcontrol`

Goal:

- move current false-sharing configuration under the shared architecture capability

Minimum target scope:

- support `shared: true` style package selectors
- preserve current package-level shared-dir analysis under new config

Recommended split:

- phase A: config migration only, behavior mostly preserved
- phase B: decide whether to upgrade from package-level to symbol-level analysis

Why split:

- configuration unification is lower risk than semantic expansion
- symbol-level analysis is the true parity jump and deserves its own proposal if needed

Open questions:

- whether package-level analysis remains sufficient until symbol-level analysis lands
- whether symbol-level analysis is required for v1 `archcontrol` or can follow later

Dependencies:

- `design-archcontrol-schema`
- `introduce-archcontrol-name`

### Change 7: `upgrade-false-sharing-to-symbol-level`

Goal:

- close the biggest semantic gap with upstream `no-false-sharing`

Why separate:

- current Go rule is package-level
- upstream rule is exported-symbol-level
- this is the most analysis-heavy architecture gap

Scope:

- count consumers per exported symbol rather than per shared package
- design Go-specific treatment for internal package usage
- keep the config stable while deepening the analysis

Dependencies:

- `migrate-false-sharing-into-archcontrol`

### Change 8: `add-archcontrol-export-contracts`

Goal:

- add Go-specific export contract enforcement using the architecture model

Target scope:

- allow regex contracts for exported top-level names in matched package groups
- limit enforcement to appropriate package surfaces

Open questions:

- whether contracts apply to every matched package or only designated public packages
- whether methods should be included or only top-level exported declarations

Dependencies:

- `design-archcontrol-schema`
- ideally after import-boundary work, since both will share selector semantics

### Change 9: `add-no-single-use-constants`

Goal:

- add a separate analyzer for constants used once or never

Why separate from `archcontrol`:

- not an architecture concern
- can ship independently
- high value and relatively self-contained

Target scope:

- local counting for unexported constants
- package or project-aware counting for exported constants
- explicit exclusions for value kinds that are likely intentional abstractions

Dependencies:

- none
- benefits from the E2E harness only if cross-package exported-use counting is implemented

### Change 10: `specify-go-adapted-readfriendlyorder`

Goal:

- formalize the Go-specific contract for `readfriendlyorder`

Why separate from general spec backfill:

- this analyzer is already richer than a direct upstream port
- it has the most language-shaped behavior and should be explicitly documented

Scope:

- top-level helper ordering
- `init()` ordering
- constructor and method placement
- Go test ordering behavior

Dependencies:

- can be folded into `specify-existing-analyzers` if preferred

## Suggested Roadmap Order

Recommended order:

1. `add-plugin-e2e-harness`
2. `specify-existing-analyzers`
3. `design-archcontrol-schema`
4. `introduce-archcontrol-name`
5. `add-archcontrol-import-boundaries`
6. `migrate-false-sharing-into-archcontrol`
7. `add-archcontrol-export-contracts`
8. `upgrade-false-sharing-to-symbol-level`
9. `add-no-single-use-constants`

Alternative order if feature delivery matters more than clean architecture setup:

1. `add-plugin-e2e-harness`
2. `add-no-single-use-constants`
3. `design-archcontrol-schema`
4. `introduce-archcontrol-name`
5. `add-archcontrol-import-boundaries`
6. `migrate-false-sharing-into-archcontrol`
7. `add-archcontrol-export-contracts`
8. `upgrade-false-sharing-to-symbol-level`

## Recommended Capability Grouping For Specs

If these are turned into OpenSpec capabilities, this grouping is likely cleanest:

- `unicode-literal-hygiene`
- `read-friendly-order`
- `architecture-control`
- `false-sharing`
- `single-use-constants`

Two grouping options are reasonable.

### Option A: keep false-sharing inside `architecture-control`

Pros:

- matches the unified config direction
- fewer capability boundaries

Cons:

- false-sharing semantics may evolve faster than import/export boundary semantics

### Option B: separate `false-sharing` capability under shared config

Pros:

- easier to spec and change independently
- cleaner if package-level and symbol-level behavior need phased evolution

Cons:

- slightly more cross-referencing between specs

Recommendation:

Prefer option B for OpenSpec even if the runtime config is unified.

## Concrete Findings To Preserve In Future Specs

- `no-whitebox-testing` should not be directly ported to Go.
- `depguard` is useful ecosystem overlap but not a replacement for a package-subtree architecture model.
- `readfriendlyorder` is already a Go adaptation, not a failed parity port.
- `boundarycontrol` currently provides only package-level shared-package signals.
- the true parity gap for false sharing is symbol-level consumer analysis.
- `no-single-use-constants` remains an unfilled and worthwhile gap.
- unified architecture config is valuable, but it should be package-centric, not file-centric.

## Open Questions To Resolve During Proposals

- Should `archcontrol` be one umbrella analyzer or a shared config consumed by several analyzers?
- Should unmatched packages be ignored or denied by default under the architecture model?
- Should `archcontrol` keep package-level shared-package analysis only, or later add symbol-level analysis?
- Should export contracts apply to all matched packages or only explicitly public package groups?
- How much compatibility should be preserved for architecture-related analyzer naming after consolidating on `boundarycontrol`?
- Is symbol-level false-sharing required before `archcontrol` is considered complete, or is it a follow-up enhancement?

## Short Recommendation

If only a few near-term changes should be proposed first, the highest-leverage sequence is:

1. add the E2E harness
2. define the shared `archcontrol` schema
3. evolve `boundarycontrol` into `archcontrol` and add real import-boundary control
4. migrate false-sharing config into `archcontrol`
5. add export contracts
6. add `no-single-use-constants`

That sequence keeps the architecture story coherent while still letting the remaining gaps land as separately reviewable changes.
