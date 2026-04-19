## Context

Today the repository has two different architecture-adjacent runtime entry points:

- `boundarycontrol` parses selector-owned `architecture` settings and enforces import rules inside the discovered module.
- `nofalsesharing` parses standalone `shared-dirs`, `mode`, and `module-root` settings, loads the full package graph, and reports package-level false-sharing diagnostics.

The current spec layout no longer matches the intended product shape. `architecture-control` mixes general selector semantics with import-policy behavior, while `false-sharing` describes behavior that the user now wants hosted by `boundarycontrol` after the standalone plugin is removed.

This change has two constraints:

- remove the standalone `nofalsesharing` plugin completely rather than keeping a compatibility wrapper
- preserve the current package-level false-sharing threshold and package-level consumer model while the configuration model moves under `boundarycontrol`

## Goals / Non-Goals

**Goals:**
- Make `boundarycontrol` the only runtime plugin for the shared architecture model and migrated package-level false-sharing checks.
- Move false-sharing configuration into selector-owned `architecture` settings so shared package declarations live next to the rest of the boundary model.
- Split the spec surface so `boundarycontrol` covers only module and selector semantics, `import-control` covers import rules, and `no-false-sharing` covers false-sharing behavior implemented through `boundarycontrol`.
- Preserve existing false-sharing threshold and `_test.go` exclusion while standardizing migrated behavior on package-level consumer counting only.
- Keep the implementation small by reusing the existing false-sharing analysis logic where possible.

**Non-Goals:**
- No symbol-level false-sharing analysis.
- No new standalone `import-control` or `no-false-sharing` runtime plugin.
- No backward-compatibility layer that keeps `nofalsesharing` configuration or plugin registration working after this change lands.
- No redesign of selector syntax beyond what is needed to host shared package declarations.
- No support for migrated `file` mode.

## Decisions

### Keep one runtime plugin: `boundarycontrol`

`boundarycontrol` remains the only plugin/analyzer name for architecture-aware enforcement. The implementation will absorb the package-level false-sharing pass and `plugin/module.go` will stop registering `nofalsesharing`.

Why this approach:

- it removes the duplicate entry point the user explicitly wants gone
- it keeps configuration ownership in one place
- it avoids introducing another user-facing plugin name while the specs are being clarified

Alternative considered:

- keep `nofalsesharing` as a deprecated wrapper that forwards into `boundarycontrol`
- rejected because it prolongs parallel configuration paths and contradicts the requested full removal

### Extend selector policy with shared-package metadata

`boundarycontrol.Config` will continue to use an `architecture` mapping keyed by selectors, but each selector policy will grow a shared-package marker. The minimum v1 shape is:

```yaml
architecture:
  shared:
    shared: true
  feature/api:
    imports:
      - shared
```

Design rules for the new fields:

- `shared: true` marks the selector-owned subtree as a false-sharing candidate set
- migrated false-sharing always counts consumers by importing package path, equivalent to current `dir` mode
- selectors that do not opt into `shared: true` participate only in import ownership and selector resolution

Why this approach:

- it matches the existing selector model instead of adding a second top-level config namespace
- it preserves the package-subtree ownership direction already established for architecture settings
- it keeps the false-sharing migration focused on config unification instead of semantic expansion

Alternatives considered:

- keep `shared-dirs` as a separate top-level setting under `boundarycontrol`
- rejected because it preserves the split configuration model this change is trying to remove

- add a nested `false-sharing` object under each selector
- rejected for now because the current migrated behavior only needs a shared marker, so a flatter shape is simpler

- keep a `mode` field with only `dir` support or temporary `file` compatibility
- rejected because the user wants `file` mode removed rather than carried forward under a new name

### Split capabilities without splitting runtime enforcement

The OpenSpec capabilities will be reorganized, but the runtime analyzer will not be split.

- `boundarycontrol` spec will own module discovery, selector syntax, selector ownership, precedence, and shared-selector semantics
- `import-control` spec will own deep-import and allowed-import requirements that are currently mixed into `architecture-control`
- `no-false-sharing` spec will own consumer counting, threshold, and configuration validation requirements for the migrated dir-only behavior that now runs through `boundarycontrol`

Why this approach:

- it gives each spec one clear responsibility
- it matches the user's requested naming cleanup
- it avoids unnecessary code churn from creating new analyzers just to mirror spec boundaries

Alternative considered:

- create a separate runtime analyzer for `import-control`
- rejected because the current code already uses one boundary-aware analyzer and this change is primarily a migration and cleanup slice

### Move false-sharing internals out of `pkg/nofalsesharing`

The reusable package-graph analysis should move into `pkg/boundarycontrol` or a boundarycontrol-local helper file, and the standalone `pkg/nofalsesharing` package should be removed once nothing imports it.

Implementation direction:

- keep the current package-graph algorithm largely intact while dropping file-based consumer grouping
- translate shared selectors into the equivalent set of package prefixes the existing algorithm expects
- run the false-sharing pass only when at least one selector is marked `shared: true`
- report migrated false-sharing diagnostics through `boundarycontrol`

Why this approach:

- it preserves the important package-level behavior while simplifying configuration and implementation
- it avoids leaving a misleading package name behind after the plugin is removed
- it keeps `boundarycontrol` code organized by moving the false-sharing logic into dedicated files even though the plugin is unified

Alternative considered:

- keep `pkg/nofalsesharing` as an internal helper package
- rejected because the surviving runtime surface would no longer be named `nofalsesharing`, so keeping that package would make ownership harder to follow

### Make the migration explicitly breaking

The change will not accept legacy `nofalsesharing` plugin configuration. Users must move to `boundarycontrol` selector-based configuration.

Why this approach:

- the user explicitly asked for full plugin removal
- config translation would add extra code and tests for a path the project does not want to preserve
- there is no persisted state or data migration to protect, only configuration updates

Alternative considered:

- support both config shapes for one release
- rejected because it weakens the cleanup and keeps spec and runtime behavior harder to reason about

## Risks / Trade-offs

- [Breaking configuration change] -> Document the new selector-based config in specs and tests, and remove the old plugin in the same change so there is one clear destination.
- [Users may rely on `file` mode semantics] -> Treat this as an intentional scope reduction, document the removal clearly, and cover the remaining `dir` behavior with migrated tests.
- [Broader `boundarycontrol` implementation surface] -> Keep import-control logic and false-sharing logic in separate files with separate tests even though they share one analyzer entry point.
- [Spec names diverge from runtime plugin names] -> State explicitly in the specs that `import-control` and `no-false-sharing` are capabilities implemented by `boundarycontrol`, not standalone analyzers.
- [Package-graph analysis adds cost to `boundarycontrol`] -> Run the false-sharing pass only when shared selectors are configured and keep the existing once-per-run aggregation strategy.

## Migration Plan

1. Create new spec files for `boundarycontrol`, `import-control`, and `no-false-sharing`, and retire the old `architecture-control` and `false-sharing` capability layout.
2. Extend `boundarycontrol` config types and validation so selector policies can declare `shared` alongside `imports`, with dir-based consumer counting as the only supported false-sharing mode.
3. Move the current false-sharing implementation into boundarycontrol-owned code, preserving the existing package-level threshold and diagnostics while removing file-based consumer grouping.
4. Update `plugin/module.go` to stop registering `nofalsesharing` and to decode only the new `boundarycontrol` configuration shape.
5. Move or rewrite existing `nofalsesharing` tests under `boundarycontrol` coverage, and add spec-oriented tests for the new config validation boundaries.
6. Update docs and examples that refer to `nofalsesharing` or the old capability names.

Rollback strategy:

- before release, revert the change set if downstream config churn is larger than expected
- after release, restoration would require reintroducing the removed plugin, so this change should ship only once tests and migration docs are complete

## Open Questions

- None that block this slice. The remaining larger question is the later symbol-level false-sharing follow-up, which stays intentionally out of scope here.
