# US-BEN-0407 — Add a new built-in metric by implementing Metric interface; verify in result JSON

*author: $USER | 2026-03-28*

---

## Identity

| Field     | Value                                                          |
|-----------|----------------------------------------------------------------|
| ID        | US-BEN-0407                                                    |
| Title     | Add built-in metric via Metric interface; verify in result JSON |
| Persona   | OSS Go Developer (Contributor)                                 |
| Trigger   | Contributor wants a new always-available metric (e.g. output_lines) |

---

## Trigger

Contributor adds `output_lines` as a built-in metric (counts lines in candidate raw output).
They implement `Metric` interface; register it; verify it appears in result JSON without any
plugin config.

---

## Acceptance Criteria

1. `Metric` interface defined in `internal/metrics/metrics.go`; exported; godoc comment present.
2. Contributor adds `OutputLines` struct in `internal/metrics/builtin.go`; implements `Metric`.
3. New metric registered in `internal/metrics/registry.go` (or init block); available by name.
4. `ben run --metric output_lines` includes `output_lines` in result JSON without config.
5. Result JSON: `candidates[*].metrics.output_lines` is an integer (float64 in JSON).
6. Metric value is correct: matches `strings.Count(output, "\n") + 1` for non-empty output.
7. `go test ./internal/metrics/...` includes a test for `OutputLines.Measure()`; passes.
8. No change to `cmd/` required; metric auto-discovered via registry.

---

## Interface Contract Referenced

`Metric` interface:

```go
// Metric measures one dimension of a candidate's raw output.
type Metric interface {
    Name() string
    Measure(ctx context.Context, raw RawOutput) (float64, error)
}

// RawOutput carries the captured stdout, stderr, exit code, and wall time.
type RawOutput struct {
    Stdout   string
    Stderr   string
    ExitCode int
    WallMS   int64
}
```

---

## Test Requirements

- Unit: `tests/unit/metric_output_lines_test.go` — table-driven; verify line counts for
  empty string, single line, multi-line, trailing newline edge cases.
- E2E: `ben run` with `--metric output_lines` on a known candidate; assert value in JSON.
- `go test ./internal/metrics/...` hermetic; no external deps.

---

## Happy Path Steps

1. Contributor reads `internal/metrics/metrics.go` — understands `Metric` interface.
2. Adds `outputLines` struct in `internal/metrics/builtin.go`; implements `Name()` and
   `Measure()`.
3. Registers in `internal/metrics/registry.go`: `Register(&outputLines{})`.
4. Writes unit test; runs `task test ./internal/metrics/...` — green.
5. `ben run --candidates "echo -e 'a\nb\nc'" --metric output_lines --format json`
6. Result JSON shows `candidates[0].metrics.output_lines == 3`.

---

## Failure Path + Expected Behavior

| Failure                                    | Expected behavior                                          |
|--------------------------------------------|------------------------------------------------------------|
| `Measure()` returns error                  | Metric recorded as `null` in result; run continues         |
| Name collision with existing metric        | Registration panics at startup with "duplicate metric name"|
| Contributor forgets to call Register()     | `ben run --metric output_lines` exits 1: "unknown metric"  |
| Metric impl not thread-safe                | `go test -race ./internal/metrics/...` catches data race   |

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0407_test.go
func:      TestUS_BEN_0407_NewBuiltinMetricInResultJSON
setup:
  - use built-in CLI adapter
  - candidate cmd: "printf 'line1\nline2\nline3\n'"
  - metrics: [output_lines]
  - scorer: raw
asserts:
  - ben run exits 0
  - result JSON: candidates[0].metrics["output_lines"] == 3
  - result JSON: candidates[0].error == null
  - go test ./internal/metrics/... exits 0 (run as sub-test or pre-condition)
```
