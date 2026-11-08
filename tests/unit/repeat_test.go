package unit_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/bench"
	"hop.top/ben/internal/reporter"
	"hop.top/ben/internal/run"
	"hop.top/ben/internal/spec"
)

func TestStats_MedianP95(t *testing.T) {
	trials := []map[string]float64{{"m": 1}, {"m": 2}, {"m": 3}, {"m": 4}, {"m": 100}}
	a := run.ComputeStats(trials)["m"]
	assert.Equal(t, 5, a.N)
	assert.InDelta(t, 22.0, a.Mean, 1e-9)
	assert.InDelta(t, 3.0, a.Median, 1e-9)
	assert.InDelta(t, 80.8, a.P95, 1e-9) // linear interpolation: 4 + 0.8*96

	even := run.ComputeStats([]map[string]float64{{"m": 4}, {"m": 1}, {"m": 3}, {"m": 2}})["m"]
	assert.InDelta(t, 2.5, even.Median, 1e-9)
	one := run.ComputeStats([]map[string]float64{{"m": 7}})["m"]
	assert.Equal(t, 7.0, one.P95)

	for kind, want := range map[string]float64{"": 22, "mean": 22, "median": 3, "p95": 80.8} {
		assert.InDelta(t, want, a.Pick(kind), 1e-9, kind)
		assert.InDelta(t, want, run.AggregateMetrics(trials, kind)["m"], 1e-9, kind)
	}
}

// scripted returns latency values from vals in call order and counts calls.
type scripted struct {
	vals    []float64
	calls   int32
	running int32
	overlap bool
}

func (s *scripted) Run(_ context.Context, _ spec.Candidate, _ map[string]string) (*adapter.Result, error) {
	if atomic.AddInt32(&s.running, 1) > 1 {
		s.overlap = true
	}
	defer atomic.AddInt32(&s.running, -1)
	i := int(atomic.AddInt32(&s.calls, 1)) - 1
	v := s.vals[i%len(s.vals)]
	return &adapter.Result{Output: "o", DurationMs: int64(v)}, nil
}

func TestMeasure_DefaultIsSingleRun(t *testing.T) {
	s := &scripted{vals: []float64{5}}
	out := bench.Measure(context.Background(), s, spec.Candidate{Name: "a"}, nil,
		[]string{"latency_ms"}, bench.Options{})
	require.NoError(t, out.Err)
	assert.EqualValues(t, 1, s.calls)
	assert.Equal(t, 5.0, out.Metrics["latency_ms"])
	require.Len(t, out.Trials, 1)
	assert.Equal(t, 1, out.Stats["latency_ms"].N)
}

func TestMeasure_WarmupDiscardedRunsSequential(t *testing.T) {
	// 2 warmup calls (values 1000,1000) then 5 measured.
	s := &scripted{vals: []float64{1000, 1000, 10, 20, 30, 40, 50}}
	out := bench.Measure(context.Background(), s, spec.Candidate{Name: "a"}, nil,
		[]string{"latency_ms"}, bench.Options{Repeat: 5, Warmup: 2, Aggregate: "median"})
	require.NoError(t, out.Err)
	assert.EqualValues(t, 7, s.calls)
	assert.False(t, s.overlap)
	require.Len(t, out.Trials, 5, "warmup runs are not sampled")
	assert.Equal(t, 10.0, out.Trials[0]["latency_ms"])
	assert.Equal(t, 5, out.Stats["latency_ms"].N)
	assert.Equal(t, 30.0, out.Stats["latency_ms"].Median)
	assert.Equal(t, 30.0, out.Metrics["latency_ms"])
}

func TestMeasure_AggregateChoice(t *testing.T) {
	vals := []float64{1, 2, 3, 4, 100}
	for kind, want := range map[string]float64{"median": 3, "mean": 22, "": 22} {
		s := &scripted{vals: vals}
		out := bench.Measure(context.Background(), s, spec.Candidate{Name: "a"}, nil,
			[]string{"latency_ms"}, bench.Options{Repeat: 5, Aggregate: kind})
		assert.InDelta(t, want, out.Metrics["latency_ms"], 1e-9, kind)
	}
}

func TestSpec_RepeatFields(t *testing.T) {
	var s spec.Spec
	src := "name: x\nrepeat: 20\nwarmup: 3\naggregate: p95\ncandidates:\n  - {name: a, adapter: cli, cmd: c, prepare: 'rm -f f', input_file: in.json}\n"
	require.NoError(t, yaml.Unmarshal([]byte(src), &s))
	assert.Equal(t, 20, s.Repeat)
	assert.Equal(t, 3, s.Warmup)
	assert.Equal(t, "p95", s.Aggregate)
	assert.Equal(t, "rm -f f", s.Candidates[0].Prepare)
	assert.Equal(t, "in.json", s.Candidates[0].InputFile)
}

func TestSpec_RepeatValidation(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"neg repeat": "repeat: -1\n",
		"neg warmup": "warmup: -2\n",
		"bad agg":    "aggregate: max\n",
	} {
		p := filepath.Join(dir, "s.yaml")
		require.NoError(t, os.WriteFile(p, []byte("name: x\n"+body+"candidates:\n  - {name: a, adapter: cli, cmd: c}\n"), 0o644))
		_, err := spec.Load(p)
		assert.Error(t, err, name)
	}
}

func TestCLI_PrepareRunsUntimedBeforeEachRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	c := spec.Candidate{Name: "p", Adapter: "cli",
		Prepare: "echo prep >> " + log,
		Cmd:     "echo run >> " + log}
	out := bench.Measure(context.Background(), adapter.NewCLI(), c, nil,
		[]string{"latency_ms"}, bench.Options{Repeat: 2, Warmup: 1})
	require.NoError(t, out.Err)
	b, _ := os.ReadFile(log)
	assert.Equal(t, strings.Repeat("prep\nrun\n", 3), string(b))
}

func TestCLI_PrepareFailureIsError(t *testing.T) {
	a := adapter.NewCLI()
	_, err := a.Run(context.Background(),
		spec.Candidate{Name: "p", Adapter: "cli", Prepare: "exit 3", Cmd: "true"}, nil)
	assert.Error(t, err)
}

func TestCLI_InputFileFeedsStdin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cat")
	}
	f := filepath.Join(t.TempDir(), "in.json")
	require.NoError(t, os.WriteFile(f, []byte(`{"k":"v"}`), 0o644))
	r, err := adapter.NewCLI().Run(context.Background(),
		spec.Candidate{Name: "s", Adapter: "cli", Cmd: "cat", InputFile: "{{input.f}}"},
		map[string]string{"f": f})
	require.NoError(t, err)
	assert.Equal(t, `{"k":"v"}`, r.Output)
}

func TestCLI_InputFileMissingIsError(t *testing.T) {
	_, err := adapter.NewCLI().Run(context.Background(),
		spec.Candidate{Name: "s", Adapter: "cli", Cmd: "cat", InputFile: "/no/such/file"}, nil)
	assert.Error(t, err)
}

func TestRunJSON_OldShapeDecodes(t *testing.T) {
	old := `{"run_id":"r","suite":"s","suite_version":1,"timestamp":"2026-01-01T00:00:00Z",
	"scorer":{"strategy":"raw"},"candidates":[{"name":"a","metrics":{"latency_ms":5},"score":null,"rank":null,"raw_output":""}],
	"winner":null,"metadata":{"host":"h","ben_version":"0.1.0"}}`
	var r run.Run
	require.NoError(t, json.Unmarshal([]byte(old), &r))
	assert.Equal(t, 5.0, r.Candidates[0].Metrics["latency_ms"])
	assert.Nil(t, r.Candidates[0].MetricStats)
	b, _ := json.Marshal(r)
	assert.NotContains(t, string(b), "warmup")
	assert.NotContains(t, string(b), "aggregate")
}

func TestTableReporter_MedianPlusMinusStddev(t *testing.T) {
	rk := 1
	r := &run.Run{Candidates: []run.CandidateResult{{
		Name: "a", Rank: &rk,
		Metrics:     map[string]float64{"latency_ms": 2},
		MetricStats: run.ComputeStats([]map[string]float64{{"latency_ms": 1}, {"latency_ms": 2}, {"latency_ms": 3}}),
	}}}
	rep, err := reporter.New("table")
	require.NoError(t, err)
	var sb strings.Builder
	require.NoError(t, rep.Report(&sb, r))
	assert.Contains(t, sb.String(), "latency_ms=2±1")
	assert.Contains(t, sb.String(), "n=3")
}

func TestE2E_RepeatWarmupFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "ben")
	out, err := exec.Command("go", "build", "-buildvcs=false", "-o", bin, "hop.top/ben/cmd/ben").CombinedOutput()
	require.NoError(t, err, "%s", out)
	cmd := exec.Command(bin, "run", "--task", "t",
		"--candidates", "a=cli=true,b=cli=true",
		"--metric", "latency_ms", "--repeat", "6", "--warmup", "2",
		"--scorer", "single:latency_ms", "--format", "json", "--quiet")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())
	raw, err := cmd.Output()
	require.NoError(t, err, "%s", raw)
	var res run.Run
	require.NoError(t, json.Unmarshal(raw, &res))
	assert.Equal(t, 2, res.Warmup)
	for _, c := range res.Candidates {
		assert.Len(t, c.Trials, 6)
		assert.Equal(t, 6, c.MetricStats["latency_ms"].N)
	}
}
