## Context

gounslop currently calls `register.Plugin` four times in `plugin/module.go`, producing four independent golangci-lint linters. Each test suite sets `s.Linter` to the individual analyzer name (e.g. `"boundarycontrol"`), and the E2E harness `renderConfig` generates a `.golangci.yml` that enables exactly that linter name under `linters.settings.custom`.

The proposal calls for replacing this with a single `register.Plugin("gounslop", ...)` call that returns all analyzers by default, with opt-out via a `disable` list. This affects `plugin/module.go`, the E2E harness config generation in `internal/ruletest/harness.go`, all four `plugin_test.go` suites, and the repo's own `.golangci.yml`.

## Goals / Non-Goals

**Goals:**
- Single `register.Plugin("gounslop", ...)` call replaces all four current registrations
- All analyzers enabled by default; users opt out via `disable` list
- Per-analyzer settings (currently only `boundarycontrol.architecture`) accessible as flat top-level keys
- `//nolint` granularity preserved per analyzer name
- E2E tests continue to exercise individual analyzers in isolation
- Self-linting config updated to use the single `gounslop` linter

**Non-Goals:**
- Changing any analyzer's detection logic or diagnostics
- Adding new analyzers as part of this change
- Supporting `enable` list (enable-all-by-default with opt-out is the chosen model)
- Backward compatibility shim for the old per-analyzer linter names

## Decisions

### Decision 1: Unified settings schema shape

The plugin constructor receives a single `any` from golangci-lint's `settings` block. The decoded struct:

```go
type gounslopSettings struct {
    Disable      []string                                `json:"disable"`
    Architecture map[string]boundarycontrolPolicySettings `json:"architecture"`
}
```

User-facing YAML:

```yaml
linters:
  enable:
    - gounslop
  settings:
    custom:
      gounslop:
        type: "module"
        settings:
          disable:
            - nospecialunicode
          architecture:
            pkg/*:
              imports: ["internal/*"]
```

**Rationale**: Analyzer-specific settings live as flat top-level keys. `architecture` is already an unambiguous key belonging to `boundarycontrol` -- no need to nest it under an analyzer name. This keeps the settings schema shallow and means the `architecture` key in the unified settings is identical to what users had before in the per-analyzer config. The `disable` list is a top-level array rather than per-analyzer `enabled: bool` flags because the common case is "enable everything" and the uncommon case is "turn off one or two."

If a future analyzer needs its own settings key, it gets added at the same top level. Keys just need to be distinct across analyzers.

**Alternative considered**: Nesting under analyzer names (`boundarycontrol: {architecture: ...}`). Rejected because it adds an unnecessary nesting level, and `architecture` is already unambiguous. Also rejected: a `rules` map with `enabled` and `settings` sub-keys per analyzer (like revive) -- too much nesting for a plugin where only one analyzer currently has settings.

### Decision 2: Validation of disable list entries

The constructor validates that every entry in `disable` matches a known analyzer name. Unknown names produce a startup error. This prevents silent misconfiguration where a typo means an analyzer stays enabled when the user thought they disabled it.

**Rationale**: Fail-fast on bad config is consistent with how `boundarycontrol` already validates its `architecture` map.

### Decision 3: BuildAnalyzers returns only enabled analyzers

The `BuildAnalyzers()` method returns a `[]*analysis.Analyzer` slice containing only non-disabled analyzers. Disabled analyzers are simply omitted from the slice. This means golangci-lint never runs them -- no diagnostics, no `//nolint` interactions, no load overhead for their specific logic.

**Rationale**: golangci-lint's analyzer runner iterates the slice from `BuildAnalyzers`. Omission is the cleanest way to disable. The alternative (returning all analyzers and wrapping `Run` to no-op) would still cause golangci-lint to schedule and invoke them.

### Decision 4: GetLoadMode returns LoadModeTypesInfo unconditionally

Two of four analyzers (`boundarycontrol`, `readfriendlyorder`) require type info. `GetLoadMode` returns a single value for all analyzers in the plugin. Returning `LoadModeTypesInfo` always is the only correct choice -- returning `LoadModeSyntax` when type-aware analyzers are enabled would break them.

Even if a user disables both type-aware analyzers, the overhead of loading type info for syntax-only analyzers is negligible (golangci-lint loads types once per package and shares across all linters in the run).

**Alternative considered**: Dynamically choosing load mode based on which analyzers are enabled. Rejected because the performance difference is immaterial and the added complexity (tracking which analyzers need types) isn't worth it for a four-analyzer plugin.

### Decision 5: E2E harness changes -- remove Linter field, add EnableOnly

The harness `ruletest.Suite` currently has a `Linter` field that each test suite sets to its analyzer name. With the single plugin:

- `Linter` is removed as a public field; the harness hardcodes `"gounslop"` internally
- A new `EnableOnly` field (`[]string`) is added: when set, the harness generates settings with `disable` containing all known analyzer names except those in `EnableOnly`
- `GivenConfig` continues to work for per-analyzer settings (e.g. `architecture`) and is merged into the generated gounslop settings

The harness `renderConfig` function changes from:

```yaml
linters:
  enable: [boundarycontrol]
  settings:
    custom:
      boundarycontrol:
        type: module
        settings: {architecture: ...}
```

to:

```yaml
linters:
  enable: [gounslop]
  settings:
    custom:
      gounslop:
        type: module
        settings:
          disable: [nospecialunicode, nounicodeescape, readfriendlyorder]
          architecture: ...
```

**Rationale**: Tests need to isolate individual analyzers so that diagnostics from other analyzers don't interfere. `EnableOnly` makes this explicit. Hardcoding the linter name eliminates a field that would be `"gounslop"` in every single test suite.

The harness needs to know the full analyzer name list to compute the disable complement. This list is defined as a constant slice in the harness (or in `plugin/`). This is a small maintenance cost: when a new analyzer is added, the list must be updated. But this is desirable -- it forces the test infrastructure to be aware of new analyzers.

**Alternative considered**: Keeping `Linter` as a public field set to `"gounslop"` everywhere. Rejected because it would be pure boilerplate -- every suite would set it to the same value.

### Decision 6: GivenConfig merging strategy

Currently `GivenConfig` sets arbitrary `map[string]any` settings that go directly into the linter's settings block. With the unified plugin, test-supplied settings from `GivenConfig` need to merge with harness-generated settings (the `disable` list from `EnableOnly`).

The merge strategy: the harness builds the top-level settings map with the `disable` key, then copies all keys from the test's `GivenConfig` map into it. If the test provides its own `disable` key, it overrides the harness-generated one (test has full control).

For `boundarycontrol` tests, `GivenConfig` currently passes `{"architecture": {...}}`. Because `architecture` is a flat top-level key in the unified settings schema (not nested under an analyzer name), these calls remain unchanged.

**Rationale**: Minimal harness change. The flat settings schema means boundarycontrol tests don't need any `GivenConfig` updates at all -- the `architecture` key they already pass is the same key the unified plugin expects.

### Decision 7: simplePlugin struct is replaced, not extended

The current `simplePlugin` struct holds a single `*analysis.Analyzer` and a load mode. The new plugin struct holds a slice:

```go
type gounslopPlugin struct {
    analyzers []*analysis.Analyzer
}

func (p *gounslopPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
    return p.analyzers, nil
}

func (p *gounslopPlugin) GetLoadMode() string {
    return register.LoadModeTypesInfo
}
```

`newSyntaxPlugin`, `newTypesPlugin`, and `simplePlugin` are all deleted.

### Decision 8: Self-linting config migration

The repo's `.golangci.yml` changes from enabling `boundarycontrol` and `readfriendlyorder` as separate linters to enabling `gounslop` once with `nospecialunicode` and `nounicodeescape` in the disable list (they flag their own test data).

```yaml
linters:
  enable:
    - gounslop
    # ... other standard linters ...
  settings:
    custom:
      gounslop:
        type: "module"
        settings:
          disable:
            - nospecialunicode
            - nounicodeescape
          architecture:
            plugin:
              imports: ["pkg/*"]
            pkg/*:
              imports: ["internal/*"]
```

## Risks / Trade-offs

**[Load mode overhead for syntax-only configs]** If a user enables only `nospecialunicode` (disabling all type-aware analyzers), they pay for type-checking they don't need. **Mitigation**: This is an unlikely configuration and the overhead is small. Acceptable trade-off for simpler code.

**[Breaking change for existing users]** Any project currently referencing `boundarycontrol` or `readfriendlyorder` as linter names in `.golangci.yml` must migrate. **Mitigation**: Document the migration in release notes. The migration is mechanical: replace N linter entries with one `gounslop` entry, move `architecture` up one level into the unified settings.

**[Analyzer name list maintenance]** The E2E harness needs to know all analyzer names to compute the disable complement for `EnableOnly`. Adding a new analyzer requires updating this list. **Mitigation**: This is a feature, not a bug -- it forces explicit registration of new analyzers in the test infrastructure. A compile-time or test-time check can verify the list matches what `BuildAnalyzers` returns.

**[Flat settings key collisions]** Future analyzers with settings must use distinct top-level keys. **Mitigation**: Analyzer settings keys are chosen at design time. If a collision ever arises, the key can be prefixed with the analyzer name. For now, `architecture` and `disable` are the only keys and are unambiguous.
