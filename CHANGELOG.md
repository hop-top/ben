# Changelog

All notable changes to this project will be documented in this file.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

---

## [0.1.0] - 2026-03-28

### Added

**CLI**
- `ben run` — run benchmark suite from YAML spec file or inline flags
- `ben run --task / --candidates / --metric / --scorer / --input` — inline mode
- `ben compare <run-a> <run-b>` — diff two historical runs
- `ben query --suite --last` — query runs from local storage
- `ben suite list` / `ben suite show <name>` — browse known suites
- `ben registry push <run-id>` / `ben registry pull` — share runs via registry
- Global flags: `--format json|yaml|table`, `--quiet`, `--config`

**Adapters**
- `cli` adapter — spawns shell commands; captures stdout, stderr, exit code, latency
- `llm` adapter — calls LLM APIs; captures tokens, cost, model output
- `eva` adapter — wraps `eva run` as a ben candidate for standard eval suites

**Metrics**
- `latency_ms` — wall-clock execution time
- `exit_code` — process exit code (cli adapter)
- `output_size` — byte length of stdout
- `tokens` — total tokens consumed (llm adapter)
- `cost_usd` — estimated USD cost (llm adapter)
- `quality_score` — 0–1 LLM judge metric (config-declared plugin)

**Scorers**
- `single:<metric>` — rank by a single metric
- `weighted:<m>=<w>,...` — weighted sum across multiple metrics
- `raw` — no ranking; emit metrics only

**Reporters**
- `table` reporter — human-readable terminal table (default)
- `json` reporter — machine-readable JSON to stdout
- `yaml` reporter — YAML output

**Plugin system**
- Binary plugin discovery: `ben-adapter-*` and `ben-reporter-*` on PATH auto-detected
- Stdio JSON protocol for adapter and reporter plugins
- Config-declared metric plugins (`llm_judge` type) via `plugins.metrics` in `ben.yaml`

**Storage**
- SQLite-backed run store with JSON run files
- Global store: `~/.local/share/ben/`
- Project-local store: `.ben/` (auto-detected; takes priority over global)
- Run indexing for fast `ben query` lookups

**Spec format**
- YAML suite spec: `name`, `description`, `version`, `task`, `candidates`, `metrics`,
  `scorer`, `registry`
- Template interpolation in `cmd` fields: `{{input.<key>}}`

**Agent interface**
- `--format json` guarantees valid JSON to stdout; all diagnostics to stderr
- `--quiet` suppresses stderr for clean pipeline use
- Exit code contract: `0` = successful run; `1` = ben error
- `winner` field in JSON output as primary agent decision signal

[0.1.0]: https://github.com/ideacrafterslabs/ben/releases/tag/v0.1.0
