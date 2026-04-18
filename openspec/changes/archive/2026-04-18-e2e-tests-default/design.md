## Context

The project has 5 analyzers under `pkg/`: `nodeepimports`, `nofalsesharing`, `nospecialunicode`, `nounicodeescape`, and `readfriendlyorder`. Only `nofalsesharing` has E2E tests; the other four use Go's `analysistest` framework with `testdata/` directories and `// want` annotations. The E2E harness in `internal/plugine2e/` is well-established and provides scenario-based testing that runs through golangci-lint's full pipeline.

Current test approaches:
- **analysistest** (4 analyzers): `analyzer_test.go` + `testdata/src/` directories with `// want` annotations and `.golden` files
- **E2E** (nofalsesharing only): `plugin_e2e_test.go` with `//go:build e2e` tag, using `plugine2e.Scenario` / `plugine2e.Run`

The Makefile has a separate `e2e` target that builds `custom-gcl` and runs tests with `-tags=e2e`.

## Goals / Non-Goals

**Goals:**
- E2E tests become the default and only test approach for all analyzers
- `make test` runs E2E tests (after building `custom-gcl` if needed)
- E2E tests cover the majority of cases currently covered by `analysistest`
- Test files live alongside analyzer code without build tags

**Non-Goals:**
- 100% test coverage parity with old tests — some edge cases (especially suggested-fix golden comparisons) may be deferred if the fix output is hard to assert in E2E
- Refactoring the `internal/plugine2e` harness beyond minor adjustments
- Changing analyzer behavior or configuration

## Decisions

### 1. E2E test file naming and location

Each analyzer package gets a single `plugin_test.go` using `plugine2e.Scenario` / `plugine2e.Run`, as table-driven tests. The `_test.go` suffix ensures Go's test runner picks them up, and the separate package import (`package nofalsesharing_test`) keeps them as integration tests.

**Alternative considered**: Keep tests inside the package (`package nofalsesharing`). Rejected because E2E tests exercise the public plugin interface, not internals.

### 2. Inline test data instead of testdata directories

E2E scenarios embed file contents directly as string map values in the test code, matching the existing `nofalsesharing` pattern. No `testdata/` directories needed. This makes test intent immediately visible and avoids the indirection of separate files.

### 3. Build `custom-gcl` as test prerequisite

The `Makefile` `test` target will depend on `custom-gcl` (same as current `e2e` target), ensuring the binary exists before `go test ./...` runs. The separate `e2e` target is removed.

**Alternative considered**: Use `go test -exec` wrapper. Rejected — unnecessary complexity for a build dependency.

### 4. Suggested-fix assertions

For analyzers that produce suggested fixes (`nospecialunicode`, `nounicodeescape`, `readfriendlyorder`), E2E tests will assert the diagnostic output (exit code 1 + output contains specific messages). Asserting the actual diff content of suggested fixes is not directly feasible via the CLI output, so fix correctness is verified indirectly through diagnostic messages. If needed, this can be supplemented later.

### 5. Phased approach

Migrate one analyzer at a time, verifying each passes before moving on. Order: `nodeepimports` (simplest) → `nospecialunicode` → `nounicodeescape` → `readfriendlyorder` (most complex).

## Risks / Trade-offs

- **[Losing golden-file fix assertions]** E2E tests cannot easily assert suggested-fix diffs like `analysistest.RunWithSuggestedFixes` does. → Accept this limitation for now; verify fix correctness via diagnostic messages. Can add fix-level testing separately if needed.
- **[Test speed]** E2E tests invoke the `custom-gcl` binary per scenario, which is slower than in-process `analysistest`. → Acceptable because the test suite is small and each run takes <1s.
- **[Build dependency]** `go test ./...` requires `custom-gcl` to exist. Running bare `go test` without Makefile will fail. → Document this clearly. The Makefile handles it.
- **[Test isolation]** Each E2E scenario gets a fresh temp workspace, so isolation is inherent.