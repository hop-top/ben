# US-BEN-0203 — View results as human-readable table (default output)

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0203
**Title:** View results as human-readable table (default output)
**Persona:** Solo Developer
**Trigger:** Dev runs ben without `--format`; expects readable terminal output, not JSON.

---

## Acceptance Criteria

1. `ben run ...` with no `--format` flag prints table to stdout.
2. Table has columns: `candidate`, one column per metric, `score`, `rank`.
3. Winner row visually distinct (e.g. asterisk prefix or `[winner]` label).
4. No raw JSON or YAML emitted to stdout.
5. `--format table` explicit flag produces identical output to default.
6. Table renders cleanly in 80-column terminal (no wrapping of core columns).

---

## Metrics Exercised

- `latency_ms` — primary column in table

---

## Scorer Strategy

- `single:latency_ms` (illustrative; table format independent of scorer)

---

## Happy Path

```pseudocode
ben run \
  --task "list files" \
  --candidates "ls -la","find . -maxdepth 1" \
  --metric latency_ms

// stdout (example shape — not literal):
// candidate          latency_ms   score   rank
// ---------------    ----------   -----   ----
// * ls -la           12           1.000   1
//   find . -maxd..   18           0.667   2
//
// exit: 0
```

---

## Failure Path

- **No TTY (piped):** `ben run ... | cat`; expected: table still emitted (not JSON
  auto-switch); dev must pass `--format json` explicitly for machine-readable output.
- **Single candidate:** table with one row; no rank comparison; `winner` == that candidate.
- **Long candidate name:** name truncated with `...` in table; full name in JSON output.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0203_test.go
func:      TestUS_BEN_0203_TableOutput
```

Asserts:
- stdout does NOT start with `{` or `---` (not JSON/YAML).
- stdout contains string `candidate` (header row present).
- stdout contains string `latency_ms` (metric column header).
- stdout contains string `rank` (rank column header).
- stdout contains a line matching rank `1` marker for winner candidate.
- exit code == 0.
- `--format table` flag produces byte-identical stdout to no-flag invocation.
