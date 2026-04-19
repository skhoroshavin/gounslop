## 1. Extend Boundarycontrol Configuration

- [x] 1.1 Add `exports` support to `pkg/boundarycontrol` policy/config types, normalization, and compiled selector policy state.
- [x] 1.2 Update `plugin/module.go` settings decoding so `boundarycontrol` accepts selector-level `exports` lists and returns actionable errors for invalid settings shapes.
- [x] 1.3 Compile and validate `exports` regex patterns during `boundarycontrol` config validation, including deterministic full-name matching semantics.

## 2. Implement Export-Contract Enforcement

- [x] 2.1 Resolve the owning selector for the current package and skip export-control evaluation when that selector has no compiled `exports` patterns.
- [x] 2.2 Collect exported top-level package-scope declarations from the current package and exclude unexported declarations and exported methods.
- [x] 2.3 Report one diagnostic per exported declaration whose name matches none of the owning selector's `exports` patterns.

## 3. Rebuild Coverage For Export Contracts

- [x] 3.1 Add `pkg/boundarycontrol` tests for selector-level `exports` parsing, invalid regex rejection, and wrong-type configuration failures.
- [x] 3.2 Add analyzer tests that cover matching top-level exports, violating exports, ignored unexported declarations, and exported methods being excluded.
- [x] 3.3 Add plugin or E2E coverage that exercises `boundarycontrol` export contracts through real configuration, including at least one configuration-error scenario.

## 4. Align Specs And Validate The Change

- [x] 4.1 Update any implementation-facing docs or examples that describe `boundarycontrol` architecture policy without the new `exports` capability.
- [x] 4.2 Run the most targeted `boundarycontrol` tests first, then run `make lint && make test` to validate the completed export-contract change end to end.
