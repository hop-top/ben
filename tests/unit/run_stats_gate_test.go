package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/run"
)

func TestComputeStats_SingleTrial(t *testing.T) {
	stats := run.ComputeStats([]map[string]float64{{"latency_ms": 12}})
	require.Contains(t, stats, "latency_ms")
	s := stats["latency_ms"]
	assert.Equal(t, 1, s.N)
	assert.Equal(t, 12.0, s.Mean)
	assert.Equal(t, 0.0, s.Stddev, "sample stddev is 0 for n=1")
	assert.Equal(t, 12.0, s.Min)
	assert.Equal(t, 12.0, s.Max)
}

func TestComputeStats_MultiTrial(t *testing.T) {
	stats := run.ComputeStats([]map[string]float64{
		{"m": 1}, {"m": 2}, {"m": 3},
	})
	s := stats["m"]
	assert.Equal(t, 3, s.N)
	assert.InDelta(t, 2.0, s.Mean, 1e-9)
	assert.InDelta(t, 1.0, s.Stddev, 1e-9, "sample stddev of 1,2,3 is 1")
	assert.Equal(t, 1.0, s.Min)
	assert.Equal(t, 3.0, s.Max)
}

func TestComputeStats_MetricMissingFromSomeTrials(t *testing.T) {
	stats := run.ComputeStats([]map[string]float64{
		{"a": 1, "b": 5}, {"a": 3},
	})
	assert.Equal(t, 2, stats["a"].N)
	assert.Equal(t, 1, stats["b"].N, "N reflects trials carrying the metric")
}

func TestMeanMetrics(t *testing.T) {
	m := run.MeanMetrics([]map[string]float64{{"x": 2}, {"x": 4}})
	assert.InDelta(t, 3.0, m["x"], 1e-9)
}

func mkRun(id string, name string, metrics map[string]float64) *run.Run {
	return &run.Run{
		RunID:      id,
		Candidates: []run.CandidateResult{{Name: name, Metrics: metrics}},
	}
}

func TestGate_PassWithinThreshold(t *testing.T) {
	a := mkRun("A", "c", map[string]float64{"latency_ms": 100})
	b := mkRun("B", "c", map[string]float64{"latency_ms": 105})
	res := run.Gate(a, b, run.GateSpec{Thresholds: map[string]float64{"latency_ms": 10}})
	assert.True(t, res.Pass)
	require.Len(t, res.Checks, 1)
	assert.False(t, res.Checks[0].Regression)
	assert.Equal(t, run.DirectionMin, res.Checks[0].Direction, "latency defaults to lower-is-better")
}

func TestGate_RegressionDirectionMin(t *testing.T) {
	a := mkRun("A", "c", map[string]float64{"latency_ms": 100})
	b := mkRun("B", "c", map[string]float64{"latency_ms": 150})
	res := run.Gate(a, b, run.GateSpec{Thresholds: map[string]float64{"latency_ms": 10}})
	assert.False(t, res.Pass)
	assert.True(t, res.Checks[0].Regression)
	assert.Contains(t, res.Checks[0].Reason, "latency_ms")
}

func TestGate_RegressionDirectionMaxDefault(t *testing.T) {
	// accuracy defaults to higher-is-better; a drop beyond threshold fails.
	a := mkRun("A", "c", map[string]float64{"accuracy": 0.9})
	b := mkRun("B", "c", map[string]float64{"accuracy": 0.7})
	res := run.Gate(a, b, run.GateSpec{Thresholds: map[string]float64{"accuracy": 0.05}})
	assert.False(t, res.Pass)

	// An improvement never regresses regardless of magnitude.
	b2 := mkRun("B2", "c", map[string]float64{"accuracy": 0.99})
	res2 := run.Gate(a, b2, run.GateSpec{Thresholds: map[string]float64{"accuracy": 0.05}})
	assert.True(t, res2.Pass)
}

func TestGate_DirectionOverride(t *testing.T) {
	// Force accuracy to lower-is-better; the same drop now passes.
	a := mkRun("A", "c", map[string]float64{"accuracy": 0.9})
	b := mkRun("B", "c", map[string]float64{"accuracy": 0.7})
	res := run.Gate(a, b, run.GateSpec{
		Thresholds: map[string]float64{"accuracy": 0.05},
		Directions: map[string]run.Direction{"accuracy": run.DirectionMin},
	})
	assert.True(t, res.Pass)
}

func TestGate_MissingMetricFails(t *testing.T) {
	a := mkRun("A", "c", map[string]float64{"accuracy": 0.9})
	b := mkRun("B", "c", map[string]float64{})
	res := run.Gate(a, b, run.GateSpec{Thresholds: map[string]float64{"accuracy": 0.05}})
	assert.False(t, res.Pass)
	assert.Contains(t, res.Checks[0].Reason, "missing")
}

func TestGate_MissingCandidateFails(t *testing.T) {
	a := mkRun("A", "c", map[string]float64{"accuracy": 0.9})
	b := mkRun("B", "other", map[string]float64{"accuracy": 0.9})
	res := run.Gate(a, b, run.GateSpec{Thresholds: map[string]float64{"accuracy": 0.05}})
	assert.False(t, res.Pass)
	assert.Contains(t, res.Checks[0].Reason, "absent")
}
