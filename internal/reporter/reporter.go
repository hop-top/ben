// Package reporter formats a run.Run for human or machine consumption.
package reporter

import (
	"fmt"
	"io"

	"hop.top/ben/internal/run"
)

// Reporter writes a Run result to w.
type Reporter interface {
	Report(w io.Writer, r *run.Run) error
}

// New returns a Reporter for the given format string.
// Valid values: "json", "yaml", "table".
func New(format string) (Reporter, error) {
	switch format {
	case "json":
		return &jsonReporter{}, nil
	case "yaml":
		return &yamlReporter{}, nil
	case "table":
		return &tableReporter{}, nil
	default:
		return nil, fmt.Errorf("reporter: unknown format %q (valid: json, yaml, table)", format)
	}
}
