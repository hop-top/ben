# AGENTS.md — ben

Agent-facing notes for using ben mid-task. Humans read
[README.md](README.md) and [docs/contributing.md](docs/contributing.md);
this file documents the machine contract.

## Conventions compliance

ben satisfies `~/.ops/docs/cli-conventions-with-kit.md` —
§3.5 (side-effect annotations), §6.4 (structured error envelope),
§6.5 (JSONL progress events), §6.6 (`_meta` provenance envelope),
§7.4 (`config path` / `config paths` subcommands), §8.5 (idempotency
annotations), §8.6 (delegation policy + `--confirm` / `--max-ops` /
`--policy` globals), §10 (group taxonomy in `--help`), and §13
(capability manifest + schema-version contract; current
`schemaVersion: 1.1`).

Introspect at runtime:

```
ben spec --format json    # full manifest
ben spec --version        # short: just schemaVersion
```

Trip-wire regression test: `tests/unit/conventions_test.go` walks
the manifest on every CI run and fails if any leaf is missing
`kit/side-effect` or `kit/idempotent`, or if `schemaVersion`
drifts unnoticed.

## Machine output contracts

Set `--format json|yaml` (or use `--quiet` to suppress stderr) for
deterministic output. All diagnostics go to stderr; stdout is
data only.

### Structured errors (§6.4)

On failure, every leaf emits the same JSON envelope to stdout when
`--format json` is set:

```json
{
  "error": {
    "code": "BEN_NO_RUN",
    "message": "run \"bogus-id\" not found",
    "suggested_fix": "list available runs with `ben list`",
    "alternatives": [],
    "exit_code": 3
  }
}
```

Ben-specific codes (defined in `internal/clierr`):

- `BEN_NO_RUN` — run id not in local store. Exit 3 (NOT_FOUND).
- `BEN_NO_SUITE` — suite name not registered. Exit 3 (NOT_FOUND).

Generic codes (`NOT_FOUND`, `USAGE`, etc.) come straight from kit.
Disambiguate on `error.code`; shell scripts can still grep
`exit_code: 3` for the NOT_FOUND family.

### Progress events (§6.5)

`ben run` emits per-phase JSONL events to stderr. Five phases:

| Phase           | Emitted when                                      |
|-----------------|---------------------------------------------------|
| `load_spec`     | YAML suite file (or inline flags) parsed          |
| `run_candidate` | Once per candidate, with `index` / `total`        |
| `score`         | Scorer strategy applied                           |
| `persist`       | Run record written to local store                 |
| `report`        | Final report rendered                             |

Render selection:

- `--quiet` → events discarded.
- `--progress-format jsonl|human` → explicit override.
- `--format json|yaml` → JSONL on stderr (auto-inherits).
- Default (no `--format`, no `--quiet`) → human-readable on stderr.

### Provenance envelope (§6.6)

`ben list` and `ben show` wrap their `--format json|yaml` payload
in a `_meta` envelope:

```json
{
  "data": [...],
  "_meta": {
    "source": "ben.local-store",
    "fetched_at": "2026-05-19T20:00:00Z",
    "method": "sqlite_query"
  }
}
```

Rows may originate from `ben run` (locally produced) or be mirrored
from `ben registry pull`; either way they're served out of the
SQLite store, which is what `source` advertises. Table output is
unchanged.

## Config paths

Ben's config precedence chain (highest first):

| Scope   | Path                              |
|---------|-----------------------------------|
| project | `./.ben/config.yaml`              |
| user    | `$XDG_CONFIG_HOME/ben/config.yaml`|
| system  | `/etc/ben/config.yaml`            |

`ben config paths --format json` prints the chain with per-entry
`exists`/`source`/`scope`. `-c <path>` overrides the chain entirely
(kit semantics).

The project layer is caller-context-aware via the `KIT_INVOKED_AS`
env var (kit v0.4.0-alpha.3+, surfaced via `root.InvokedAs()`):

| `KIT_INVOKED_AS`  | Project config path  |
|-------------------|----------------------|
| (unset/standalone)| `./.ben/config.yaml` |
| `hop`             | `./.hop/ben.yaml`    |
| `tlc`             | `./.tlc/ben.yaml`    |

Callers (tlc, hop, etc.) export `KIT_INVOKED_AS=<tool>` before
exec'ing ben. Only one project-layer entry wins per invocation
(kit constraint). Standalone is the fallback for any unknown value.

## Versioning

`ben --version` prints the value of `var version` in `cmd/ben/main.go`,
defaulted to `"dev"` for local builds. Release builds inject the tag
via `-X main.version=<tag>` ldflag from goreleaser. Agents that depend
on a specific ben version can capability-negotiate via
`ben spec --version` (returns the schemaVersion, currently `1.1`).

## Delegation safety (§8.6)

Three global flags control agent-driven invocations:

- `--confirm` — write/destructive leaves require explicit confirm.
- `--max-ops <N>` — cap the number of side-effecting operations.
- `--policy <name>` — load
  `$XDG_CONFIG_HOME/ben/policies/<name>.yaml`; kit's policy
  middleware gates per-leaf behaviour.

Kit installs these globally via
`cli.WithPolicy(cli.DefaultPolicyLoader("ben"))`.

## Leaf inventory

Eleven leaves total. Every one is annotated with `kit/side-effect`
(§3.5) and `kit/idempotent` (§8.5):

| Leaf                | side-effect | idempotent  |
|---------------------|-------------|-------------|
| `ben compare`       | read        | yes         |
| `ben config path`   | read        | yes         |
| `ben config paths`  | read        | yes         |
| `ben list`          | read        | yes         |
| `ben registry pull` | write       | yes         |
| `ben registry push` | write       | conditional |
| `ben run`           | write       | no          |
| `ben show`          | read        | yes         |
| `ben spec`          | read        | yes         |
| `ben suite list`    | read        | yes         |
| `ben suite show`    | read        | yes         |

The two non-idempotent leaves (`ben run`, `ben registry push`)
carry a "Safe retry:" preamble in their Long help describing the
side effects of a repeat invocation.
