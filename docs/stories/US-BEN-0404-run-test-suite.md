# US-BEN-0404 — Run test suite locally via 'task test'; all green first try

*author: $USER | 2026-03-28*

---

## Identity

| Field     | Value                                          |
|-----------|------------------------------------------------|
| ID        | US-BEN-0404                                    |
| Title     | Run test suite locally; all green first try    |
| Persona   | OSS Go Developer (Contributor)                 |
| Trigger   | Contributor clones repo; runs tests before any change |

---

## Trigger

Contributor wants confidence the baseline is green before making changes. They run
`task test` (or `go test ./...`) immediately after clone; expect zero failures and no
external service dependencies.

---

## Acceptance Criteria

1. `task test` defined in `Taskfile.yml`; runs `go test ./...` (or targeted equivalent).
2. All tests pass on first run; no setup beyond `go mod download`.
3. No test requires a running service (no Postgres, no Docker, no network calls).
4. Tests that need a temp binary compile it within the test via `go build` into `t.TempDir()`.
5. `go test ./...` completes in < 60 seconds on M-series Mac (cold, no cache).
6. `task lint` also passes (vet + staticcheck or equivalent).
7. `task test` exit code mirrors `go test` exit code.

---

## Interface Contracts Referenced

No interface contracts directly; this story validates that all existing impls
(`Adapter`, `Metric`, `Scorer`, `Reporter`) have passing tests at baseline.

---

## Test Requirements

- This story IS the test requirement. Existing unit + e2e suite must be hermetic.
- `tests/unit/` — pure Go, no external deps.
- `tests/e2e/` — compiles helper binaries in-process; no Docker.
- Coverage gate: `go test -coverprofile=cover.out ./...`; report generated; no hard floor
  for v0.1 but coverage must be measured.

---

## Happy Path Steps

1. `git clone github.com/hop-top/ben && cd ben`
2. `go mod download` — all deps fetched; no private module auth required.
3. `task test` — runs; all lines say `ok  hop.top/ben/...`.
4. `task lint` — runs; exits 0.
5. Contributor modifies one file; re-runs `task test`; still green.

---

## Failure Path + Expected Behavior

| Failure                                  | Expected behavior                                        |
|------------------------------------------|----------------------------------------------------------|
| Test requires network / running service  | Test skips (`t.Skip`) if env var not set; CI controls   |
| `task` binary absent                     | Doc says: `brew install go-task/tap/go-task`; fall back |
|                                          | to `go test ./...` directly                              |
| Test flaky due to timing                 | Flaky test is a bug; contributor files issue; maintainer |
|                                          | fixes before merge                                       |
| `go mod download` needs private module   | Build fails with clear message; maintainer makes dep     |
|                                          | public or uses `replace` directive                       |

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0404_test.go
func:      TestUS_BEN_0404_TestSuiteGreenOnFirstRun
asserts:
  - exec "go test ./..." from repo root exits 0
  - exec "go test ./..." produces no "FAIL" lines
  - exec "go vet ./..." exits 0
  - no test in tests/unit/ or tests/e2e/ calls t.Fatal with "requires running service"
    (static check: grep for sentinel string in source)
note: this test is itself hermetic; it is a meta-test validating hermeticity invariant
```
