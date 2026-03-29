# US-BEN-0104 — Use --format json in CI pipeline; parse winner field

**ID:** US-BEN-0104
**Title:** Use --format json in CI pipeline and parse winner field
**Persona:** Platform Engineer
**Trigger:** CI step needs a machine-readable result from ben to gate a deploy or emit a
metric; engineer pipes `--format json` stdout through `jq` to extract `winner`.

---

## Acceptance Criteria

1. `ben run --suite <name> --format json --quiet` exits 0 on successful run.
2. Stdout is valid JSON (parseable by `jq`).
3. Stdout contains no non-JSON content (no progress bars, no ANSI codes).
4. Stderr is empty when `--quiet` is set.
5. `winner` field is present in top-level JSON object.
6. `winner` is a non-null string when scorer is not `raw`.
7. CI can extract winner via: `ben run ... --format json | jq -r '.winner'`.
8. Exit code 1 only on ben errors (bad config, missing adapter), not on candidate failures.

---

## Metrics Exercised

- `latency_ms` (builtin)
- `cost_usd` (llm adapter builtin, if applicable)

---

## Scorer Strategy

`weighted` — configured in suite YAML

---

## Happy Path Steps

```
1. CI justfile step:
     WINNER=$(ben run --suite codebase-indexing --format json --quiet | jq -r '.winner')
     echo "Winner: $WINNER"

2. ben runs suite; all output to stdout as single JSON object.
3. jq extracts .winner string ("xray" or "grep").
4. CI step passes; WINNER variable available for downstream steps.
5. Optional: CI emits WINNER as a job output / annotation.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| Candidate fails (non-zero exit) | Exit 0; JSON includes candidate error field; winner from remaining candidates |
| Ben config missing adapter | Exit 1; stderr: adapter error message; stdout: empty |
| `--format` value invalid | Exit 1; stderr: "unknown format: <val>"; stdout: empty |
| `--quiet` not set; progress to stderr | Stderr may have logs; stdout still valid JSON |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0104_test.go`
**Test func:** `TestUS_BEN_0104_FormatJSONInCI`

Asserts:
- `ben run --suite <name> --format json --quiet` exits 0.
- Stdout bytes parse as `map[string]interface{}` without error.
- Stdout contains no ANSI escape sequences.
- Top-level key `winner` exists and is a non-empty string.
- Stderr is empty (len == 0) when `--quiet` flag present.
- `echo '<stdout>' | jq -r '.winner'` returns same value as `result["winner"]`.
- `ben run --format bad-value` exits 1; stderr non-empty; stdout empty.
