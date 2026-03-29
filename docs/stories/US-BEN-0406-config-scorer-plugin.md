# US-BEN-0406 — Add config-declared scorer plugin (e.g. pareto); verify ben loads from ben.yaml

*author: $USER | 2026-03-28*

---

## Identity

| Field     | Value                                                       |
|-----------|-------------------------------------------------------------|
| ID        | US-BEN-0406                                                 |
| Title     | Add config-declared scorer plugin; verify loaded from ben.yaml |
| Persona   | OSS Go Developer (Contributor)                              |
| Trigger   | Contributor wants a custom scoring strategy not built-in    |

---

## Trigger

Contributor implements a `pareto` scorer (Pareto frontier ranking). They declare it in
`ben.yaml` under `plugins.scorers`; expect ben to load and invoke it without source changes.

---

## Acceptance Criteria

1. `ben.yaml` supports `plugins.scorers[].name` + `plugins.scorers[].import` fields.
2. `import` value is a binary name resolved on PATH (same mechanism as adapter/reporter).
3. `ben run --suite pareto-test.yaml` with `scorer: {strategy: pareto}` loads the plugin.
4. Ben sends scorer input via stdin; scorer returns a `float64` score per candidate via stdout.
5. Result JSON shows `score` field populated from plugin scorer; `winner` field set correctly.
6. If scorer binary missing, `ben run` exits 1 with "scorer plugin 'pareto' not found".
7. Built-in scorers (`single`, `weighted`, `raw`) not affected by config-declared plugins.
8. Plugin loaded once per run; not re-spawned per candidate (batch protocol).

---

## Interface Contract Referenced

Config-declared scorer plugin protocol:

```
ben.yaml declaration:
  plugins:
    scorers:
      - name: pareto
        import: ben-plugin-pareto   # binary on PATH

stdin  (ben → scorer): {"candidates":[{"name":"...","metrics":{...}},...]}
stdout (scorer → ben): {"scores":{"xray":0.91,"grep":0.31}}
```

Scorer binary must exit 0; non-zero treated as scorer error; run exits 1.

---

## Test Requirements

- Unit: `tests/unit/plugin_config_test.go` — parse `ben.yaml` scorer block; verify loader
  resolves binary name and constructs correct stdin payload.
- E2E: compile minimal scorer binary; declare in `ben.yaml`; run ben; verify scores in result.
- `go test ./internal/plugin/...` passes hermetically.

---

## Happy Path Steps

1. Contributor writes `ben-plugin-pareto` binary; reads JSON candidates; writes scores JSON.
2. Builds binary: `go build -o /tmp/ben-plugin-pareto ./cmd/pareto`
3. Adds to `~/.config/ben/ben.yaml`:
   ```yaml
   plugins:
     scorers:
       - name: pareto
         import: ben-plugin-pareto
   ```
4. Creates suite YAML with `scorer: {strategy: pareto}`.
5. `PATH=/tmp:$PATH ben run --suite pareto-test.yaml --format json`
6. Result JSON: each candidate has `score` from pareto binary; `winner` set.

---

## Failure Path + Expected Behavior

| Failure                                    | Expected behavior                                         |
|--------------------------------------------|-----------------------------------------------------------|
| `import` binary not on PATH                | `ben run` exits 1; stderr: "scorer plugin 'pareto' not found" |
| Scorer returns partial scores (missing cand) | Ben logs warning; missing candidates get score 0         |
| Scorer returns invalid JSON                | Ben exits 1; stderr: "scorer plugin parse error"          |
| `ben.yaml` missing `plugins` block         | Ben uses built-in scorer only; no error                   |

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0406_test.go
func:      TestUS_BEN_0406_ConfigDeclaredScorerPlugin
setup:
  - compile scorer stub (testdata/scorer-stub/main.go):
      reads {"candidates":[...]} from stdin
      writes {"scores":{"a":0.9,"b":0.1}} to stdout
      exits 0
  - place as "ben-plugin-pareto-stub" in tmp dir; prepend to PATH
  - write ben.yaml in tmp config dir declaring scorer with import
  - write suite YAML with scorer strategy: pareto-stub; two candidates a, b
asserts:
  - ben run exits 0
  - result JSON: candidates["a"].score == 0.9
  - result JSON: candidates["b"].score == 0.1
  - result JSON: winner == "a"
  - remove binary; re-run → exits 1; stderr contains "not found"
```
