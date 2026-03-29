package metrics

import "hop.top/ben/internal/adapter"

func init() {
	Register(latencyMetric{})
	Register(exitCodeMetric{})
	Register(outputSizeMetric{})
}

// latencyMetric returns the wall-clock duration of the run in milliseconds.
type latencyMetric struct{}

func (latencyMetric) Name() string                    { return "latency_ms" }
func (latencyMetric) Collect(r *adapter.Result) float64 { return float64(r.DurationMs) }

// exitCodeMetric returns the process exit code.
type exitCodeMetric struct{}

func (exitCodeMetric) Name() string                    { return "exit_code" }
func (exitCodeMetric) Collect(r *adapter.Result) float64 { return float64(r.ExitCode) }

// outputSizeMetric returns the byte length of the combined output.
type outputSizeMetric struct{}

func (outputSizeMetric) Name() string                    { return "output_size" }
func (outputSizeMetric) Collect(r *adapter.Result) float64 { return float64(len(r.Output)) }
