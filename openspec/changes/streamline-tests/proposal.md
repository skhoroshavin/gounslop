## Why

Current analyzer E2E tests require verbose `ruletest.Scenario` structs with repeated boilerplate (linter name, module path, file contents as single strings, manual expectation structs). Each test case repeats the same configuration and assertion pattern, making tests harder to read and write.

## What Changes

- Introduce a fluent test harness that integrates with `testify/suite`, exposing builder-style methods: `GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `ShouldPass`, `ShouldFailWith`
- `GivenFile` accepts a path and variadic lines (joined into file content), replacing inline multi-line string maps
- `LintCode` is a convenience that creates a temp file from variadic lines and lints it, removing the need for `GivenFile` + `LintFile` in single-file tests
- `GivenConfig` sets linter settings once per test, removing per-scenario `Settings` repetition
- `ShouldPass` and `ShouldFailWith` replace manual `Expectation` struct construction with readable assertions
- `internal/ruletest/` — existing package extended with fluent builder API alongside existing `Scenario`/`Run` API

## Capabilities

### New Capabilities

### Modified Capabilities
- `plugin-e2e-harness`: Add fluent builder API (`GivenConfig`, `GivenFile`, `LintFile`, `LintCode`, `ShouldPass`, `ShouldFailWith`) to the existing harness

## Impact

- `internal/ruletest/` — existing package unchanged, new harness wraps it
- `pkg/*/plugin_test.go` — all five analyzer test files will migrate to the new harness
- No production code changes; this is test-only
- No breaking changes to existing `ruletest.Scenario`/`ruletest.Run` API (kept for backward compatibility)