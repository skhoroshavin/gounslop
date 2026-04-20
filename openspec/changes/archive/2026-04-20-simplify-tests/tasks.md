## 1. Define exported settings types

- [x] 1.1 Create `pkg/gounslop/config.go` with exported `Config` struct (fields: `Disable []string`, `Architecture map[string]PolicyConfig`) with both `json` and `yaml` struct tags
- [x] 1.2 Add exported `PolicyConfig` struct (fields: `Imports []string`, `Exports []string`, `Shared bool`, `Mode *string`) with both `json` and `yaml` struct tags

## 2. Update harness to use typed settings

- [x] 2.1 Change `Suite.settings` field type from `map[string]any` to `GounslopSettings`
- [x] 2.2 Change `scenarioInput.Settings` field type from `map[string]any` to `GounslopSettings`
- [x] 2.3 Change `GivenConfig` signature from `map[string]any` to `GounslopSettings` and update the method body to store a copy of the struct
- [x] 2.4 Update `renderConfig` to use `GounslopSettings` directly in the YAML config struct instead of `map[string]any`, replacing the `customLinter.Settings` field type
- [x] 2.5 Replace `mergeSettings` with struct-level merge logic: compute `Disable` from `EnableOnly` complement, then overlay test-supplied `GounslopSettings` fields
- [x] 2.6 Remove `copyAnyMap` helper (no longer needed)
- [x] 2.7 Update `SetupTest` to reset settings to zero value instead of `nil`

## 3. Update plugin to use shared settings types

- [x] 3.1 Replace `gounslopSettings` and `boundarycontrolPolicySettings` in `plugin/module.go` with `gounslop.Config` and `gounslop.PolicyConfig`
- [x] 3.2 Update `newGounslopPlugin` to use `register.DecodeSettings[gounslop.Config]`
- [x] 3.3 Update `toConfig` function to accept `gounslop.Config` and convert to `boundarycontrol.Config`
- [x] 3.4 Add import of `github.com/skhoroshavin/gounslop/pkg/gounslop` to `plugin/module.go`

## 4. Update all test call sites

- [x] 4.1 Update `pkg/boundarycontrol/plugin_test.go`: convert all `GivenConfig(map[string]any{...})` calls to `GivenConfig(gounslop.Config{Architecture: map[string]gounslop.PolicyConfig{...}})`
- [x] 4.2 Update `pkg/boundarycontrol/false_sharing_plugin_test.go`: convert all `GivenConfig(map[string]any{...})` calls to typed struct literals
- [x] 4.3 Verify and update any other `*_plugin_test.go` files that call `GivenConfig` (nospecialunicode, nounicodeescape, readfriendlyorder have no `GivenConfig` calls - verified)

## 5. Validate

- [x] 5.1 Run `make lint && make test` and fix any issues
