// Package adapter defines the Adapter interface and shared result types.
package adapter

import (
	"context"

	"hop.top/ben/internal/spec"
)

// Result holds the outcome of running one candidate.
type Result struct {
	Output     string
	ExitCode   int
	Err        error
	DurationMs int64
}

// Adapter executes a candidate and returns a Result.
// A non-zero exit code is not an error — it is recorded in Result.ExitCode.
// Err is only set for infrastructure failures (e.g. command not found, context cancelled).
type Adapter interface {
	Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error)
}
