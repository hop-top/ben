# US-BEN-0108 — Run suite with registry.push: false; confirm no remote push

**ID:** US-BEN-0108
**Title:** Run suite with registry.push: false and confirm no remote push occurs
**Persona:** Platform Engineer
**Trigger:** Engineer wants local-only runs during development; must be sure no results are
sent to the shared registry; confirms opt-in semantics of registry push.

---

## Acceptance Criteria

1. Suite YAML with `registry: push: false` does not trigger any network call to registry.
2. Run completes and result is stored locally only.
3. No HTTP request is made to registry endpoint during or after the run.
4. `ben query --suite <name> --last 1` returns the local result.
5. Explicitly passing `--registry.push=false` on CLI also suppresses push.
6. If `push: false` and a prior `ben registry push` was queued, this run does not push.

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
1. Engineer creates .ben/suites/local-only.yaml with:
     name: local-only
     version: 1
     registry:
       push: false
     ...rest of spec...

2. Engineer runs:
     ben run --suite local-only --format json

3. Ben executes suite; stores result in ~/.local/share/ben/runs/<run-id>.json.
4. No outbound HTTP call to registry endpoint.
5. Result available via: ben query --suite local-only --last 1 --format json
6. Exit 0; result JSON contains run_id, winner, candidates.
```

---

## Failure Path + Expected Behavior

| Failure | Expected |
|---------|----------|
| Registry endpoint unreachable (and push: true) | Exit 0 for run; post-run push error logged to stderr |
| `push: false` but env var `BEN_REGISTRY_PUSH=true` | CLI/file setting takes precedence; no push |
| Local storage write fails (disk full) | Exit 1; stderr: "storage error: ..."; no push attempt |

---

## E2E Test Spec

**File:** `tests/e2e/stories/US_BEN_0108_test.go`
**Test func:** `TestUS_BEN_0108_RegistryPushFalse`

Asserts:
- Test intercepts HTTP calls via a local stub server; stub records any incoming requests.
- `ben run --suite local-only --format json` exits 0.
- Stub server received 0 requests (no push).
- Local run file exists at `$BEN_DATA_DIR/runs/<run_id>.json`.
- `ben query --suite local-only --last 1 --format json` returns array of length 1.
- When suite has `registry: push: true` and stub reachable: stub receives exactly 1 POST.
- When `--registry.push=false` CLI flag overrides `push: true` in YAML: stub receives 0 requests.
