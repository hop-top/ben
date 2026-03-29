# US-BEN-0207 — Run with --format yaml; verify YAML output shape

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0207
**Title:** Run with --format yaml; verify YAML output shape
**Persona:** Solo Developer
**Trigger:** Dev wants to pipe ben output into a YAML-consuming script or config tool.

---

## Acceptance Criteria

1. `ben run ... --format yaml` exits 0 and emits valid YAML to stdout.
2. YAML root keys present: `run_id`, `suite`, `timestamp`, `scorer`, `candidates`, `winner`.
3. Each candidate entry has: `name`, `metrics`, `score`, `rank`.
4. `metrics` map contains at least the requested metrics with numeric values.
5. No JSON or table output mixed into stdout.
6. stderr clean (no log noise) on happy path.

---

## Metrics Exercised

- `latency_ms`
- `exit_code`

---

## Scorer Strategy

- `single:latency_ms`

---

## Happy Path

```pseudocode
ben run \
  --task "echo test" \
  --candidates "echo a","echo b" \
  --metric latency_ms,exit_code \
  --scorer single:latency_ms \
  --format yaml

// stdout (YAML):
// run_id: 01HX...
// suite: ""
// timestamp: 2026-03-28T...
// scorer:
//   strategy: single
//   weights:
//     latency_ms: 1.0
// candidates:
//   - name: echo a
//     metrics:
//       latency_ms: 8
//       exit_code: 0
//     score: 1.0
//     rank: 1
//   - name: echo b
//     ...
// winner: echo a
//
// exit: 0
```

---

## Failure Path

- **--format yaml with --quiet:** YAML still emitted to stdout; stderr suppressed; exit 0.
- **All candidates fail:** YAML still emitted; each candidate has `error` key set;
  `winner` is null.
- **Invalid --format value:** `--format toml`; exit 1;
  stderr: `error: unknown format "toml"; valid: table, json, yaml`.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0207_test.go
func:      TestUS_BEN_0207_FormatYAML
```

Asserts:
- Exit code == 0.
- stdout parses as valid YAML (use `gopkg.in/yaml.v3` unmarshal in test).
- Parsed struct has `RunID` non-empty string.
- `Candidates` slice length == number of `--candidates` flags given.
- Each candidate `Metrics["latency_ms"]` > 0.
- Each candidate `Metrics["exit_code"]` is 0.
- `Winner` field is non-empty string matching one candidate name.
- stdout does NOT contain `{` as first non-whitespace byte (not JSON).
