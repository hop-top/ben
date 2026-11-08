// Package scorer ranks benchmark candidates by their collected metrics.
package scorer

import (
	"fmt"
	"sort"
	"strings"

	"hop.top/ben/internal/metrics"
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
//
// Contract for Score:
//   - Must return a slice of the same length as the input; one ScoredResult
//     per input CandidateResult in any order.
//   - Ordering is stable: ties preserve the original input order.
//   - Rank 1 is best; ranks are contiguous starting at 1 (no gaps).
//   - Rank 0 means no ranking was applied (raw mode); Score is also 0.
//   - Must not mutate the input slice or any CandidateResult it contains.
type Scorer interface {
	Score(candidates []CandidateResult) []ScoredResult
}

// Parse returns a Scorer for the given strategy string and weights map.
//
// Supported strategies:
//   - "single:<metric>"       — rank best first by one metric: ascending for
//     lower-is-better metrics (latency, cpu, memory, cost), descending otherwise
//   - "single:<metric>:asc"   — explicit ascending, overriding the metric's direction
//   - "single:<metric>:desc"  — explicit descending, overriding the metric's direction
//   - "weighted"              — min-max normalise each metric (inverted for
//     lower-is-better ones so 1 is always best), weighted sum, rank descending
//   - "raw"                   — no ranking; all ranks 0, all scores 0
//
// Direction comes from metrics.DirectionOf; undeclared metrics are
// higher-is-better.
func Parse(strategy string, weights map[string]float64) (Scorer, error) {
	if strategy == "raw" {
		return &rawScorer{}, nil
	}
	if metric, ok := strings.CutPrefix(strategy, "single:"); ok {
		explicit := dirAuto
		if m, dir, hasDir := strings.Cut(metric, ":"); hasDir {
			switch dir {
			case "asc":
				explicit = dirAsc
			case "desc":
				explicit = dirDesc
			default:
				return nil, fmt.Errorf("scorer: unknown direction %q (valid: asc, desc)", dir)
			}
			metric = m
		}
		if metric == "" {
			return nil, fmt.Errorf("scorer: single strategy requires a metric name")
		}
		return &singleScorer{metric: metric, dir: explicit}, nil
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

// singleScorer ranks candidates by one metric, best first. dir overrides the
// metric's registered direction when the strategy carries :asc or :desc.
type singleScorer struct {
	metric string
	dir    explicitDir
}

type explicitDir int

const (
	dirAuto explicitDir = iota // follow metrics.DirectionOf
	dirAsc
	dirDesc
)

func (s *singleScorer) Score(candidates []CandidateResult) []ScoredResult {
	out := make([]ScoredResult, len(candidates))
	for i, c := range candidates {
		out[i] = ScoredResult{CandidateResult: c, Score: c.Metrics[s.metric]}
	}
	// Stable sort best first. A candidate that did not
	// report the metric (e.g. max_rss_kb unavailable) ranks after all that did.
	lower := metrics.DirectionOf(s.metric) == metrics.LowerIsBetter
	switch s.dir {
	case dirAsc:
		lower = true
	case dirDesc:
		lower = false
	}
	has := func(r ScoredResult) bool { _, ok := r.Metrics[s.metric]; return ok }
	sort.SliceStable(out, func(i, j int) bool {
		hi, hj := has(out[i]), has(out[j])
		if hi != hj {
			return hi
		}
		if lower {
			return out[i].Score < out[j].Score
		}
		return out[i].Score > out[j].Score
	})
	// Assign ranks 1..N.
	for i := range out {
		out[i].Rank = i + 1
	}
	return out
}

// weightedScorer min-max normalises each metric and computes a weighted sum.
// Higher score = better for every metric (lower-is-better ones are inverted);
// rank 1 = highest score.
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
				if metrics.DirectionOf(name) == metrics.LowerIsBetter {
					norm = 1 - norm
				}
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

// SingleMetric extracts the metric name from a "single:<metric>[:dir]"
// strategy string. ok is false for every other strategy shape. Callers
// use it to validate the scored metric against collected metrics before
// scoring (a bogus metric must be an error, not a silent all-zero score).
func SingleMetric(strategy string) (string, bool) {
	rest, ok := strings.CutPrefix(strategy, "single:")
	if !ok {
		return "", false
	}
	if m, _, hasDir := strings.Cut(rest, ":"); hasDir {
		return m, true
	}
	return rest, true
}
