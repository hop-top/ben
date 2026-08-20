package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/scorer"
)

func TestSingleScorer_DescRanksHigherFirst(t *testing.T) {
	sc, err := scorer.Parse("single:accuracy:desc", nil)
	require.NoError(t, err)
	out := sc.Score([]scorer.CandidateResult{
		{Name: "low", Metrics: map[string]float64{"accuracy": 0.2}},
		{Name: "high", Metrics: map[string]float64{"accuracy": 0.9}},
	})
	byName := map[string]int{}
	for _, r := range out {
		byName[r.Name] = r.Rank
	}
	assert.Equal(t, 1, byName["high"], "desc: highest accuracy ranks first")
	assert.Equal(t, 2, byName["low"])
}

func TestSingleScorer_ExplicitAscMatchesDefault(t *testing.T) {
	def, err := scorer.Parse("single:latency_ms", nil)
	require.NoError(t, err)
	asc, err := scorer.Parse("single:latency_ms:asc", nil)
	require.NoError(t, err)
	in := []scorer.CandidateResult{
		{Name: "slow", Metrics: map[string]float64{"latency_ms": 90}},
		{Name: "fast", Metrics: map[string]float64{"latency_ms": 10}},
	}
	rankOf := func(rs []scorer.ScoredResult, name string) int {
		for _, r := range rs {
			if r.Name == name {
				return r.Rank
			}
		}
		return -1
	}
	assert.Equal(t, rankOf(def.Score(in), "fast"), rankOf(asc.Score(in), "fast"))
	assert.Equal(t, 1, rankOf(asc.Score(in), "fast"))
}

func TestParse_UnknownDirectionErrors(t *testing.T) {
	_, err := scorer.Parse("single:accuracy:sideways", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sideways")
}

func TestSingleMetric(t *testing.T) {
	m, ok := scorer.SingleMetric("single:accuracy:desc")
	require.True(t, ok)
	assert.Equal(t, "accuracy", m)

	m, ok = scorer.SingleMetric("single:latency_ms")
	require.True(t, ok)
	assert.Equal(t, "latency_ms", m)

	_, ok = scorer.SingleMetric("weighted")
	assert.False(t, ok)
	_, ok = scorer.SingleMetric("raw")
	assert.False(t, ok)
}
