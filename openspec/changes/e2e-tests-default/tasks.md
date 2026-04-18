## 1. Makefile Update

- [x] 1.1 Make the `test` target depend on `custom-gcl` (so the binary is built before running tests)
- [x] 1.2 Remove the separate `e2e` target from the Makefile

## 2. Migrate nodeepimports to E2E Tests

- [x] 2.1 Create `pkg/nodeepimports/plugin_test.go` with E2E test scenarios covering: one-level deep import passes, deep import flagged, different top-level scope passes, test file import passes
- [x] 2.2 Delete `pkg/nodeepimports/analyzer_test.go`
- [x] 2.3 Delete `pkg/nodeepimports/testdata/` directory
- [x] 2.4 Run `make lint && make test` and verify all nodeepimports E2E tests pass

## 3. Migrate nospecialunicode to E2E Tests

- [x] 3.1 Create `pkg/nospecialunicode/plugin_test.go` with E2E test scenarios covering: ASCII string passes, special Unicode flagged, raw string flagged, multiple banned characters
- [x] 3.2 Delete `pkg/nospecialunicode/analyzer_test.go`
- [x] 3.3 Delete `pkg/nospecialunicode/testdata/` directory
- [x] 3.4 Run `make lint && make test` and verify all nospecialunicode E2E tests pass

## 4. Migrate nounicodeescape to E2E Tests

- [x] 4.1 Create `pkg/nounicodeescape/plugin_test.go` with E2E test scenarios covering: literal Unicode passes, escape flagged, raw string not flagged
- [x] 4.2 Delete `pkg/nounicodeescape/analyzer_test.go`
- [x] 4.3 Delete `pkg/nounicodeescape/testdata/` directory
- [x] 4.4 Run `make lint && make test` and verify all nounicodeescape E2E tests pass

## 5. Migrate readfriendlyorder to E2E Tests

- [x] 5.1 Create `pkg/readfriendlyorder/plugin_test.go` with E2E test scenarios covering: correct order passes, incorrect top-level order flagged, method ordering, init function ordering, TestMain ordering, cyclic dependencies exempt
- [x] 5.2 Delete `pkg/readfriendlyorder/analyzer_test.go`
- [x] 5.3 Delete `pkg/readfriendlyorder/testdata/` directory
- [x] 5.4 Run `make lint && make test` and verify all readfriendlyorder E2E tests pass

## 6. Clean Up nofalsesharing E2E Test

- [x] 6.1 Remove the `//go:build e2e` tag from `pkg/nofalsesharing/plugin_e2e_test.go`
- [x] 6.2 Rename `pkg/nofalsesharing/plugin_e2e_test.go` to `pkg/nofalsesharing/plugin_test.go`
- [x] 6.3 Run `make lint && make test` and verify all nofalsesharing E2E tests pass

## 7. Final Validation

- [x] 7.1 Run `make lint && make test` and verify all tests pass
- [x] 7.2 Verify no `testdata/` directories remain under `pkg/`
- [x] 7.3 Verify no `analysistest` imports remain in any test files under `pkg/`
- [x] 7.4 Verify no `//go:build e2e` tags remain anywhere in the project