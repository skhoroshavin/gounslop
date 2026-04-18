## Context

`gounslop` currently tests analyzer behavior with `analysistest`, which works well for AST- and type-level checks inside a single package tree. It does not give the repository a shared way to exercise full-plugin behavior against temporary Go modules, generated `golangci-lint` configuration, or multi-package repository layouts.

This gap matters most for the planned architecture-oriented analyzers and config work, where correctness depends on real package graphs, plugin registration, and configuration decoding. The repository already has a stable way to build the custom linter binary through `make custom-gcl`, so the design should reuse that path instead of introducing a separate test-only execution model.

## Goals / Non-Goals

**Goals:**

- Add one reusable E2E harness that package tests can call to create temporary repositories and run `./custom-gcl run ./...` against them.
- Support three scenario types in the first version: successful runs with diagnostics, runs with no diagnostics, and expected command/configuration failures.
- Keep assertions stable by normalizing temp-directory-specific output before tests compare results.
- Keep scenario definitions compact and readable so new cases are easy to review inline in Go tests.
- Establish one representative seed suite for an existing analyzer so later changes extend the same pattern instead of inventing local helpers.

**Non-Goals:**

- Replacing `analysistest` for analyzer internals or suggested-fix coverage.
- Designing a large fixture framework with multiple storage backends, golden management tools, or custom DSLs.
- Changing analyzer behavior, plugin registration behavior, or public runtime configuration as part of this change.

## Decisions

### 1. Put the harness in a shared internal test helper package

Create a small shared package under `internal/` for E2E test support so any package test in the module can reuse it without exporting production APIs. This keeps the harness close to the repository root, avoids circular dependencies on analyzer packages, and makes it clear that the code exists only for repository-level tests.

Alternative considered:

- Put helpers in each analyzer package: rejected because it duplicates temp-workspace and command-running logic.
- Use shell scripts instead of Go helpers: rejected because test setup, assertions, and temp-file lifecycle become harder to compose with `go test`.

### 2. Use compact inline scenarios with explicit files and generated config

The first version should represent each scenario in Go as a small struct containing file contents, optional module path, linter settings, and expected outcome. The harness will materialize those files into a temp directory, generate `go.mod` when needed, and write `.golangci.yml` from the scenario settings.

This keeps the initial version small and readable, especially while the repository only needs a few seed cases. It also makes configuration-error scenarios straightforward because a test can provide raw invalid settings or malformed values without maintaining a separate fixture directory tree. If examples ever become too large for comfortable inline review, the contents can be moved behind `go:embed` without changing the harness API, but that should remain an exception rather than a primary mode.

Alternative considered:

- Copy entire fixture directories from disk: deferred because it adds I/O helpers and fixture management overhead before there is enough case volume to justify it.

### 3. Keep build orchestration outside Go tests

The Go harness should only execute an existing `custom-gcl` binary against the temp repository. It should not invoke `make` or perform binary build orchestration itself. Repository-level command wiring can add a dedicated `make e2e` target that depends on `custom-gcl`, so local and CI entrypoints still get the right build ordering without embedding Makefile behavior in test code.

Alternative considered:

- Invoke `make custom-gcl` from Go tests: rejected because build orchestration belongs in repository commands such as `make e2e`, not in the harness API.
- Invoke `golangci-lint custom` directly from the harness: rejected because it duplicates build logic already maintained outside the test helper.

### 4. Assert only on actionable success or failure output

The harness should capture exit status, stdout, and stderr, then normalize temp-root paths to repository-relative placeholders before assertions. The primary test contract should be simple: either no error was raised, or an error was raised and its message is readable enough to act on directly. Assertions should therefore focus on expected diagnostics or failure fragments that a human or LLM could use immediately, instead of snapshotting full raw process output.

Alternative considered:

- Provide exact-output snapshots as the main assertion style: rejected because incidental formatting and temp paths make them brittle without adding much confidence.
- Assert on raw combined output: rejected because temp paths and incidental formatting make failures noisy and fragile.

### 5. Seed the harness with `nofalsesharing` plugin-level scenarios

The first consumer should be `nofalsesharing`, because it already exercises the kinds of behavior the harness is meant to support: multi-package module loading, plugin-level configuration, and command failures caused by invalid configuration decoding. A small initial suite can cover:

- a multi-package shared-package violation
- a passing multi-package case with multiple consumers
- a configuration-error case using an invalid settings type in `.golangci.yml`

This gives the harness immediate value for the architecture roadmap without waiting for `archcontrol` to exist.

Alternative considered:

- Seed with `nospecialunicode` or `nounicodeescape`: rejected because they do not stress multi-package behavior.
- Seed with `nodeepimports`: possible later, but `nofalsesharing` better exercises both workspace setup and configuration handling.

## Risks / Trade-offs

- More expensive test runs than `analysistest` -> Mitigation: build `custom-gcl` once, keep the initial suite small, and reserve the harness for cases that need full-plugin execution.
- Scenario structs may become verbose as coverage grows -> Mitigation: keep the first API minimal, optimize for short inline definitions, and treat `go:embed` as a narrow escape hatch rather than adding fixture-directory support.
- Output format changes in `golangci-lint` may make assertions brittle -> Mitigation: assert on normalized diagnostics and failure fragments, not the full raw process transcript.
- E2E tests depend on the custom binary build path staying healthy -> Mitigation: keep build orchestration in repository commands such as `make e2e`, which can depend on `custom-gcl` without teaching the Go harness how to build binaries.

## Migration Plan

1. Add the shared `internal` E2E harness with temp-workspace creation, file writing, config generation, command execution, and output normalization.
2. Add an initial `nofalsesharing` E2E suite covering one failing multi-package case, one passing case, and one config-error case.
3. Add repository command wiring, such as `make e2e`, that ensures `custom-gcl` exists before E2E tests run.
4. Document when to use the harness versus `analysistest` so future changes extend the shared pattern.

No user-facing migration is required. If the harness proves too costly or awkward, rollback is limited to removing the helper and the seed E2E tests.

## Open Questions

- None for this phase. The first version should keep assertions centered on actionable success or failure messages and should stay with compact inline scenarios.
