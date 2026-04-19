## 1. Reshape Boundarycontrol Configuration

- [x] 1.1 Extend `pkg/boundarycontrol` config types, normalization, and validation so selector policies support `shared: true` alongside `imports`.
- [x] 1.2 Reject removed migrated false-sharing options such as selector-level `mode`, and keep validation errors actionable under `boundarycontrol`.
- [x] 1.3 Update `plugin/module.go` settings decoding so `boundarycontrol` is the only accepted runtime configuration path for architecture-aware and migrated false-sharing behavior.

## 2. Move False-Sharing Runtime Into Boundarycontrol

- [x] 2.1 Move the reusable package-graph false-sharing analysis out of `pkg/nofalsesharing` into `pkg/boundarycontrol`-owned code.
- [x] 2.2 Adapt the migrated false-sharing analysis to discover shared packages from selector ownership with `shared: true` and count consumers by importing package path only.
- [x] 2.3 Integrate the migrated false-sharing pass into `boundarycontrol` execution so it runs only when shared selectors are configured and reports preserved package-level diagnostics.

## 3. Remove Nofalsesharing Surface

- [x] 3.1 Remove `nofalsesharing` plugin registration and legacy settings structs from `plugin/module.go`.
- [x] 3.2 Delete the obsolete `pkg/nofalsesharing` plugin package files once all logic has been absorbed or replaced.
- [x] 3.3 Update any repository references, fixtures, or examples that still mention the removed `nofalsesharing` plugin or its old config shape.

## 4. Rebuild Test Coverage Around The New Shape

- [x] 4.1 Add or update `boundarycontrol` tests for shared selector parsing, selector ownership of shared subtrees, and rejection of removed `mode` settings.
- [x] 4.2 Migrate the existing package-level false-sharing E2E coverage into `boundarycontrol` tests, keeping dir-style consumer counting, `_test.go` exclusion, single-consumer failure, and no-consumer failure cases.
- [x] 4.3 Keep import-control coverage passing under the split spec model and ensure combined import and false-sharing behavior still works through one `boundarycontrol` plugin.

## 5. Align Specs And Project Documentation

- [x] 5.1 Update any implementation-facing docs or comments that still describe `architecture-control`, `false-sharing`, or `file` mode as active behavior.
- [x] 5.2 Confirm the repository examples and docs show the new `boundarycontrol` selector-based shared-package configuration.
- [x] 5.3 Run the targeted Go tests first, then run `make lint && make test` to verify the completed migration end to end.
