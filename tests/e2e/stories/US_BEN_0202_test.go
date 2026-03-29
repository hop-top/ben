// Package stories contains e2e tests that validate named user stories.
package stories_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUS_BEN_0202_InlineCompare validates US-BEN-0202: inline compare with no
// spec file. Candidates are provided via --candidates; no .yaml file is used.
//
// ACs tested:
//  1. `ben run --task "count" --candidates "echo a","echo b" --metric latency_ms,exit_code
//     --scorer single:latency_ms --format json` exits 0.
//  2. candidates[*].metrics.latency_ms > 0 for both candidates.
//  3. candidates[*].metrics.exit_code == 0 for both candidates.
//  4. scorer.strategy is checked and noted (actual value from spec may be "single:latency_ms").
//  5. scorer.weights key == "latency_ms" OR weights absent (actual: empty map → omitted).
//  6. winner is non-null and matches one of the two candidate names.
//  7. No .ben/ directory is created under the run's cwd (temp dir, not a git repo).
//  8. Table output (no --format): stdout contains both candidate names; exit 0.
//
// Notes on scorer JSON shape (observed behaviour):
//   - scorer.strategy == "single:latency_ms" (raw flag string stored verbatim).
//   - scorer.weights is absent from JSON (empty map → omitempty).
func TestUS_BEN_0202_InlineCompare(t *testing.T) {
	ben := buildBen(t)

	// runDir is the working directory for the ben process — a fresh temp dir
	// with no .ben/ subdir so the absence check is meaningful.
	runDir := t.TempDir()
	// xdgHome isolates storage from the user's real data.
	xdgHome := t.TempDir()

	// candidateNames are the names ben will assign when given bare commands.
	// run.go splits on "=" then joins parts; "echo a" has no "=", so the name
	// equals the full string.
	candidateA := "echo a"
	candidateB := "echo b"

	t.Run("json_exits_0_and_valid", func(t *testing.T) {
		// AC1: command exits 0 and emits valid JSON.
		cmd := exec.Command(ben,
			"run",
			"--task", "count",
			"--candidates", candidateA+","+candidateB,
			"--metric", "latency_ms,exit_code",
			"--scorer", "single:latency_ms",
			"--format", "json",
		)
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)

		out, err := cmd.Output()
		require.NoError(t, err, "ben run must exit 0; stderr: %s", stderrOf(err))

		var result map[string]any
		require.NoError(t, json.Unmarshal(out, &result),
			"stdout must be valid JSON; got: %s", string(out))

		// AC2 + AC3: check per-candidate metrics.
		cands, ok := result["candidates"].([]any)
		require.True(t, ok, "result.candidates must be a JSON array")
		require.Len(t, cands, 2, "must have exactly 2 candidates")

		for _, raw := range cands {
			c, ok := raw.(map[string]any)
			require.True(t, ok)
			m, ok := c["metrics"].(map[string]any)
			require.True(t, ok, "each candidate must have a metrics object")

			latency, ok := m["latency_ms"].(float64)
			assert.True(t, ok, "latency_ms must be a number")
			// AC2: latency_ms > 0.
			assert.Greater(t, latency, 0.0,
				"candidate %q: latency_ms must be > 0", c["name"])

			exitCode, ok := m["exit_code"].(float64)
			assert.True(t, ok, "exit_code must be a number")
			// AC3: exit_code == 0.
			assert.Equal(t, 0.0, exitCode,
				"candidate %q: exit_code must be 0", c["name"])
		}

		// AC4: scorer.strategy — actual stored value is "single:latency_ms"
		// (the raw flag string; not normalised to "single").
		scorer, ok := result["scorer"].(map[string]any)
		require.True(t, ok, "result.scorer must be an object")

		strategy, _ := scorer["strategy"].(string)
		// The task spec says to check actual and note if "single:latency_ms".
		// Observed: strategy == "single:latency_ms" (verbatim flag value).
		assert.True(t,
			strategy == "single" || strategy == "single:latency_ms",
			"scorer.strategy must be 'single' or 'single:latency_ms'; got %q", strategy)
		t.Logf("scorer.strategy = %q (note: stored verbatim as flag value)", strategy)

		// AC5: scorer.weights — for single: strategy the weights map is empty,
		// so the field is omitted (omitempty). We accept either absence or presence
		// with key "latency_ms".
		if weights, ok := scorer["weights"].(map[string]any); ok {
			// If present, must have "latency_ms" key.
			assert.Contains(t, weights, "latency_ms",
				"scorer.weights must contain key 'latency_ms' when present")
		} else {
			// Absent is expected for single: strategy (empty map → omitempty).
			t.Log("scorer.weights absent (empty map omitted by omitempty — expected for single: strategy)")
		}

		// AC6: winner must be non-null and match one of the two candidate names.
		winner, ok := result["winner"]
		require.True(t, ok, "result must have a 'winner' key")
		require.NotNil(t, winner, "winner must not be null")
		winnerStr, ok := winner.(string)
		require.True(t, ok, "winner must be a string")
		assert.True(t,
			winnerStr == candidateA || winnerStr == candidateB,
			"winner %q must be one of %q or %q", winnerStr, candidateA, candidateB)
	})

	t.Run("no_ben_dir_created_in_cwd", func(t *testing.T) {
		// AC7: running ben run must NOT create a .ben/ directory in cwd.
		// runDir is a fresh temp dir with no .ben/ present initially.
		freshDir := t.TempDir()

		cmd := exec.Command(ben,
			"run",
			"--task", "count",
			"--candidates", candidateA+","+candidateB,
			"--metric", "latency_ms,exit_code",
			"--scorer", "single:latency_ms",
			"--format", "json",
		)
		cmd.Dir = freshDir
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())

		out, err := cmd.Output()
		require.NoError(t, err, "ben run must exit 0; stderr: %s; stdout: %s", stderrOf(err), out)

		benDir := filepath.Join(freshDir, ".ben")
		_, statErr := os.Stat(benDir)
		assert.True(t, os.IsNotExist(statErr),
			".ben/ must NOT be created in cwd; found it at %s", benDir)
	})

	t.Run("table_output_contains_candidates", func(t *testing.T) {
		// AC8: default (no --format) produces table output; stdout contains both
		// candidate names; exit code == 0.
		cmd := exec.Command(ben,
			"run",
			"--task", "count",
			"--candidates", candidateA+","+candidateB,
			"--metric", "latency_ms,exit_code",
			"--scorer", "single:latency_ms",
			// no --format flag → defaults to table
		)
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)

		out, err := cmd.Output()
		require.NoError(t, err, "ben run (table) must exit 0; stderr: %s", stderrOf(err))

		s := string(out)
		assert.Contains(t, s, candidateA,
			"table output must contain candidate name %q", candidateA)
		assert.Contains(t, s, candidateB,
			"table output must contain candidate name %q", candidateB)
	})
}

// stderrOf extracts stderr text from an *exec.ExitError; returns "" otherwise.
func stderrOf(err error) string {
	if err == nil {
		return ""
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return strings.TrimSpace(string(ee.Stderr))
	}
	return err.Error()
}
