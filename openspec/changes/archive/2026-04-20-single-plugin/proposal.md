## Why

gounslop currently registers each analyzer as a separate golangci-lint module plugin. Users must individually enumerate every analyzer in their `linters.enable` list and declare a separate `linters.settings.custom` block (with `type: "module"`) for each one -- even analyzers with zero configuration. This creates adoption friction: adding gounslop to a project means knowing all analyzer names upfront, and every new analyzer the project ships requires users to update their config to opt in. There is no way to say "enable all gounslop rules."

Consolidating into a single `gounslop` plugin with all analyzers enabled by default solves this. Users add one linter entry, get all rules, and opt out of specific ones if needed. New analyzers become available on upgrade without config changes.

## What Changes

- **BREAKING**: The four current `register.Plugin` calls (`boundarycontrol`, `nospecialunicode`, `nounicodeescape`, `readfriendlyorder`) are replaced by a single `register.Plugin("gounslop", ...)` call
- The single plugin constructor builds all four `*analysis.Analyzer` instances by default, applying per-analyzer settings from a unified settings struct
- A `disable` list in settings allows users to opt out of individual analyzers by name
- Per-analyzer settings (currently only `boundarycontrol.architecture`) move under a nested key in the unified settings struct
- `GetLoadMode()` returns `LoadModeTypesInfo` unconditionally (required by `boundarycontrol` and `readfriendlyorder`; negligible overhead for syntax-only analyzers when type-aware analyzers are co-enabled)
- `//nolint` granularity is preserved: each returned `*analysis.Analyzer` keeps its own `Name` field, so `//nolint:boundarycontrol` and `//nolint:nospecialunicode` continue to work
- Self-linting config (`.golangci.yml`) and E2E test harness config generation (`internal/ruletest`) are updated to use the single `gounslop` linter name
- The E2E harness `ruletest.Suite` field `Linter` always becomes `"gounslop"`; a new field (e.g. `Rule` or `Analyzers`) controls which analyzer(s) are active for a given test suite via settings

## Capabilities

### New Capabilities

- `unified-plugin`: Defines the single-plugin registration model, unified settings schema, analyzer enable/disable behavior, and load-mode strategy

### Modified Capabilities

- `plugin-e2e-harness`: The harness must generate `.golangci.yml` referencing the single `gounslop` linter and route per-analyzer settings through the unified settings schema

## Impact

- **Plugin registration** (`plugin/module.go`): Major rewrite -- single constructor, unified settings struct, multi-analyzer `BuildAnalyzers` return
- **User-facing config** (`.golangci.yml`): **BREAKING** -- existing configs referencing `boundarycontrol`, `readfriendlyorder`, etc. as separate linters must migrate to a single `gounslop` entry
- **E2E harness** (`internal/ruletest/`): Config generation changes to emit the unified linter name and route settings
- **E2E tests** (`pkg/*/plugin_test.go`): Suite setup changes from `s.Linter = "boundarycontrol"` to `s.Linter = "gounslop"` with analyzer-specific enablement
- **Analyzer packages** (`pkg/*/analyzer.go`): No changes -- analyzers remain independent `*analysis.Analyzer` values
- **Dependencies**: No new dependencies
