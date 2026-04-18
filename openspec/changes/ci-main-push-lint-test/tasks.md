## 1. Workflow Setup

- [x] 1.1 Create `.github/workflows/ci.yml` with `push` trigger limited to `main`
- [x] 1.2 Name the workflow `Test`
- [x] 1.3 Add shared job steps for repository checkout and Go setup from `go.mod`

## 2. CI Job Implementation

- [x] 2.1 Add `lint` job that runs `make lint`
- [x] 2.2 Add `test` job that runs `make test`
- [x] 2.3 Ensure lint and test jobs execute independently so failures are reported per job

## 3. Verification

- [x] 3.1 Validate workflow syntax and repository lint/test commands locally where possible
- [ ] 3.2 Push to a branch and confirm the workflow behavior is ready for `main` push execution
