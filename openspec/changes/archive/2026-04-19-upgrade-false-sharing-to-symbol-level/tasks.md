## 1. Extend False-Sharing Analysis Inputs

- [x] 1.1 Update `pkg/boundarycontrol` false-sharing package loading so the cached module-wide pass has the syntax and type information needed to resolve exported symbol references.
- [x] 1.2 Add shared-package symbol discovery that enumerates exported package-owned declarations and skips packages that expose no exported symbols.
- [x] 1.3 Introduce a stable cached symbol identity format that can be matched back to declaration positions during normal analyzer passes.

## 2. Implement Symbol-Level Consumer Counting

- [x] 2.1 Replace package-level consumer aggregation with per-exported-symbol consumer tracking keyed by shared package and symbol identity.
- [x] 2.2 Count one direct non-test importing package path as one external consumer even when multiple files reference the same exported symbol.
- [x] 2.3 Treat a reference from another declaration inside the same shared package as an additional valid consumer for that exported symbol.
- [x] 2.4 Ignore `_test.go` references when computing exported-symbol consumer counts.

## 3. Report Symbol-Level Diagnostics

- [x] 3.1 Change false-sharing reporting so diagnostics are emitted at exported symbol declarations instead of once per shared package.
- [x] 3.2 Format diagnostics to mention only the direct consuming package path when a symbol has exactly one external consumer and keep the no-consumer message deterministic.
- [x] 3.3 Ensure packages that import the same shared package but use different exported symbols still produce symbol-specific violations.

## 4. Rebuild Coverage For The New Semantics

- [x] 4.1 Update `pkg/boundarycontrol` tests to cover single-consumer, no-consumer, and multi-consumer symbol-level behavior.
- [x] 4.2 Add focused tests for internal shared-package references counting as valid use, `_test.go` references not counting, and shared packages with no exported symbols producing no diagnostics.
- [x] 4.3 Add coverage for representative symbol forms that rely on type information, including exported functions, types, vars/consts, and exported methods.

## 5. Align Docs And Validate The Change

- [x] 5.1 Update README and any implementation-facing docs that still describe package-level false-sharing behavior.
- [x] 5.2 Run the most targeted `boundarycontrol` tests first, then run `make lint && make test` to validate the completed symbol-level upgrade end to end.
