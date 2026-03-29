package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"hop.top/ben/internal/run"
)

// binaryReporterRequest is written to the plugin's stdin.
type binaryReporterRequest struct {
	Run *run.Run `json:"run"`
}

// BinaryReporter delegates formatting to a ben-reporter-* binary.
// The plugin reads {"run":{...}} from stdin and writes formatted output to stdout.
type BinaryReporter struct {
	binPath string
}

// NewBinaryReporter returns a BinaryReporter that invokes binPath.
func NewBinaryReporter(binPath string) *BinaryReporter {
	return &BinaryReporter{binPath: binPath}
}

// Report implements Reporter. It writes the run as JSON to the plugin's stdin
// and passes the plugin's stdout through to w.
func (b *BinaryReporter) Report(w io.Writer, r *run.Run) error {
	req := binaryReporterRequest{Run: r}
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("binary reporter: marshal request: %w", err)
	}

	cmd := exec.Command(b.binPath)
	cmd.Stdin = bytes.NewReader(reqData)

	out, runErr := cmd.Output()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return fmt.Errorf("binary reporter %q: exit %d: %s",
				b.binPath, exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return fmt.Errorf("binary reporter %q: %w", b.binPath, runErr)
	}

	_, err = w.Write(out)
	return err
}
