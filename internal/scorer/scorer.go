// Package scorer ranks benchmark candidates by their collected metrics.
package scorer

import (
	"fmt"
	"sort"
	"strings"
)

// CandidateResult holds raw metric values for one candidate.
type CandidateResult struct {
	Name      string
	Metrics   map[string]float64
	RawOutput string
	Err       error
}

// ScoredResult extends CandidateResult with a computed score and rank.
// Rank 1 is best. Rank 0 means scoring was not applied (raw mode).
type ScoredResult struct {
	CandidateResult
	Score float64
	Rank  int
}

// Scorer ranks a slice of CandidateResult values.
type Scorer interface {
	Score(candidates []CandidateResult) []ScoredResult
}

// Parse returns a Scorer for the given strategy string and weights map.
//
// Supported strategies:
//   - "single:<metric>"  — rank ascending by one metric (lower = better)
//   - "weighted"         — min-max normalise each metric, compute weighted sum, rank descending
//   - "raw"              — no ranking; all ranks 0, all scores 0
func Parse(strategy string, weights map[string]float64) (Scorer, error) {
	if strategy == "raw" {
		return &rawScorer{}, nil
	}
	if metric, ok := strings.CutPrefix(strategy, "single:"); ok {
		if metric == "" {
			return nil, fmt.Errorf("scorer: single strategy requires a metric name")
		}
		return &singleScorer{metric: metric}, nil
	}
	if strategy == "weighted" {
		if len(weights) == 0 {
			return nil, fmt.Errorf("scorer: weighted strategy requires at least one weight")
		}
		return &weightedScorer{weights: weights}, nil
	}
	return nil, fmt.Errorf("scorer: unknown strategy %q (valid: single:<metric>, weighted, raw)", strategy)
}

// rawScorer assigns rank 0 and score 0 to all candidates.
type rawScorer struct{}

func (s *rawScorer) Score(candidates []CandidateResult) []ScoredResult {
	out := make([]ScoredResult, len(candidates))
	for i, c := range candidates {
		out[i] = ScoredResult{CandidateResult: c, Score: 0, Rank: 0}
	}
	return out
}

// singleScorer ranks candidates by one metric ascending (lower = better, e.g. latency).
type singleScorer struct {
	metric string
}

func (s *singleScorer) Score(candidates []CandidateResult) []ScoredResult {
	out := make([]ScoredResult, len(candidates))
	for i, c := range candidates {
		out[i] = ScoredResult{CandidateResult: c, Score: c.Metrics[s.metric]}
	}
	// Stable sort ascending by the metric value.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score < out[j].Score
	})
	// Assign ranks 1..N.
	for i := range out {
		out[i].Rank = i + 1
	}
	return out
}

// weightedScorer min-max normalises each metric and computes a weighted sum.
// Higher score = better; rank 1 = highest score.
type weightedScorer struct {
	weights map[string]float64
}

func (s *weightedScorer) Score(candidates []CandidateResult) []ScoredResult {
	if len(candidates) == 0 {
		return nil
	}

	// Collect min/max per metric across all candidates.
	type minmax struct{ min, max float64 }
	ranges := map[string]minmax{}
	for name := range s.weights {
		mn, mx := candidates[0].Metrics[name], candidates[0].Metrics[name]
		for _, c := range candidates[1:] {
			v := c.Metrics[name]
			if v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
		}
		ranges[name] = minmax{mn, mx}
	}

	// Compute weighted score for each candidate.
	out := make([]ScoredResult, len(candidates))
	totalWeight := 0.0
	for _, w := range s.weights {
		totalWeight += w
	}

	for i, c := range candidates {
		var score float64
		for name, w := range s.weights {
			r := ranges[name]
			var norm float64
			if r.max == r.min {
				// All candidates tied on this metric — contribute 0.5 (neutral).
				norm = 0.5
			} else {
				norm = (c.Metrics[name] - r.min) / (r.max - r.min)
			}
			score += (w / totalWeight) * norm
		}
		out[i] = ScoredResult{CandidateResult: c, Score: score}
	}

	// Stable sort descending by score (higher = better).
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})

	for i := range out {
		out[i].Rank = i + 1
	}
	return out
}
