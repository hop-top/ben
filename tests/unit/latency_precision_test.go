package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/metrics"
	"hop.top/ben/internal/spec"
)

func TestLatency_SubMillisecondFromDuration(t *testing.T) {
	m, ok := metrics.Get("latency_ms")
	require.True(t, ok)
	r := &adapter.Result{Duration: 1500 * time.Microsecond, DurationMs: 1}
	assert.InDelta(t, 1.5, m.Collect(r), 1e-9)

	r = &adapter.Result{Duration: 250 * time.Microsecond}
	assert.InDelta(t, 0.25, m.Collect(r), 1e-9)
}

func TestLatency_FallsBackToDurationMs(t *testing.T) {
	m, _ := metrics.Get("latency_ms")
	assert.InDelta(t, 340.0, m.Collect(&adapter.Result{DurationMs: 340}), 1e-9)
}

func TestCLI_RecordsSubMsDuration(t *testing.T) {
	r, err := adapter.NewCLI().Run(context.Background(),
		spec.Candidate{Name: "t", Adapter: "cli", Cmd: "true"}, nil)
	require.NoError(t, err)
	assert.Greater(t, r.Duration, time.Duration(0))
	assert.Equal(t, r.Duration.Milliseconds(), r.DurationMs)
	m, _ := metrics.Get("latency_ms")
	assert.InDelta(t, float64(r.Duration)/float64(time.Millisecond), m.Collect(r), 1e-9)
}
