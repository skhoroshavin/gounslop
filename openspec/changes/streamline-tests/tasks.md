## 1. Replace the ruletest harness API

- [ ] 1.1 Refactor `internal/ruletest/harness.go` to introduce `ruletest.Suite` with exported suite-level configuration (`Linter`, `ModulePath`, `GoVersion`) and per-test state for files, settings, temporary workspace, and last execution result
- [ ] 1.2 Implement `SetupTest()` on `ruletest.Suite` to clear per-test state before each suite test method runs
- [ ] 1.3 Implement `GivenConfig(settings map[string]any)` and `GivenFile(path string, lines ...string)` on `ruletest.Suite`
- [ ] 1.4 Implement `LintFile(path string)` and `LintCode(lines ...string)` on `ruletest.Suite`, including deterministic generated filenames for `LintCode`
- [ ] 1.5 Implement `FixFile(path string)` and `FixCode(lines ...string)` on `ruletest.Suite`, capturing fixed file contents for later assertions

## 2. Add suite assertions and remove the old API

- [ ] 2.1 Implement `ShouldPass()`, `ShouldFailWith(fragments ...string)`, and `ShouldProduce(lines ...string)` on `ruletest.Suite`
- [ ] 2.2 Make suite assertions fail clearly when called before any lint or fix execution
- [ ] 2.3 Remove `Scenario`, `Expectation`, `Run`, `Execute`, `ExecuteFix`, and `AssertResult` from `internal/ruletest` once the new suite methods cover their behavior
- [ ] 2.4 Keep normalized output, exit-code handling, config rendering, workspace creation, and fixed-file reading behavior intact while moving them behind the suite API

## 3. Migrate analyzer test suites

- [ ] 3.1 Update `pkg/nodeepimports/plugin_test.go` to embed `ruletest.Suite`, configure `Linter`, and replace scenario structs with `GivenFile`/`LintFile`/`LintCode` and suite assertions
- [ ] 3.2 Update `pkg/nofalsesharing/plugin_test.go` to embed `ruletest.Suite`, configure `Linter`, and replace scenario structs with the new suite API
- [ ] 3.3 Update `pkg/nounicodeescape/plugin_test.go` to migrate both lint and fix cases to `LintCode`/`FixCode` plus suite assertions
- [ ] 3.4 Update `pkg/nospecialunicode/plugin_test.go` to migrate both lint and fix cases to `LintCode`/`FixCode` plus suite assertions
- [ ] 3.5 Update `pkg/readfriendlyorder/plugin_test.go` to migrate multi-file, lint, and fix scenarios to the new suite API

## 4. Validate behavior end to end

- [ ] 4.1 Run targeted Go tests for the migrated analyzer packages and fix any harness regressions
- [ ] 4.2 Run `make lint` and `make test` to verify the replacement harness and all migrated suites pass end to end
