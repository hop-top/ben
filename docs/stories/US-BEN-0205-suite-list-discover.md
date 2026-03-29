# US-BEN-0205 — Use ben suite list to discover available suites

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0205
**Title:** Use ben suite list to discover available suites
**Persona:** Solo Developer
**Trigger:** Dev wants to know what pre-built or saved suites are available before running.

---

## Acceptance Criteria

1. `ben suite list` exits 0 and prints a list of suite names to stdout.
2. List includes suites from both `~/.local/share/ben/suites/` and `.ben/suites/` if present.
3. Each entry shows: name, description (if set), source (global / project-local).
4. Empty result (no suites installed) prints a friendly message, not an error; exit 0.
5. `--format json` flag emits JSON array of suite objects.

---

## Metrics Exercised

- (none — discovery command, no benchmark execution)

---

## Scorer Strategy

- (not applicable)

---

## Happy Path

```pseudocode
// assume one suite installed globally, one project-local

ben suite list
// stdout (table):
// name              source    description
// ----------------  --------  --------------------------------
// codebase-index   global    Compare xray vs grep orientation
// dep-compare      project   Latency diff across dep versions
//
// exit: 0

ben suite list --format json
// stdout: JSON array with name, description, source, path fields
// exit: 0
```

---

## Failure Path

- **No suites anywhere:** stdout: `No suites found. Create one in .ben/suites/ or
  ~/.local/share/ben/suites/.`; exit 0.
- **Malformed suite YAML:** that entry shows name + `[parse error]` in description;
  does not abort list; remaining valid suites shown; exit 0.
- **Project-local dir missing:** only global suites shown; no error for absent `.ben/`.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0205_test.go
func:      TestUS_BEN_0205_SuiteList
```

Asserts:
- `ben suite list` exits 0 in a dir with a pre-placed `.ben/suites/test.yaml`.
- stdout contains the suite name from `test.yaml`.
- stdout contains string `project` (source label).
- `ben suite list --format json` exits 0; stdout parses as valid JSON array.
- JSON array element has fields: `name` (string), `source` (string), `description` (string).
- `ben suite list` in a temp dir with no suites exits 0 and stdout is non-empty
  (friendly message, not empty string).
