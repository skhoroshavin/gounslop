## 1. Boundarycontrol Configuration Surface

- [x] 1.1 Replace `boundarycontrol` plugin settings decoding with the new `architecture` map shape and remove support for `module-root` and `selectors`
- [x] 1.2 Normalize the `architecture` map into compiled selector policies that `pkg/boundarycontrol` can validate and evaluate
- [x] 1.3 Update owner-resolution logic so overlapping selectors are resolved by specificity only, without declaration-order tie-breaking

## 2. Module Discovery And Scope Classification

- [x] 2.1 Add nearest-`go.mod` discovery for each analyzed package and parse the owning module path from the `module` directive
- [x] 2.2 Cache discovered module contexts, including nested module paths beneath each owning module directory
- [x] 2.3 Use the discovered module context to classify imports as in-module, external, or owned by a nested module, and fail clearly when no enclosing `go.mod` is found

## 3. Boundarycontrol Enforcement Updates

- [x] 3.1 Switch deep-import evaluation to use the discovered module path instead of configured `module-root`
- [x] 3.2 Enforce `architecture` policy matching for in-module imports while preserving unmatched importers as `imports: []`
- [x] 3.3 Preserve the immediate-child import allowance and exclude external and nested-module imports from boundarycontrol violations

## 4. Remove Nodeepimports Surface Area

- [x] 4.1 Remove `nodeepimports` from plugin registration, settings types, and plugin registration tests
- [x] 4.2 Remove `pkg/nodeepimports` implementation and its plugin E2E tests
- [x] 4.3 Update repository-owned config and docs to use only `boundarycontrol`, including `.golangci.yml`, `README.md`, and related mapping docs

## 5. E2E Coverage And Verification

- [x] 5.1 Update `boundarycontrol` E2E tests to use the `architecture` map and cover allowed imports, undeclared imports, and same-scope deep-import violations
- [x] 5.2 Add multi-module E2E coverage for nearest-`go.mod` discovery, nested-module exclusion, and missing-`go.mod` failure behavior
- [x] 5.3 Update plugin E2E coverage expectations and any harness setup needed to support scenarios with multiple `go.mod` files
- [x] 5.4 Run `make lint && make test` and fix any resulting failures until the full suite passes
