package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/scorer"
)

func candidates(names []string, metric string, values []float64) []scorer.CandidateResult {
	out := make([]scorer.CandidateResult, len(names))
	for i, n := range names {
		out[i] = scorer.CandidateResult{
			Name:    n,
			Metrics: map[string]float64{metric: values[i]},
		}
	}
	return out
}

func TestSingle_LowestGetsRank1(t *testing.T) {
	s, err := scorer.Parse("single:latency_ms", nil)
	require.NoError(t, err)

	cands := candidates(
		[]string{"slow", "fast", "medium"},
		"latency_ms",
		[]float64{300, 100, 200},
	)
	results := s.Score(cands)
	require.Len(t, results, 3)

	byName := map[string]scorer.ScoredResult{}
	for _, r := range results {
		byName[r.Name] = r
	}
	assert.Equal(t, 1, byName["fast"].Rank)
	assert.Equal(t, 2, byName["medium"].Rank)
	assert.Equal(t, 3, byName["slow"].Rank)
}

func TestSingle_EqualValues_StableOrder(t *testing.T) {
	s, err := scorer.Parse("single:latency_ms", nil)
	require.NoError(t, err)

	cands := candidates(
		[]string{"a", "b", "c"},
		"latency_ms",
		[]float64{100, 100, 100},
	)
	results := s.Score(cands)
	require.Len(t, results, 3)
	// All rank 1,2,3 assigned; no panic or duplication
	ranks := []int{results[0].Rank, results[1].Rank, results[2].Rank}
	assert.ElementsMatch(t, []int{1, 2, 3}, ranks)
}

func TestSingle_EmptyMetricName(t *testing.T) {
	_, err := scorer.Parse("single:", nil)
	assert.Error(t, err)
}

func TestWeighted_NormalisationAndRanking(t *testing.T) {
	// Use 3 candidates + unequal weights so one clearly wins.
	// latency weight=0.3, quality weight=0.7 → quality dominates.
	weights := map[string]float64{
		"latency_ms":    0.3,
		"quality_score": 0.7,
	}
	s, err := scorer.Parse("weighted", weights)
	require.NoError(t, err)

	cands := []scorer.CandidateResult{
		// "a": worst latency (norm=1.0), best quality (norm=1.0)
		// score = 0.3*(1.0/1.0) + 0.7*(1.0/1.0) = 0.3 + 0.7 = 1.0 (normalised by totalWeight=1)
		{Name: "a", Metrics: map[string]float64{"latency_ms": 300, "quality_score": 0.9}},
		// "b": best latency (norm=0.0), worst quality (norm=0.0)
		// score = 0.3*0 + 0.7*0 = 0
		{Name: "b", Metrics: map[string]float64{"latency_ms": 100, "quality_score": 0.1}},
		// "c": middle values
		{Name: "c", Metrics: map[string]float64{"latency_ms": 200, "quality_score": 0.5}},
	}
	results := s.Score(cands)
	require.Len(t, results, 3)

	byName := map[string]scorer.ScoredResult{}
	for _, r := range results {
		byName[r.Name] = r
	}
	// "a" has best quality (dominant metric) → rank 1
	assert.Equal(t, 1, byName["a"].Rank)
	// "b" has worst quality → rank 3
	assert.Equal(t, 3, byName["b"].Rank)
	// "c" is in the middle → rank 2
	assert.Equal(t, 2, byName["c"].Rank)
	assert.Greater(t, byName["a"].Score, byName["c"].Score)
	assert.Greater(t, byName["c"].Score, byName["b"].Score)
}

func TestWeighted_AllTied_Score05(t *testing.T) {
	weights := map[string]float64{"latency_ms": 1.0}
	s, err := scorer.Parse("weighted", weights)
	require.NoError(t, err)

	cands := candidates(
		[]string{"a", "b"},
		"latency_ms",
		[]float64{100, 100},
	)
	results := s.Score(cands)
	require.Len(t, results, 2)
	// When all tied, norm=0.5 for both → scores equal
	assert.InDelta(t, 0.5, results[0].Score, 1e-9)
	assert.InDelta(t, 0.5, results[1].Score, 1e-9)
}

func TestWeighted_NoWeights_Error(t *testing.T) {
	_, err := scorer.Parse("weighted", map[string]float64{})
	assert.Error(t, err)
}

func TestRaw_AllRanksZero(t *testing.T) {
	s, err := scorer.Parse("raw", nil)
	require.NoError(t, err)

	cands := candidates(
		[]string{"a", "b", "c"},
		"latency_ms",
		[]float64{100, 200, 50},
	)
	results := s.Score(cands)
	require.Len(t, results, 3)
	for _, r := range results {
		assert.Equal(t, 0, r.Rank, "raw scorer: rank must be 0")
		assert.InDelta(t, 0.0, r.Score, 1e-9, "raw scorer: score must be 0")
	}
}

func TestParse_UnknownStrategy(t *testing.T) {
	_, err := scorer.Parse("banana", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown strategy")
}
