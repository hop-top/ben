# US-BEN-0109 — Verify full result schema fields in JSON output

**ID:** US-BEN-0109
**Title:** Verify full result schema fields in JSON output
**Persona:** Platform Engineer
**Trigger:** Engineer integrating ben into a pipeline or parsing results programmatically
needs confidence that all documented schema fields are always present and correctly typed.

---

## Acceptance Criteria

1. `ben run --format json` output contains all top-level fields: `run_id`, `suite`,
   `suite_version`, `timestamp`, `scorer`, `candidates`, `winner`, `metadata`.
2. `run_id` is a non-empty string (ULID format).
3. `timestamp` is a valid RFC3339 datetime string.
4. `scorer` object contains `strategy` (string) and `weights` (object or null).
5. Each candidate object contains: `name`, `metrics`, `score`, `rank`, `raw_output`, `error`.
6. `metrics` is an object; each key is a metric name; each value is a number or null.
7. `score` is a float; `rank` is a positive integer.
8. `raw_output` is a string (may be empty); `error` is string or null.
9. `metadata` contains `host` (string) and `ben_version` (string).
10. `winner` is a string matching a candidate name, or null when scorer is `raw`.

---

## Metrics Exercised

- `latency_ms` (builtin)
- `exit_code` (builtin)
- `output_size` (builtin)

---

## Scorer Strategy

`weighted` — `latency_ms=0.5`, `exit_code=0.3`, `output_size=0.2`

---

## Happy Path Steps

```
1. Engineer runs:
     ben run \
       --task "List files" \
       --candidates ls,find \
       --metric latency_ms,exit_code,output_size \
       --scorer weighted:latency_ms=0.5,exit_code=0.3,output_size=0.2 \
       --format json

2. Ben runs both candidates; captures all three metrics.
3. Scorer assigns scores and ranks.
4. Output JSON emitted to stdout.
5. Engineer pipes to: jq 'keys'  — confirms top-level keys present.
6. Engineer pipes to: jq '.candidates[].metrics | keys'  — confirms metric keys.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| Candidate errors mid-run | `error` field set; `metrics` may be partial; `score` = 0; `rank` = last |
| Metric plugin returns null | `metrics.<name>` = null; scorer treats null as 0 for weighted calc |
| `raw_output` exceeds 1MB | Truncated to 1MB; `metadata.raw_output_truncated: true` added |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0109_test.go`
**Test func:** `TestUS_BEN_0109_FullResultSchema`

Asserts:
- Stdout parses as JSON object.
- Top-level keys present: `run_id`, `suite`, `suite_version`, `timestamp`, `scorer`,
  `candidates`, `winner`, `metadata`.
- `run_id` matches ULID regex `^[0-9A-Z]{26}$`.
- `timestamp` parses as time.RFC3339 without error.
- `scorer.strategy` is non-empty string.
- `candidates` is array of length >= 1.
- Each candidate has string `name`, object `metrics`, float64 `score`,
  int `rank` >= 1, string `raw_output`, `error` is string or null.
- `metrics` object has keys `latency_ms`, `exit_code`, `output_size`; values are numbers.
- `metadata.host` is non-empty string; `metadata.ben_version` matches semver pattern.
- `winner` is one of the candidate names or null.
