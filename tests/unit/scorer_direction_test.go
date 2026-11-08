package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/metrics"
	"hop.top/ben/internal/scorer"
)

func TestWeighted_LowerIsBetterMetricFastestWins(t *testing.T) {
	for _, m := range []string{"latency_ms", "cpu_ms", "cpu_user_ms", "cpu_sys_ms", "max_rss_kb", "cost_usd"} {
		t.Run(m, func(t *testing.T) {
			s, err := scorer.Parse("weighted", map[string]float64{m: 1})
			require.NoError(t, err)
			out := s.Score([]scorer.CandidateResult{
				{Name: "slow", Metrics: map[string]float64{m: 300}},
				{Name: "fast", Metrics: map[string]float64{m: 100}},
			})
			assert.Equal(t, "fast", out[0].Name)
			assert.Equal(t, 1, out[0].Rank)
			assert.Greater(t, out[0].Score, out[1].Score)
		})
	}
}

func TestWeighted_HigherIsBetterMetricHighestWins(t *testing.T) {
	s, err := scorer.Parse("weighted", map[string]float64{"quality_score": 1})
	require.NoError(t, err)
	out := s.Score([]scorer.CandidateResult{
		{Name: "bad", Metrics: map[string]float64{"quality_score": 0.1}},
		{Name: "good", Metrics: map[string]float64{"quality_score": 0.9}},
	})
	assert.Equal(t, "good", out[0].Name)
}

func TestSingle_HigherIsBetterMetricDescends(t *testing.T) {
	s, err := scorer.Parse("single:quality_score", nil)
	require.NoError(t, err)
	out := s.Score([]scorer.CandidateResult{
		{Name: "bad", Metrics: map[string]float64{"quality_score": 0.1}},
		{Name: "good", Metrics: map[string]float64{"quality_score": 0.9}},
	})
	assert.Equal(t, "good", out[0].Name)
}

func TestMetricDirection(t *testing.T) {
	for _, n := range []string{"latency_ms", "cpu_ms", "cpu_user_ms", "cpu_sys_ms", "max_rss_kb", "cost_usd", "tokens", "exit_code"} {
		assert.Equal(t, metrics.LowerIsBetter, metrics.DirectionOf(n), n)
	}
	for _, n := range []string{"quality_score", "accuracy", "no_such_metric"} {
		assert.Equal(t, metrics.HigherIsBetter, metrics.DirectionOf(n), n)
	}
	metrics.SetDirection("my_plugin_ms", metrics.LowerIsBetter)
	assert.Equal(t, metrics.LowerIsBetter, metrics.DirectionOf("my_plugin_ms"))
}

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

func TestSingleScorer_ExplicitDirectionOverridesMetric(t *testing.T) {
	in := []scorer.CandidateResult{
		{Name: "fast", Metrics: map[string]float64{"latency_ms": 10}},
		{Name: "slow", Metrics: map[string]float64{"latency_ms": 90}},
	}
	sc, err := scorer.Parse("single:latency_ms:desc", nil)
	require.NoError(t, err)
	assert.Equal(t, "slow", sc.Score(in)[0].Name)
}
