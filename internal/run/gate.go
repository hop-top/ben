package run

import (
	"fmt"
	"sort"
)

// Direction states which way a metric improves.
type Direction string

const (
	// DirectionMax — higher is better (accuracy, score, throughput).
	DirectionMax Direction = "max"
	// DirectionMin — lower is better (latency_ms, cost_usd).
	DirectionMin Direction = "min"
)

// DefaultDirection returns the conventional improvement direction for a
// metric: lower-is-better for latency_ms and cost_usd, higher-is-better
// for everything else. Callers override per metric via GateSpec.Directions.
func DefaultDirection(metric string) Direction {
	switch metric {
	case "latency_ms", "cost_usd":
		return DirectionMin
	default:
		return DirectionMax
	}
}

// GateSpec configures a regression gate between a baseline run (A) and a
// candidate run (B). Thresholds maps metric name → allowed slack: movement
// in the WORSE direction beyond the threshold is a regression. Directions
// overrides DefaultDirection per metric.
type GateSpec struct {
	Thresholds map[string]float64
	Directions map[string]Direction
}

// GateCheck is one (candidate, metric) verdict.
type GateCheck struct {
	Candidate  string    `json:"candidate" yaml:"candidate"`
	Metric     string    `json:"metric" yaml:"metric"`
	Direction  Direction `json:"direction" yaml:"direction"`
	Threshold  float64   `json:"threshold" yaml:"threshold"`
	Baseline   float64   `json:"baseline" yaml:"baseline"`
	Value      float64   `json:"value" yaml:"value"`
	Delta      float64   `json:"delta" yaml:"delta"`
	Regression bool      `json:"regression" yaml:"regression"`
	Reason     string    `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// GateResult is the full gate outcome. Pass is false if ANY check
// regressed — including structural failures (missing metric or missing
// candidate), which are regressions by definition: a gate that cannot
// see a metric must not pass it.
type GateResult struct {
	Pass   bool        `json:"pass" yaml:"pass"`
	Checks []GateCheck `json:"checks" yaml:"checks"`
}

// Gate evaluates spec against baseline run a and candidate run b.
// Candidates are matched by name; every candidate in a is checked.
func Gate(a, b *Run, spec GateSpec) GateResult {
	bByName := map[string]CandidateResult{}
	for _, c := range b.Candidates {
		bByName[c.Name] = c
	}

	metricNames := make([]string, 0, len(spec.Thresholds))
	for m := range spec.Thresholds {
		metricNames = append(metricNames, m)
	}
	sort.Strings(metricNames)

	res := GateResult{Pass: true}
	for _, ca := range a.Candidates {
		cb, found := bByName[ca.Name]
		for _, m := range metricNames {
			dir := DefaultDirection(m)
			if d, ok := spec.Directions[m]; ok {
				dir = d
			}
			check := GateCheck{
				Candidate: ca.Name,
				Metric:    m,
				Direction: dir,
				Threshold: spec.Thresholds[m],
			}
			va, okA := ca.Metrics[m]
			var vb float64
			okB := false
			if found {
				vb, okB = cb.Metrics[m]
			}
			switch {
			case !found:
				check.Regression = true
				check.Reason = fmt.Sprintf("candidate %q absent from run %s", ca.Name, b.RunID)
			case !okA || !okB:
				check.Regression = true
				side := "baseline"
				if okA {
					side = "candidate"
				}
				check.Reason = fmt.Sprintf("metric %q missing on %s side", m, side)
			default:
				check.Baseline = va
				check.Value = vb
				check.Delta = vb - va
				switch dir {
				case DirectionMin: // lower is better; regression = grew past slack
					check.Regression = check.Delta > spec.Thresholds[m]
				default: // max: higher is better; regression = dropped past slack
					check.Regression = check.Delta < -spec.Thresholds[m]
				}
				if check.Regression {
					check.Reason = fmt.Sprintf("%s moved %+.4g against direction %s (threshold %.4g)",
						m, check.Delta, dir, spec.Thresholds[m])
				}
			}
			if check.Regression {
				res.Pass = false
			}
			res.Checks = append(res.Checks, check)
		}
	}
	return res
}
