package stories_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUS_BEN_0201_InstallFirstBenchmark validates the first-benchmark happy path
// and the no-candidates error path (story US-BEN-0201).
func TestUS_BEN_0201_InstallFirstBenchmark(t *testing.T) {
	ben := buildBen(t)

	t.Run("table_mode_exits_zero_contains_candidates", func(t *testing.T) {
		dataDir := t.TempDir()

		// Use name=adapter=cmd form; two candidates: "echo a" and "echo b".
		// --quiet silences kit's per-phase progress events (§6.5) so we
		// can keep the strict "stderr is empty" acceptance criterion.
		cmd := exec.Command(ben,
			"run",
			"--quiet",
			"--task", "echo hello",
			"--candidates", "echo a=cli=echo a,echo b=cli=echo b",
			"--metric", "latency_ms",
			"--scorer", "single:latency_ms",
		)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		// AC: exits 0
		require.NoError(t, err, "ben run exited non-zero; stderr: %s", stderr.String())

		// AC: stderr is empty on happy path (with --quiet to silence progress events)
		assert.Empty(t, stderr.String(), "unexpected stderr output")

		out := stdout.String()

		// AC: stdout (table mode) contains both candidate names
		assert.Contains(t, out, "echo a", "stdout missing candidate 'echo a'")
		assert.Contains(t, out, "echo b", "stdout missing candidate 'echo b'")

		// AC: each row has a non-zero latency_ms value visible in output.
		// Table renders metrics as "latency_ms=<N>" in a Metrics cell.
		// Verify at least one row has a non-zero value.
		foundNonZero := false
		for _, line := range strings.Split(out, "\n") {
			// Look for "latency_ms=<N>" where N > 0.
			idx := strings.Index(line, "latency_ms=")
			if idx < 0 {
				continue
			}
			rest := line[idx+len("latency_ms="):]
			// Trim any trailing whitespace or table chars.
			rest = strings.FieldsFunc(rest, func(r rune) bool {
				return r == ' ' || r == '\t' || r == '|' || r == ','
			})[0]
			var v float64
			if _, scanErr := parseFloat(rest, &v); scanErr == nil && v > 0 {
				foundNonZero = true
				break
			}
		}
		assert.True(t, foundNonZero, "expected at least one non-zero latency_ms value in table output:\n%s", out)
	})

	t.Run("json_format_winner_non_empty", func(t *testing.T) {
		dataDir := t.TempDir()

		// --quiet silences kit's per-phase progress events on stderr
		// (§6.5) so the strict "empty stderr" assertion still holds.
		cmd := exec.Command(ben,
			"run",
			"--quiet",
			"--task", "echo hello",
			"--candidates", "echo a=cli=echo a,echo b=cli=echo b",
			"--metric", "latency_ms",
			"--scorer", "single:latency_ms",
			"--format", "json",
		)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		out, err := cmd.Output()
		require.NoError(t, err, "ben run --format json exited non-zero; stderr: %s", stderr.String())

		// AC: stderr is empty on happy path (with --quiet to silence progress events)
		assert.Empty(t, stderr.String(), "unexpected stderr output")

		var result map[string]any
		require.NoError(t, json.Unmarshal(out, &result), "output is not valid JSON: %s", out)

		// AC: winner field is non-empty string
		winner, ok := result["winner"]
		require.True(t, ok, "JSON result missing 'winner' field")
		winnerStr, ok := winner.(string)
		require.True(t, ok, "'winner' field is not a string: %v", winner)
		assert.NotEmpty(t, winnerStr, "'winner' field is empty")
	})

	t.Run("no_candidates_exits_one_stderr_contains_candidate_or_required", func(t *testing.T) {
		cmd := exec.Command(ben,
			"run",
			"--task", "echo hello",
			"--metric", "latency_ms",
		)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()

		// AC: exits 1
		require.Error(t, err, "expected non-zero exit when no --candidates provided")
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.ExitCode())

		// AC: stderr contains "candidate" or "required"
		stderrStr := strings.ToLower(stderr.String())
		hasCandidateOrRequired := strings.Contains(stderrStr, "candidate") || strings.Contains(stderrStr, "required")
		assert.True(t, hasCandidateOrRequired,
			"expected stderr to contain 'candidate' or 'required', got: %q", stderr.String())
	})
}

// parseFloat attempts to parse s as a float64, writing into v.
// Returns (n, err) like fmt.Sscanf.
func parseFloat(s string, v *float64) (int, error) {
	return fmt.Sscanf(s, "%f", v)
}
