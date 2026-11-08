// Package bench runs one candidate repeatedly and aggregates its metrics.
package bench

import (
	"context"
	"math"

	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/metrics"
	"hop.top/ben/internal/run"
	"hop.top/ben/internal/spec"
)

// Options controls repetition. Zero value = one measured run, no warmup,
// mean (single-shot behaviour).
type Options struct {
	Repeat    int    // measured runs; <1 means 1
	Warmup    int    // discarded runs before measuring
	Aggregate string // mean (default) | median | p95
}

// Outcome is the result of measuring one candidate.
type Outcome struct {
	Output  string // output of the last run
	Metrics map[string]float64
	Trials  []map[string]float64 // one map per measured run
	Stats   map[string]run.MetricStat
	Err     error // first infrastructure error; stops further runs
}

// Measure runs a Warmup times (discarded) then Repeat times, strictly
// sequentially, collecting names from each measured result. Metrics holds
// the configured aggregate of the trials; Trials and Stats keep the spread.
func Measure(ctx context.Context, a adapter.Adapter, c spec.Candidate,
	input map[string]string, names []string, opt Options) *Outcome {
	repeat := max(opt.Repeat, 1)
	out := &Outcome{}

	for i := 0; i < opt.Warmup+repeat; i++ {
		res, err := a.Run(ctx, c, input)
		if res != nil {
			out.Output = res.Output
		}
		if res != nil && i >= opt.Warmup {
			out.Trials = append(out.Trials, collect(res, names))
		}
		if err != nil {
			out.Err = err
			break
		}
	}

	out.Metrics = run.AggregateMetrics(out.Trials, opt.Aggregate)
	out.Stats = run.ComputeStats(out.Trials)
	return out
}

// collect gathers the named metrics from one result: plugin-reported values
// first, then registered builtins. NaN (unmeasured) values are omitted.
func collect(res *adapter.Result, names []string) map[string]float64 {
	got := map[string]float64{}
	if pd, ok := res.Metadata["plugin_directions"].(map[string]string); ok {
		for k, d := range pd {
			if dir, ok := metrics.ParseDirection(d); ok {
				metrics.SetDirection(k, dir)
			}
		}
	}
	if pm, ok := res.Metadata["plugin_metrics"].(map[string]float64); ok {
		for k, v := range pm {
			got[k] = v
		}
	}
	for _, n := range names {
		if _, ok := got[n]; ok {
			continue
		}
		if m, ok := metrics.Get(n); ok {
			if v := m.Collect(res); !math.IsNaN(v) {
				got[n] = v
			}
		}
	}
	return got
}
