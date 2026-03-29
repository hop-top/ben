# US-BEN-0101 — Run ben to compare two CLI tools (latency + output quality)

**ID:** US-BEN-0101
**Title:** Compare two CLI tools by latency and output quality
**Persona:** Platform Engineer
**Trigger:** Engineer wants to decide between two CLI tools (e.g. xray vs grep) for a
recurring task; needs objective latency and quality data before committing.

---

## Acceptance Criteria

1. `ben run` with `--candidates xray,grep`, `--metric latency_ms,quality_score`
   completes exit 0.
2. Result JSON contains both candidates with `latency_ms` and `quality_score` values.
3. `winner` field is set to the higher-scored candidate (not null).
4. `rank` field is `1` for winner, `2` for loser.
5. `latency_ms` is a positive integer (ms wall-clock).
6. `quality_score` is a float in [0, 1].
7. Stderr is empty when `--quiet` is passed.
8. Stdout is valid JSON when `--format json` is passed.

---

## Metrics Exercised

- `latency_ms` (builtin)
- `quality_score` (llm_judge plugin)

---

## Scorer Strategy

`weighted` — default weights: `latency_ms=0.3`, `quality_score=0.7`

---

## Happy Path Steps

```
1. Engineer runs:
     ben run \
       --task "Find all HTTP handler functions" \
       --candidates xray,grep \
       --metric latency_ms,quality_score \
       --scorer weighted:latency_ms=0.3,quality_score=0.7 \
       --input.repo ./testdata/sample-repo \
       --format json

2. Ben executes xray and grep against the task in parallel.
3. latency_ms captured per candidate via wall-clock timing.
4. quality_score evaluated by configured llm_judge plugin.
5. Scorer computes weighted score; assigns rank 1 to winner.
6. Ben emits result JSON to stdout; exits 0.
7. Engineer reads winner field; adopts winning tool.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| One candidate binary not on PATH | That candidate: `error` field set, metrics null, rank last; other candidate ranked 1; exit 0 |
| llm_judge API unreachable | Run aborts with exit 1; stderr: "quality_score plugin error: ..."; stdout: no partial JSON |
| Both candidates fail | Both have `error` set; `winner: null`; exit 0 |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0101_test.go`
**Test func:** `TestUS_BEN_0101_CompareTwoCLITools`

Asserts:
- Exit code 0.
- Stdout parses as valid JSON.
- `candidates` array length == 2.
- Each candidate has `metrics.latency_ms > 0`.
- Each candidate has `metrics.quality_score` in [0.0, 1.0].
- `winner` is one of `["xray", "grep"]` (not null, not empty string).
- Candidate with `rank == 1` matches `winner`.
- Candidate with `rank == 2` has lower `score` than rank-1 candidate.
- Stderr is empty when `--quiet` flag set.
