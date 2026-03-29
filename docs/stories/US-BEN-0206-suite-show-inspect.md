# US-BEN-0206 — Use ben suite show <name> to inspect suite before running

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0206
**Title:** Use ben suite show <name> to inspect suite before running
**Persona:** Solo Developer
**Trigger:** Dev found a suite via `ben suite list`; wants to review its config before
             committing to a run.

---

## Acceptance Criteria

1. `ben suite show <name>` exits 0 and prints suite config to stdout.
2. Output includes: name, description, version, task prompt, candidates list,
   metrics list, scorer strategy.
3. Candidate adapter type and command template shown per candidate.
4. `--format yaml` flag dumps raw YAML spec.
5. Unknown suite name → exit 1, stderr `error: suite "<name>" not found`.

---

## Metrics Exercised

- (none — inspection command, no benchmark execution)

---

## Scorer Strategy

- (not applicable; shows configured scorer from suite spec)

---

## Happy Path

```pseudocode
ben suite show dep-compare
// stdout (human table/structured):
// Suite:       dep-compare
// Description: Latency diff across dep versions
// Version:     1
// Task:        count lines in file
// Candidates:
//   wc-l    adapter=cli   cmd="wc -l {{input.file}}"
//   cat-wc  adapter=cli   cmd="cat {{input.file}} | wc -l"
// Metrics:     latency_ms, exit_code
// Scorer:      single:latency_ms
//
// exit: 0

ben suite show dep-compare --format yaml
// stdout: raw YAML spec content
// exit: 0
```

---

## Failure Path

- **Suite not found:** `ben suite show no-such-suite`; exit 1;
  stderr: `error: suite "no-such-suite" not found`.
- **Malformed YAML:** exit 1; stderr: `error: failed to parse suite "bad-suite": <detail>`.
- **Ambiguous name (same name in global + project):** project-local takes precedence;
  stdout includes note `(project-local)` vs `(global)`.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0206_test.go
func:      TestUS_BEN_0206_SuiteShow
```

Asserts:
- `ben suite show <fixture-suite-name>` exits 0.
- stdout contains `name:` field matching suite name.
- stdout contains `candidates` section with at least one entry.
- stdout contains `metrics` list.
- `--format yaml` output parses as valid YAML with `name` key.
- `ben suite show nonexistent` exits 1 and stderr matches `not found`.
