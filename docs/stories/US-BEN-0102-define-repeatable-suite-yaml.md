# US-BEN-0102 — Define repeatable suite in YAML; re-run across repo changes

**ID:** US-BEN-0102
**Title:** Define repeatable suite in YAML and re-run across repo changes
**Persona:** Platform Engineer
**Trigger:** Engineer wants a stable, version-controlled benchmark spec that can be re-run
after code changes to detect regressions; checked into `.ben/suites/`.

---

## Acceptance Criteria

1. Suite YAML at `.ben/suites/<name>.yaml` is discovered and loaded by `ben run --suite <name>`.
2. `suite_version` from YAML appears in result JSON as `suite_version`.
3. Re-running the same suite after a repo change produces a new `run_id` with a new
   `timestamp`; prior run is not mutated.
4. Result stored locally; `ben query --suite <name> --last 2` shows both runs.
5. Exit code 0 on each successful run.
6. Spec validation error (missing required field) produces exit 1 with descriptive stderr.

---

## Metrics Exercised

- `latency_ms` (builtin)
- `exit_code` (builtin)

---

## Scorer Strategy

`weighted` — as declared in suite YAML

---

## Happy Path Steps

```
1. Engineer creates .ben/suites/codebase-indexing.yaml with:
     name: codebase-indexing
     version: 1
     task:
       prompt: "Find all HTTP handler functions"
       input:
         repo: ./testdata/sample-repo
     candidates:
       - name: xray
         adapter: cli
         cmd: "xray explore --search {{input.prompt}} --path {{input.repo}}"
       - name: grep
         adapter: cli
         cmd: "grep -r 'func.*Handler' {{input.repo}}"
     metrics: [latency_ms, exit_code]
     scorer:
       strategy: weighted
       weights: {latency_ms: 0.5, exit_code: 0.5}

2. Engineer runs: ben run --suite codebase-indexing --format json
3. Ben loads spec; executes candidates; stores result as run-A.
4. Engineer modifies repo code; runs again.
5. Ben produces new run-B with new run_id, same suite name + version.
6. ben query --suite codebase-indexing --last 2 returns both run-A and run-B.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| YAML missing `name` field | Exit 1; stderr: "spec validation error: name is required" |
| Suite name not found in any suite dir | Exit 1; stderr: "suite not found: <name>" |
| `input.repo` path does not exist | Exit 0; candidate error: "repo path not found"; winner based on remaining metrics |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0102_test.go`
**Test func:** `TestUS_BEN_0102_RepeatableSuiteYAML`

Asserts:
- First `ben run --suite codebase-indexing --format json` exits 0.
- Result JSON: `suite == "codebase-indexing"`, `suite_version == 1`.
- `run_id` is non-empty string.
- Second run produces different `run_id` and later `timestamp`.
- `ben query --suite codebase-indexing --last 2 --format json` returns array of length 2.
- Both result objects have distinct `run_id` values.
- Invalid YAML (missing `name`) exits 1; stderr contains "validation error".
