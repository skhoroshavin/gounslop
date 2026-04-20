## Why

`GivenConfig` accepts `map[string]any`, forcing test authors to write untyped nested maps that mirror the plugin's `gounslopSettings` JSON structure. This is verbose, error-prone (no compile-time checking), and duplicates the settings schema that already exists in `plugin/module.go`. Tests should use the same typed struct the plugin defines, so adding a new setting field only requires updating one type.

## What Changes

- **BREAKING**: Change `GivenConfig` to accept the typed `gounslopSettings` struct (or an identical exported type) instead of `map[string]any`
- Update all existing test call sites to use the typed struct
- Remove or simplify the `mergeSettings` / `copyAnyMap` helpers that currently operate on `map[string]any`

## Capabilities

### New Capabilities

(None)

### Modified Capabilities

- `plugin-e2e-harness`: `GivenConfig` signature changes from `map[string]any` to a typed config struct, updating the requirement that defines the `GivenConfig` method contract

## Impact

- `internal/ruletest/harness.go`: `GivenConfig` signature and internal config rendering logic
- `plugin/module.go`: `gounslopSettings` may need to be exported or a shared config type introduced
- All `*_plugin_test.go` files under `pkg/`: call sites updated from `map[string]any` literals to typed struct literals
- `internal/ruletest` scenario/rendering pipeline that currently serializes `map[string]any` to YAML