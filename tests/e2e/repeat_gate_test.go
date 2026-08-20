package e2e_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunRepeat_StatsAndTrials — --repeat N produces N trials and
// metric_stats with n == N; metrics stay the per-metric mean.
func TestRunRepeat_StatsAndTrials(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	cmd := exec.Command(ben,
		"run",
		"--task", "repeat smoke",
		"--candidates", "fast=cli=echo hi",
		"--metric", "latency_ms,exit_code",
		"--scorer", "single:latency_ms",
		"--repeat", "3",
		"--format", "json",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := cmd.Output()
	require.NoError(t, err, "ben run --repeat failed: %s", out)

	var result struct {
		Candidates []struct {
			Name        string                    `json:"name"`
			Metrics     map[string]float64        `json:"metrics"`
			Trials      []map[string]float64      `json:"trials"`
			MetricStats map[string]map[string]any `json:"metric_stats"`
		} `json:"candidates"`
	}
	require.NoError(t, json.Unmarshal(out, &result))
	require.Len(t, result.Candidates, 1)
	c := result.Candidates[0]
	assert.Len(t, c.Trials, 3, "three trials persisted")
	require.Contains(t, c.MetricStats, "latency_ms")
	assert.EqualValues(t, 3, c.MetricStats["latency_ms"]["n"])
	// metrics carries the mean of the trials.
	sum := 0.0
	for _, tr := range c.Trials {
		sum += tr["latency_ms"]
	}
	assert.InDelta(t, sum/3.0, c.Metrics["latency_ms"], 1e-9)
}

// TestCompare_FailOnRegression — run-b regresses exit_code (0 → 1);
// the gate exits 4 with the BEN_REGRESSION code in the error envelope.
func TestCompare_FailOnRegression(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	runOnce := func(candidateCmd string) string {
		cmd := exec.Command(ben,
			"run",
			"--task", "gate smoke",
			"--candidates", "c=cli="+candidateCmd,
			"--metric", "exit_code,latency_ms",
			"--scorer", "raw",
			"--format", "json",
		)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
		out, err := cmd.Output()
		require.NoError(t, err, "ben run failed: %s", out)
		var r map[string]any
		require.NoError(t, json.Unmarshal(out, &r))
		return r["run_id"].(string)
	}

	idA := runOnce("true")  // exit_code 0
	idB := runOnce("false") // exit_code 1 — regression under direction min

	cmd := exec.Command(ben,
		"compare", idA, idB,
		"--fail-on-regression",
		"--threshold", "exit_code=0",
		"--direction", "exit_code=min",
		"--format", "json",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "gate must fail; stdout=%s stderr=%s", stdout.String(), stderr.String())
	// Repo convention (matches BEN_NO_RUN): the process exits 1 for any
	// error; the FAMILY exit code travels in the structured envelope
	// (`"exit_code": 4`), which is what scripts branch on.
	assert.Equal(t, 1, exitErr.ExitCode())
	combined := stdout.String() + stderr.String()
	assert.Contains(t, combined, "BEN_REGRESSION")
	assert.Contains(t, combined, `"exit_code": 4`)
	assert.Contains(t, combined, `"pass": false`)
}

// TestCompare_GatePassesWithinThreshold — same run compared to itself
// passes and exits 0, with the gate section present.
func TestCompare_GatePassesWithinThreshold(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	mk := exec.Command(ben,
		"run",
		"--task", "gate smoke",
		"--candidates", "c=cli=true",
		"--metric", "exit_code,latency_ms",
		"--scorer", "raw",
		"--format", "json",
	)
	mk.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := mk.Output()
	require.NoError(t, err)
	var r map[string]any
	require.NoError(t, json.Unmarshal(out, &r))
	id := r["run_id"].(string)

	cmd := exec.Command(ben,
		"compare", id, id,
		"--fail-on-regression",
		"--threshold", "exit_code=0",
		"--format", "json",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err = cmd.Output()
	require.NoError(t, err, "self-compare must pass the gate: %s", out)

	var diff map[string]any
	require.NoError(t, json.Unmarshal(out, &diff))
	gate, ok := diff["gate"].(map[string]any)
	require.True(t, ok, "gate section present when gating requested")
	assert.Equal(t, true, gate["pass"])
}

// TestCompare_ThresholdWithoutGateFlagIsUsageError.
func TestCompare_ThresholdWithoutGateFlagIsUsageError(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	cmd := exec.Command(ben,
		"compare", "a", "b",
		"--threshold", "exit_code=0",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Contains(t, stderr.String()+err.Error(), "fail-on-regression")
}
