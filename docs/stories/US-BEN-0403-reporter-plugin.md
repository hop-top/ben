# US-BEN-0403 — Implement a new reporter plugin; verify stdio JSON protocol

*author: $USER | 2026-03-28*

---

## Identity

| Field     | Value                                              |
|-----------|----------------------------------------------------|
| ID        | US-BEN-0403                                        |
| Title     | Implement reporter plugin; verify stdio JSON protocol |
| Persona   | OSS Go Developer (Contributor)                     |
| Trigger   | Contributor wants custom output format (e.g. markdown) |

---

## Trigger

Contributor needs ben results in a format not built-in (markdown, HTML, CSV). They write a
`ben-reporter-*` binary; expect ben to discover and invoke it with the full run payload.

---

## Acceptance Criteria

1. Contributor creates `ben-reporter-myreport` binary; places it on PATH.
2. `ben run --format myreport` discovers and invokes the binary automatically.
3. Ben sends full run payload to reporter stdin: `{"run":{...}}` (complete result schema).
4. Reporter writes formatted output to stdout; ben relays that to its own stdout unchanged.
5. Reporter stderr forwarded to ben stderr; does not interfere with formatted output.
6. If reporter binary missing, `ben run --format myreport` exits 1; clear error to stderr.
7. Built-in reporters (`json`, `table`, `yaml`) not affected by reporter plugins on PATH.
8. Discovery scan matches prefix `ben-reporter-` only.

---

## Interface Contract Referenced

Binary stdio JSON protocol (reporter direction):

```
stdin  (ben → reporter): {"run":{<full result schema>}}
stdout (reporter → stdout): <any formatted text — relayed verbatim>
```

Reporter must exit 0; non-zero treated as reporter error; ben exits 1.

---

## Test Requirements

- Unit: `tests/unit/plugin_reporter_test.go` — mock binary; verify JSON payload sent.
- E2E: compile minimal Go binary; place on PATH; verify formatted output reaches stdout.
- `go test ./internal/plugin/...` passes with no external services.

---

## Happy Path Steps

1. Contributor writes `ben-reporter-md` — reads `{"run":{...}}`; writes markdown table.
2. `go build -o /tmp/ben-reporter-md ./cmd/md && export PATH=/tmp:$PATH`
3. `ben run --suite mytest.yaml --format md` — ben discovers `ben-reporter-md`.
4. Ben sends `{"run":{...}}` JSON to reporter stdin.
5. Reporter writes markdown to stdout; ben relays verbatim.
6. Terminal shows markdown-formatted table.

---

## Failure Path + Expected Behavior

| Failure                              | Expected behavior                                              |
|--------------------------------------|----------------------------------------------------------------|
| Reporter binary not on PATH          | `ben run --format md` exits 1; stderr: "reporter 'md' not found" |
| Reporter exits non-zero              | Ben exits 1; reporter's stderr forwarded                       |
| Reporter writes invalid/no output    | Ben exits 1; stderr: "reporter produced no output"             |
| Built-in name collision (e.g. json)  | Built-in takes precedence; binary ignored                      |

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0403_test.go
func:      TestUS_BEN_0403_ReporterPluginProtocol
setup:
  - compile minimal Go binary (testdata/reporter-stub/main.go):
      reads JSON from stdin ({"run":{...}})
      writes "STUB_REPORT" to stdout
      exits 0
  - place binary as "ben-reporter-stub" in tmp dir; prepend to PATH
asserts:
  - ben run --suite testdata/simple.yaml --format stub exits 0
  - stdout contains "STUB_REPORT"
  - stdout does NOT contain ben's default table output
  - remove binary; re-run --format stub → exits 1; stderr contains "not found"
```
