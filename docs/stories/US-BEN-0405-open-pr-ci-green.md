# US-BEN-0405 — Open PR; CI green; merged in <= 2 review cycles

*author: $USER | 2026-03-28*

---

## Identity

| Field     | Value                                              |
|-----------|----------------------------------------------------|
| ID        | US-BEN-0405                                        |
| Title     | Open PR; CI green; merged in <= 2 review cycles    |
| Persona   | OSS Go Developer (Contributor)                     |
| Trigger   | Contributor has working implementation; opens PR   |

---

## Trigger

Contributor finishes a new adapter, metric, or scorer. They open a PR; expect CI to pass
on first push and the PR to be reviewed and merged within two round-trips.

---

## Acceptance Criteria

1. CI pipeline defined (GitHub Actions or equivalent); runs on every PR push.
2. CI runs `task test` + `task lint`; both must be green for merge.
3. CI checks `go vet ./...` and `staticcheck ./...` (or golangci-lint equivalent).
4. PR template prompts: interface implemented, tests added, `go test ./...` output.
5. Maintainer reviews within 5 business days of CI-green PR.
6. Contributor needs at most 2 review cycles (initial + one round of fixes) to merge.
7. No manual steps required beyond pushing a branch and opening a PR.
8. CI matrix covers at least go@latest on linux; darwin optional.

---

## Interface Contracts Referenced

No new interface; CI validates that contributed impl satisfies existing `Adapter`,
`Metric`, or `Scorer` interface at compile time (type assertion tests).

---

## Test Requirements

- CI must run the full `go test ./...` suite including e2e tests.
- Coverage report uploaded to CI artifact; not a merge gate in v1.
- E2E tests that require PATH-placed binaries compile them within the test.

---

## Happy Path Steps

1. `git hop add feat/my-adapter` — new worktree branch.
2. Implement adapter; add unit + e2e tests; run `task test` locally — green.
3. `git push origin feat/my-adapter`; open PR on GitHub.
4. CI triggers automatically; completes in < 5 minutes.
5. CI is green; maintainer reviews; requests minor changes.
6. Contributor pushes fixes; CI re-runs; green; maintainer merges.
7. Feature available in next release.

---

## Failure Path + Expected Behavior

| Failure                               | Expected behavior                                           |
|---------------------------------------|-------------------------------------------------------------|
| CI red on first push                  | Contributor sees failing step in CI logs; fixes + re-pushes |
| Maintainer unresponsive > 5 days      | Contributor pings via issue; tracked as process gap         |
| > 2 review cycles needed              | Maintainer scopes review to interface contract only;        |
|                                       | style nits deferred to follow-up                            |
| CI matrix version mismatch            | go.mod specifies minimum Go version; CI uses go@latest      |
|                                       | and go@min; both must pass                                  |

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0405_test.go
func:      TestUS_BEN_0405_CIPipelineDefinedAndPasses
asserts:
  - .github/workflows/ci.yml (or equivalent) exists
  - workflow file contains "go test" step
  - workflow file contains "go vet" or "golangci-lint" step
  - workflow triggers on: [push, pull_request]
  - exec "go build ./..." from repo root exits 0 (build gate)
note: full CI invocation not run in e2e; file existence + build gate sufficient
```
