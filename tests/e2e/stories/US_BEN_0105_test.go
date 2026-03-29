package stories_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCapturingID105 executes ben run and returns the run_id from JSON output.
// (named to avoid collision with other helpers in the package)
func runCapturingID105(t *testing.T, ben, dataDir string) string {
	t.Helper()
	cmd := exec.Command(ben,
		"run",
		"--task", "echo hello",
		"--candidates", "fast=cli=echo fast,slow=cli=sleep 0.05 && echo slow",
		"--metric", "latency_ms,exit_code",
		"--scorer", "single:latency_ms",
		"--format", "json",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := cmd.Output()
	require.NoError(t, err, "ben run failed: %s", out)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result), "run output not valid JSON: %s", out)
	id, ok := result["run_id"].(string)
	require.True(t, ok, "run_id missing from run output")
	return id
}

// TestUS_BEN_0105_CompareTwoRunIDs validates the compare command for US-BEN-0105.
func TestUS_BEN_0105_CompareTwoRunIDs(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	// Setup: execute suite twice; capture run_id from each run.
	idA := runCapturingID105(t, ben, dataDir)
	idB := runCapturingID105(t, ben, dataDir)

	t.Run("json_exits_0", func(t *testing.T) {
		cmd := exec.Command(ben, "compare", idA, idB, "--format", "json")
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
		out, err := cmd.Output()
		require.NoError(t, err, "ben compare exited non-zero: %s", out)

		var diff map[string]any
		require.NoError(t, json.Unmarshal(out, &diff), "output is not valid JSON: %s", out)

		// run_id_a and run_id_b present and match.
		assert.Equal(t, idA, diff["run_id_a"], "run_id_a mismatch")
		assert.Equal(t, idB, diff["run_id_b"], "run_id_b mismatch")

		// candidates array present with at least one entry.
		cands, ok := diff["candidates"].([]any)
		require.True(t, ok, "candidates key missing or wrong type")
		require.NotEmpty(t, cands, "candidates must not be empty")

		// Each candidate has numeric metric deltas.
		for _, raw := range cands {
			c, ok := raw.(map[string]any)
			require.True(t, ok, "candidate entry not an object")
			assert.NotEmpty(t, c["candidate"], "candidate name missing")

			metrics, ok := c["metrics"].([]any)
			require.True(t, ok, "metrics key missing or wrong type")
			require.NotEmpty(t, metrics, "metrics must not be empty")

			for _, mRaw := range metrics {
				m, ok := mRaw.(map[string]any)
				require.True(t, ok, "metric entry not an object")
				assert.Contains(t, m, "metric")
				_, hasRunA := m["run_a"].(float64)
				_, hasRunB := m["run_b"].(float64)
				_, hasDelta := m["delta"].(float64)
				assert.True(t, hasRunA, "run_a must be numeric in metric %v", m["metric"])
				assert.True(t, hasRunB, "run_b must be numeric in metric %v", m["metric"])
				assert.True(t, hasDelta, "delta must be numeric in metric %v", m["metric"])
			}
		}
	})

	t.Run("nonexistent_id_exits_1_stderr_contains_run_not_found", func(t *testing.T) {
		cmd := exec.Command(ben, "compare", "nonexistent-id", idB)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

		var combinedOut []byte
		outPipe, _ := cmd.StderrPipe()
		require.NoError(t, cmd.Start())
		buf := make([]byte, 4096)
		n, _ := outPipe.Read(buf)
		combinedOut = buf[:n]
		err := cmd.Wait()

		require.Error(t, err, "expected non-zero exit")
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.ExitCode())
		// stderr must reference the missing run ID.
		assert.True(t,
			strings.Contains(string(combinedOut), "nonexistent-id") ||
				strings.Contains(string(exitErr.Stderr), "nonexistent-id"),
			"stderr should name the missing run ID; got: %q / %q",
			string(combinedOut), string(exitErr.Stderr),
		)
	})

	t.Run("same_id_exits_0_all_deltas_zero", func(t *testing.T) {
		cmd := exec.Command(ben, "compare", idA, idA, "--format", "json")
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
		out, err := cmd.Output()
		require.NoError(t, err, "ben compare same IDs exited non-zero: %s", out)

		var diff map[string]any
		require.NoError(t, json.Unmarshal(out, &diff), "output not valid JSON: %s", out)

		cands, ok := diff["candidates"].([]any)
		require.True(t, ok, "candidates key missing or wrong type")
		require.NotEmpty(t, cands)

		for _, raw := range cands {
			c := raw.(map[string]any)
			metrics, ok := c["metrics"].([]any)
			require.True(t, ok)
			for _, mRaw := range metrics {
				m := mRaw.(map[string]any)
				delta, ok := m["delta"].(float64)
				require.True(t, ok, "delta must be float64 in metric %v", m["metric"])
				assert.Equal(t, 0.0, delta,
					"delta must be 0 when comparing run to itself (metric %v)", m["metric"])
			}
		}
	})

	t.Run("table_format_exits_0_nonempty_stdout", func(t *testing.T) {
		cmd := exec.Command(ben, "compare", idA, idB, "--format", "table")
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
		out, err := cmd.Output()
		require.NoError(t, err, "ben compare --format table exited non-zero")
		assert.NotEmpty(t, strings.TrimSpace(string(out)), "table output must not be empty")
	})
}
