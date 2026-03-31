# ben Cheatsheet

General-purpose benchmarking CLI. Answers: "which approach is better, and by how much?"
Scannable in 30 seconds.

---

## Concepts

```
Suite     — YAML spec file: task + candidates + metrics + scorer
Candidate — one approach being benchmarked (CLI tool, LLM, adapter)
Adapter   — how ben invokes a candidate: cli | llm | eva | <plugin>
Metric    — what to measure per candidate run
Scorer    — how to rank candidates: raw | single:<metric> | weighted
Run       — one execution record; persisted to .ben/ or XDG data dir
```

---

## Config

Default search order: `$HOME/.config/ben/ben.yaml` then `.ben/ben.yaml`

```bash
ben --config ./custom.yaml run ...   # explicit override
```

Key config fields:

```yaml
format: table                        # default output format: table | json | yaml
registry:
  url: https://registry.example.com  # required for push/pull
plugins:
  metrics:
    - name: quality
      type: llm_judge
      model: claude-sonnet-4-6
      prompt: "Rate 1-10: {{output}}"
```

---

## Run

### Inline (quick test)

```bash
ben run --task "sort a list of numbers" \
        --candidates "jq=cli=jq -s 'sort'" \
        --candidates "python=cli=python3 sort.py"

# inline with scorer
ben run --task "echo hello" \
        --candidates "a=cli=echo hello" \
        --candidates "b=cli=printf hello" \
        --metric latency_ms \
        --scorer single:latency_ms

# pass input key=value to template
ben run --suite suites/gsm8k.yaml \
        --input dataset=suites/testdata/gsm8k-sample.yaml
```

### From suite YAML

```bash
ben run --suite ./suites/gsm8k.yaml
ben run --suite .ben/suites/my-suite.yaml
```

### Candidate format

```
name=adapter=cmd      (flag form)
name,adapter,cmd      (spec form)

adapter: cli          shell command; cmd = the command string
         llm          LLM call; set model in spec
         eva          evaluation harness; cmd = dataset path
         <plugin>     binary on PATH named ben-adapter-<name>
```

### Output formats

```bash
ben run --suite ... --format table    # default human-readable
ben run --suite ... --format json
ben run --suite ... --format yaml
ben run --suite ... --format <plugin> # binary reporter: ben-reporter-<name>
```

---

## Suites

```bash
ben suite list                       # all known suites (XDG + .ben/suites/)
ben suite list --format json
ben suite show <name>                # human-readable detail
ben suite show <name> --format yaml  # full spec dump
```

Suite YAML structure:

```yaml
name: my-suite
description: "..."
version: 1

task:
  prompt: "describe the task"
  input:
    key: value                        # {{input.key}} in candidate cmd

candidates:
  - name: tool-a
    adapter: cli
    cmd: "tool-a --flag {{input.key}}"
  - name: model-x
    adapter: llm
    model: claude-sonnet-4-6

metrics:
  - latency_ms
  - exit_code
  - output_size
  - output_lines
  - accuracy                          # eva adapter only
  - cost_usd                          # eva adapter only
  - <plugin-metric>                   # llm_judge or binary plugin

scorer:
  strategy: single                    # single:<metric> | weighted | raw
  weights:
    latency_ms: 0.7
    output_size: 0.3

registry:
  push: false
```

Suite scan dirs (in order):
1. `~/.local/share/ben/suites/`
2. `.ben/suites/`

---

## Built-in Metrics

| Metric         | What it measures                       |
|----------------|----------------------------------------|
| `latency_ms`   | wall-clock duration in ms              |
| `exit_code`    | process exit code                      |
| `output_size`  | byte length of combined output         |
| `output_lines` | line count of output                   |
| `accuracy`     | fraction correct (eva adapter)         |
| `cost_usd`     | LLM token cost in USD (eva/llm)        |

### Scorers

| Strategy                         | Meaning                                 |
|----------------------------------|-----------------------------------------|
| `raw`                            | no ranking; collect metrics only        |
| `single:<metric>`                | rank by one metric (lower = better)     |
| `weighted:<m>=<w>,<m>=<w>,...`   | weighted sum; rank highest score first  |

---

## Query (Past Runs)

```bash
ben query                            # 10 most recent runs
ben query --last 25
ben query --suite gsm8k              # filter by suite name
ben query --format json
```

---

## Compare (Two Runs)

```bash
ben compare <run-id-a> <run-id-b>
ben compare <run-id-a> <run-id-b> --format json
```

Shows per-candidate, per-metric delta (B − A) and winner in each run.

---

## Registry (Share Runs)

Requires `registry.url` in config.

```bash
ben registry push <run-id>           # push local run to remote
ben registry pull                    # pull all runs from remote
ben registry pull --suite gsm8k      # filter by suite
```

---

## Plugins

### Adapter plugins

Binary on PATH named `ben-adapter-<name>`:

```bash
# invoked by ben as:
ben-adapter-playwright --task "..." --candidate-name "..."
```

### Reporter plugins

Binary on PATH named `ben-reporter-<name>`:

```bash
ben run --suite ... --format playwright   # invokes ben-reporter-playwright
```

### Metric plugins (llm_judge)

Declared in `ben.yaml`:

```yaml
plugins:
  metrics:
    - name: quality
      type: llm_judge
      model: claude-sonnet-4-6
      prompt: "Rate quality 1–10: {{output}}"
```

Reference in suite YAML: `metrics: [quality]`

---

## Data Storage

```
.ben/              project-local (auto-detected if dir exists)
~/.local/share/ben/ XDG fallback (global)
```

Run record key fields: `run_id`, `suite`, `timestamp`, `winner`, `scorer`,
`candidates[].metrics`, `candidates[].score`, `candidates[].rank`

---

## Error Reference

| Symptom | Fix |
|---------|-----|
| `unknown adapter "x"` | add `ben-adapter-x` binary to PATH |
| `metric "x" unavailable` | check metric name; ensure plugin loaded |
| `registry.url not configured` | add to ben.yaml under `registry.url` |
| `--suite and --task are mutually exclusive` | use one or the other |
| `missing required field: name` | YAML spec missing `name:` field |
| `missing required field: candidates` | spec needs at least one candidate |
| Suite not found | check `ben suite list`; verify `.ben/suites/` or XDG path |
