// Package stories contains e2e tests that validate named user stories.
package stories_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUS_BEN_0203_TableOutput validates that `ben run` produces human-readable
// table output (US-BEN-0203) with no-flag and explicit --format table invocations.
//
// ACs tested:
//  1. stdout does NOT start with '{' or '---' (not JSON/YAML).
//  2. stdout contains "Name" (table header column).
//  3. stdout contains "latency_ms" (metric value in Metrics column).
//  4. stdout contains "Rank" (rank column header).
//  5. exit code == 0.
//  6. --format table produces equivalent table-shaped output as no-flag invocation.
func TestUS_BEN_0203_TableOutput(t *testing.T) {
	ben := buildBen(t)

	run := func(t *testing.T, extraArgs ...string) []byte {
		t.Helper()
		args := []string{
			"run",
			"--task", "echo hello",
			"--candidates", "alpha=cli=echo alpha,beta=cli=echo beta",
			"--metric", "latency_ms",
			"--scorer", "single:latency_ms",
		}
		args = append(args, extraArgs...)
		cmd := exec.Command(ben, args...)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())
		out, err := cmd.Output()
		require.NoError(t, err, "ben run exited non-zero; stderr: %s", func() string {
			if ee, ok := err.(*exec.ExitError); ok {
				return string(ee.Stderr)
			}
			return ""
		}())
		return out
	}

	// AC5 + AC1–4: no --format flag → defaults to table.
	t.Run("no_flag_defaults_to_table", func(t *testing.T) {
		out := run(t)
		s := string(out)

		// AC5: exit 0 (enforced by require.NoError above).

		// AC1: output does not start with JSON '{' or YAML '---'.
		trimmed := strings.TrimSpace(s)
		assert.False(t, strings.HasPrefix(trimmed, "{"), "stdout must not start with '{' (got JSON?): %q", trimmed)
		assert.False(t, strings.HasPrefix(trimmed, "---"), "stdout must not start with '---' (got YAML?): %q", trimmed)

		// AC2: header row contains "Name".
		assert.Contains(t, s, "Name", "expected 'Name' column header in table output")

		// AC3: metric value present in Metrics column (format: key=value).
		assert.Contains(t, s, "latency_ms", "expected 'latency_ms' in Metrics column")

		// AC4: rank column header present.
		assert.Contains(t, s, "Rank", "expected 'Rank' column header in table output")
	})

	// AC6: --format table produces the same table-shaped output.
	t.Run("explicit_format_table", func(t *testing.T) {
		out := run(t, "--format", "table")
		s := string(out)

		trimmed := strings.TrimSpace(s)
		assert.False(t, strings.HasPrefix(trimmed, "{"), "stdout must not start with '{': %q", trimmed)
		assert.False(t, strings.HasPrefix(trimmed, "---"), "stdout must not start with '---': %q", trimmed)

		assert.Contains(t, s, "Name", "expected 'Name' column header with --format table")
		assert.Contains(t, s, "latency_ms", "expected 'latency_ms' in Metrics column with --format table")
		assert.Contains(t, s, "Rank", "expected 'Rank' column header with --format table")
	})
}
