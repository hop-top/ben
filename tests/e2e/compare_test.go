package e2e_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCapturingID executes a ben run and returns the run_id from JSON output.
func runCapturingID(t *testing.T, ben, dataDir string, extraArgs ...string) string {
	t.Helper()
	args := []string{
		"run",
		"--task", "echo hello",
		"--candidates", "fast=cli=echo fast,slow=cli=sleep 0.05 && echo slow",
		"--metric", "latency_ms,exit_code",
		"--scorer", "single:latency_ms",
		"--format", "json",
	}
	args = append(args, extraArgs...)

	cmd := exec.Command(ben, args...)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := cmd.Output()
	require.NoError(t, err, "ben run failed: %s", out)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	id, ok := result["run_id"].(string)
	require.True(t, ok, "run_id missing from output")
	return id
}

func TestCompareCommand_JSON(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	idA := runCapturingID(t, ben, dataDir)
	idB := runCapturingID(t, ben, dataDir)

	cmd := exec.Command(ben, "compare", idA, idB, "--format", "json")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := cmd.Output()
	require.NoError(t, err, "ben compare failed: %s", out)

	var diff map[string]any
	require.NoError(t, json.Unmarshal(out, &diff), "output is not valid JSON: %s", out)

	// Both run IDs present.
	assert.Equal(t, idA, diff["run_id_a"])
	assert.Equal(t, idB, diff["run_id_b"])

	// Candidates array present and metrics compared.
	cands, ok := diff["candidates"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, cands)

	first := cands[0].(map[string]any)
	metrics, ok := first["metrics"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, metrics)
	m0 := metrics[0].(map[string]any)
	assert.Contains(t, m0, "metric")
	assert.Contains(t, m0, "run_a")
	assert.Contains(t, m0, "run_b")
	assert.Contains(t, m0, "delta")
}

func TestCompareCommand_Table(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	idA := runCapturingID(t, ben, dataDir)
	idB := runCapturingID(t, ben, dataDir)

	cmd := exec.Command(ben, "compare", idA, idB, "--format", "table")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := cmd.Output()
	require.NoError(t, err)

	s := string(out)
	assert.True(t, strings.Contains(s, idA) || strings.Contains(s, "Run A:"))
	assert.Contains(t, s, "fast")
}

func TestCompareCommand_MissingRunID_ExitsOne(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	// Save one real run, compare with a bogus ID.
	idA := runCapturingID(t, ben, dataDir)

	cmd := exec.Command(ben, "compare", idA, "nonexistent-id", "--format", "json")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	err := cmd.Run()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
}
