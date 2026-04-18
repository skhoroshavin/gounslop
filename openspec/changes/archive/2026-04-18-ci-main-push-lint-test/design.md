## Context

This repository is a Go-based lint plugin project with existing quality checks exposed through `make lint` and `make test`. Today these checks are primarily run locally, so pushes to `main` can succeed even when lint or tests fail in a clean environment. The change introduces a GitHub Actions workflow to execute the same local quality gates on every push to `main`.

## Goals / Non-Goals

**Goals:**
- Run linting automatically on each push to `main`.
- Run unit/integration tests automatically on each push to `main`.
- Keep CI behavior aligned with local development commands by invoking Make targets.
- Use a deterministic Go toolchain version compatible with the repository.

**Non-Goals:**
- Adding pull request, tag, or scheduled workflow triggers.
- Refactoring lint/test commands or analyzer implementation details.
- Introducing multi-version Go test matrices.

## Decisions

- Use a single workflow file under `.github/workflows/ci.yml`.
  - Rationale: keeps repository automation discoverable and minimal for one CI capability.
  - Alternative considered: split lint and test into separate workflow files; rejected to avoid duplicate setup and maintenance overhead.
- Name the GitHub Actions workflow `Test`.
  - Rationale: provides a concise health indicator label in the GitHub Actions UI.
  - Alternative considered: generic `CI` naming; rejected to align with the requested pipeline name.
- Trigger workflow only on `push` to `main`.
  - Rationale: matches explicit requested behavior and avoids changing contributor workflow beyond main branch protection checks.
  - Alternative considered: include pull request triggers; deferred as future enhancement.
- Implement checks as separate jobs (`lint` and `test`) sharing the same setup pattern.
  - Rationale: clearer failure isolation and potential parallel execution to reduce total runtime.
  - Alternative considered: run both commands in one job; rejected because one failure would hide status separation.
- Use `actions/checkout` and `actions/setup-go` with Go version from `go.mod`.
  - Rationale: keeps CI version source-of-truth in repository and reduces drift.
  - Alternative considered: hardcoded Go version in workflow; rejected due to duplicate version maintenance.

## Risks / Trade-offs

- [Workflow runtime may be slower because lint builds custom tooling] -> Mitigation: rely on existing make target behavior and GitHub Actions caching support from `setup-go`.
- [Main-branch-only trigger provides delayed feedback before merge] -> Mitigation: accepted trade-off for current solo-maintainer workflow.
- [Local/CI environment differences could still produce surprises] -> Mitigation: use official Go setup action and existing Makefile commands to keep parity high.

## Migration Plan

1. Add `.github/workflows/ci.yml` with `push` trigger for `main`.
2. Define `lint` and `test` jobs, each checking out code, setting up Go from `go.mod`, and running its target command.
3. Push change and confirm workflow runs on `main` push.
4. If workflow fails unexpectedly, rollback by reverting the workflow file commit.

## Open Questions

- None for this iteration.
