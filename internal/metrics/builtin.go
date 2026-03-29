package metrics

import (
	"strings"

	"hop.top/ben/internal/adapter"
)

func init() {
	Register(latencyMetric{})
	Register(exitCodeMetric{})
	Register(outputSizeMetric{})
	Register(outputLinesMetric{})
}

// latencyMetric returns the wall-clock duration of the run in milliseconds.
type latencyMetric struct{}

func (latencyMetric) Name() string                       { return "latency_ms" }
func (latencyMetric) Collect(r *adapter.Result) float64  { return float64(r.DurationMs) }

// exitCodeMetric returns the process exit code.
type exitCodeMetric struct{}

func (exitCodeMetric) Name() string                      { return "exit_code" }
func (exitCodeMetric) Collect(r *adapter.Result) float64 { return float64(r.ExitCode) }

// outputSizeMetric returns the byte length of the combined output.
type outputSizeMetric struct{}

func (outputSizeMetric) Name() string                      { return "output_size" }
func (outputSizeMetric) Collect(r *adapter.Result) float64 { return float64(len(r.Output)) }

// outputLinesMetric returns the number of lines in the output.
//
// Counting rules:
//   - empty string → 0
//   - "foo" (no newline) → 1
//   - "a\nb\n" (trailing newline) → 2
//   - "a\nb\nc" (no trailing newline) → 3
type outputLinesMetric struct{}

func (outputLinesMetric) Name() string { return "output_lines" }
func (outputLinesMetric) Collect(r *adapter.Result) float64 {
	s := r.Output
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if s[len(s)-1] != '\n' {
		n++
	}
	return float64(n)
}
