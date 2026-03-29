# US-BEN-0110 — Run with scorer: single:<metric>; confirm ranking by one metric

**ID:** US-BEN-0110
**Title:** Run with scorer single:<metric> and confirm ranking by one metric only
**Persona:** Platform Engineer
**Trigger:** Engineer wants to rank candidates purely by latency (or any single metric)
without weighting; uses `single:<metric>` scorer for unambiguous, single-dimension ranking.

---

## Acceptance Criteria

1. `--scorer single:latency_ms` ranks candidates solely by `latency_ms` (lower = better).
2. `score` value in result equals the raw metric value (or normalized equivalent).
3. `winner` is the candidate with lowest `latency_ms`.
4. `rank` 1 assigned to lowest-latency candidate regardless of other metric values.
5. When two candidates tie on the metric, lower rank assigned to earlier candidate name
   (alphabetical tiebreak) and a stderr warning emitted.
6. Non-existent metric name in `single:<metric>` exits 1 with descriptive stderr.

---

## Metrics Exercised

- `latency_ms` (builtin) — sole scoring metric
- `quality_score` (collected but not used in scoring)

---

## Scorer Strategy

`single:latency_ms`

---

## Happy Path Steps

```
1. Engineer runs:
     ben run \
       --task "Index codebase" \
       --candidates xray,grep \
       --metric latency_ms,quality_score \
       --scorer single:latency_ms \
       --format json

2. Ben executes both candidates; captures latency_ms and quality_score.
3. Scorer ignores quality_score; ranks by latency_ms only.
4. Candidate with lower latency_ms gets rank 1 and is winner.
5. Result JSON: scorer.strategy == "single", scorer.metric == "latency_ms".
6. quality_score present in metrics but not in scorer weights.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| `--scorer single:nonexistent_metric` | Exit 1; stderr: "scorer error: metric 'nonexistent_metric' not in run" |
| Tie on latency_ms | Exit 0; alphabetical tiebreak; stderr warning: "tie on latency_ms: tiebreak applied" |
| Candidate latency_ms is null (error) | Null treated as worst (max); other candidate wins |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0110_test.go`
**Test func:** `TestUS_BEN_0110_ScorerSingleMetric`

Asserts:
- `ben run --scorer single:latency_ms --format json` exits 0.
- `scorer.strategy == "single"` in result JSON.
- `scorer.metric == "latency_ms"` (or equivalent field) in result JSON.
- Candidate with smallest `metrics.latency_ms` has `rank == 1`.
- `winner` matches the rank-1 candidate name.
- `quality_score` present in candidate metrics (collected) but absent from scorer weights.
- `--scorer single:bogus` exits 1; stderr contains "metric 'bogus' not in run".
- Controlled tie scenario (both latency_ms == 100): exit 0; exactly one candidate has rank 1;
  stderr contains "tie" or "tiebreak".
