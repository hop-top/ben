package metrics

import (
	"math"

	"hop.top/ben/internal/adapter"
)

// Resource-usage metrics read the child rusage recorded on adapter.Result.
// All are lower-is-better. When the adapter or platform could not measure a
// value, Collect returns NaN; callers must omit NaN metrics (never store 0).

func init() {
	Register(cpuUserMetric{})
	Register(cpuSysMetric{})
	Register(cpuMetric{})
	Register(maxRSSMetric{})
}

type cpuUserMetric struct{}

func (cpuUserMetric) Direction() Direction { return LowerIsBetter }
func (cpuUserMetric) Name() string         { return "cpu_user_ms" }
func (cpuUserMetric) Collect(r *adapter.Result) float64 {
	if r.CPUUserMs == nil {
		return math.NaN()
	}
	return *r.CPUUserMs
}

type cpuSysMetric struct{}

func (cpuSysMetric) Direction() Direction { return LowerIsBetter }
func (cpuSysMetric) Name() string         { return "cpu_sys_ms" }
func (cpuSysMetric) Collect(r *adapter.Result) float64 {
	if r.CPUSysMs == nil {
		return math.NaN()
	}
	return *r.CPUSysMs
}

// cpuMetric is user + system CPU time.
type cpuMetric struct{}

func (cpuMetric) Direction() Direction { return LowerIsBetter }
func (cpuMetric) Name() string         { return "cpu_ms" }
func (cpuMetric) Collect(r *adapter.Result) float64 {
	if r.CPUUserMs == nil && r.CPUSysMs == nil {
		return math.NaN()
	}
	var sum float64
	if r.CPUUserMs != nil {
		sum += *r.CPUUserMs
	}
	if r.CPUSysMs != nil {
		sum += *r.CPUSysMs
	}
	return sum
}

// maxRSSMetric is peak resident set size in KiB.
type maxRSSMetric struct{}

func (maxRSSMetric) Direction() Direction { return LowerIsBetter }
func (maxRSSMetric) Name() string         { return "max_rss_kb" }
func (maxRSSMetric) Collect(r *adapter.Result) float64 {
	if r.MaxRSSKB == nil {
		return math.NaN()
	}
	return float64(*r.MaxRSSKB)
}
