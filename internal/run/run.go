// Package run defines the Run result type shared across storage, reporter, and scorer.
package run

import "time"

// ScorerConfig holds the scoring strategy and optional per-metric weights.
type ScorerConfig struct {
	Strategy string             `json:"strategy"`
	Weights  map[string]float64 `json:"weights,omitempty"`
}

// CandidateResult holds per-candidate metrics, score, rank, and raw output.
type CandidateResult struct {
	Name      string             `json:"name"`
	Metrics   map[string]float64 `json:"metrics"`
	Score     float64            `json:"score"`
	Rank      int                `json:"rank"`
	RawOutput string             `json:"raw_output"`
	Error     string             `json:"error,omitempty"`
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
	Winner       string            `json:"winner"` // empty = raw mode / no winner
	Metadata     Metadata          `json:"metadata"`
}
