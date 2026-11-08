package run

import (
	"math"
	"sort"
)

// ComputeStats aggregates per-trial metric maps into per-metric summary
// statistics. Only metrics present in at least one trial appear in the
// result; a metric absent from some trials is summarised over the trials
// that carry it (N reflects that count).
func ComputeStats(trials []map[string]float64) map[string]MetricStat {
	if len(trials) == 0 {
		return nil
	}
	values := map[string][]float64{}
	for _, tr := range trials {
		for k, v := range tr {
			values[k] = append(values[k], v)
		}
	}
	out := make(map[string]MetricStat, len(values))
	for k, vs := range values {
		out[k] = statOf(vs)
	}
	return out
}

// MeanMetrics collapses per-trial metric maps into the per-metric mean,
// the shape scorers and reporters consume.
func MeanMetrics(trials []map[string]float64) map[string]float64 {
	stats := ComputeStats(trials)
	out := make(map[string]float64, len(stats))
	for k, s := range stats {
		out[k] = s.Mean
	}
	return out
}

// AggregateMetrics collapses per-trial metric maps into one value per metric
// using kind: "mean" (default, also ""), "median" or "p95".
func AggregateMetrics(trials []map[string]float64, kind string) map[string]float64 {
	stats := ComputeStats(trials)
	out := make(map[string]float64, len(stats))
	for k, s := range stats {
		out[k] = s.Pick(kind)
	}
	return out
}

// Pick returns the aggregate named by kind: "mean" (default, also ""),
// "median" or "p95".
func (s MetricStat) Pick(kind string) float64 {
	switch kind {
	case "median":
		return s.Median
	case "p95":
		return s.P95
	default:
		return s.Mean
	}
}

// percentile returns the p-quantile (0..1) of sorted s by linear
// interpolation between closest ranks.
func percentile(s []float64, p float64) float64 {
	if len(s) == 1 {
		return s[0]
	}
	pos := p * float64(len(s)-1)
	lo, hi := int(math.Floor(pos)), int(math.Ceil(pos))
	return s[lo] + (pos-float64(lo))*(s[hi]-s[lo])
}

func statOf(vs []float64) MetricStat {
	n := len(vs)
	sum := 0.0
	mn, mx := vs[0], vs[0]
	for _, v := range vs {
		sum += v
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	mean := sum / float64(n)
	sd := 0.0
	if n > 1 {
		acc := 0.0
		for _, v := range vs {
			d := v - mean
			acc += d * d
		}
		sd = math.Sqrt(acc / float64(n-1))
	}
	sorted := append([]float64(nil), vs...)
	sort.Float64s(sorted)
	return MetricStat{
		Median: percentile(sorted, 0.5), P95: percentile(sorted, 0.95),
		Mean: mean, Stddev: sd, Min: mn, Max: mx, N: n,
	}
}

// MetricNames returns the sorted union of metric names across trials.
func MetricNames(trials []map[string]float64) []string {
	seen := map[string]struct{}{}
	for _, tr := range trials {
		for k := range tr {
			seen[k] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
