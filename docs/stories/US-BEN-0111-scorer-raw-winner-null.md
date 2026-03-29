# US-BEN-0111 — Run with scorer: raw; confirm winner is null

**ID:** US-BEN-0111
**Title:** Run with scorer raw and confirm winner is null
**Persona:** Platform Engineer
**Trigger:** Engineer exploring a new metric or adapter wants raw metric values without any
scoring or ranking applied; uses `scorer: raw` to inspect unprocessed data.

---

## Acceptance Criteria

1. `--scorer raw` produces a result where `winner == null`.
2. All candidates still present in `candidates` array with their raw metrics.
3. `score` field on each candidate is null (or omitted).
4. `rank` field on each candidate is null (or omitted).
5. `scorer.strategy == "raw"` in result JSON.
6. All metrics collected normally; only scoring/ranking skipped.
7. Exit code 0 on successful run.

---

## Metrics Exercised

- `latency_ms` (builtin)
- `exit_code` (builtin)
- `output_size` (builtin)

---

## Scorer Strategy

`raw` — no scoring applied

---

## Happy Path Steps

```
1. Engineer runs:
     ben run \
       --task "List files" \
       --candidates ls,find \
       --metric latency_ms,exit_code,output_size \
       --scorer raw \
       --format json

2. Ben executes both candidates; captures all three metrics per candidate.
3. No scorer applied; no score computed; no rank assigned.
4. winner field in result set to null.
5. Engineer inspects raw metric values to decide which metric to use for ranking.
6. Engineer re-runs with --scorer single:latency_ms once metric is understood.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| One candidate errors | Error candidate has metrics: null; winner still null; exit 0 |
| All candidates error | All metrics null; winner null; exit 0 (all errors in result) |
| `--scorer raw` combined with `--scorer single:x` | Exit 1; stderr: "conflicting scorer flags" |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0111_test.go`
**Test func:** `TestUS_BEN_0111_ScorerRawWinnerNull`

Asserts:
- `ben run --scorer raw --format json` exits 0.
- `scorer.strategy == "raw"` in result JSON.
- `winner == null` (JSON null, not string "null").
- Each candidate in `candidates[]` has metrics object with numeric values.
- `score` is null or key absent for all candidates.
- `rank` is null or key absent for all candidates.
- `metrics.latency_ms`, `metrics.exit_code`, `metrics.output_size` all present and numeric.
- When one candidate errors: `candidates[i].error` is non-null string; `winner` still null.
