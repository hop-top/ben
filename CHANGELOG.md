# Changelog

All notable changes to this project will be documented in this file.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

---

## [Unreleased]

### Added

- `ben list` — replaces `ben query`; lists recent runs from local storage
- `ben show <run-id>` — fetch a single run by id
- `ben config path` / `ben config paths` — inspect config file precedence
  chain (project → user → system) provided by kit/console/cli/config
- `ben spec` / `ben spec --version` — emit machine-readable capability
  manifest per kit conventions §13; agents use this for compatibility
  negotiation before issuing other commands
- `--dry-run` on `run`, `registry push`, `registry pull` — emits a
  structured Plan describing intended effects without applying them
- Side-effect (`kit/side-effect`) and idempotency (`kit/idempotent`)
  annotations on every leaf command; available via `ben spec`
- `--confirm`, `--max-ops`, `--policy` global flags wired by kit for
  agent-driven invocations (delegation safety, conventions §8.6)
- Group taxonomy in `--help`: EXECUTE, RESULTS, CATALOG, REGISTRY;
  MANAGEMENT (config, spec, completion) auto-hidden
- Hints after successful commands (silenced by `--no-hints` or
  non-TTY)
- Structured JSON error envelope (`code`/`message`/`exit_code`) under
  `--format json|yaml`
- Config layering picks up `$XDG_CONFIG_HOME/ben/ben.yaml` and
  `/etc/ben/ben.yaml` in addition to project-local files

### Changed

**BREAKING**

- `ben query` removed; callers must use `ben list` (no deprecation
  alias). `--suite` and `--last` flags are unchanged.
- Registry push/pull completion messages move from stdout to stderr
  (via slog) so `--format json` pipelines stay clean.
- Errors for missing runs/suites and missing `registry.url` now use
  structured exit codes (3 NOT_FOUND, 2 USAGE) instead of generic 1.
- Kit upgrade: imports moved from `hop.top/kit/<pkg>` to
  `hop.top/kit/go/<area>/<pkg>` (kit's March 2026 restructure).

### Fixed

- Plugin-supplied custom metrics are now passed through to the
  candidate result instead of being rejected by metric validation.

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
