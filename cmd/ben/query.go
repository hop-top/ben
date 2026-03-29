package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/run"
	"hop.top/ben/internal/storage"
	"hop.top/kit/output"
)

// queryRow is one row in the query results table.
type queryRow struct {
	RunID     string `table:"RunID"     json:"run_id"   yaml:"run_id"`
	Suite     string `table:"Suite"     json:"suite"    yaml:"suite"`
	Timestamp string `table:"Timestamp" json:"timestamp" yaml:"timestamp"`
	Winner    string `table:"Winner"    json:"winner"   yaml:"winner"`
	Score     string `table:"Score"     json:"score"    yaml:"score"`
}

func queryCmd(v *viper.Viper) *cobra.Command {
	var (
		suiteName string
		last      int
	)
	cmd := &cobra.Command{
		Use:   "query",
		Short: "List recent benchmark runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return queryRuns(cmd.Context(), v, suiteName, last)
		},
	}
	cmd.Flags().StringVar(&suiteName, "suite", "", "Filter by suite name")
	cmd.Flags().IntVar(&last, "last", 10, "Number of most recent runs to return")
	return cmd
}

func queryRuns(ctx context.Context, v *viper.Viper, suiteName string, last int) error {
	dataDir, err := resolveDataDir()
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}
	store, err := storage.Open(dataDir)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	runs, err := store.List(ctx, suiteName, last)
	if err != nil {
		return fmt.Errorf("list runs: %w", err)
	}

	format := v.GetString("format")
	if format == "table" {
		return renderQueryTable(os.Stdout, runs)
	}
	return output.Render(os.Stdout, format, runs)
}

func renderQueryTable(w *os.File, runs []*run.Run) error {
	if len(runs) == 0 {
		_, _ = fmt.Fprintln(w, "no runs found")
		return nil
	}
	rows := make([]queryRow, 0, len(runs))
	for _, r := range runs {
		score := ""
		winner := ""
		if r.Winner != nil {
			winner = *r.Winner
			for _, c := range r.Candidates {
				if c.Name == winner && c.Score != nil {
					score = fmt.Sprintf("%.4f", *c.Score)
					break
				}
			}
		}
		rows = append(rows, queryRow{
			RunID:     r.RunID,
			Suite:     r.Suite,
			Timestamp: r.Timestamp.Format("2006-01-02T15:04:05Z"),
			Winner:    winner,
			Score:     score,
		})
	}
	return output.Render(w, output.Table, rows)
}
