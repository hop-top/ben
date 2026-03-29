# US-BEN-0202 — Compare two deps with single inline command (no spec file)

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0202
**Title:** Compare two deps with single inline command; use --task, --candidates,
           --metric, --scorer inline flags
**Persona:** Solo Developer
**Trigger:** Dev needs to pick between two CLI tools; wants one-liner; no YAML.

---

## Acceptance Criteria

1. `ben run --task <desc> --candidates A,B --metric latency_ms,exit_code
   --scorer single:latency_ms` completes exit 0.
2. Both candidates executed; results include `latency_ms` and `exit_code` columns.
3. Scorer `single:latency_ms` applied; lower latency_ms = rank 1.
4. `winner` field in result matches rank-1 candidate name.
5. No `.ben/` or spec file created or required.

---

## Metrics Exercised

- `latency_ms`
- `exit_code`

---

## Scorer Strategy

- `single:latency_ms` — lowest value wins; passed via `--scorer` flag

---

## Happy Path

```pseudocode
ben run \
  --task "count lines in file" \
  --candidates "wc -l ./testdata/sample.txt","cat ./testdata/sample.txt | wc -l" \
  --metric latency_ms,exit_code \
  --scorer single:latency_ms

// stdout: table; 2 rows; latency_ms + exit_code columns; winner highlighted
// exit: 0
```

---

## Failure Path

- **Unknown scorer name:** `--scorer unknown:metric`; expected: exit 1,
  stderr `error: unknown scorer strategy "unknown"`.
- **Metric not captured by adapter:** e.g. `--metric tokens` on CLI adapter;
  expected: exit 1 or warning: `metric "tokens" unavailable for cli adapter`.
- **All candidates exit non-zero:** result still emitted; all `exit_code` != 0;
  winner selected by latency_ms anyway; overall exit 0.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0202_test.go
func:      TestUS_BEN_0202_InlineCompare
```

Asserts:
- Command exits 0.
- stdout (table mode) contains both candidate names.
- `--format json` output: `candidates[*].metrics.latency_ms` > 0 for both.
- `--format json` output: `candidates[*].metrics.exit_code` == 0 for both.
- `scorer.strategy` == `"single"`, `scorer.weights` key == `"latency_ms"`.
- `winner` is non-null and matches one of the two candidate names.
- No file written under `.ben/` or `~/.local/share/ben/suites/`.
