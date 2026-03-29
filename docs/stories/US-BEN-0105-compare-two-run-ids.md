# US-BEN-0105 — Compare two run IDs side-by-side (ben compare)

**ID:** US-BEN-0105
**Title:** Compare two run IDs side-by-side using ben compare
**Persona:** Platform Engineer
**Trigger:** Engineer ran the suite before and after a code change; wants to diff metric
values and winner across the two runs to confirm improvement or detect regression.

---

## Acceptance Criteria

1. `ben compare <run-a> <run-b> --format json` exits 0 when both run IDs exist.
2. Output JSON contains both run objects (keyed or as array).
3. Each run object includes `run_id`, `timestamp`, `winner`, `candidates` with metrics.
4. Delta values (metric differences) are present per candidate per metric.
5. Exit 1 if either run ID does not exist; stderr names the missing ID.
6. `--format table` renders a human-readable side-by-side comparison.

---

## Metrics Exercised

- `latency_ms` (builtin)
- `quality_score` (llm_judge plugin)

---

## Scorer Strategy

Scorer from each run's stored result (may differ between runs; compare shows both).

---

## Happy Path Steps

```
1. Engineer has run-A (before change) and run-B (after change) stored locally.
2. Engineer runs:
     ben compare <run-A-id> <run-B-id> --format json

3. Ben loads both results from local storage.
4. Output JSON:
     {
       "run_a": { "run_id": "...", "winner": "xray", "candidates": [...] },
       "run_b": { "run_id": "...", "winner": "grep", "candidates": [...] },
       "delta": {
         "xray": { "latency_ms": -40, "quality_score": +0.12 },
         "grep": { "latency_ms": +20, "quality_score": -0.05 }
       }
     }

5. Engineer sees winner changed from xray → grep; quality_score of xray improved.
6. Engineer decides whether to promote or revert the change.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| run-A ID not found | Exit 1; stderr: "run not found: <id>" |
| run-B ID not found | Exit 1; stderr: "run not found: <id>" |
| Both IDs identical | Exit 0; delta is all zeros; stderr warning: "comparing run to itself" |
| Candidates differ between runs | Delta computed for intersection; extra candidates noted in stderr |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0105_test.go`
**Test func:** `TestUS_BEN_0105_CompareTwoRunIDs`

Asserts:
- Setup: execute suite twice; capture run_id from each run.
- `ben compare <id-A> <id-B> --format json` exits 0.
- Stdout parses as JSON object with keys `run_a`, `run_b`, `delta`.
- `run_a.run_id == id-A`, `run_b.run_id == id-B`.
- `delta` contains at least one candidate key; each has numeric metric deltas.
- `ben compare nonexistent-id <id-B>` exits 1; stderr contains "run not found".
- `ben compare <id-A> <id-A> --format json` exits 0; all delta values == 0.
