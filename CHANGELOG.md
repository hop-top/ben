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
  (conventions §7.4)
- `ben spec` / `ben spec --version` — emit machine-readable capability
  manifest per kit conventions §13; agents use this for compatibility
  negotiation before issuing other commands
- `--dry-run` on `run`, `registry push`, `registry pull` — emits a
  structured Plan describing intended effects without applying them
- Side-effect (`kit/side-effect`) and idempotency (`kit/idempotent`)
  annotations on every one of the 11 leaf commands per conventions
  §3.5 (read|write|destructive|interactive) and §8.5
  (yes|no|conditional); available via `ben spec`. Trip-wire test at
  `tests/unit/conventions_test.go` enforces both invariants
- `--confirm`, `--max-ops`, `--policy` global flags wired by kit for
  agent-driven invocations (delegation safety, conventions §8.6).
  Policies load from `$XDG_CONFIG_HOME/ben/policies/<name>.yaml` via
  `cli.WithPolicy(cli.DefaultPolicyLoader("ben"))`
- Group taxonomy in `--help`: EXECUTE, RESULTS, CATALOG, REGISTRY;
  MANAGEMENT (config, spec, completion) auto-hidden
- Hints after successful commands (silenced by `--no-hints` or
  non-TTY)
- Structured JSON error envelope under `--format json|yaml`
  (conventions §6.4): `{error: {code, message, suggested_fix,
  alternatives, exit_code}}`. Ben-specific codes
  `BEN_NO_RUN` / `BEN_NO_SUITE` (exit 3 NOT_FOUND family) live in
  `internal/clierr` alongside kit's generic codes
- JSONL progress events on stderr for `ben run` (conventions §6.5):
  five phases `load_spec`, `run_candidate`, `score`, `persist`,
  `report`. Render selection via `--quiet` / `--progress-format` /
  `--format json` (auto-JSONL) / human default
- `_meta` provenance envelope on `ben list` + `ben show` JSON/YAML
  output (conventions §6.6): `{data: ..., _meta: {source,
  fetched_at, method}}`. Source `ben.local-store`,
  method `sqlite_query`. Table output unchanged
- Config layering chain finalized — project `./.ben/config.yaml`,
  user `$XDG_CONFIG_HOME/ben/config.yaml`, system
  `/etc/ben/config.yaml`. `ben config paths --format json` reports
  the active chain. `-c <path>` overrides discovery entirely.
  Caller-context routing (`.hop/ben.yaml` under hop, `.tlc/ben.yaml`
  under tlc) is wired in once kit ships `root.InvokedAs()` —
  tracked as T-0091
- Release pipeline: adopted `hop-top/.github` reusable workflows —
  `release-please.yml` (App-token mint via
  `actions/create-github-app-token@v1`), `publish.yml` calling
  `publish-on-tag.yml@v0`, `goreleaser-on-tag.yml@v0` paired with
  `.goreleaser.yaml`. Tag shape `ben/v<version>` (enforced via
  `tag-separator: /` in release-please-config). Prerelease channel
  seeded at `0.2.0-alpha.0` in `.github/.release-please-manifest.json`.
  Manual web-side steps tracked in `.github/RELEASE-BOOTSTRAP.md`
- `var version = "dev"` at `cmd/ben/main.go` package scope; goreleaser
  injects the release tag via `-X main.version=<tag>` ldflag

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
- `hop.top/kit` dependency pinned to `kit/v0.4.0-alpha.2`; the
  local `replace` directive in `go.mod` is removed. Local development
  against unreleased kit revisions uses a commented-out `replace`
  example in `go.mod`.
- `schemaVersion` bumped `1.0` → `1.1` (additive per conventions
  §13.2 MINOR rule — progress events, `_meta` envelope, policy flags;
  no commands or fields removed).

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
