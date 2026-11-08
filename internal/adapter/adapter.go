// Package adapter defines the Adapter interface and shared result types.
package adapter

import (
	"context"
	"time"

	"hop.top/ben/internal/spec"
)

// Result holds the outcome of running one candidate.
//
// ExitCode captures the process exit status; a non-zero value is a normal
// benchmark observation, not a failure. Err is reserved for infrastructure
// problems only (binary not found, context cancelled, I/O errors).
// DurationMs is wall-clock milliseconds measured by the adapter.
// Metadata is a side-channel for adapter-specific data (e.g. token counts,
// model name) that metrics can read without coupling to a specific adapter.
type Result struct {
	Output     string
	ExitCode   int
	Err        error
	DurationMs int64
	// Duration is the same wall-clock time at full (ns) precision; adapters
	// that measure time set it alongside DurationMs. latency_ms is derived
	// from it as fractional ms, falling back to DurationMs when zero.
	Duration time.Duration
	Metadata map[string]any

	// Optional resource usage of the waited child process. Nil means the
	// adapter or platform could not measure it; the matching metric is then
	// omitted rather than reported as 0.
	CPUUserMs *float64 // user-mode CPU time, ms
	CPUSysMs  *float64 // kernel-mode CPU time, ms
	MaxRSSKB  *int64   // peak resident set size, KiB
}

// Adapter executes a single benchmark candidate and returns a Result.
//
// Contract for Run:
//   - A non-zero exit code MUST NOT be returned as an error; it goes into
//     Result.ExitCode so scorers can treat it as a metric if desired.
//   - Return a non-nil error only for infrastructure failures: binary not
//     found, permission denied, or context cancellation/deadline exceeded.
//   - On context cancellation ctx.Err() is the canonical error to return.
//   - Result must always be non-nil even when error is non-nil, so callers
//     can inspect partial output and elapsed time.
type Adapter interface {
	Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error)
}
