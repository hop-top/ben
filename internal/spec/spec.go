// Package spec loads and validates ben suite spec files.
package spec

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec is a ben suite specification.
type Spec struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Version     int               `yaml:"version"`
	Task        Task              `yaml:"task"`
	Candidates  []Candidate       `yaml:"candidates"`
	Metrics     []string          `yaml:"metrics"`
	Scorer      Scorer            `yaml:"scorer"`
	Registry    RegistryConfig    `yaml:"registry"`
}

// Task describes the benchmark prompt and its inputs.
type Task struct {
	Prompt string            `yaml:"prompt"`
	Input  map[string]string `yaml:"input"`
}

// Candidate is one approach being benchmarked.
type Candidate struct {
	Name    string `yaml:"name"`
	Adapter string `yaml:"adapter"`
	Cmd     string `yaml:"cmd"`
	Model   string `yaml:"model"`
}

// Scorer configures how candidates are ranked.
type Scorer struct {
	Strategy string             `yaml:"strategy"`
	Weights  map[string]float64 `yaml:"weights"`
}

// RegistryConfig controls optional push/pull to shared registry.
type RegistryConfig struct {
	Push bool `yaml:"push"`
}

// Load reads a YAML spec file from path and validates required fields.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("spec: read %s: %w", path, err)
	}
	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("spec: parse %s: %w", path, err)
	}
	if err := validate(&s); err != nil {
		return nil, fmt.Errorf("spec: %w", err)
	}
	return &s, nil
}

func validate(s *Spec) error {
	if s.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if len(s.Candidates) == 0 {
		return fmt.Errorf("missing required field: candidates (must have at least one)")
	}
	for i, c := range s.Candidates {
		if c.Name == "" {
			return fmt.Errorf("candidate[%d]: missing name", i)
		}
		if c.Adapter == "" {
			return fmt.Errorf("candidate %q: missing adapter", c.Name)
		}
	}
	return nil
}

// Template expands {{input.KEY}} placeholders in tmpl using the provided input map.
func Template(tmpl string, input map[string]string) string {
	for k, v := range input {
		tmpl = strings.ReplaceAll(tmpl, "{{input."+k+"}}", v)
	}
	return tmpl
}

// FromFlags holds values supplied via inline CLI flags (--task, --candidates, etc.).
type FromFlags struct {
	Task       string
	Candidates []string // name=adapter:cmd or just name for path-based lookup
	Metrics    []string
	Scorer     string
	Input      map[string]string
}

// ToSpec converts inline flag values into a Spec. Candidates format:
// "name,adapter,cmd" or simple "name" (adapter defaults to "cli", cmd = name).
func (f *FromFlags) ToSpec() (*Spec, error) {
	if f.Task == "" && len(f.Candidates) == 0 {
		return nil, fmt.Errorf("fromflags: --task or --candidates required")
	}

	candidates := make([]Candidate, 0, len(f.Candidates))
	for _, raw := range f.Candidates {
		parts := strings.SplitN(raw, ",", 3)
		c := Candidate{
			Name:    parts[0],
			Adapter: "cli",
			Cmd:     parts[0],
		}
		if len(parts) >= 2 {
			c.Adapter = parts[1]
		}
		if len(parts) >= 3 {
			c.Cmd = parts[2]
		}
		candidates = append(candidates, c)
	}

	scorerStrategy := f.Scorer
	scorerWeights := map[string]float64{}
	if pairs, ok := strings.CutPrefix(f.Scorer, "weighted:"); ok {
		scorerStrategy = "weighted"
		for pair := range strings.SplitSeq(pairs, ",") {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) != 2 {
				continue
			}
			var w float64
			if _, err := fmt.Sscanf(kv[1], "%f", &w); err == nil {
				scorerWeights[kv[0]] = w
			}
		}
	}

	input := f.Input
	if input == nil {
		input = map[string]string{}
	}

	s := &Spec{
		Name: "inline",
		Task: Task{
			Prompt: f.Task,
			Input:  input,
		},
		Candidates: candidates,
		Metrics:    f.Metrics,
		Scorer: Scorer{
			Strategy: scorerStrategy,
			Weights:  scorerWeights,
		},
		Version: 1,
	}

	if len(s.Candidates) == 0 {
		return nil, fmt.Errorf("fromflags: no candidates specified")
	}

	return s, nil
}
