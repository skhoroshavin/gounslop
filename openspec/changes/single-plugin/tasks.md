## 1. Plugin Registration

- [x] 1.1 Replace `plugin/module.go`: delete `simplePlugin`, `newSyntaxPlugin`, `newTypesPlugin`, `newBoundarycontrolPlugin`; implement single `register.Plugin("gounslop", newGounslopPlugin)` with `gounslopSettings` struct (flat `Disable` + `Architecture` fields), disable-list validation, conditional boundarycontrol flag setup, and `gounslopPlugin` returning filtered `[]*analysis.Analyzer`
- [x] 1.2 Verify `make lint && make test` passes with the new plugin (will fail until harness and config are updated -- do this after tasks 2 and 3)

## 2. E2E Harness

- [x] 2.1 Update `ruletest.Suite`: remove `Linter` field, add `EnableOnly []string` field, hardcode linter name as `"gounslop"` in `renderConfig`
- [x] 2.2 Add known analyzer name list to the harness and implement `EnableOnly` → `disable` complement logic in `renderConfig`, with validation that `EnableOnly` entries are known names
- [x] 2.3 Implement settings merge: harness-generated `disable` list merged with test-supplied `GivenConfig` settings (test `disable` key overrides harness-generated one)
- [x] 2.4 Reset `EnableOnly` in `SetupTest`

## 3. Self-Linting Config

- [x] 3.1 Update `.golangci.yml`: replace `boundarycontrol` and `readfriendlyorder` linter entries with single `gounslop` entry; move `architecture` into unified settings; add `disable: [nospecialunicode, nounicodeescape]`
- [x] 3.2 Remove the separate `readfriendlyorder` custom linter settings block

## 4. E2E Test Suites

- [x] 4.1 Update `pkg/boundarycontrol/plugin_test.go`: replace `s.Linter = "boundarycontrol"` with `s.EnableOnly = []string{"boundarycontrol"}` in `SetupTest`
- [x] 4.2 Update `pkg/nospecialunicode/plugin_test.go`: replace `s.Linter = "nospecialunicode"` with `s.EnableOnly = []string{"nospecialunicode"}`
- [x] 4.3 Update `pkg/nounicodeescape/plugin_test.go`: replace `s.Linter = "nounicodeescape"` with `s.EnableOnly = []string{"nounicodeescape"}`
- [x] 4.4 Update `pkg/readfriendlyorder/plugin_test.go`: replace `s.Linter = "readfriendlyorder"` with `s.EnableOnly = []string{"readfriendlyorder"}`

## 5. Validation

- [x] 5.1 Run `make lint && make test` -- all tests pass, self-linting passes
- [x] 5.2 Verify `custom-gcl` binary rebuilds cleanly with the single plugin registration
