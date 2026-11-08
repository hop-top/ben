package metrics

import "sync"

// Direction says which way a metric improves.
type Direction int

const (
	// HigherIsBetter is the default for metrics that declare nothing
	// (quality_score, accuracy, plugin metrics).
	HigherIsBetter Direction = iota
	// LowerIsBetter marks cost-like metrics (latency, CPU, memory, spend).
	LowerIsBetter
)

// Directional is an optional interface a Metric implements to declare its
// direction. Metrics that do not implement it are HigherIsBetter.
type Directional interface {
	Direction() Direction
}

var (
	dirMu   sync.RWMutex
	dirOver = map[string]Direction{}
)

// SetDirection declares the direction of a metric by name. It is how metrics
// ben does not own (binary adapter metrics, config plugins) take part in
// ranking; it overrides a registered metric's own declaration.
func SetDirection(name string, d Direction) {
	dirMu.Lock()
	defer dirMu.Unlock()
	dirOver[name] = d
}

// DirectionOf returns the direction for the named metric: an explicit
// SetDirection wins, then a registered Directional metric, else
// HigherIsBetter.
func DirectionOf(name string) Direction {
	dirMu.RLock()
	d, ok := dirOver[name]
	dirMu.RUnlock()
	if ok {
		return d
	}
	if m, ok := Get(name); ok {
		if dm, ok := m.(Directional); ok {
			return dm.Direction()
		}
	}
	return HigherIsBetter
}

// ParseDirection maps "lower"/"higher" (config and plugin protocol spelling)
// to a Direction. ok is false for anything else.
func ParseDirection(s string) (Direction, bool) {
	switch s {
	case "lower":
		return LowerIsBetter, true
	case "higher":
		return HigherIsBetter, true
	}
	return HigherIsBetter, false
}
