# Changelog

All notable changes to this project will be documented in this file.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

---

## [0.2.0-alpha.1](https://github.com/hop-top/ben/compare/ben/v0.2.0-alpha.0...ben/v0.2.0-alpha.1) (2026-05-20)


### ⚠ BREAKING CHANGES

* **cli:** 'ben query' is removed; callers must use 'ben list'. Suite/registry subcommands and other top-level commands keep their existing names.

### Features

* ben suite list/show commands ([b725426](https://github.com/hop-top/ben/commit/b725426385419260c7ec06c12137779f4d705b90))
* ben v0.1.0 — benchmarking CLI (all 5 phases) ([69642cb](https://github.com/hop-top/ben/commit/69642cb96a84d189a6133f9e4324f1cd29b83a14))
* binary plugin discovery and stdio protocol ([2f9c8c1](https://github.com/hop-top/ben/commit/2f9c8c1e1f7e6e47c25f226cc716abf6925bd5ed))
* **cli:** expose 'ben spec' capability manifest via toolspec/cli ([909e2a2](https://github.com/hop-top/ben/commit/909e2a2001211a49e684bccdfd4505e373dbf0f2))
* **cli:** wire --dry-run on write leaves with structured Plan output ([14d607b](https://github.com/hop-top/ben/commit/14d607bf0cefdee903b16ad91339c6d0650d60af))
* eva adapter ([ec2dfeb](https://github.com/hop-top/ben/commit/ec2dfeb7faa303ab3358b43b1666311386b2f0c8))
* eva benchmark suite specs (MMLU, GSM8K, HellaSwag) ([ac7c544](https://github.com/hop-top/ben/commit/ac7c5444803c862010839c290789aad395aedf29))
* LLM adapter (Anthropic + OpenAI via net/http) ([0cf2a54](https://github.com/hop-top/ben/commit/0cf2a546308d29ec34f3138f5e20ee6972fc82c1))
* LLM metrics (tokens, cost_usd), llm_judge, config plugin loader ([905ae35](https://github.com/hop-top/ben/commit/905ae35b0aaa46469d3b4282c93b5ea72ef53c97))
* local registry index in SQLite ([025d0c6](https://github.com/hop-top/ben/commit/025d0c6103fd225913ca90b50c8399d0f6865050))
* output_lines metric, llm_judge plugin, nullable winner/score/rank ([8f52556](https://github.com/hop-top/ben/commit/8f52556eafacf7f36119d7e7c1f49b48ddbd1c90))
* **output:** adopt kit logging, structured errors, and hint registry ([8a2f306](https://github.com/hop-top/ben/commit/8a2f306da4c46a186b72812d0246331b28c9695e))
* registry push/pull commands and HTTP client ([e08c345](https://github.com/hop-top/ben/commit/e08c345fbcbd95f6e34b25d35be739b0bc90a9e0))
* reporters, run/compare/query commands, e2e tests ([cd9850e](https://github.com/hop-top/ben/commit/cd9850ea926e77512bd2f624e515cb10d95bae3c))
* run type and SQLite storage via kit/sqlstore ([17358ed](https://github.com/hop-top/ben/commit/17358ed16793678c2dcf49047ceae4e8f5325764))
* spec loader, CLI adapter, built-in metrics, scorers ([f5c82a9](https://github.com/hop-top/ben/commit/f5c82a95437f43fd096f4ce947bf40e6f3bb941e))


### Bug Fixes

* go.work at hops/ level for gopls workspace; fix yaml import ([5d78b82](https://github.com/hop-top/ben/commit/5d78b8263e865846140cce7653522372bdf5c5e6))
* lint — CutPrefix, SplitSeq, maps.Copy; viper to direct dep ([4717c14](https://github.com/hop-top/ben/commit/4717c1436f8e6e6e3a3105eb71b0a6ded61b7809))
* remove duplicate metrics blank import in run.go ([08aee56](https://github.com/hop-top/ben/commit/08aee56bbea0aa166aabd1d6606fc862d417e319))
* remove unused io/fs import in suite.go ([7c3e124](https://github.com/hop-top/ben/commit/7c3e1242c6a033f71a21722055e3b41cd53fbb9b))
* **run:** pass through plugin-supplied custom metrics ([d696bf1](https://github.com/hop-top/ben/commit/d696bf1856248c6d4a932a3b0399628176b1ea43))


### Code Refactoring

* **cli:** align command surface with kit conventions ([b4a95ef](https://github.com/hop-top/ben/commit/b4a95ef99ebfe412e79e002b4f68fad7d0e3a515))

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
- Config layering chain finalized — user
  `$XDG_CONFIG_HOME/ben/config.yaml`, system `/etc/ben/config.yaml`,
  and a caller-context-aware project layer driven by `KIT_INVOKED_AS`
  (kit v0.4.0-alpha.3+, surfaced via `root.InvokedAs()`):
  `./.ben/config.yaml` standalone, `./.hop/ben.yaml` under hop
  umbrella, `./.tlc/ben.yaml` under tlc workspace. Only one
  project-layer entry wins per invocation (kit constraint).
  `ben config paths --format json` reports the active chain;
  `-c <path>` overrides discovery entirely
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
- `hop.top/kit` dependency pinned to `kit/v0.4.0-alpha.3`; the
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
