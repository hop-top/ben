package reporter

import (
	"io"

	"hop.top/ben/internal/run"
	"hop.top/kit/output"
)

// yamlReporter renders a Run as YAML.
type yamlReporter struct{}

func (y *yamlReporter) Report(w io.Writer, r *run.Run) error {
	return output.Render(w, output.YAML, r)
}
