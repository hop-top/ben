package unit_test

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/metrics"
	"hop.top/ben/internal/scorer"
	"hop.top/ben/internal/spec"
)

func TestMaxRSSToKiB(t *testing.T) {
	tests := []struct {
		goos string
		raw  int64
		want int64
	}{
		{"linux", 20480, 20480},
		{"darwin", 20 << 20, 20480},
		{"ios", 20 << 20, 20480},
		{"freebsd", 20480, 20480},
		{"openbsd", 20480, 20480},
		{"linux", 0, 0},
		{"darwin", -5, 0},
	}
	for _, tc := range tests {
		t.Run(tc.goos, func(t *testing.T) {
			assert.Equal(t, tc.want, adapter.MaxRSSToKiB(tc.goos, tc.raw))
		})
	}
}

func buildBurn(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "burn")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", bin, "./testdata/burn").CombinedOutput()
	require.NoError(t, err, "build burn: %s", out)
	return bin
}

func requireUsage(t *testing.T, r *adapter.Result) {
	t.Helper()
	require.NotNil(t, r.CPUUserMs)
	require.NotNil(t, r.CPUSysMs)
	cpu := metricVal(t, "cpu_ms", r)
	assert.GreaterOrEqual(t, cpu, 30.0)
	assert.InDelta(t, *r.CPUUserMs+*r.CPUSysMs, cpu, 1e-6)
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		require.NotNil(t, r.MaxRSSKB)
		assert.GreaterOrEqual(t, *r.MaxRSSKB, int64(15000))
	}
}

func metricVal(t *testing.T, name string, r *adapter.Result) float64 {
	t.Helper()
	m, ok := metrics.Get(name)
	require.True(t, ok, name)
	return m.Collect(r)
}

func TestCLI_RusageShellMode(t *testing.T) {
	bin := buildBurn(t)
	a := adapter.NewCLI()
	r, err := a.Run(context.Background(),
		spec.Candidate{Name: "burn", Adapter: "cli", Cmd: `"` + bin + `" a`}, nil)
	require.NoError(t, err)
	assert.Equal(t, "a", r.Output)
	requireUsage(t, r)
}

func TestCLI_RusageArgvMode(t *testing.T) {
	bin := buildBurn(t)
	a := adapter.NewCLI()
	r, err := a.Run(context.Background(),
		spec.Candidate{Name: "burn", Adapter: "cli", Cmd: bin,
			Args: []string{"hello world", "{{input.x}}", "$HOME", "*"}},
		map[string]string{"x": "a b"})
	require.NoError(t, err)
	// No shell: no word splitting, no expansion, no globbing.
	assert.Equal(t, "hello world|a b|$HOME|*", r.Output)
	requireUsage(t, r)
}

func TestSpec_ArgsPresenceSelectsArgvMode(t *testing.T) {
	var with, without struct {
		C spec.Candidate `yaml:"c"`
	}
	require.NoError(t, yaml.Unmarshal([]byte("c: {name: x, adapter: cli, cmd: burn, args: []}"), &with))
	assert.NotNil(t, with.C.Args)
	require.NoError(t, yaml.Unmarshal([]byte("c: {name: x, adapter: cli, cmd: burn}"), &without))
	assert.Nil(t, without.C.Args)
}

func TestRusageMetrics_AbsentIsNaN(t *testing.T) {
	r := &adapter.Result{}
	for _, n := range []string{"cpu_user_ms", "cpu_sys_ms", "cpu_ms", "max_rss_kb"} {
		assert.True(t, math.IsNaN(metricVal(t, n, r)), n)
	}
}

func TestRusageMetrics_Values(t *testing.T) {
	u, s, rss := 12.5, 7.5, int64(2048)
	r := &adapter.Result{CPUUserMs: &u, CPUSysMs: &s, MaxRSSKB: &rss}
	assert.Equal(t, 12.5, metricVal(t, "cpu_user_ms", r))
	assert.Equal(t, 7.5, metricVal(t, "cpu_sys_ms", r))
	assert.Equal(t, 20.0, metricVal(t, "cpu_ms", r))
	assert.Equal(t, 2048.0, metricVal(t, "max_rss_kb", r))
}

func TestBinaryAdapter_ReportsUsage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script plugin")
	}
	p := filepath.Join(t.TempDir(), "ben-adapter-fake")
	script := "#!/bin/sh\ncat >/dev/null\nprintf '{\"output\":\"x\",\"cpu_user_ms\":3.5,\"cpu_sys_ms\":1.5,\"max_rss_kb\":999}'\n"
	require.NoError(t, os.WriteFile(p, []byte(script), 0o755))
	r, err := adapter.NewBinaryAdapter(p).Run(context.Background(), spec.Candidate{Name: "f"}, nil)
	require.NoError(t, err)
	require.NotNil(t, r.CPUUserMs)
	assert.Equal(t, 3.5, *r.CPUUserMs)
	assert.Equal(t, 1.5, *r.CPUSysMs)
	assert.Equal(t, int64(999), *r.MaxRSSKB)
	assert.Equal(t, 5.0, metricVal(t, "cpu_ms", r))
}

func TestSingleScorer_MissingMetricRanksLast(t *testing.T) {
	sc, err := scorer.Parse("single:max_rss_kb", nil)
	require.NoError(t, err)
	out := sc.Score([]scorer.CandidateResult{
		{Name: "none", Metrics: map[string]float64{}},
		{Name: "big", Metrics: map[string]float64{"max_rss_kb": 900}},
		{Name: "small", Metrics: map[string]float64{"max_rss_kb": 100}},
	})
	assert.Equal(t, []string{"small", "big", "none"}, []string{out[0].Name, out[1].Name, out[2].Name})
}
