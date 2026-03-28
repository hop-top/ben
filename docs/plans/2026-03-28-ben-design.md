# Ben Design
*2026-03-28*

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan
> task-by-task.

**Goal:** General-purpose benchmarking tool — answers "which approach is better, and by how much?"
for any measurable task: tools, implementations, deps, LLM calls, agents.

**Architecture:** Ben is a standalone CLI in the hop-top family, sibling to eva. It runs N
candidates against a task, captures configurable metrics, applies a per-run scoring strategy, and
emits structured results. Every core concept (suites, candidates, metrics, scorers, adapters,
reporters, registry) is extensible via plugins or config. Shared infrastructure (config, storage,
reporting) is written for eventual extraction into `hop-top/kit`.

**Tech Stack:** Go, SQLite (local storage), YAML (spec files), stdio JSON protocol (plugins),
Typer-style CLI via cobra.

---

## Core Concepts

| Concept | Description |
|---------|-------------|
| Suite | Named, repeatable benchmark; defined in YAML or registered by plugin |
| Candidate | One approach being measured (xray, grep, idx, claude-sonnet, etc.) |
| Metric | Measurable dimension: `latency_ms`, `tokens`, `cost_usd`, `quality_score`, ... |
| Run | One execution of a suite against all candidates; produces a result set |
| Scorer | Configurable per run: `single:<metric>`, `weighted:<m>=<w>,...`, `raw` |
| Adapter | How ben executes a candidate: `cli`, `llm`, `function`, or custom binary |
| Reporter | Output formatter: `table`, `json`, `yaml`, or custom binary |
| Registry | Local + optional shared store for runs; agents share institutional memory |

---

## CLI Surface

```
ben run --suite <name>                          # run from spec file
ben run --task "<desc>" \
        --candidates xray,grep \
        --metric latency_ms,quality_score \
        --scorer weighted:latency_ms=0.3,quality_score=0.7 \
        --input.repo .
ben compare <run-a> <run-b>
ben query --suite <name> --last 10
ben suite list
ben suite show <name>
ben registry push <run-id>
ben registry pull --suite <name>
```

Global flags:
- `--format json|yaml|table` (default: table for humans, json for agents)
- `--quiet` — suppress stderr for pipeline use
- `--config <path>` — override config file

Exit codes:
- `0` — successful run (even if all candidates fail — failures are in result)
- `1` — ben error (bad config, missing adapter, etc.)

---

## Spec File Shape

```yaml
name: codebase-indexing
description: Compare xray vs grep for initial codebase orientation
version: 1

task:
  prompt: "Find all HTTP handler functions in this repo"
  input:
    repo: ./testdata/sample-repo

candidates:
  - name: xray
    adapter: cli
    cmd: "xray explore --search {{input.prompt}} --path {{input.repo}}"
  - name: grep
    adapter: cli
    cmd: "grep -r 'func.*Handler' {{input.repo}}"

metrics:
  - latency_ms
  - tokens
  - cost_usd
  - quality_score          # plugin-defined

scorer:
  strategy: weighted
  weights:
    latency_ms: 0.3
    cost_usd: 0.2
    quality_score: 0.5

registry:
  push: false
```

---

## Result Schema

```yaml
run_id: 01HX...
suite: codebase-indexing
suite_version: 1
timestamp: 2026-03-28T11:00:00Z
scorer: {strategy: weighted, weights: {latency_ms: 0.3, cost_usd: 0.2, quality_score: 0.5}}

candidates:
  - name: xray
    metrics:
      latency_ms: 340
      cost_usd: 0.0012
      quality_score: 0.91
    score: 0.847
    rank: 1
    raw_output: "..."
    error: null

  - name: grep
    metrics:
      latency_ms: 180
      cost_usd: 0.0
      quality_score: 0.43
    score: 0.31
    rank: 2
    raw_output: "..."

winner: xray              # null if scorer=raw
metadata:
  host: macbook-pro
  ben_version: 0.1.0
```

---

## Storage Layout

Global (cross-project):
```
~/.local/share/ben/
  runs/
    <run-id>.json
  registry/
    cache/
  suites/
    *.yaml
```

Project-local (when run from a repo):
```
.ben/
  suites/
  runs/
```

---

## Plugin Architecture

### Config-declared plugins (metrics, scorers)

In `~/.config/ben/ben.yaml` or `.ben/ben.yaml`:

```yaml
plugins:
  metrics:
    - name: quality_score
      type: llm_judge
      model: claude-sonnet-4-6
      prompt: "Rate the relevance of this output 0-1: {{output}}"
    - name: semantic_similarity
      import: ben-plugin-similarity    # binary on PATH

  scorers:
    - name: pareto
      import: ben-plugin-pareto

  reporters:
    - name: markdown
      import: ben-plugin-md-report
```

### Binary plugins (adapters, reporters)

Ben discovers `ben-adapter-*` and `ben-reporter-*` binaries on PATH.
Stdio JSON protocol:

```
# ben → adapter (stdin)
{"action": "run", "candidate": {...}, "input": {...}}

# adapter → ben (stdout)
{"metrics": {"latency_ms": 340, "quality_score": 0.91}, "output": "..."}
```

Same protocol for reporters:
```
# ben → reporter (stdin)
{"run": {...}}

# reporter → stdout (formatted output)
```

---

## Agent-Callable Interface

Agents call ben mid-task with `--format json`. Ben guarantees:
- exit 0 on successful run; exit 1 on ben errors only
- `--format json` emits valid JSON to stdout; all logs/errors to stderr
- `winner` field is the agent's primary decision signal (null = raw mode)
- `--quiet` suppresses stderr for clean pipelines

**Registry as shared memory:** agents push results after runs so future agents can pull
community baselines without re-running:

```bash
ben registry pull --suite codebase-indexing
ben run --suite codebase-indexing --input.repo . --format json
ben registry push <run-id>
```

---

## Phase Plan

### Phase 1: Core + CLI Adapter

**Goal:** Working `ben run` with CLI adapter, local storage, JSON+table output.

**Deliverables:**
- `ben run` (spec file + inline args)
- CLI adapter (`adapter: cli`)
- Built-in metrics: `latency_ms`, `exit_code`, `output_size`
- Scorers: `single`, `weighted`, `raw`
- Local SQLite storage
- Reporters: `json`, `table`
- `ben compare <run-a> <run-b>`
- `ben query --suite <name> --last N`

**Files:**
- Create: `cmd/ben/main.go`
- Create: `cmd/ben/run.go`
- Create: `cmd/ben/compare.go`
- Create: `cmd/ben/query.go`
- Create: `internal/spec/spec.go` — suite spec loader/validator
- Create: `internal/adapter/cli.go` — CLI adapter
- Create: `internal/metrics/builtin.go` — latency_ms, exit_code, output_size
- Create: `internal/scorer/scorer.go` — single, weighted, raw
- Create: `internal/storage/storage.go` — SQLite runs store
- Create: `internal/reporter/json.go`
- Create: `internal/reporter/table.go`
- Create: `tests/unit/spec_test.go`
- Create: `tests/unit/scorer_test.go`
- Create: `tests/unit/adapter_cli_test.go`
- Create: `tests/e2e/run_test.go`
- Create: `tests/e2e/compare_test.go`

### Phase 2: LLM Adapter + Quality Metrics

**Goal:** `ben run --candidates claude-sonnet,gpt-4o` works; quality scoring via LLM judge.

**Deliverables:**
- `llm` adapter (wraps LLM call; model specified in candidate config)
- Built-in metrics: `tokens`, `cost_usd`
- Built-in `quality_score` metric via LLM judge (config-declared, llm_judge type)
- Config-declared metric plugin loading

**Files:**
- Create: `internal/adapter/llm.go`
- Create: `internal/metrics/llm.go` — tokens, cost_usd
- Create: `internal/metrics/llm_judge.go` — quality_score
- Create: `internal/plugin/config.go` — config-declared plugin loader
- Edit: `internal/spec/spec.go` — extend for llm adapter fields
- Create: `tests/unit/adapter_llm_test.go`
- Create: `tests/unit/metrics_llm_test.go`
- Create: `tests/e2e/run_llm_test.go`

### Phase 3: Plugin System

**Goal:** Binary plugin protocol; `ben-adapter-*` and `ben-reporter-*` discovery.

**Deliverables:**
- Binary plugin discovery (PATH scan for `ben-adapter-*`, `ben-reporter-*`)
- Stdio JSON protocol for adapter and reporter plugins
- Config-declared scorer plugins
- `ben suite list` / `ben suite show`

**Files:**
- Create: `internal/plugin/binary.go` — discovery + stdio protocol
- Create: `internal/plugin/registry.go` — plugin registry
- Edit: `cmd/ben/run.go` — wire binary adapter/reporter plugins
- Create: `cmd/ben/suite.go` — suite list/show
- Create: `tests/unit/plugin_binary_test.go`
- Create: `tests/e2e/plugin_test.go`

### Phase 4: Registry

**Goal:** Local registry index; opt-in push/pull to shared community registry.

**Deliverables:**
- Local registry index (SQLite)
- `ben registry push <run-id>`
- `ben registry pull --suite <name>`
- Shared registry client (HTTP, opt-in)
- Agents accumulate institutional memory across sessions

**Files:**
- Create: `internal/registry/local.go`
- Create: `internal/registry/remote.go` — HTTP client
- Create: `cmd/ben/registry.go`
- Edit: `internal/storage/storage.go` — registry index
- Create: `tests/unit/registry_test.go`
- Create: `tests/e2e/registry_test.go`

### Phase 5: Eva Integration

**Goal:** Eva benchmark suites (MMLU, GSM8K, HellaSwag) become ben suites; supersedes
`eva benchmark` plan tasks.

**Deliverables:**
- `eva` adapter built-in (wraps `eva run` as a ben candidate)
- Eva benchmark suite specs (MMLU, GSM8K, HellaSwag) as `.ben/suites/*.yaml`
- Deprecate / close out `eva benchmark` TLC tasks (T-0196 through T-0243)

**Files:**
- Create: `internal/adapter/eva.go`
- Create: `suites/mmlu.yaml`
- Create: `suites/gsm8k.yaml`
- Create: `suites/hellaswag.yaml`
- Create: `tests/e2e/adapter_eva_test.go`

---

## Risks

- Plugin stdio protocol adds latency per metric; budget carefully for multi-candidate runs.
- LLM judge quality_score adds cost; must be opt-in and budgeted per run.
- Registry shared state requires trust model; v1 push/pull is opt-in, no auth required.
- Shared infrastructure with hop-top/kit extraction is future work; avoid tight coupling now.

---

## Success Criteria

- Agent can call `ben run --candidates xray,grep --metric latency_ms,quality_score --format json`
  and get a machine-readable winner in one command.
- Developer can define a repeatable suite in YAML, run it, and query historical results.
- A new metric, scorer, adapter, or reporter can be added as a binary on PATH with no ben
  changes.
- Registry push/pull lets agents share tooling benchmarks across sessions and projects.
- Eva benchmark suites (MMLU, GSM8K, HellaSwag) run through ben without bespoke eva code.
