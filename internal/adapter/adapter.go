// Package adapter defines the Adapter interface and shared result types.
package adapter

import (
	"context"

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
	Metadata   map[string]any
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
