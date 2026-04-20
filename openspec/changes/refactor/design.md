## Context

The repository currently organizes analyzers as separate packages (`pkg/boundarycontrol/`, `pkg/nospecialunicode/`, `pkg/nounicodeescape/`, `pkg/readfriendlyorder/`) and registers each as a distinct named analyzer with golangci-lint. Tests live alongside implementation in each package (`pkg/*/plugin_test.go`). The `ruletest.Suite` provides an `EnableOnly []string` mechanism to run tests with a subset of analyzers enabled, but this flexibility has allowed incompatibilities between analyzers to go undetected.

This refactor addresses structural debt: tests should live in a dedicated directory, `EnableOnly` should be removed so all E2E tests run with every rule enabled, and the `pkg/` layout should group code by functional concern rather than by analyzer name.

## Goals / Non-Goals

**Goals:**
- Move all E2E tests into a top-level `tests/` directory, separated from implementation
- Remove the `EnableOnly` mechanism so E2E tests always run with all analyzers enabled
- Split `pkg/boundarycontrol/` into functional groups: `importcontrol/`, `exportcontrol/`, `nofalsesharing/`
- Keep `nospecialunicode/` and `nounicodeescape/` as separate packages
- Reorganize `pkg/readfriendlyorder/` as a single functional rule package
- Extract common analyzer infrastructure into `pkg/analyzer/`
- Return multiple analyzers from the unified plugin, each with its own name for `# nolint` granularity
- Inject caches from the root package rather than using package-level singletons
- Use new analyzer names in the user-facing `disable` config list
- Eliminate all unit tests; all testing goes through the unified E2E framework

**Non-Goals:**
- Do not change the external behavior of any individual rule
- Do not change the user-facing settings shape (`disable` list and `architecture` map)
- Do not introduce a new plugin name; `gounslop` remains the single registered plugin
- Do not provide migration tools for user code referencing old package names

## Decisions

### Decision: Package hierarchy by dependency layer

**Choice:**
- `pkg/analyzer/` — common analyzer infrastructure. Holds module-context discovery, architecture config compilation, selector parsing, and generic AST fixer utilities. Does not import any other package under `pkg/`.
- `pkg/importcontrol/` — import boundary checking. Imports `pkg/analyzer` for module context, compiled config, and selector matching.
- `pkg/exportcontrol/` — export contract checking. Imports `pkg/analyzer` for the same shared infrastructure.
- `pkg/nofalsesharing/` — false-sharing detection. Imports `pkg/analyzer` for module context and compiled config.
- `pkg/readfriendlyorder/` — code ordering rules (top-level, method, init, test ordering). Imports `pkg/analyzer` for generic fixer utilities.
- `pkg/nospecialunicode/` — special unicode character detection. Self-contained; does not import `pkg/analyzer`.
- `pkg/nounicodeescape/` — unicode escape sequence detection. Self-contained; does not import `pkg/analyzer`.
- `pkg/gounslop/` — root package. Defines `Config`, the plugin constructor, and `BuildAnalyzers`. Imports anything from `pkg/`. Initializes caches and injects them into analyzer closures.
- `plugin/` — golangci-lint plugin entrypoint. Imports only `pkg/gounslop`.
- `tests/` — flat directory with E2E test files: `tests/importcontrol_test.go`, `tests/exportcontrol_test.go`, `tests/nofalsesharing_test.go`, `tests/readfriendlyorder_test.go`, `tests/nospecialunicode_test.go`, `tests/nounicodeescape_test.go`. Imports only `pkg/gounslop` and `tests/rule`.
- `tests/rule/` — reusable E2E harness (formerly `internal/ruletest/`). Imports only `pkg/gounslop`.

**Rationale:** The layered structure enforces dependency direction. `pkg/analyzer` contains only generic infrastructure; concrete rules live in their own packages and import only the shared base; `pkg/gounslop` sits at the top to wire everything together. The split of `boundarycontrol` into three functional packages removes the monolithic package while avoiding code duplication through the shared `pkg/analyzer` layer.

### Decision: Multiple analyzers returned from plugin with injected caches

**Choice:** The plugin returns multiple `*analysis.Analyzer` values, one per rule, each preserving its own name (`importcontrol`, `exportcontrol`, `nofalsesharing`, `readfriendlyorder`, `nospecialunicode`, `nounicodeescape`). Caches are instantiated in `pkg/gounslop/` and injected via closures. No package-level `sync.Map` singletons remain.

**Self-linting note:** `nospecialunicode` and `nounicodeescape` remain disabled in `.golangci.yml` for self-linting because they flag their own test data. The analyzer names in `disable` do not change.

**Example wiring in `pkg/gounslop/`:**
```go
func BuildAnalyzers(cfg Config) ([]*analysis.Analyzer, error) {
    modCache := analyzer.NewModuleContextCache()
    fsCache := nofalsesharing.NewCache()

    all := []*analysis.Analyzer{
        {
            Name: "importcontrol",
            Run: func(pass *analysis.Pass) (any, error) {
                return importcontrol.Run(pass, modCache, cfg)
            },
        },
        {
            Name: "exportcontrol",
            Run: func(pass *analysis.Pass) (any, error) {
                return exportcontrol.Run(pass, modCache, cfg)
            },
        },
        {
            Name: "nofalsesharing",
            Run: func(pass *analysis.Pass) (any, error) {
                return nofalsesharing.Run(pass, modCache, fsCache, cfg)
            },
        },
        // ... etc
    }
    return filterByDisable(all, cfg.Disable), nil
}
```

**Rationale:** Multiple analyzers preserve `# nolint:<rule>` granularity, which is valuable to users. Cache injection eliminates singletons, making the code testable and race-safe. The `go/analysis` framework shares `inspect.Analyzer` results across all analyzers, so the AST is not re-walked per analyzer.

### Decision: `pkg/analyzer/` absorbs boundarycontrol shared infrastructure

**Choice:** The following components from current `pkg/boundarycontrol/` move into `pkg/analyzer/`:
- `module_context.go` — module discovery, `go.mod` parsing, nested-module scanning, import-path classification (`ModuleContextCache`)
- `analyzer.go` config layer — `Config`, `Policy`, compiled config types, selector parsing (`parseKeySelector`, `parseImportSelector`, `resolveOwner`, `matchesImportSelector`, etc.)
- Generic fixer utilities from `pkg/readfriendlyorder/fixer.go` — `declRange`, `readFileSource`, `buildSwapFix`, `buildReorderFix`, `buildMoveFix`

**What stays in concrete rule packages:**
- `importcontrol/` — AST traversal for import specs, deep-import logic, diagnostic reporting
- `exportcontrol/` — exported symbol enumeration, regex matching, diagnostic reporting
- `nofalsesharing/` — symbol consumer counting, cross-package reference analysis, diagnostic reporting
- `readfriendlyorder/` — ordering rules (init, top-level, method, test), concrete fix computation
- `nospecialunicode/` — all existing code (self-contained)
- `nounicodeescape/` — all existing code (self-contained)

**Rationale:** The shared infrastructure is substantial (~660 lines) and used by three related rules. Moving it to `pkg/analyzer/` avoids duplication and gives that package meaningful content. The generic fixer utilities also belong there since they could theoretically be reused by any rule that provides `SuggestedFixes`.

### Decision: Cache injection replaces singletons

**Choice:** All caches become struct types with constructors in their respective packages:
- `pkg/analyzer/` provides `ModuleContextCache` with `NewModuleContextCache()`
- `pkg/nofalsesharing/` provides `Cache` with `NewCache()`

`pkg/gounslop/` creates instances and passes them into analyzer closures. No `var` package-level caches remain.

**Rationale:** Package-level mutable state (singleton `sync.Map`) is an anti-pattern. Injected caches are testable, allow per-plugin-instance isolation, and eliminate hidden global dependencies.

### Decision: Dashes in internal flag names

**Choice:** Analyzer names match their package names exactly (`importcontrol`, `exportcontrol`, `nofalsesharing`, `readfriendlyorder`, `nospecialunicode`, `nounicodeescape`). These are used as `*analysis.Analyzer.Name` values and in the user-facing `disable` config list.

**Rationale:** Using package names as analyzer names eliminates ambiguity. One name refers to one concept everywhere: package, analyzer name, and config key.

### Decision: Fixer organization

**Choice:** Generic fixer utilities (`declRange`, `readFileSource`, `buildSwapFix`, `buildReorderFix`, `buildMoveFix`) move to `pkg/analyzer/`. Rule-specific fix computation (e.g., `computeTopLevelReorderFix`) stays in `pkg/readfriendlyorder/`.

**Rationale:** The generic utilities are AST-manipulation helpers with no domain knowledge. Keeping them in a shared package makes them available to future rules. The concrete fix logic stays with the rule it serves.

### Decision: Tests live in flat `tests/` directory

**Choice:**
- `tests/importcontrol_test.go`, `tests/exportcontrol_test.go`, `tests/nofalsesharing_test.go`, `tests/readfriendlyorder_test.go`, `tests/nospecialunicode_test.go`, `tests/nounicodeescape_test.go`
- `tests/rule/` contains the reusable E2E harness

**Rationale:** Separating tests from implementation keeps `pkg/` clean and focused. A flat `tests/` directory avoids unnecessary package proliferation while still grouping E2E coverage by analyzer. `go test ./...` from the repo root automatically discovers and runs all test packages.

### Decision: No unit tests anywhere

**Choice:** All test coverage is provided through the E2E framework in `tests/`. No `_test.go` files remain in `pkg/`. The existing `analysistest`-style or direct unit test patterns are eliminated.

**Rationale:** A single testing approach reduces cognitive overhead. E2E tests exercise the actual plugin binary and config, catching issues that unit tests miss (config parsing, binary wiring, cross-analyzer interactions).

### Decision: Remove `EnableOnly` from harness entirely

**Choice:** The `EnableOnly` field and associated `disableComplement`/`validateEnableOnly` logic are removed from the harness. All E2E tests run with all analyzers enabled. The harness always generates a config with no `disable` list unless the test explicitly calls `GivenConfig` with one.

**Rationale:** Running all analyzers together catches cross-analyzer incompatibilities. The `EnableOnly` mechanism allowed tests to hide these problems. Removing it forces test authors to write test data that is valid under all rules.

### Decision: Rewrite test data to avoid cross-analyzer conflicts

**Choice:** Test cases in `tests/nounicodeescape_test.go` that use literal Unicode characters must use characters **not** on the `nospecialunicode` banned list (e.g., `é`, `ñ`, `中` instead of `—`). All test data across suites is audited to ensure no accidental cross-analyzer triggers.

**Rationale:** This enables the "all rules always on" testing strategy without `EnableOnly` workarounds. It is a one-time data fix (~3-4 test cases).

## Risks / Trade-offs

[Risk] Package moves break internal import paths
→ Mitigation: All internal imports are updated in one refactor. `go mod tidy` validates the module graph.

[Risk] Multiple analyzers cause repeated AST walks
→ Mitigation: The `go/analysis` framework deduplicates analyzer dependencies. All analyzers require `inspect.Analyzer`; the inspection result is computed once and shared.

[Risk] Cache injection adds parameter plumbing
→ Mitigation: The closures in `pkg/gounslop/` hide the plumbing from rule implementations. Rule packages see only `Run(pass, cache, cfg)` signatures.

[Risk] Tests without `EnableOnly` may produce noisy failures
→ Mitigation: The initial migration includes a pass over all test data to eliminate cross-analyzer conflicts. After that, any new conflict is a legitimate bug.

## Migration Plan

1. Create `pkg/analyzer/` with:
   - Module context discovery (from `pkg/boundarycontrol/module_context.go`)
   - Config compilation and selector parsing (from `pkg/boundarycontrol/analyzer.go`)
   - Generic fixer utilities (from `pkg/readfriendlyorder/fixer.go`)
   - `ModuleContextCache` struct with constructor

2. Create `pkg/importcontrol/`, `pkg/exportcontrol/`, `pkg/nofalsesharing/`:
   - Move rule-specific logic from `pkg/boundarycontrol/`
   - Import `pkg/analyzer` for shared infrastructure
   - `pkg/nofalsesharing/` provides `Cache` struct with constructor

3. Refactor `pkg/readfriendlyorder/`:
   - Import generic fixers from `pkg/analyzer/`
   - Keep rule-specific ordering logic and concrete fix computation

4. Create `tests/rule/` from `internal/ruletest/`:
   - Remove `EnableOnly` field and associated logic
   - Update imports to `pkg/gounslop`

5. Move and adapt tests:
   - Move `pkg/boundarycontrol/plugin_test.go` and `false_sharing_plugin_test.go` content into `tests/importcontrol_test.go`, `tests/exportcontrol_test.go`, `tests/nofalsesharing_test.go`
   - Move `pkg/nospecialunicode/plugin_test.go` → `tests/nospecialunicode_test.go`
   - Move `pkg/nounicodeescape/plugin_test.go` → `tests/nounicodeescape_test.go`
   - Move `pkg/readfriendlyorder/plugin_test.go` → `tests/readfriendlyorder_test.go`
   - Rewrite `nounicodeescape` test data to avoid banned characters
   - Ensure all tests run with all analyzers enabled

6. Refactor `pkg/gounslop/`:
   - Keep `Config` and `PolicyConfig` structs
   - Implement `BuildAnalyzers` returning multiple analyzers with injected caches
   - Implement `disable` list filtering using new analyzer names

7. Update `plugin/` to import only `pkg/gounslop`

8. Remove old packages: `pkg/boundarycontrol/`, `pkg/*/plugin_test.go`, `internal/ruletest/`

9. Update `Makefile`, `AGENTS.md`, and `README.md` to document:
    - `make test` runs `go test ./...` (covers `tests/`)
    - No unit tests in `pkg/`
    - All testing through E2E framework
    - New package layout and analyzer names
    - Self-linting (`make lint`) disables `nospecialunicode` and `nounicodeescape` to avoid flagging their own test data

10. Run full test suite: `make lint && make test`

11. Update `openspec/specs/unified-plugin/spec.md` and `openspec/specs/plugin-e2e-harness/spec.md` to reflect:
    - Multiple analyzers returned from plugin
    - `EnableOnly` removed from harness
    - Tests in `tests/` directory

## Open Questions

1. Should `readfriendlyorder` be split further (e.g., separate packages for top-level vs. method ordering)? Not yet — will think about it in the future.
