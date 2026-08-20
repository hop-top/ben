package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/clierr"
	"hop.top/ben/internal/run"
	"hop.top/ben/internal/storage"
	"hop.top/kit/go/console/output"
)

// metricDiff holds a per-metric comparison between two runs for one candidate.
type metricDiff struct {
	Metric string  `table:"Metric" json:"metric" yaml:"metric"`
	RunA   float64 `table:"Run-A"  json:"run_a"  yaml:"run_a"`
	RunB   float64 `table:"Run-B"  json:"run_b"  yaml:"run_b"`
	Delta  float64 `table:"Delta"  json:"delta"  yaml:"delta"`
}

// candidateDiff compares one candidate across two runs.
type candidateDiff struct {
	Candidate string       `json:"candidate" yaml:"candidate"`
	WinnerA   bool         `json:"winner_a"  yaml:"winner_a"`
	WinnerB   bool         `json:"winner_b"  yaml:"winner_b"`
	Metrics   []metricDiff `json:"metrics"   yaml:"metrics"`
}

// compareDiff is the structured result of a compare operation. Gate is
// present only when --fail-on-regression requested a gate evaluation.
type compareDiff struct {
	RunIDA     string          `json:"run_id_a" yaml:"run_id_a"`
	RunIDB     string          `json:"run_id_b" yaml:"run_id_b"`
	WinnerA    string          `json:"winner_a" yaml:"winner_a"`
	WinnerB    string          `json:"winner_b" yaml:"winner_b"`
	Candidates []candidateDiff `json:"candidates" yaml:"candidates"`
	Gate       *run.GateResult `json:"gate,omitempty" yaml:"gate,omitempty"`
}

func compareCmd(v *viper.Viper) *cobra.Command {
	var (
		failOnRegression bool
		thresholdKV      map[string]string
		directionKV      map[string]string
	)
	cmd := &cobra.Command{
		Use:   "compare <run-a> <run-b>",
		Short: "Compare two runs side-by-side",
		Long: `Compare two benchmark runs and surface per-candidate score
deltas. Useful for spotting regressions between iterations.

Regression gate: with --fail-on-regression, run-a is treated as the
baseline and run-b as the candidate. Each --threshold metric is checked
per candidate; movement in the WORSE direction beyond the threshold is a
regression and the command exits 4 (BEN_REGRESSION). Directions default
to lower-is-better for latency_ms and cost_usd, higher-is-better for
everything else; override per metric with --direction. A metric or
candidate missing on either side also fails the gate — a gate that
cannot see a metric must not pass it.`,
		Args: cobra.ExactArgs(2),
		Annotations: map[string]string{
			"kit/side-effect":    "read",
			"kit/idempotent":     "yes",
			"kit/top-level-verb": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var gs *run.GateSpec
			if failOnRegression {
				spec, err := parseGateSpec(thresholdKV, directionKV)
				if err != nil {
					return err
				}
				gs = spec
			} else if len(thresholdKV) > 0 || len(directionKV) > 0 {
				return output.UsageError("--threshold/--direction require --fail-on-regression")
			}
			return compareRuns(cmd.Context(), v, args[0], args[1], gs)
		},
	}
	f := cmd.Flags()
	f.BoolVar(&failOnRegression, "fail-on-regression", false, "Exit 4 when run-b regresses vs run-a on any --threshold metric")
	f.StringToStringVar(&thresholdKV, "threshold", nil, "Per-metric allowed slack: metric=delta (repeatable, comma-separated)")
	f.StringToStringVar(&directionKV, "direction", nil, "Per-metric improvement direction: metric=min|max (default: min for latency_ms/cost_usd, max otherwise)")
	return cmd
}

// parseGateSpec converts the flag maps into a run.GateSpec.
// --fail-on-regression without at least one --threshold entry is a usage
// error: an empty gate would pass vacuously and hide regressions.
func parseGateSpec(thresholds, directions map[string]string) (*run.GateSpec, error) {
	if len(thresholds) == 0 {
		return nil, output.UsageError("--fail-on-regression requires at least one --threshold metric=delta")
	}
	spec := &run.GateSpec{
		Thresholds: map[string]float64{},
		Directions: map[string]run.Direction{},
	}
	for m, raw := range thresholds {
		var f float64
		if _, err := fmt.Sscanf(raw, "%g", &f); err != nil || f < 0 {
			return nil, output.UsageError(fmt.Sprintf("--threshold %s=%s: delta must be a non-negative number", m, raw))
		}
		spec.Thresholds[m] = f
	}
	for m, raw := range directions {
		switch raw {
		case "min":
			spec.Directions[m] = run.DirectionMin
		case "max":
			spec.Directions[m] = run.DirectionMax
		default:
			return nil, output.UsageError(fmt.Sprintf("--direction %s=%s: direction must be min or max", m, raw))
		}
	}
	return spec, nil
}

func compareRuns(ctx context.Context, v *viper.Viper, idA, idB string, gs *run.GateSpec) error {
	dataDir, err := resolveDataDir()
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}
	store, err := storage.Open(dataDir)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	runA, err := store.Get(ctx, idA)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return clierr.NoRun(idA)
		}
		return fmt.Errorf("load run %s: %w", idA, err)
	}
	runB, err := store.Get(ctx, idB)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return clierr.NoRun(idB)
		}
		return fmt.Errorf("load run %s: %w", idB, err)
	}

	diff := buildDiff(runA, runB)
	if gs != nil {
		gr := run.Gate(runA, runB, *gs)
		diff.Gate = &gr
	}
	format := v.GetString("format")

	var renderErr error
	if format == "table" {
		renderErr = renderCompareTable(os.Stdout, diff)
	} else {
		renderErr = output.Render(os.Stdout, format, diff)
	}
	if renderErr != nil {
		return renderErr
	}
	if diff.Gate != nil && !diff.Gate.Pass {
		for _, c := range diff.Gate.Checks {
			if c.Regression {
				detail := c.Reason
				if detail == "" {
					detail = fmt.Sprintf("%s/%s delta %+.4g", c.Candidate, c.Metric, c.Delta)
				}
				return clierr.Regression(fmt.Sprintf("%s (candidate %q)", detail, c.Candidate))
			}
		}
	}
	return nil
}

func buildDiff(a, b *run.Run) compareDiff {
	// Index b candidates by name for O(1) lookup.
	bByName := map[string]run.CandidateResult{}
	for _, c := range b.Candidates {
		bByName[c.Name] = c
	}

	var diffs []candidateDiff
	for _, ca := range a.Candidates {
		cb, ok := bByName[ca.Name]
		if !ok {
			cb = run.CandidateResult{Name: ca.Name, Metrics: map[string]float64{}}
		}
		// Collect union of metric keys.
		keys := map[string]struct{}{}
		for k := range ca.Metrics {
			keys[k] = struct{}{}
		}
		for k := range cb.Metrics {
			keys[k] = struct{}{}
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)

		mds := make([]metricDiff, 0, len(sorted))
		for _, k := range sorted {
			va := ca.Metrics[k]
			vb := cb.Metrics[k]
			mds = append(mds, metricDiff{
				Metric: k,
				RunA:   va,
				RunB:   vb,
				Delta:  vb - va,
			})
		}
		diffs = append(diffs, candidateDiff{
			Candidate: ca.Name,
			WinnerA:   a.Winner != nil && *a.Winner == ca.Name,
			WinnerB:   b.Winner != nil && *b.Winner == ca.Name,
			Metrics:   mds,
		})
	}

	winnerA := ""
	if a.Winner != nil {
		winnerA = *a.Winner
	}
	winnerB := ""
	if b.Winner != nil {
		winnerB = *b.Winner
	}
	return compareDiff{
		RunIDA:     a.RunID,
		RunIDB:     b.RunID,
		WinnerA:    winnerA,
		WinnerB:    winnerB,
		Candidates: diffs,
	}
}

func renderCompareTable(w *os.File, diff compareDiff) error {
	_, _ = fmt.Fprintf(w, "Run A: %s  winner=%s\n", diff.RunIDA, diff.WinnerA)
	_, _ = fmt.Fprintf(w, "Run B: %s  winner=%s\n\n", diff.RunIDB, diff.WinnerB)
	for _, cd := range diff.Candidates {
		_, _ = fmt.Fprintf(w, "Candidate: %s", cd.Candidate)
		if cd.WinnerA {
			_, _ = fmt.Fprintf(w, " [winner in A]")
		}
		if cd.WinnerB {
			_, _ = fmt.Fprintf(w, " [winner in B]")
		}
		_, _ = fmt.Fprintln(w)
		if len(cd.Metrics) > 0 {
			if err := output.Render(w, output.Table, cd.Metrics); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintln(w)
	}
	if diff.Gate != nil {
		verdict := "PASS"
		if !diff.Gate.Pass {
			verdict = "FAIL"
		}
		_, _ = fmt.Fprintf(w, "Gate: %s\n", verdict)
		for _, c := range diff.Gate.Checks {
			mark := "ok"
			if c.Regression {
				mark = "REGRESSION"
			}
			_, _ = fmt.Fprintf(w, "  [%s] %s/%s dir=%s threshold=%.4g baseline=%.4g value=%.4g delta=%+.4g %s\n",
				mark, c.Candidate, c.Metric, c.Direction, c.Threshold, c.Baseline, c.Value, c.Delta, c.Reason)
		}
	}
	return nil
}
