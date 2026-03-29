// Package stories contains e2e tests keyed to individual user stories.
package stories_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// yamlCandidate mirrors the YAML shape of run.CandidateResult.
// yaml.v3 serialises run.CandidateResult using lowercased field names
// (no yaml tags on the struct), so keys are: name, metrics, score, rank, etc.
type yamlCandidate struct {
	Name    string             `yaml:"name"`
	Metrics map[string]float64 `yaml:"metrics"`
	Score   *float64           `yaml:"score"`
	Rank    *int               `yaml:"rank"`
}

// yamlRun mirrors the YAML shape of run.Run.
// Keys match lowercased Go field names (yaml.v3 default, no yaml tags on source struct).
type yamlRun struct {
	RunID      string          `yaml:"runid"`
	Candidates []yamlCandidate `yaml:"candidates"`
	Winner     *string         `yaml:"winner"`
}

// TestUS_BEN_0207_FormatYAML validates US-BEN-0207:
// ben run --format yaml emits valid YAML (not JSON) with the expected shape.
func TestUS_BEN_0207_FormatYAML(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	cmd := exec.Command(ben,
		"run",
		"--task", "echo test",
		"--candidates", "echo a,echo b",
		"--metric", "latency_ms,exit_code",
		"--scorer", "single:latency_ms",
		"--format", "yaml",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

	out, err := cmd.Output()

	// AC: exit code == 0.
	require.NoError(t, err, "ben run --format yaml exited non-zero: %s", out)

	// AC: stdout does NOT start with '{' (not JSON).
	trimmed := strings.TrimLeft(string(out), " \t\r\n")
	require.NotEmpty(t, trimmed, "stdout must not be empty")
	assert.NotEqual(t, byte('{'), trimmed[0],
		"first non-whitespace byte must not be '{' (output looks like JSON): %s", trimmed[:min(80, len(trimmed))])

	// AC: stdout parses as valid YAML.
	var result yamlRun
	require.NoError(t, yaml.Unmarshal(out, &result),
		"stdout is not valid YAML: %s", out)

	// AC: RunID is a non-empty string.
	assert.NotEmpty(t, result.RunID, "runid must be non-empty")

	// AC: candidates slice length == 2.
	require.Len(t, result.Candidates, 2, "expected 2 candidates")

	// AC: each candidate Metrics["latency_ms"] > 0 and Metrics["exit_code"] == 0.
	for _, c := range result.Candidates {
		assert.Greater(t, c.Metrics["latency_ms"], float64(0),
			"candidate %q: latency_ms must be > 0", c.Name)
		assert.Equal(t, float64(0), c.Metrics["exit_code"],
			"candidate %q: exit_code must be 0", c.Name)
	}

	// AC: Winner field is a non-empty string matching one candidate name.
	require.NotNil(t, result.Winner, "winner must not be nil")
	winner := *result.Winner
	assert.NotEmpty(t, winner, "winner must not be empty string")

	names := make([]string, len(result.Candidates))
	for i, c := range result.Candidates {
		names[i] = c.Name
	}
	assert.Contains(t, names, winner,
		"winner %q must match one of the candidate names %v", winner, names)
}

