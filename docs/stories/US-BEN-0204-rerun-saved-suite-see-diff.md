# US-BEN-0204 — Re-run saved suite after updating a dep; see diff

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0204
**Title:** Re-run saved suite after updating a dep; see diff
**Persona:** Solo Developer
**Trigger:** Dev updated a dep version; wants to confirm it's faster/better than prior run.

---

## Acceptance Criteria

1. A suite YAML in `.ben/suites/` can be re-run with `ben run --suite <name>`.
2. Two runs of same suite produce distinct `run_id` values stored in local storage.
3. `ben compare <run-a> <run-b>` prints diff: per-candidate delta for each metric.
4. Diff shows direction (improved / regressed / unchanged) per metric per candidate.
5. Both runs retrievable after session restart (persisted in `~/.local/share/ben/` or
   `.ben/runs/`).

---

## Metrics Exercised

- `latency_ms`
- `exit_code`

---

## Scorer Strategy

- `single:latency_ms` (suite-configured)

---

## Happy Path

```pseudocode
// suite file: .ben/suites/dep-compare.yaml
// (two candidates: old-version, new-version of same tool)

ben run --suite dep-compare
// stores run_id: RUN_A

// (simulate dep update — swap cmd in suite or use different binary path)

ben run --suite dep-compare
// stores run_id: RUN_B

ben compare RUN_A RUN_B
// stdout: diff table; each candidate row shows delta latency_ms
// exit: 0
```

---

## Failure Path

- **Non-existent run_id:** `ben compare BAD_ID RUN_B`; expected: exit 1,
  stderr `error: run "BAD_ID" not found`.
- **Same run_id twice:** `ben compare RUN_A RUN_A`; expected: exit 1,
  stderr `error: cannot compare a run to itself`.
- **Suite file deleted after run:** `ben compare` still works — runs stored independently
  of suite file existence.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0204_test.go
func:      TestUS_BEN_0204_RerunSavedSuiteDiff
```

Asserts:
- Two successive `ben run --suite <name>` invocations each exit 0.
- Each emits a distinct `run_id` (captured from `--format json` output).
- `ben compare <run-a> <run-b>` exits 0.
- Compare stdout contains both `run_id` values as reference labels.
- Compare stdout contains `latency_ms` delta column.
- `ben compare NONEXISTENT RUN_B` exits 1 and stderr contains `not found`.
