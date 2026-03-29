# Story ↔ Phase Task Coverage Matrix

*author: $USER | 2026-03-28*

Source of truth: `docs/plans/2026-03-28-ben-design.md` (T-0001..T-0017) +
`docs/stories/US-BEN-{01,02,04}xx`.

---

## Phase Task → Stories

### Phase 1: Core + CLI Adapter

| Task   | Title (abbrev)                     | Stories covered                                          |
|--------|------------------------------------|----------------------------------------------------------|
| T-0001 | Repo scaffold; global flags        | US-BEN-0201, US-BEN-0107, US-BEN-0104                   |
| T-0002 | Spec loader (`internal/spec`)      | US-BEN-0102, US-BEN-0109                                 |
| T-0003 | CLI adapter (`internal/adapter`)   | US-BEN-0101, US-BEN-0201, US-BEN-0202, US-BEN-0208      |
| T-0004 | Built-in metrics (latency/exit/sz) | US-BEN-0101, US-BEN-0208, US-BEN-0109, US-BEN-0111      |
| T-0005 | Scorers (single/weighted/raw)      | US-BEN-0110, US-BEN-0111, US-BEN-0209, US-BEN-0202      |
| T-0006 | Storage (`internal/storage`)       | US-BEN-0102, US-BEN-0103, US-BEN-0108, US-BEN-0210      |
| T-0007 | Reporters (json/table/yaml)        | US-BEN-0104, US-BEN-0203, US-BEN-0207                   |
| T-0008 | `ben run` command                  | US-BEN-0101..0111, US-BEN-0201, US-BEN-0202, US-BEN-0208|
| T-0009 | `ben compare` + `ben query`        | US-BEN-0103, US-BEN-0105, US-BEN-0204                   |

### Phase 2: LLM Adapter + Quality Metrics

| Task   | Title (abbrev)                     | Stories covered                                          |
|--------|------------------------------------|----------------------------------------------------------|
| T-0010 | LLM adapter (`internal/adapter`)   | US-BEN-0106 (partial — adapter side)                    |
| T-0011 | LLM metrics (tokens/cost/judge)    | US-BEN-0106, US-BEN-0101 (quality_score path)            |

### Phase 3: Plugin System

| Task   | Title (abbrev)                     | Stories covered                                          |
|--------|------------------------------------|----------------------------------------------------------|
| T-0012 | Binary plugin discovery + stdio    | US-BEN-0402, US-BEN-0403, US-BEN-0406                   |
| T-0013 | `ben suite list` / `ben suite show`| US-BEN-0205, US-BEN-0206                                 |

### Phase 4: Registry

| Task   | Title (abbrev)                     | Stories covered                                          |
|--------|------------------------------------|----------------------------------------------------------|
| T-0014 | Local registry index               | US-BEN-0108 (push:false / local-only)                   |
| T-0015 | Registry push/pull commands        | US-BEN-0108 (push:true path)                             |

### Phase 5: Eva Integration

| Task   | Title (abbrev)                     | Stories covered                                          |
|--------|------------------------------------|----------------------------------------------------------|
| T-0016 | Eva adapter (`internal/adapter`)   | **NONE** — no story exists for eva adapter               |
| T-0017 | Eva suite specs (MMLU/GSM8K/HellaS)| **NONE** — no story exists for eva suite specs           |

---

## Story → Phase Task(s)

### Series 01xx — Platform Engineer

| Story        | Title (abbrev)                        | Implementing tasks         |
|--------------|---------------------------------------|----------------------------|
| US-BEN-0101  | Compare two CLI tools (latency+qual)  | T-0001,T-0003,T-0004,T-0005,T-0007,T-0008,T-0011 |
| US-BEN-0102  | Repeatable suite YAML                 | T-0002,T-0006,T-0008       |
| US-BEN-0103  | Query last N runs; spot regressions   | T-0006,T-0009              |
| US-BEN-0104  | --format json in CI pipeline          | T-0001,T-0007,T-0008       |
| US-BEN-0105  | Compare two run IDs (ben compare)     | T-0009                     |
| US-BEN-0106  | quality_score via llm_judge           | T-0010,T-0011              |
| US-BEN-0107  | --config override per environment     | T-0001,T-0002              |
| US-BEN-0108  | registry.push: false; no remote push  | T-0006,T-0014,T-0015       |
| US-BEN-0109  | Verify full result schema fields      | T-0002,T-0004,T-0005,T-0007,T-0008 |
| US-BEN-0110  | Scorer single:<metric>; one-dim rank  | T-0005,T-0008              |
| US-BEN-0111  | Scorer raw; winner == null            | T-0004,T-0005,T-0008       |

### Series 02xx — Solo Developer

| Story        | Title (abbrev)                        | Implementing tasks         |
|--------------|---------------------------------------|----------------------------|
| US-BEN-0201  | Install ben; run first benchmark      | T-0001,T-0003,T-0004,T-0005,T-0007,T-0008 |
| US-BEN-0202  | Inline compare; no spec file          | T-0003,T-0005,T-0008       |
| US-BEN-0203  | Human-readable table output (default) | T-0007,T-0008              |
| US-BEN-0204  | Re-run saved suite; see diff          | T-0002,T-0006,T-0008,T-0009 |
| US-BEN-0205  | ben suite list; discover suites       | T-0013                     |
| US-BEN-0206  | ben suite show; inspect before run    | T-0013                     |
| US-BEN-0207  | --format yaml output shape            | T-0007,T-0008              |
| US-BEN-0208  | exit_code + output_size metrics       | T-0003,T-0004,T-0008       |
| US-BEN-0209  | Scorer single:latency_ms; speed wins  | T-0005,T-0008              |
| US-BEN-0210  | Storage dirs created on first run     | T-0006                     |

### Series 04xx — OSS Go Developer (Contributor)

| Story        | Title (abbrev)                        | Implementing tasks         |
|--------------|---------------------------------------|----------------------------|
| US-BEN-0401  | Explore codebase; find interfaces     | T-0002*,T-0003*,T-0005* (partial — see GAP-01) |
| US-BEN-0402  | Binary adapter plugin; PATH discovery | T-0012                     |
| US-BEN-0403  | Reporter plugin; stdio JSON protocol  | T-0012                     |
| US-BEN-0404  | Run test suite locally; all green     | T-0001* (partial — see GAP-02) |
| US-BEN-0405  | Open PR; CI green; merged ≤2 cycles   | **NONE** (see GAP-03)      |
| US-BEN-0406  | Config-declared scorer plugin         | T-0012                     |
| US-BEN-0407  | New built-in metric via interface     | T-0004* (partial — see GAP-04) |

*partial: existing task covers implementation but not the specific deliverable (docs/CI/registry).

---

## Coverage Gaps

### GAP-01 — No task for `docs/contributing.md` + interface godoc (→ US-BEN-0401)

US-BEN-0401 AC5/6 require `docs/contributing.md` cross-referencing all three interfaces and
`go doc` exits 0 with interface definition. T-0002/T-0003/T-0005 define the interfaces in
code but no task produces the contributing doc.
**New task needed:** P3 (post-interface-impl); tags `ben`, `phase3`.

---

### GAP-02 — No task for hermetic Taskfile.yml test gate (→ US-BEN-0404)

US-BEN-0404 AC1–7 require `task test`, `task lint`, `go test ./...` < 60s, coverage gate.
T-0001 scaffolds `Taskfile.yml` with build/test/lint targets but no task explicitly owns
the hermeticity contract (no external deps, temp binaries compiled in-test, coverage report).
**New task needed:** P1 (before Phase 2); tags `ben`, `phase1`.

---

### GAP-03 — No task for CI pipeline / GitHub Actions (→ US-BEN-0405)

US-BEN-0405 requires `.github/workflows/ci.yml` with `go test`, `go vet`/golangci-lint,
PR trigger, go@latest on linux. No existing task creates this file.
**New task needed:** P1 (after scaffold); tags `ben`, `phase1`.

---

### GAP-04 — No task for metrics registry / Metric interface + auto-discovery (→ US-BEN-0407)

US-BEN-0407 AC3/8 require `internal/metrics/registry.go` (Register fn, auto-discovery by
name, no cmd/ change). T-0004 implements specific built-in metrics but does not define the
registry or the exported `Metric` interface file separately. The `Adapter`, `Metric`,
`Scorer` interfaces must be exported + godoc'd per US-BEN-0401 and US-BEN-0407.
**New task needed:** P1 (before T-0004); tags `ben`, `phase1`.

---

### GAP-05 — No stories for Phase 5 (T-0016, T-0017)

T-0016 (eva adapter) and T-0017 (eva suite specs) have zero story coverage. Phase 5
deliverables are unverifiable without stories that define acceptance criteria.
Stories needed: US-BEN-05xx series (eva adapter + eva suite specs).
**Deferred:** story authoring is a separate concern; flag for product backlog.

---

## Summary

| Dimension             | Count |
|-----------------------|-------|
| Phase tasks total     | 17    |
| Phase tasks with coverage | 15 |
| Phase tasks with NO coverage | 2 (T-0016, T-0017) |
| Stories total         | 29    |
| Stories fully covered | 24    |
| Stories with gap      | 5 (0401,0402*,0403*,0404,0405,0406*,0407) |
| Gaps requiring new tasks | 4 (GAP-01..04) |
| Gaps requiring new stories | 1 (GAP-05: Phase 5) |

*0402,0403,0406 are fully covered by T-0012; only 0401,0404,0405,0407 have partial/no coverage.
