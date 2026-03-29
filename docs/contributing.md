# Contributing to ben

## Quick start

```
# clone
git clone https://github.com/ideacrafterslabs/ben hops/ben-phase1
cd hops

# go.work — needed because kit is a local replace
# go.work already exists in the hops/ root; verify it lists ./ben-phase1
go work use ./ben-phase1

# run tests (race detector on)
cd ben-phase1
task test
```

`task check` runs lint + test in sequence; must pass before any PR.

## Interfaces

| Interface  | File                                | Purpose                              |
|------------|-------------------------------------|--------------------------------------|
| `Adapter`  | `internal/adapter/adapter.go`       | Execute one candidate, return Result |
| `Metric`   | `internal/metrics/metric.go`        | Collect one float64 from a Result    |
| `Scorer`   | `internal/scorer/scorer.go`         | Rank a slice of CandidateResult      |
| `Reporter` | `internal/reporter/reporter.go`     | Write a Run to an io.Writer          |

Read each file's godoc for the full contract before implementing.

## Adding an adapter

1. Create `internal/adapter/<name>.go`.
2. Define a struct; implement `Run(ctx, c, input) (*Result, error)`.
   - Non-zero exit → `Result.ExitCode`; never return it as an error.
   - Context cancellation → return `ctx.Err()` as error.
   - Always return a non-nil `*Result` even on error.
3. Open `cmd/ben/run.go`; locate the `switch c.Adapter` block.
4. Add a case for the new adapter name that wires the struct:
   ```
   case "<name>":
       adpt = adapter.New<Name>()
   ```
5. `task test` — add a unit test in `internal/adapter/<name>_test.go`.

## Adding a metric

1. Create `internal/metrics/<name>.go` (or add to `builtin.go` if trivial).
2. Define a struct; implement `Name() string` and `Collect(*Result) float64`.
   - `Name()` must be unique, stable, no spaces/dots.
   - `Collect()` must never panic; return 0 or math.NaN() for missing data.
3. Register via `init()`:
   ```
   func init() { Register(<name>Metric{}) }
   ```
4. `task test` — add a unit test proving Name uniqueness and Collect output.

## Adding a scorer

1. Create `internal/scorer/<name>.go`.
2. Define a struct; implement `Score([]CandidateResult) []ScoredResult`.
   - Same length in, same length out.
   - Stable sort on ties; rank 1 = best; no gaps.
   - Do not mutate input.
3. Open `internal/scorer/scorer.go`; extend `Parse()` with a new case:
   ```
   if strategy == "<name>" {
       return &<name>Scorer{}, nil
   }
   ```
4. Update the error message in `Parse()` to list the new strategy.
5. `task test` — add a table-driven test in `internal/scorer/<name>_test.go`.

## Adding a reporter

1. Create `internal/reporter/<name>.go`.
2. Define a struct; implement `Report(w io.Writer, r *run.Run) error`.
   - All output to `w`; diagnostics to stderr only.
   - Return error only on write failure or encoding failure with partial output.
3. Open `internal/reporter/reporter.go`; add a case to `New()`:
   ```
   case "<name>":
       return &<name>Reporter{}, nil
   ```
4. Update the error message in `New()` to list the new format.
5. `task test` — add a unit test in `internal/reporter/<name>_test.go`.

## Binary plugins (preview)

Phase 3 will add PATH-based plugin discovery. A binary named `ben-adapter-*`
on PATH will be auto-detected and wrapped as an Adapter. No action needed now;
the interface contract above will remain stable across the transition.

## PR checklist

- [ ] `task check` passes locally (lint + test, race detector on)
- [ ] PR description links the relevant story from `docs/stories/`
- [ ] Every new exported symbol (interface method, struct field, func) has godoc
- [ ] New interface implementation has a unit test
- [ ] `go mod tidy` run; go.sum committed
