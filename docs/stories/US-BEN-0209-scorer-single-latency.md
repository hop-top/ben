# US-BEN-0209 — Run with scorer: single:latency_ms; fastest candidate wins

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0209
**Title:** Run with scorer: single:latency_ms; fastest candidate wins
**Persona:** Solo Developer
**Trigger:** Dev cares only about speed; wants clearest possible winner signal.

---

## Acceptance Criteria

1. `--scorer single:latency_ms` sets scorer strategy to `single` with metric `latency_ms`.
2. Candidate with lowest `latency_ms` gets rank 1 and is named as `winner`.
3. Score field is normalised 0–1 (1.0 = best, 0.0 = worst); single-candidate → 1.0.
4. Ties broken deterministically (e.g. by candidate declaration order).
5. `scorer` object in JSON result: `{strategy: "single", weights: {latency_ms: 1.0}}`.

---

## Metrics Exercised

- `latency_ms` — sole ranking dimension

---

## Scorer Strategy

- `single:latency_ms` — lowest latency_ms wins

---

## Happy Path

```pseudocode
ben run \
  --task "sleep test" \
  --candidates "sleep 0.01","sleep 0.05","sleep 0.1" \
  --metric latency_ms \
  --scorer single:latency_ms \
  --format json

// result:
// winner: "sleep 0.01"
// candidates sorted by latency_ms ascending
// rank: 1 → sleep 0.01, 2 → sleep 0.05, 3 → sleep 0.1
// score: 1.0 → 0.2 → 0.1 (normalised; exact formula TBD by impl)
// exit: 0
```

---

## Failure Path

- **Metric not in --metric list:** `--scorer single:quality_score` but `--metric latency_ms`
  only; exit 1; stderr: `error: scorer metric "quality_score" not in requested metrics`.
- **Scorer applied to raw run:** `--scorer raw` — winner == null; score == null; exit 0.
- **One candidate errors (exit != 0):** that candidate's latency_ms still captured
  (time-to-failure); scored normally unless `--skip-errors` flag set.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0209_test.go
func:      TestUS_BEN_0209_ScorerSingleLatency
```

Asserts:
- Command exits 0.
- JSON `scorer.strategy` == `"single"`.
- JSON `scorer.weights` has key `latency_ms` with value `1.0`.
- JSON `winner` equals the candidate with minimum `metrics.latency_ms`.
- Candidate with minimum latency_ms has `rank == 1`.
- All scores in range [0.0, 1.0].
- Highest score candidate == winner.
- `--scorer single:nonexistent_metric` exits 1 and stderr contains `not in requested metrics`.
