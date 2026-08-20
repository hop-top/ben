// Package run defines the Run result type shared across storage, reporter, and scorer.
package run

import "time"

// ScorerConfig holds the scoring strategy and optional per-metric weights.
type ScorerConfig struct {
	Strategy string             `json:"strategy"`
	Weights  map[string]float64 `json:"weights,omitempty"`
}

// CandidateResult holds per-candidate metrics, score, rank, and raw output.
// Score and Rank are nil when no scoring was applied (raw mode).
//
// Metrics holds the per-metric MEAN across trials (identical to the single
// observation when --repeat is 1). Trials preserves the raw per-trial
// values and MetricStats summarises them; both are additive fields — a
// pre-repetition run JSON without them still decodes.
type CandidateResult struct {
	Name        string                `json:"name"`
	Metrics     map[string]float64    `json:"metrics"`
	Score       *float64              `json:"score"`
	Rank        *int                  `json:"rank"`
	RawOutput   string                `json:"raw_output"`
	Error       string                `json:"error,omitempty"`
	Trials      []map[string]float64  `json:"trials,omitempty"`
	MetricStats map[string]MetricStat `json:"metric_stats,omitempty"`
}

// MetricStat summarises one metric across N trials. Stddev is the sample
// standard deviation (n-1 denominator); 0 when N == 1.
type MetricStat struct {
	Mean   float64 `json:"mean"`
	Stddev float64 `json:"stddev"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	N      int     `json:"n"`
}

// Metadata captures environment info at run time.
type Metadata struct {
	Host       string `json:"host"`
	BenVersion string `json:"ben_version"`
}

// Run is a single execution of a suite against all candidates.
// It is the canonical result type stored, reported, and scored.
type Run struct {
	RunID        string            `json:"run_id"`
	Suite        string            `json:"suite"`
	SuiteVersion int               `json:"suite_version"`
	Timestamp    time.Time         `json:"timestamp"`
	Scorer       ScorerConfig      `json:"scorer"`
	Candidates   []CandidateResult `json:"candidates"`
	Winner       *string           `json:"winner"` // nil = raw mode / no winner
	Metadata     Metadata          `json:"metadata"`
}
