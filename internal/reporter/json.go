package reporter

import (
	"io"

	"hop.top/ben/internal/run"
	"hop.top/kit/output"
)

// jsonReporter renders a Run as indented JSON.
// All run fields are emitted; logs go to stderr only.
type jsonReporter struct{}

func (j *jsonReporter) Report(w io.Writer, r *run.Run) error {
	return output.Render(w, output.JSON, r)
}
