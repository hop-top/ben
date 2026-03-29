package reporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"hop.top/ben/internal/run"
	"hop.top/kit/output"
)

// tableRow is a single row in the ranked candidate table.
// Metric values are flattened into a formatted string for display.
type tableRow struct {
	Rank    int    `table:"Rank"`
	Name    string `table:"Name"`
	Score   string `table:"Score"`
	Metrics string `table:"Metrics"`
	Error   string `table:"Error"`
}

// tableReporter renders a ranked candidate table.
type tableReporter struct{}

func (t *tableReporter) Report(w io.Writer, r *run.Run) error {
	rows := make([]tableRow, 0, len(r.Candidates))
	for _, c := range r.Candidates {
		rows = append(rows, toTableRow(c))
	}
	// Sort by rank; rank 0 (raw mode) goes last sorted by name.
	sort.SliceStable(rows, func(i, j int) bool {
		ri, rj := rows[i].Rank, rows[j].Rank
		if ri == 0 && rj == 0 {
			return rows[i].Name < rows[j].Name
		}
		if ri == 0 {
			return false
		}
		if rj == 0 {
			return true
		}
		return ri < rj
	})
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no candidates")
		return err
	}
	return output.Render(w, output.Table, rows)
}

func toTableRow(c run.CandidateResult) tableRow {
	// Format metrics as "key=val key=val" sorted by key.
	keys := make([]string, 0, len(c.Metrics))
	for k := range c.Metrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%.4g", k, c.Metrics[k]))
	}
	return tableRow{
		Rank:    c.Rank,
		Name:    c.Name,
		Score:   fmt.Sprintf("%.4f", c.Score),
		Metrics: strings.Join(parts, " "),
		Error:   c.Error,
	}
}
