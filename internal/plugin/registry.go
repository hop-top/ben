package plugin

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/reporter"
)

// PluginRegistry holds discovered binary adapters and reporters.
type PluginRegistry struct {
	adapters  map[string]adapter.Adapter
	reporters map[string]reporter.Reporter
}

// NewRegistry returns an empty PluginRegistry.
func NewRegistry() *PluginRegistry {
	return &PluginRegistry{
		adapters:  make(map[string]adapter.Adapter),
		reporters: make(map[string]reporter.Reporter),
	}
}

// RegisterAdapter adds an adapter under name.
func (r *PluginRegistry) RegisterAdapter(name string, a adapter.Adapter) {
	r.adapters[name] = a
}

// RegisterReporter adds a reporter under name.
func (r *PluginRegistry) RegisterReporter(name string, rep reporter.Reporter) {
	r.reporters[name] = rep
}

// LookupAdapter returns the adapter for name and whether it was found.
func (r *PluginRegistry) LookupAdapter(name string) (adapter.Adapter, bool) {
	a, ok := r.adapters[name]
	return a, ok
}

// LookupReporter returns the reporter for name and whether it was found.
func (r *PluginRegistry) LookupReporter(name string) (reporter.Reporter, bool) {
	rep, ok := r.reporters[name]
	return rep, ok
}

// DiscoverAll scans PATH for ben-adapter-* and ben-reporter-* binaries,
// constructs the appropriate wrappers, and returns a populated registry.
func DiscoverAll() *PluginRegistry {
	reg := NewRegistry()

	for _, name := range DiscoverAdapters() {
		binPath, err := lookupBinary("ben-adapter-" + name)
		if err != nil {
			continue
		}
		reg.RegisterAdapter(name, adapter.NewBinaryAdapter(binPath))
	}

	for _, name := range DiscoverReporters() {
		binPath, err := lookupBinary("ben-reporter-" + name)
		if err != nil {
			continue
		}
		reg.RegisterReporter(name, reporter.NewBinaryReporter(binPath))
	}

	return reg
}

// lookupBinary resolves a binary name to its full path via exec.LookPath,
// then returns the absolute path.
func lookupBinary(name string) (string, error) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("plugin: %q not found on PATH: %w", name, err)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p, nil
	}
	return abs, nil
}
