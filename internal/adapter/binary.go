package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"hop.top/ben/internal/spec"
)

// binaryAdapterRequest is written to the plugin's stdin.
type binaryAdapterRequest struct {
	Action    string         `json:"action"`
	Candidate spec.Candidate `json:"candidate"`
	Input     map[string]any `json:"input"`
}

// binaryAdapterResponse is read from the plugin's stdout.
type binaryAdapterResponse struct {
	Metrics map[string]float64 `json:"metrics"`
	Output  string             `json:"output"`

	// Directions optionally declares, per reported metric, "lower" or
	// "higher" is better. Undeclared plugin metrics are higher-is-better.
	Directions map[string]string `json:"directions,omitempty"`

	// Optional resource usage; omit when unmeasured.
	CPUUserMs *float64 `json:"cpu_user_ms,omitempty"`
	CPUSysMs  *float64 `json:"cpu_sys_ms,omitempty"`
	MaxRSSKB  *int64   `json:"max_rss_kb,omitempty"`
}

// BinaryAdapter runs a ben-adapter-* binary on PATH using the stdio JSON protocol.
type BinaryAdapter struct {
	binPath string
}

// NewBinaryAdapter returns a BinaryAdapter that invokes binPath.
func NewBinaryAdapter(binPath string) *BinaryAdapter {
	return &BinaryAdapter{binPath: binPath}
}

// Run implements Adapter. It writes the request JSON to stdin and reads
// the response JSON from stdout. Stderr is passed through to os.Stderr.
// DurationMs is the wall-clock duration of the subprocess.
// ExitCode is taken from the process exit code.
func (b *BinaryAdapter) Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error) {
	inputAny := make(map[string]any, len(input))
	for k, v := range input {
		inputAny[k] = v
	}

	req := binaryAdapterRequest{
		Action:    "run",
		Candidate: c,
		Input:     inputAny,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return &Result{}, fmt.Errorf("binary adapter: marshal request: %w", err)
	}

	cmd := exec.CommandContext(ctx, b.binPath)
	cmd.Stdin = io.Reader(bytes.NewReader(reqData))
	cmd.Stderr = os.Stderr

	start := time.Now()
	out, runErr := cmd.Output()
	elapsed := time.Since(start)

	result := &Result{
		DurationMs: elapsed.Milliseconds(),
		Duration:   elapsed,
	}

	if runErr != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			// Try to parse partial output if any.
			_ = parseAdapterResponse(out, result)
			return result, nil
		}
		return result, fmt.Errorf("binary adapter %q: %w", b.binPath, runErr)
	}

	result.ExitCode = 0
	if err := parseAdapterResponse(out, result); err != nil {
		return result, fmt.Errorf("binary adapter %q: decode response: %w", b.binPath, err)
	}
	return result, nil
}

func parseAdapterResponse(data []byte, r *Result) error {
	if len(data) == 0 {
		return nil
	}
	var resp binaryAdapterResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	r.Output = resp.Output
	if len(resp.Directions) > 0 {
		if r.Metadata == nil {
			r.Metadata = make(map[string]any)
		}
		r.Metadata["plugin_directions"] = resp.Directions
	}
	r.CPUUserMs = resp.CPUUserMs
	r.CPUSysMs = resp.CPUSysMs
	r.MaxRSSKB = resp.MaxRSSKB
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}
	for k, v := range resp.Metrics {
		r.Metadata["plugin_metric_"+k] = v
	}
	// Store raw metrics map for callers that want them.
	r.Metadata["plugin_metrics"] = resp.Metrics
	return nil
}
