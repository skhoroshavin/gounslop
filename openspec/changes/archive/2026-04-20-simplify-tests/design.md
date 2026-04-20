## Context

The test harness's `GivenConfig` currently accepts `map[string]any`, which tests populate with untyped nested maps that mirror the plugin's `gounslopSettings` JSON schema. The plugin already defines a typed struct (`gounslopSettings` in `plugin/module.go`) with JSON tags matching the golangci-lint settings format. The harness serializes test-supplied settings to YAML for `.golangci.yml` generation.

The core constraint: `internal/ruletest` cannot import `plugin` (the `plugin` package imports `pkg/*` analyzers; a reverse dependency would be circular). Similarly, `plugin` shouldn't import `internal/ruletest`. So the shared settings type must live in a third location both can reach.

## Goals / Non-Goals

**Goals:**
- Replace `GivenConfig(map[string]any)` with `GivenConfig(GounslopSettings)` using a typed struct
- Export the settings type so test files in `pkg/*` can use it directly
- Keep the YAML rendering path working (settings → `.golangci.yml` custom linter settings)
- Merge `EnableOnly`-derived `disable` list with test-supplied `Disable` field

**Non-Goals:**
- Changing the plugin's internal `gounslopSettings` unexported type (it can stay as-is with a conversion)
- Adding validation to `GivenConfig` (the plugin already validates on load)
- Supporting config types for hypothetical future plugins

## Decisions

### Decision 1: Location of the exported settings type

Create `GounslopSettings` and `PolicySettings` as exported types in `internal/ruletest/settings.go`.

**Why not `plugin/`?** `internal/ruletest` cannot import `plugin` — that would create a cycle since `plugin` imports `pkg/*` and tests in `pkg/*` import `ruletest`. The `plugin` package is also an external-facing entrypoint; test-only config types don't belong there.

**Why not a new shared package?** A separate package (e.g., `internal/config/`) would work but adds a package for two structs used only by `ruletest` and `plugin`. The `ruletest` package already owns the test config rendering pipeline, so it's the natural home.

**Why not move `gounslopSettings` out of `plugin/`?** `plugin/module.go` uses `gounslopSettings` with `register.DecodeSettings[gounslopSettings]` and the `toConfig()` method. Moving it would require `plugin` to import the new location. This works but introduces an `internal/` dependency in the public `plugin` package. Keeping a private alias in `plugin` that converts from the exported `ruletest.GounslopSettings` is cleaner.

**Chosen approach**: Define exported types in `internal/ruletest/settings.go`. In `plugin/module.go`, replace `gounslopSettings` with a private wrapper that converts from `ruletest.GounslopSettings` for `toConfig()`, or simply use `ruletest.GounslopSettings` directly with `register.DecodeSettings`.

### Decision 2: YAML serialization strategy

Change `renderConfig` to serialize the typed `GounslopSettings` struct directly via `yaml.Marshal` using `yaml` struct tags, instead of converting to `map[string]any` first.

**Why not round-trip through `map[string]any`?** The current path is: test provides `map[string]any` → `mergeSettings` produces merged `map[string]any` → `renderConfig` puts it in a `customLinter.Settings map[string]any` → `yaml.Marshal`. With a typed struct, we can marshal directly, eliminating `mergeSettings`, `copyAnyMap`, and the intermediate `map[string]any` plumbing.

**Chosen approach**: `GounslopSettings` gets `yaml` struct tags (identical to the `json` tags it already has in `plugin/module.go`). The harness builds a `GounslopSettings` value: if `EnableOnly` is set, it computes `Disable` as the complement; then it overlays any non-zero fields from the test-supplied `GounslopSettings`. The `renderConfig` function uses the typed struct directly in its YAML config struct, replacing `map[string]any` with `GounslopSettings`.

### Decision 3: Merge strategy for EnableOnly + GivenConfig

Replace `mergeSettings` with struct-level merge: the harness computes a base `GounslopSettings{Disable: disableComplement(enableOnly)}`, then overlays the test-supplied settings on top. Since `Disable` from the test takes precedence (test has full control), the overlay simply replaces the `Disable` field if the test supplies one. `Architecture` is always taken from the test-supplied struct (zero value means no architecture config).

**Why not method merging?** A `Merge` method would couple the settings type to harness internals. The overlay is trivial (two fields: `Disable` and `Architecture`), so inline logic in `renderConfig` is sufficient.

### Decision 4: PolicySettings type sharing

Export `PolicySettings` alongside `GounslopSettings` in `internal/ruletest/settings.go`. This replaces the unexported `boundarycontrolPolicySettings` in `plugin/module.go`. The `plugin` package will reference `ruletest.PolicySettings` and convert it to `boundarycontrol.Policy` via the existing `toConfig()` logic.

## Risks / Trade-offs

- **`plugin` importing `internal/ruletest`?** → No. `plugin/module.go` will reference the settings types. Since `internal/ruletest` is under `internal/`, only code within the module can import it. `plugin/` is in the same module, so this is allowed by Go's access rules. However, it creates a conceptual coupling between the public plugin package and the test harness internals. **Mitigation**: The settings types are pure data with no harness logic — they're a shared schema definition, not a harness dependency. If this coupling becomes problematic, the types can move to a separate `internal/config/` package later with no test changes.

- **JSON vs YAML tag divergence** → The settings types need both `json` tags (for `register.DecodeSettings`) and `yaml` tags (for harness config rendering). **Mitigation**: Use both tag sets on the same struct fields (`json:"..." yaml:"..."`). Since golangci-lint passes settings as JSON-decoded `any` values and the harness writes YAML, both tag sets must match the same key names. Currently they already match (the JSON keys in `plugin/module.go` are the same strings that appear in the YAML output).

- **Breaking change for external test authors** → `GivenConfig` signature changes. **Mitigation**: This is an internal package (`internal/ruletest`); no external consumers exist.
