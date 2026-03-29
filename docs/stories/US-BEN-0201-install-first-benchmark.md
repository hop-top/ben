# US-BEN-0201 — Install ben; run first benchmark in under 5 min

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0201
**Title:** Install ben; run first benchmark in under 5 min
**Persona:** Solo Developer
**Trigger:** Dev wants to compare two CLI tools; installs ben cold; no prior config.

---

## Acceptance Criteria

1. `ben` binary available on PATH within 2 min of starting install (go install or brew).
2. `ben run --task "find files" --candidates grep,find --metric latency_ms` completes with
   exit code 0.
3. Table printed to stdout; two candidate rows visible; latency_ms column populated.
4. Total wall time from fresh shell to first result output < 5 min.
5. No suite YAML required; inline flags sufficient.

---

## Metrics Exercised

- `latency_ms` — wall time per candidate execution

---

## Scorer Strategy

- `single:latency_ms` (implicit default when one metric given)

---

## Happy Path

```pseudocode
go install hop.top/ben@latest
// or: brew install hop-top/tap/ben

ben run \
  --task "find all .go files in current dir" \
  --candidates "grep -r --include='*.go' ." \
  --candidates "find . -name '*.go'" \
  --metric latency_ms

// stdout: table with columns: candidate | latency_ms | score | rank
// exit: 0
```

---

## Failure Path

- **Binary not on PATH after install:** user sees `command not found: ben`; expected: install
  docs include `export PATH` snippet; ben emits no output (not running yet).
- **No candidates given:** `ben run --task "..." --metric latency_ms` with no `--candidates`;
  expected: exit 1, stderr `error: at least one candidate required`.
- **Candidate command fails:** exit_code != 0 on one candidate; expected: that candidate shows
  `error: exit 1` in result row; other candidates still scored; overall exit 0.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0201_test.go
func:      TestUS_BEN_0201_InstallFirstBenchmark
```

Asserts:
- `ben run --task "echo hello" --candidates "echo a","echo b" --metric latency_ms` exits 0.
- stdout contains a table with exactly 2 candidate rows.
- each row has a non-zero `latency_ms` value.
- `winner` field in JSON re-run (`--format json`) is non-empty string.
- stderr is empty (no unexpected log noise on happy path).
