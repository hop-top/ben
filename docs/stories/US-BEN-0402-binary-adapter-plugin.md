# US-BEN-0402 — Implement a new binary adapter plugin; verify PATH discovery + stdio JSON

*author: $USER | 2026-03-28*

---

## Identity

| Field     | Value                                                      |
|-----------|------------------------------------------------------------|
| ID        | US-BEN-0402                                                |
| Title     | Implement binary adapter plugin; verify discovery + protocol |
| Persona   | OSS Go Developer (Contributor)                             |
| Trigger   | Contributor wants to add a new adapter without modifying ben |

---

## Trigger

Contributor has a tool that isn't a simple CLI command. They want to wrap it as a
`ben-adapter-*` binary so ben can use it without a code change to the main repo.

---

## Acceptance Criteria

1. Contributor creates `ben-adapter-mytest` binary; places it on PATH.
2. `ben run` discovers the binary automatically — no config change required.
3. Ben sends `{"action":"run","candidate":{...},"input":{...}}` to adapter stdin (JSON).
4. Adapter writes `{"metrics":{...},"output":"..."}` to stdout; ben parses and stores it.
5. Result JSON includes the adapter's returned metrics under the candidate entry.
6. If adapter exits non-zero, ben records `error` in result; overall run exits 0.
7. If adapter binary not on PATH, `ben run` exits 1 with a clear error message to stderr.
8. Discovery scan matches prefix `ben-adapter-` only; no false positives.

---

## Interface Contract Referenced

Binary stdio JSON protocol (adapter direction):

```
stdin  (ben → adapter): {"action":"run","candidate":{...},"input":{...}}
stdout (adapter → ben): {"metrics":{"latency_ms":340,"quality_score":0.91},"output":"..."}
```

Adapter must exit 0 on success; non-zero signals adapter error (recorded, run continues).

---

## Test Requirements

- Unit: `tests/unit/plugin_binary_test.go` — mock binary; verify JSON serialisation round-trip.
- E2E: compile minimal Go binary implementing the protocol; place on PATH; run ben against it.
- `go test ./internal/plugin/...` passes with no external services.

---

## Happy Path Steps

1. Contributor writes `ben-adapter-mytest` (minimal Go binary: read stdin, write JSON stdout).
2. `go build -o /tmp/ben-adapter-mytest ./cmd/mytest && export PATH=/tmp:$PATH`
3. `ben run --candidates mytest --metric latency_ms --task "echo hello"` — no suite YAML.
4. Ben discovers `ben-adapter-mytest` on PATH via prefix scan.
5. Ben invokes binary; passes JSON on stdin.
6. Binary writes metrics JSON to stdout; ben captures result.
7. `ben run --format json` output contains `candidates[0].metrics.latency_ms`.

---

## Failure Path + Expected Behavior

| Failure                              | Expected behavior                                             |
|--------------------------------------|---------------------------------------------------------------|
| Binary not on PATH                   | `ben run` exits 1; stderr: "adapter 'mytest' not found"      |
| Binary exits non-zero                | Result has `error` field; run exits 0; other candidates run  |
| Binary writes invalid JSON           | Ben logs parse error to stderr; candidate marked errored      |
| Binary writes to stderr              | Ben forwards stderr to its own stderr; does not fail parse    |

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0402_test.go
func:      TestUS_BEN_0402_BinaryAdapterDiscoveryAndProtocol
setup:
  - compile minimal Go binary (testdata/adapter-stub/main.go):
      reads JSON from stdin
      writes {"metrics":{"latency_ms":1},"output":"ok"} to stdout
      exits 0
  - place binary as "ben-adapter-stub" in tmp dir; prepend to PATH
asserts:
  - ben run --candidates stub --metric latency_ms exits 0
  - result JSON: candidates[0].metrics.latency_ms == 1
  - result JSON: candidates[0].error == null
  - remove binary; re-run → exits 1; stderr contains "not found"
```
