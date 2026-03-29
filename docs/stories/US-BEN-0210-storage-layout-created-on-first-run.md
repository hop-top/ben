# US-BEN-0210 — Verify global and project-local storage created on first run

*author: $USER | 2026-03-28*

---

## Story

**ID:** US-BEN-0210
**Title:** Verify global (~/.local/share/ben/) and project-local (.ben/) storage
           created on first run
**Persona:** Solo Developer
**Trigger:** Dev runs ben for the first time; expects storage dirs to auto-init with
             no manual setup.

---

## Acceptance Criteria

1. `~/.local/share/ben/runs/` dir exists after first `ben run` invocation.
2. `~/.local/share/ben/suites/` dir exists after first run (may be empty).
3. `.ben/runs/` dir created in CWD if CWD is a git repo root; absent otherwise.
4. Run result JSON file written to `~/.local/share/ben/runs/<run-id>.json`.
5. Project-local `.ben/` created only when project context detected (git repo present).
6. Repeated runs do not fail if dirs already exist (idempotent init).

---

## Metrics Exercised

- `latency_ms` (trigger metric to force a real run)

---

## Scorer Strategy

- `single:latency_ms`

---

## Happy Path

```pseudocode
// fresh env: $XDG_DATA_HOME not set; CWD is a git repo

ben run \
  --task "echo test" \
  --candidates "echo a" \
  --metric latency_ms \
  --scorer single:latency_ms

// post-run checks:
stat ~/.local/share/ben/runs/          // exists
stat ~/.local/share/ben/suites/        // exists
ls   ~/.local/share/ben/runs/          // one *.json file present
stat .ben/runs/                        // exists (git repo detected)

// second run (idempotent):
ben run --task "echo test" --candidates "echo a" --metric latency_ms
// exits 0; no "already exists" errors
```

---

## Failure Path

- **XDG_DATA_HOME set to custom path:** storage created under `$XDG_DATA_HOME/ben/`
  instead of `~/.local/share/ben/`; same subdirectory structure.
- **CWD not a git repo:** `.ben/` NOT created; runs stored only in global path; exit 0.
- **Filesystem permission denied on global path:** exit 1;
  stderr: `error: cannot create storage dir: permission denied`.

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0210_test.go
func:      TestUS_BEN_0210_StorageLayoutCreated
```

Asserts:
- After `ben run` in a temp git repo dir (init'd by test), `~/.local/share/ben/runs/`
  exists (or `$XDG_DATA_HOME/ben/runs/` if env set).
- Exactly one JSON file present in runs dir; filename matches returned `run_id`.
- JSON file parses correctly: `run_id` matches filename stem.
- `.ben/runs/` exists inside the temp git repo dir.
- Second identical `ben run` exits 0 with no error about existing dirs.
- In a non-git temp dir, `.ben/` is NOT created after `ben run`.
