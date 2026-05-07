# ben

> [!WARNING]
> **🚧 Do Not Use — History Will Be Rewritten 🚧**
>
> This repo is undergoing major restructuring as we selectively
> open-source internal tools built at
> [Idea Crafters LLC](https://ideacrafters.com). Git history **will be
> force-pushed and rewritten** multiple times. Do not fork, clone, or
> depend on this repo in any capacity until we tag a stable release.

General-purpose benchmarking tool — answers "which approach is better, and by how much?"
for any measurable task: tools, implementations, deps, LLM calls, agents.

---

## Install

```
go install hop.top/ben/cmd/ben@latest
```

---

## Quick start

```sh
# Inline run: compare two CLI tools on a task
ben run --task "Find HTTP handlers" --candidates xray,grep --metric latency_ms,quality_score \
    --scorer weighted:latency_ms=0.3,quality_score=0.7 --input repo=.

# Suite file: run a named, repeatable benchmark
ben run --suite .ben/suites/codebase-indexing.yaml

# Compare two historical runs
ben compare 01HX...abc 01HX...def

# Query last 10 runs for a suite
ben query --suite codebase-indexing --last 10
```

---

## Commands

| Command                         | Description                                           |
|---------------------------------|-------------------------------------------------------|
| `ben run`                       | Run benchmark suite or inline task against candidates |
| `ben compare <run-a> <run-b>`   | Diff two run results side-by-side                     |
| `ben query`                     | Query historical runs from local storage              |
| `ben suite list`                | List known suites (global + project-local)            |
| `ben suite show <name>`         | Show suite spec details                               |
| `ben registry push <run-id>`    | Push a run to the shared registry                     |
| `ben registry pull`             | Pull community baselines for a suite                  |

---

## Adapters

| Adapter    | How ben runs the candidate                                               |
|------------|--------------------------------------------------------------------------|
| `cli`      | Spawns a shell command; captures stdout/stderr, exit code, latency       |
| `llm`      | Calls an LLM via API; captures tokens, cost, output                      |
| `eva`      | Wraps `eva run` as a ben candidate for standard eval suites              |
| binary     | Any `ben-adapter-*` binary on PATH; communicates via stdio JSON protocol |

---

## Metrics

| Metric           | Source   | Description                                   |
|------------------|----------|-----------------------------------------------|
| `latency_ms`     | built-in | Wall-clock execution time in milliseconds     |
| `exit_code`      | built-in | Process exit code (cli adapter)               |
| `output_size`    | built-in | Byte length of stdout output                  |
| `tokens`         | llm      | Total tokens consumed (prompt + completion)   |
| `cost_usd`       | llm      | Estimated cost in USD                         |
| `quality_score`  | plugin   | 0–1 relevance score; requires llm_judge plugin|

---

## Scorers

| Scorer                        | Description                                          |
|-------------------------------|------------------------------------------------------|
| `single:<metric>`             | Rank by one metric; lowest wins for cost/latency     |
| `weighted:<m>=<w>,...`        | Weighted sum across metrics; highest score wins      |
| `raw`                         | No ranking; emit raw metrics only; winner=null       |

Examples:

```
--scorer single:latency_ms
--scorer weighted:latency_ms=0.3,cost_usd=0.2,quality_score=0.5
--scorer raw
```

---

## Spec file

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
  - quality_score

scorer:
  strategy: weighted
  weights:
    latency_ms: 0.3
    quality_score: 0.7
```

---

## Plugin protocol

Binary plugins are auto-discovered as `ben-adapter-<name>` or `ben-reporter-<name>` on PATH.
Ben communicates via newline-delimited JSON over stdio: it writes a request JSON object to the
plugin's stdin and reads the response from stdout. Adapter plugins receive
`{"action":"run","candidate":{...},"input":{...}}` and must respond with
`{"metrics":{...},"output":"..."}`. Reporter plugins receive `{"run":{...}}` and write
formatted output to stdout. Naming convention: use the adapter/reporter name as the suffix,
e.g. `ben-adapter-docker`, `ben-reporter-markdown`.

---

## Agent usage

ben is designed for programmatic use mid-task:

```sh
# Machine-readable output; all logs to stderr
ben run --suite my-suite --format json --quiet

# Parse winner directly
ben run ... --format json | jq .winner
```

- `--format json` — emits valid JSON to stdout; diagnostics to stderr only
- `--quiet` — suppresses stderr; clean for pipelines
- Exit `0` — successful run (candidate failures are in the result, not exit code)
- Exit `1` — ben error (bad config, missing adapter, etc.)
- `winner` field — primary decision signal for agents; `null` when scorer is `raw`

---

## Storage

Global (cross-project):

```
~/.local/share/ben/
  runs/          # persisted run results
  registry/      # local registry index + cache
  suites/        # global suite specs
```

Project-local (detected automatically when `.ben/` exists in cwd):

```
.ben/
  suites/        # project-scoped suite specs
  runs/          # project-scoped run results
```

Ben prefers project-local storage when `.ben/` is present; falls back to global.

---

## Contributing

See [docs/contributing.md](docs/contributing.md) for interfaces, how to add adapters/metrics/
scorers/reporters, and the PR checklist.
