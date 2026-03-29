# US-BEN-0208 — Observe exit_code and output_size metrics captured for CLI run

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0208
**Title:** Observe exit_code and output_size metrics captured for CLI run
**Persona:** Solo Developer
**Trigger:** Dev wants to verify a candidate produces correct output and doesn't crash,
             alongside timing.

---

## Acceptance Criteria

1. `--metric exit_code` captures the OS exit code of each candidate command (int).
2. `--metric output_size` captures byte count of candidate stdout (int, bytes).
3. Both metrics appear in result without any plugin configuration.
4. Failing candidate (exit != 0) has `exit_code` > 0 in metrics; `error` field set.
5. Candidate with zero output has `output_size` == 0 (not null).
6. Table and JSON output both show these metric values correctly.

---

## Metrics Exercised

- `exit_code` — OS exit code of candidate process
- `output_size` — byte count of candidate stdout

---

## Scorer Strategy

- `raw` — all metrics captured; no ranking applied; `winner` == null

---

## Happy Path

```pseudocode
ben run \
  --task "produce output" \
  --candidates "echo hello world","false" \
  --metric exit_code,output_size \
  --scorer raw

// result (JSON shape):
// candidates:
//   - name: "echo hello world"
//     metrics: {exit_code: 0, output_size: 12}   // "hello world\n" = 12 bytes
//     score: null
//     rank: null
//   - name: "false"
//     metrics: {exit_code: 1, output_size: 0}
//     error: "exit status 1"
//     score: null
// winner: null    // scorer=raw
//
// exit: 0
```

---

## Failure Path

- **Candidate hangs:** ben applies default timeout (configurable); candidate killed;
  `error: timeout` in result; `exit_code` = -1; overall exit 0.
- **Metric name typo:** `--metric exit_cde`; exit 1;
  stderr: `error: unknown metric "exit_cde"`.
- **scorer=raw with --format table:** table renders with null score/rank columns shown
  as `-`; no error.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0208_test.go
func:      TestUS_BEN_0208_ExitCodeOutputSize
```

Asserts:
- Command exits 0.
- `--format json` output: candidate `"echo hello world"` has `metrics.exit_code == 0`.
- `"echo hello world"` has `metrics.output_size == 12` (len("hello world\n")).
- Candidate `"false"` has `metrics.exit_code == 1`.
- Candidate `"false"` has `metrics.output_size == 0`.
- `winner` field in JSON == null (scorer is raw).
- `scorer.strategy` == `"raw"`.
