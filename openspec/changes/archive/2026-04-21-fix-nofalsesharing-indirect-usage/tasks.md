## 1. Core Analyzer Logic

- [x] 1.1 Implement `collectSharedTypeKeys(t types.Type, sharedPackages) []string` recursive type traversal helper in `pkg/nofalsesharing/analyzer.go`
- [x] 1.2 Implement `collectCarriers(packagesByPath, sharedPackages, moduleCtx)` Pass 1: walk all exported symbols in non-shared packages and build map of carrier symbol → shared type keys
- [x] 1.3 Extend `countSharedSymbolConsumers` / `countPackageSymbolConsumers` to perform Pass 2: for each reference to a carrier symbol, propagate the consumer package to all carried shared types
- [x] 1.4 Ensure consumer deduplication works correctly when a package both directly and indirectly references the same shared type

## 2. Test Coverage

- [x] 2.1 Add E2E test: shared type used indirectly through exported struct field reaches two consumers
- [x] 2.2 Add E2E test: shared type used indirectly through exported function signature reaches two consumers
- [x] 2.3 Add E2E test: shared type used indirectly through exported interface method reaches two consumers
- [x] 2.4 Add E2E test: shared type with direct + indirect consumer does not trigger false-sharing
- [x] 2.5 Add E2E test: carrier symbol with no external consumers does not over-count shared type
- [x] 2.6 Ensure all existing `nofalsesharing` E2E tests continue to pass

## 3. Validation

- [x] 3.1 Run `make lint` and fix any issues
- [x] 3.2 Run `make test` and confirm all tests pass
- [x] 3.3 Verify `selectorKind` case (or equivalent indirect usage) no longer triggers false-sharing if appropriately configured
