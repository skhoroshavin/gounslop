## Why

Currently only `nofalsesharing` has E2E (end-to-end) plugin tests, while the other four analyzers (`nodeepimports`, `nospecialunicode`, `nounicodeescape`, `readfriendlyorder`) rely on `analysistest`-based unit tests. E2E tests are more realistic — they run the linter through golangci-lint's full pipeline — and easier to read and reason about. This change migrates all analyzers to E2E tests as the default and only test approach, removing the old `analysistest`-based tests entirely.

## What Changes

- Add E2E test files for `nodeepimports`, `nospecialunicode`, `nounicodeescape`, and `readfriendlyorder` covering the majority of cases currently tested via `analysistest`
- Remove all `analysistest`-based `*_test.go` files and their `testdata/` directories from each analyzer package
- Remove the `e2e` build tag from all E2E test files — E2E tests become the default tests runnable via `go test ./...`
- Update the `Makefile` to remove the separate `e2e` target and make `test` depend on building `custom-gcl` before running
- **BREAKING**: Old `analysistest`-based test data directories and test files are deleted

## Capabilities

### New Capabilities
_(none)_

### Modified Capabilities
- `plugin-e2e-harness`: Expand from "at least one analyzer" to all five analyzers; remove `e2e` build tag requirement; make E2E tests the default test approach via `make test`

## Impact

- All `pkg/<analyzer>/analyzer_test.go` files removed (replaced by E2E equivalents)
- All `pkg/<analyzer>/testdata/` directories removed (scenarios move inline into E2E test code)
- `pkg/nofalsesharing/plugin_e2e_test.go` loses the `e2e` build tag; likely renamed to `plugin_test.go`
- `Makefile` test target updated to ensure `custom-gcl` is built before tests run; `e2e` target removed
- `internal/plugine2e/` stays as the shared harness (no structural changes expected)