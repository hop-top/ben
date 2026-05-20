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
	Metric  string  `table:"Metric" json:"metric" yaml:"metric"`
	RunA    float64 `table:"Run-A"  json:"run_a"  yaml:"run_a"`
	RunB    float64 `table:"Run-B"  json:"run_b"  yaml:"run_b"`
	Delta   float64 `table:"Delta"  json:"delta"  yaml:"delta"`
}

// candidateDiff compares one candidate across two runs.
type candidateDiff struct {
	Candidate string       `json:"candidate" yaml:"candidate"`
	WinnerA   bool         `json:"winner_a"  yaml:"winner_a"`
	WinnerB   bool         `json:"winner_b"  yaml:"winner_b"`
	Metrics   []metricDiff `json:"metrics"   yaml:"metrics"`
}

// compareDiff is the structured result of a compare operation.
type compareDiff struct {
	RunIDA    string          `json:"run_id_a" yaml:"run_id_a"`
	RunIDB    string          `json:"run_id_b" yaml:"run_id_b"`
	WinnerA   string          `json:"winner_a" yaml:"winner_a"`
	WinnerB   string          `json:"winner_b" yaml:"winner_b"`
	Candidates []candidateDiff `json:"candidates" yaml:"candidates"`
}

func compareCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "compare <run-a> <run-b>",
		Short: "Compare two runs side-by-side",
		Args:  cobra.ExactArgs(2),
		Annotations: map[string]string{
			"kit/side-effect": "read",
			"kit/idempotent":  "yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return compareRuns(cmd.Context(), v, args[0], args[1])
		},
	}
}

func compareRuns(ctx context.Context, v *viper.Viper, idA, idB string) error {
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
	format := v.GetString("format")

	if format == "table" {
		return renderCompareTable(os.Stdout, diff)
	}
	return output.Render(os.Stdout, format, diff)
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
	return nil
}
