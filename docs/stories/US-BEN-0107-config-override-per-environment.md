# US-BEN-0107 — Run ben with --config to override config per environment

**ID:** US-BEN-0107
**Title:** Run ben with --config to override config per environment
**Persona:** Platform Engineer
**Trigger:** Engineer runs ben in multiple environments (local dev, CI, staging); each
environment needs different model, plugin, or registry settings without modifying checked-in
config files.

---

## Acceptance Criteria

1. `ben run --config <path>` loads the specified config file instead of default.
2. Settings in the overriding config take precedence over defaults.
3. Keys absent from the override config fall back to default config values.
4. `--config` accepts absolute and relative paths.
5. Non-existent config path exits 1 with stderr: "config not found: <path>".
6. Invalid YAML in config exits 1 with stderr: "config parse error: ...".
7. Overriding `registry.push: true` in CI config causes push after run.

---

## Metrics Exercised

- `latency_ms` (builtin)

---

## Scorer Strategy

`single:latency_ms` — as declared in the override config or suite

---

## Happy Path Steps

```
1. Default .ben/ben.yaml has: registry: push: false
2. Engineer creates ci.ben.yaml with:
     registry:
       push: true
     plugins:
       metrics:
         - name: quality_score
           type: llm_judge
           model: claude-sonnet-4-6-ci
           prompt: "Rate 0-1: {{output}}"

3. CI step runs:
     ben run --suite codebase-indexing --config ./ci.ben.yaml --format json --quiet

4. Ben merges ci.ben.yaml over defaults; uses CI model for judge.
5. registry.push == true from override; ben pushes run after completion.
6. Result JSON emitted to stdout; exit 0.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| `--config /nonexistent.yaml` | Exit 1; stderr: "config not found: /nonexistent.yaml" |
| Config file has invalid YAML | Exit 1; stderr: "config parse error: ..." |
| Config overrides unknown key | Logged as stderr warning; run continues |
| Config omits `plugins` entirely | Falls back to default plugin config; run continues |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0107_test.go`
**Test func:** `TestUS_BEN_0107_ConfigOverridePerEnvironment`

Asserts:
- Default run (no `--config`) uses base config values.
- `ben run --config ./testdata/ci.ben.yaml --format json` exits 0.
- When override sets `scorer.strategy: single` and `scorer.metric: latency_ms`,
  result `scorer.strategy == "single"`.
- `ben run --config /nonexistent.yaml` exits 1; stderr contains "config not found".
- `ben run --config ./testdata/bad.yaml` (invalid YAML) exits 1; stderr contains
  "config parse error".
- Config with unknown key logs warning to stderr; exit code still 0.
