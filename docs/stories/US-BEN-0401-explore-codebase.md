# US-BEN-0401 — Explore codebase; find Adapter/Metric/Scorer interfaces quickly

*author: $USER | 2026-03-28*

---

## Identity

| Field     | Value                                    |
|-----------|------------------------------------------|
| ID        | US-BEN-0401                              |
| Title     | Explore codebase; find interfaces quickly |
| Persona   | OSS Go Developer (Contributor)           |
| Trigger   | Dev clones ben repo; wants to contribute a plugin or built-in |

---

## Trigger

Contributor clones `github.com/hop-top/ben`; needs to locate `Adapter`, `Metric`, and
`Scorer` interfaces before writing any code. No prior knowledge of the codebase.

---

## Acceptance Criteria

1. `internal/adapter/adapter.go` exports `Adapter` interface; discoverable via `go doc`.
2. `internal/metrics/metrics.go` exports `Metric` interface; discoverable via `go doc`.
3. `internal/scorer/scorer.go` exports `Scorer` interface; discoverable via `go doc`.
4. Each interface has a godoc comment describing its contract (inputs, outputs, error cases).
5. `docs/contributing.md` cross-references all three interfaces with file paths.
6. `go doc hop.top/ben/internal/adapter Adapter` exits 0 and prints interface definition.
7. Contributor can identify which file to edit within 5 minutes of cloning.

---

## Interface Contracts Referenced

| Interface | Package                       | Method signature                                  |
|-----------|-------------------------------|---------------------------------------------------|
| Adapter   | `internal/adapter`            | `Run(ctx, Candidate, Input) (Result, error)`      |
| Metric    | `internal/metrics`            | `Measure(ctx, raw RawOutput) (float64, error)`    |
| Scorer    | `internal/scorer`             | `Score(ctx, metrics map[string]float64) float64`  |

---

## Test Requirements

- Unit: `tests/unit/interfaces_test.go` — compile-time checks that built-in impls satisfy
  each interface (type assertion, not runtime).
- No external deps required; `go test ./internal/...` must pass with no services running.

---

## Happy Path Steps

1. `git clone github.com/hop-top/ben && cd ben`
2. `go doc hop.top/ben/internal/adapter` — prints package doc + `Adapter` interface.
3. `go doc hop.top/ben/internal/metrics` — prints `Metric` interface.
4. `go doc hop.top/ben/internal/scorer` — prints `Scorer` interface.
5. Read `docs/contributing.md` — confirms file paths and contract summary.
6. Open `internal/adapter/adapter.go` — see interface + existing `cli` impl in same dir.
7. Contributor starts writing their own impl.

---

## Failure Path + Expected Behavior

| Failure                                   | Expected behavior                                       |
|-------------------------------------------|---------------------------------------------------------|
| Interface not exported (lowercase)        | `go doc` returns nothing; contributor files issue       |
| Godoc comment missing                     | `go doc` prints bare signature; contributor files issue |
| `contributing.md` absent                  | Contributor must grep — acceptable but not ideal; doc   |
|                                           | gap is tracked as a separate story                      |

---

## E2E Test Spec

```
file:      tests/e2e/stories/US_BEN_0401_test.go
func:      TestUS_BEN_0401_InterfacesDiscoverable
asserts:
  - exec "go doc hop.top/ben/internal/adapter Adapter" exits 0
  - exec "go doc hop.top/ben/internal/metrics Metric"  exits 0
  - exec "go doc hop.top/ben/internal/scorer Scorer"   exits 0
  - output of each contains "interface"
  - docs/contributing.md exists and contains "internal/adapter"
```
