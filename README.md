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

ben depends on `hop.top/kit kit/v0.4.0-alpha.2`, pinned in `go.mod`
with no local override. Local development against unreleased kit
revisions uses a `replace` directive in `go.mod` (commented-out
example near the bottom of the file):

```go
// replace hop.top/kit => ../kit
```

Uncomment, point at your kit checkout, and `go mod tidy`.

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

# List last 10 runs for a suite
ben list --suite codebase-indexing --last 10

# Show one run by id
ben show 01HX...abc
```

---

## Commands

| Command                         | Description                                           |
|---------------------------------|-------------------------------------------------------|
| `ben run`                       | Run benchmark suite or inline task against candidates |
| `ben list`                      | List recent runs from local storage                   |
| `ben show <run-id>`             | Show details of one run                               |
| `ben compare <run-a> <run-b>`   | Diff two run results side-by-side                     |
| `ben suite list`                | List known suites (global + project-local)            |
| `ben suite show <name>`         | Show suite spec details                               |
| `ben registry push <run-id>`    | Push a run to the shared registry                     |
| `ben registry pull`             | Pull community baselines for a suite                  |
| `ben config path` / `paths`     | Inspect ben config file precedence                    |
| `ben spec`                      | Emit machine-readable capability manifest             |

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

## Configuration

Ben loads config from three layers, highest precedence first:

| Layer   | Path                              |
|---------|-----------------------------------|
| project | `./.ben/config.yaml`              |
| user    | `$XDG_CONFIG_HOME/ben/config.yaml`|
| system  | `/etc/ben/config.yaml`            |

Run `ben config paths --format json` to see the active chain. The
`-c <path>` flag overrides the discovery chain entirely (kit semantics
— `-c` wins over any previously discovered file). When ben is invoked
under another hop-top tool, the project-layer path will switch to that
tool's conventional location (`.hop/ben.yaml` under hop, `.tlc/ben.yaml`
under tlc) — wired in once kit ships `root.InvokedAs()` (T-0091).

---

## Release process

Release pipeline mirrors the `hop-top/.github` reusable workflows:

- `release-please.yml` watches `main`, opens a standing release PR
  that bumps the version + assembles the changelog.
- Merging that PR cuts a `ben/v<version>` tag.
- The tag push fires `publish.yml` (Go module mirror to
  `hop-top/ben`) and `goreleaser-on-tag.yml` (cross-platform binaries
  + Homebrew tap + Scoop bucket entries) in parallel.

Prerelease channel is seeded at `0.2.0-alpha.0`. See
[`.github/RELEASE-BOOTSTRAP.md`](.github/RELEASE-BOOTSTRAP.md) for
the manual web-side steps (mirror-repo creation, GitHub App
installation, org secrets) required before the first cut.

---

## Troubleshooting

**`compile: version "go1.26.1" does not match go tool version "go1.26.2"`**

Cause: stale `GOROOT` exported from an earlier `mise` shell. Quick
workaround: `env -u GOROOT go test ./...`. Long-term fix: `mise use
go@<latest>` and respawn the shell.

---

## Contributing

See [docs/contributing.md](docs/contributing.md) for interfaces, how to add adapters/metrics/
scorers/reporters, and the PR checklist.
