// Package reporter formats a run.Run for human or machine consumption.
package reporter

import (
	"fmt"
	"io"

	"hop.top/ben/internal/run"
)

// Reporter writes a Run result to w.
//
// Contract for Report:
//   - All formatted output MUST go to w; nothing is written to stdout.
//   - Diagnostic or debug messages MAY be written to stderr (os.Stderr).
//   - Return a non-nil error only when writing to w fails; encoding errors
//     that produce partial output must also return a non-nil error.
//   - A nil error guarantees that w received a complete, valid document.
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
