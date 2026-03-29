// Package metrics defines the Metric interface and a global registry.
package metrics

import (
	"maps"
	"sync"

	"hop.top/ben/internal/adapter"
)

// Metric measures one dimension of a candidate Result.
//
// Name contract:
//   - Must return a non-empty, stable string that is unique across all
//     registered metrics; it is used as the map key in CandidateResult.Metrics
//     and must be valid as a YAML/JSON object key (no spaces, no dots).
//
// Collect contract:
//   - Always returns a float64; never returns an error.
//   - If the result lacks the data needed (e.g. output is empty), return a
//     sentinel such as 0 or math.NaN() — never panic.
//   - Must be safe to call from multiple goroutines concurrently.
type Metric interface {
	Name() string
	Collect(r *adapter.Result) float64
}

var (
	mu       sync.RWMutex
	registry = map[string]Metric{}
)

// Register adds m to the built-in registry.
// Panics if a metric with the same name is already registered.
func Register(m Metric) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[m.Name()]; exists {
		panic("metrics: duplicate registration: " + m.Name())
	}
	registry[m.Name()] = m
}

// Get returns the registered metric for name, or (nil, false) if not found.
func Get(name string) (Metric, bool) {
	mu.RLock()
	defer mu.RUnlock()
	m, ok := registry[name]
	return m, ok
}

// All returns a copy of all registered metrics.
func All() map[string]Metric {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]Metric, len(registry))
	maps.Copy(out, registry)
	return out
}
