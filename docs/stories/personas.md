# Ben — Shared Personas
*author: $USER | 2026-03-28*

---

## Selected Personas

### 1. Solo Developer (Indie Hacker / Side Project)

**Rationale:** Primary human user of ben. Runs one-off comparisons to pick the best tool or
impl for their project. Values fast setup, simple CLI, readable output.

**Commands used:**
- `ben run --task "<desc>" --candidates xray,grep --metric latency_ms`
- `ben compare <run-a> <run-b>`
- `ben query --suite <name> --last 5`

**Metrics/scorers they care about:**
- `latency_ms` — fast = good for interactive use
- `quality_score` — output usefulness matters more than speed sometimes
- `scorer: single:<metric>` or `weighted` with simple weights

**How they invoke ben:**
- Inline args from shell; no suite YAML initially
- `--format table` (human-readable default)
- Promoted to spec YAML once workflow repeats

**Acceptance criteria shape:**
- `ben run` with no suite file works via inline args
- Table output readable without post-processing
- `ben compare` surfaces winner clearly
- Time-to-first-result < 2 min on fresh install

---

### 2. Platform Engineer (SaaS Backend / Infra Owner)

**Rationale:** Power user; runs repeatable suites in CI/pipelines; cares about trend data
across runs; shares results with team. Needs structured output for automation.

**Commands used:**
- `ben run --suite <name> --format json`
- `ben query --suite <name> --last 30`
- `ben registry push <run-id>` / `ben registry pull --suite <name>`
- `ben suite list` / `ben suite show <name>`

**Metrics/scorers they care about:**
- `latency_ms`, `cost_usd` — operational budget matters
- `quality_score` via LLM judge — for agent/tool comparisons
- `scorer: weighted` with cost/latency/quality trade-off configured per suite

**How they invoke ben:**
- Suite YAML checked into `.ben/suites/`
- Called from justfile / CI step; `--quiet` + `--format json` for pipeline use
- `ben registry push` after CI run to accumulate institutional memory

**Acceptance criteria shape:**
- Exit 0 on successful run even when candidates fail (failures in result)
- `--format json` emits valid JSON to stdout; errors only to stderr
- Suite spec versioned; `suite_version` in result schema
- Registry push/pull roundtrip works; pulled baselines visible in compare

---

### 3. OSS Go Developer (Contributor)

**Rationale:** Contributes new adapters, metrics, or scorer plugins to ben. Needs clean
interfaces, testable code, and reproducible runs for regression checks.

**Commands used:**
- `ben run --suite <name>` — smoke-test contributed adapter end-to-end
- `ben run --candidates <new-adapter> --metric latency_ms,exit_code`
- `just test ./internal/adapter/...` (via justfile, not ben directly)

**Metrics/scorers they care about:**
- `exit_code`, `latency_ms`, `output_size` — builtin metrics validate adapter correctness
- `scorer: raw` — see all metrics unscored during development

**How they invoke ben:**
- Direct CLI during adapter development; `--format json` to inspect raw result schema
- Integration tests via `tests/e2e/` in CI

**Acceptance criteria shape:**
- Binary plugin protocol fully documented: stdin/stdout JSON contract
- `ben-adapter-*` discovery works with adapter binary on PATH
- New adapter passes `ben run` without modifying ben source
- `go test ./...` green before and after adapter addition

---

## Excluded Personas

### Automation Builder (Email-to-Workflow Integrator)

**Excluded.** Role is specific to MXHook inbound email + webhook routing. Ben is a
general-purpose benchmarking CLI — no inbound email, no webhook delivery, no SaaS dashboard.
Zero overlap with ben's use cases; persona's pain points (payload schema, DLQ replay,
plus-addressing) are irrelevant to benchmarking.

---

### SaaS Series A Business

**Excluded.** Organizational buyer for MXHook EE. Ben is a developer CLI tool, not a
commercial product with a buying cycle, EE license tier, or multi-team deployment story
(at this stage). No ben use case maps to procurement, SLA negotiation, or vendor evaluation.
Revisit if ben gains a hosted/team registry offering.
