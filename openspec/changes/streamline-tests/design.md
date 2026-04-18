## Context

All five analyzer test suites (`pkg/*/plugin_test.go`) currently use `testify/suite` with a manual pattern: each test method constructs a `ruletest.Scenario` struct, then calls a private `runScenario` or `runFixScenario` helper that delegates to `ruletest.Execute` + `ruletest.AssertResult`. The boilerplate repeated per test case includes: linter name, module path, `Files` map with inline multi-line strings, `Settings` map, and a separate `Expect` struct. Two of the five suites also define a `runFixScenario` helper.

The harness in `internal/ruletest/harness.go` provides `Scenario`, `Expectation`, `Result`, `Run`, `Execute`, `ExecuteFix`, and `AssertResult`. The fluent builder API will replace this entirely — the old `Scenario` struct and its related functions will be removed once all test suites are migrated.

## Goals / Non-Goals

**Goals:**
- Reduce per-test-case boilerplate by providing a fluent builder that embeds into testify suite structs
- Make single-file lint tests expressible in one line of assertion code (`LintCode` + `ShouldPass`/`ShouldFailWith`)
- Make multi-file tests expressible via `GivenFile` + `LintFile` instead of `Files: map[string]string{...}` with raw newline-escaped strings
- Allow linter settings to be configured once per test via `GivenConfig` instead of repeated per scenario
- Replace the existing `ruletest.Scenario`/`Run`/`Execute`/`AssertResult` API entirely

**Non-Goals:**
- Replacing `testify/suite` with a different test framework
- Changing how `custom-gcl` is built or invoked
- Adding `analysistest` or `testdata/` fixture support

## Decisions

### 1. Custom `Suite` type replacing per-analyzer suite boilerplate

Define a `ruletest.Suite` struct that embeds `suite.Suite` and provides all test harness methods directly. Analyzer test suites embed `ruletest.Suite` instead of `suite.Suite`, and configure the linter name at construction.

```go
type Suite struct {
    suite.Suite
    linter    string
    modulePath string
    goVersion string
    // per-test state (reset in SetupTest)
    files    map[string]string
    settings map[string]any
}
```

```go
// In analyzer test file:
type NospecialunicodeE2ESuite struct {
    ruletest.Suite
}

func TestPluginE2E(t *testing.T) {
    s := new(NospecialunicodeE2ESuite)
    s.Linter = "nospecialunicode"
    suite.Run(t, s)
}
```

**Why not a separate `Builder` struct?** A separate builder adds another type to learn and embed. Putting the harness methods directly on `Suite` means `s.GivenConfig(...)`, `s.LintCode(...)`, `s.ShouldPass()` — the `s.` already exists in every test method. No extra indirection, no chaining to manage.

**Why embed `ruletest.Suite` instead of `suite.Suite`?** This replaces the repeated per-analyzer suite boilerplate (private `runScenario`/`runFixScenario` helpers, `modulePath` constants) with shared functionality on the base suite.

### 2. `SetupTest` resets per-test state

`SetupTest()` on `ruletest.Suite` clears `files` and `settings` before each test method runs. Suite-level defaults (`linter`, `modulePath`, `goVersion`) persist across the entire suite.

`GivenConfig` and `GivenFile` accumulate state in the test method. When `LintFile`/`LintCode`/`FixFile`/`FixCode` is called, it materializes the accumulated files and settings into a workspace, runs `custom-gcl`, and stores the result on the suite for assertion. There is exactly one lint or fix call per test method.

**Why `SetupTest` instead of resetting inside lint/fix calls?** testify already calls `SetupTest` before each test method — using this existing hook keeps the reset obvious and debuggable. The lint/fix methods focus on execution, not lifecycle management.

### 3. `GivenFile` accepts variadic lines joined with newlines

`GivenFile(path string, lines ...string)` joins `lines` with `"\n"` and appends `"\n"` to form file content. This replaces the current pattern of `"line1\n\nline2\n"` inline strings.

**Why not accept a single string?** Variadic lines eliminate the `\n` escaping noise that makes current test files hard to read. For cases where the author wants a raw string, they can pass a single argument.

### 4. `LintCode` generates a random filename

`LintCode(lines ...string)` is sugar for `GivenFile(randomName, lines...)` + `LintFile(randomName)`. The random name is a simple counter-based name like `lint0.go`, `lint1.go` to keep output stable and readable.

**Why not use `os.TempDir` randomness?** Counter-based names are deterministic within a test run, making diagnostics easier to read. They also avoid OS-specific path issues in normalized output.

### 5. Suite type and API shape

```go
type Suite struct {
    suite.Suite
    Linter     string       // set by embedding suite before suite.Run
    ModulePath string       // optional, defaults to "example.com/plugin-e2e"
    GoVersion  string       // optional, defaults to "1.25.6"
    files      map[string]string
    settings   map[string]any
}
```

Configuration methods (called during test method, before lint/fix):
- `GivenConfig(settings map[string]any)` — sets per-test linter settings
- `GivenFile(path string, lines ...string)` — adds a file to the workspace

Execution methods (called once per test, store result on Suite):
- `LintFile(path string)` — materializes workspace, runs `custom-gcl` (check only), stores result
- `LintCode(lines ...string)` — convenience: `GivenFile(autoName, lines...)` + `LintFile(autoName)`
- `FixFile(path string)` — materializes workspace, runs `custom-gcl --fix`, stores result
- `FixCode(lines ...string)` — convenience: `GivenFile(autoName, lines...)` + `FixFile(autoName)`

Lifecycle:
- `SetupTest()` — resets `files`, `settings`, and stored result before each test method

Assertion methods (called after execution):
- `ShouldPass()` — asserts most recent result has exit code 0 and empty output
- `ShouldFailWith(fragments ...string)` — asserts most recent result has exit code != 0 and all fragments appear in output
- `ShouldProduce(lines ...string)` — asserts that the linted file's content after fix matches the given lines (joined with `"\n"` + trailing `"\n"`); used after `FixFile`/`FixCode`


### 6. Assertion methods live on `Suite`

`ShouldPass()`, `ShouldFailWith(fragments...)`, and `ShouldProduce(lines...)` are methods on `Suite`. The execution methods (`LintFile`, `LintCode`, `FixFile`, `FixCode`) store their `Result` on the suite, and the assertion methods operate on that stored result.

This keeps the API surface on a single type — `s.LintCode(...)` then `s.ShouldPass()` — with no intermediate `LintResult` type needed.

**Why not a separate `LintResult` type?** Adding a return type just to enforce call order adds complexity. The testify/suite pattern already relies on test method ordering (call lint, then call assert) — `ShouldPass()` before any lint/fix call will simply fail the test with a clear "no result" message, which is sufficient guardrail.

**Why `ShouldProduce` instead of a map of expected fixed files?** The current API uses `Expectation.FixedFiles map[string]string` which is verbose. Since most fix tests assert on a single file, `ShouldProduce(lines...)` asserts that the linted file's content matches the given lines.

## Risks / Trade-offs

- **Fix scenarios not covered in initial API** → Mitigation: `runFixScenario` helper stays unchanged; `LintFix` + `ShouldFixTo` can be added as an extension in a follow-up
- **Builder state leakage between test methods** → Mitigation: `SetupTest()` resets per-test state before each method; suite-level defaults are set once at construction
- **Old API removal** → Mitigation: migrate all five test suites to `ruletest.Suite` first, then remove `Scenario`, `Expectation`, `Run`, `Execute`, `ExecuteFix`, `AssertResult`, and private `runScenario`/`runFixScenario` helpers in the same PR