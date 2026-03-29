// Package plugin loads config-declared metric plugins from ben.yaml.
package plugin

import (
	"fmt"

	"hop.top/ben/internal/metrics"
)

// MetricPluginConfig maps one entry under plugins.metrics in ben.yaml.
type MetricPluginConfig struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`   // "llm_judge" | "import"
	Model  string `yaml:"model"`
	Prompt string `yaml:"prompt"`
	Import string `yaml:"import"` // binary name on PATH (Phase 3)
}

// LoadMetricPlugins registers config-declared metrics into the global registry.
// Currently supports:
//   - type "llm_judge": creates an LLMJudgeMetric and registers it.
//   - type "import": binary plugin on PATH — Phase 3; skipped for now.
func LoadMetricPlugins(cfgs []MetricPluginConfig) error {
	for _, cfg := range cfgs {
		switch cfg.Type {
		case "llm_judge":
			if cfg.Name == "" {
				return fmt.Errorf("plugin: llm_judge: missing name")
			}
			if cfg.Model == "" {
				return fmt.Errorf("plugin %q: llm_judge: missing model", cfg.Name)
			}
			if cfg.Prompt == "" {
				return fmt.Errorf("plugin %q: llm_judge: missing prompt", cfg.Name)
			}
			m := metrics.NewLLMJudge(cfg.Name, cfg.Model, cfg.Prompt)
			metrics.Register(m)
		case "import":
			// Phase 3: binary plugin discovery — not yet implemented.
		default:
			return fmt.Errorf("plugin %q: unknown type %q", cfg.Name, cfg.Type)
		}
	}
	return nil
}
