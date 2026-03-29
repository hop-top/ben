# US-BEN-0103 — Query last N runs for a suite; spot regressions

**ID:** US-BEN-0103
**Title:** Query last N runs for a suite to spot regressions
**Persona:** Platform Engineer
**Trigger:** After several CI runs, engineer wants to inspect trend data for a suite to
identify whether a recent change degraded latency or quality scores.

---

## Acceptance Criteria

1. `ben query --suite <name> --last N --format json` returns an array of exactly N result
   objects (or fewer if < N runs exist).
2. Results are ordered newest-first by `timestamp`.
3. Each result object contains `run_id`, `timestamp`, `winner`, `candidates` array.
4. Each candidate in results contains `metrics` and `score`.
5. Exit code 0 when ≥ 1 run exists; exit 0 with empty array when 0 runs exist.
6. `--last` value must be a positive integer; non-integer value yields exit 1 + stderr error.

---

## Metrics Exercised

- `latency_ms` (builtin)
- `quality_score` (llm_judge plugin)

---

## Scorer Strategy

`weighted` — as declared in stored suite runs

---

## Happy Path Steps

```
1. Engineer has run the suite 5 times (run-1 .. run-5) across different commits.
2. Engineer runs:
     ben query --suite codebase-indexing --last 3 --format json
3. Ben returns array of 3 most-recent results [run-5, run-4, run-3].
4. Engineer compares winner field across results: run-3=xray, run-4=xray, run-5=grep.
5. Engineer notes xray score dropped in run-5; investigates that commit.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| Suite has 0 runs stored | Exit 0; stdout: `[]` |
| `--last 0` | Exit 1; stderr: "--last must be >= 1" |
| `--last abc` | Exit 1; stderr: "--last must be a positive integer" |
| Suite name unknown | Exit 0; stdout: `[]` (no runs stored for that name) |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0103_test.go`
**Test func:** `TestUS_BEN_0103_QueryLastNRuns`

Asserts:
- Setup: run suite 3 times with known `run_id` values captured.
- `ben query --suite <name> --last 3 --format json` exits 0.
- Stdout parses as JSON array of length 3.
- Array[0].timestamp > Array[1].timestamp > Array[2].timestamp (newest-first).
- Each element has fields: `run_id` (string), `winner` (string|null),
  `candidates` (array), `timestamp` (RFC3339 string).
- `ben query --suite <name> --last 1 --format json` returns array of length 1.
- `ben query --suite unknown-suite --last 5 --format json` returns `[]`.
- `ben query --suite <name> --last 0` exits 1; stderr contains "--last must be >= 1".
