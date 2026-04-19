## Context

`boundarycontrol` currently runs a cached module-wide false-sharing pass that only looks at package import edges. A shared package is considered healthy once two non-test package paths import it, even if those consumers rely on completely different exported APIs inside that package.

This change keeps the existing selector-based configuration model and the single `boundarycontrol` runtime entry point, but deepens the analysis so the false-sharing signal matches the actual exported API surface being shared. The repository is small and pure Go, so the main constraint is implementation complexity rather than deployment or runtime migration.

## Goals / Non-Goals

**Goals:**
- Count false-sharing consumers per exported symbol instead of per shared package.
- Keep `architecture` and `shared: true` configuration unchanged.
- Reuse the current once-per-module cache pattern so symbol analysis does not rerun independently for every package pass.
- Make diagnostics actionable enough for users to identify which exported API in a shared package is not actually shared.
- Keep the change scoped to `boundarycontrol` and its existing test/doc surfaces.

**Non-Goals:**
- No new plugin name, top-level config namespace, or selector syntax.
- No attempt to preserve package-level false-sharing semantics as an option.
- No special compatibility path for removed `mode` settings.
- No expansion into export-contract enforcement or broader API-shape linting.

## Decisions

### Run one cached module-wide symbol usage pass

The false-sharing implementation should keep the current cache shape: one module-wide analysis result per discovered module directory and compiled config. The difference is that the cached pass will load enough package metadata to inspect actual symbol references from importing packages, not just their import lists.

Implementation direction:
- load current-module packages with syntax and type information in addition to imports
- keep shared-package discovery based on selector ownership exactly as it works today
- enumerate exported symbols declared by each shared package
- walk non-test consumer packages and record which consumer package paths reference which exported shared-package symbols

Why this approach:
- it preserves the current analyzer architecture and avoids N-times repeated module scans
- it keeps import-control and false-sharing coupled only at the shared package discovery boundary
- it gives one deterministic data set that every package pass can reuse for reporting

Alternatives considered:
- recompute symbol usage during each `analysis.Pass`
- rejected because it would be substantially more expensive and would duplicate work across every analyzed package

- keep using import edges only and infer symbol sharing from package imports
- rejected because that is the semantic gap this change exists to close

### Count exported objects owned by the shared package

The counted unit should be the exported Go object defined by the shared package, not the package itself. The initial symbol set should include exported package-owned objects that can be resolved reliably from type information: exported functions, vars, consts, type names, and exported methods whose receiver type is declared in the shared package.

Design rules:
- only non-test consumer packages count
- multiple files in the same consumer package still count as one consumer for a given symbol
- unexported objects never participate in false-sharing evaluation
- shared packages with no exported symbols produce no false-sharing diagnostics
- a reference from another declaration inside the same shared package counts as valid additional use for that exported symbol
- shared packages under `internal/` are treated the same as any other shared package; if a selector marks them shared, they must still show multi-package symbol usage

Why this approach:
- it matches the proposal's intent to measure actual shared API use rather than raw package import reach
- it stays aligned with Go's type information model instead of relying on text matching
- it avoids introducing configuration branches for `internal` packages or different symbol kinds

Alternatives considered:
- count exported struct fields as independent shared symbols
- rejected because field ownership is more ambiguous for reporting and the type reference already captures the dominant API-sharing signal

- exempt `internal` packages from symbol-level checks
- rejected because `shared: true` is already an explicit opt-in and should keep one consistent meaning

### Report diagnostics at symbol declarations

The implementation should move from one package-level false-sharing diagnostic to one diagnostic per under-shared exported symbol, anchored to the symbol declaration inside the shared package. Each diagnostic should include the symbol identity and the consumer package paths that use it when there is exactly one consumer.

Implementation direction:
- cache under-shared symbol results using a stable symbol key such as package path plus object identity
- when the analyzer runs on the shared package itself, match cached results back to declaration positions in the current pass and report them there
- keep message format concise and deterministic so tests can assert exact output
- mention only direct external consumer package paths in diagnostics; do not add separate notes for test-only references

Why this approach:
- it gives users a direct pointer to the exported API surface that violates the rule
- it avoids long package-level summary messages once several symbols fail at once
- it keeps diagnostics local to the package that owns the shared API

Alternatives considered:
- emit one aggregated package-level diagnostic listing all under-shared symbols
- rejected because it becomes noisy quickly and forces users to inspect the whole package to locate the problem declarations

### Keep configuration and plugin behavior unchanged

This change should modify analysis depth only. `boundarycontrol` remains the only runtime plugin, `shared: true` remains the way packages opt into false-sharing checks, and removed `mode` settings stay unsupported.

Why this approach:
- it minimizes user migration surface to one behavioral change instead of another config rewrite
- it matches the parity-plan note that this slice should keep config stable while deepening analysis

Alternative considered:
- add per-selector symbol-level settings or thresholds
- rejected because there is no current need for new tuning knobs and they would complicate both docs and specs

## Risks / Trade-offs

- [More expensive package loading] -> Limit the symbol pass to configurations that actually contain `shared: true` selectors and keep the once-per-module cache.
- [Symbol identity can be tricky across cached package loads and `analysis.Pass`] -> Use a stable symbol key for cached results and remap to declaration positions from the current pass before reporting.
- [Behavioral breakage for packages that import a shared package but do not share the same exports] -> Call this out as breaking in the proposal/spec/docs and cover representative regressions in analyzer and plugin tests.
- [Go-specific edge cases such as method values, promoted methods, aliases, and dot imports] -> Base usage detection on `types.Info` rather than raw AST shape, and add focused tests for the supported reference forms.
- [Multiple diagnostics per package may feel noisier than the current package-level report] -> Keep messages short and declaration-anchored so the extra precision outweighs the count increase.

## Migration Plan

1. Update the `no-false-sharing` capability spec from package-level consumers to symbol-level consumers.
2. Extend the false-sharing loader so cached module analysis includes syntax and type information needed for symbol reference tracking.
3. Replace package-level consumer aggregation with exported-symbol aggregation keyed by shared package and symbol identity.
4. Change reporting in `boundarycontrol` so under-shared symbols are reported at their declarations in the owning shared package.
5. Rewrite and expand analyzer/plugin tests to cover symbol-level sharing, including cases where two packages import the same shared package but use different exports.
6. Update README and related docs that still describe package-level false-sharing.

Rollback strategy:
- before release, revert the change if the symbol-level signal proves too noisy or expensive
- after release, reverting would mean restoring package-level behavior in `boundarycontrol`, so the change should ship only with updated specs, tests, and docs

## Open Questions

- None at this stage.
