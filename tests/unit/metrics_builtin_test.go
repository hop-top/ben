package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/adapter"
	_ "hop.top/ben/internal/metrics" // trigger init() registration
	"hop.top/ben/internal/metrics"
)

func mockResult(durationMs int64, exitCode int, output string) *adapter.Result {
	return &adapter.Result{
		DurationMs: durationMs,
		ExitCode:   exitCode,
		Output:     output,
	}
}

func TestLatencyMs(t *testing.T) {
	m, ok := metrics.Get("latency_ms")
	require.True(t, ok)
	assert.Equal(t, "latency_ms", m.Name())

	r := mockResult(340, 0, "some output")
	assert.InDelta(t, 340.0, m.Collect(r), 1e-9)
}

func TestExitCode(t *testing.T) {
	m, ok := metrics.Get("exit_code")
	require.True(t, ok)
	assert.Equal(t, "exit_code", m.Name())

	tests := []struct {
		exitCode int
		want     float64
	}{
		{0, 0.0},
		{1, 1.0},
		{42, 42.0},
		{255, 255.0},
	}
	for _, tc := range tests {
		r := mockResult(100, tc.exitCode, "")
		assert.InDelta(t, tc.want, m.Collect(r), 1e-9)
	}
}

func TestOutputSize(t *testing.T) {
	m, ok := metrics.Get("output_size")
	require.True(t, ok)
	assert.Equal(t, "output_size", m.Name())

	tests := []struct {
		output string
		want   float64
	}{
		{"", 0.0},
		{"hello", 5.0},
		{"hello\nworld\n", 12.0},
	}
	for _, tc := range tests {
		r := mockResult(50, 0, tc.output)
		assert.InDelta(t, tc.want, m.Collect(r), 1e-9)
	}
}

func TestRegistry_Get_Unknown(t *testing.T) {
	_, ok := metrics.Get("nonexistent_metric")
	assert.False(t, ok)
}

func TestRegistry_All(t *testing.T) {
	all := metrics.All()
	assert.Contains(t, all, "latency_ms")
	assert.Contains(t, all, "exit_code")
	assert.Contains(t, all, "output_size")
}
