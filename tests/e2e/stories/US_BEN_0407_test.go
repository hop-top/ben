// Package stories contains e2e tests that validate named user stories.
package stories_test

// US-BEN-0407 — output_lines built-in metric.
//
// Story: the built-in metric "output_lines" counts newline-delimited lines in
// a candidate's output so users can benchmark tools by line-count productivity.
//
// Acceptance criteria:
//  1. `ben run --task "print lines" --candidates "printf 'line1\nline2\nline3\n'"
//     --metric output_lines --scorer raw --format json` exits 0.
//  2. result JSON: candidates[0].metrics["output_lines"] == 3.
//  3. result JSON: candidates[0].error is absent or empty.

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUS_BEN_0407_NewBuiltinMetricInResultJSON(t *testing.T) {
	ben := buildBen(t)

	cmd := exec.Command(ben,
		"run",
		"--task", "print lines",
		"--candidates", "printf 'line1\\nline2\\nline3\\n'",
		"--metric", "output_lines",
		"--scorer", "raw",
		"--format", "json",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())

	out, err := cmd.Output()
	// AC1: exits 0.
	require.NoError(t, err, "ben run must exit 0; stderr=%s", stderrOf(err))

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result), "output must be valid JSON: %s", out)

	cands, ok := result["candidates"].([]any)
	require.True(t, ok, "result.candidates must be an array")
	require.NotEmpty(t, cands, "candidates array must be non-empty")

	cand, ok := cands[0].(map[string]any)
	require.True(t, ok, "candidates[0] must be an object")

	// AC2: output_lines == 3.
	mRaw, hasMets := cand["metrics"]
	require.True(t, hasMets, "candidates[0].metrics must be present")
	mets, ok := mRaw.(map[string]any)
	require.True(t, ok, "candidates[0].metrics must be an object")

	lines, hasLines := mets["output_lines"]
	require.True(t, hasLines, "candidates[0].metrics must contain output_lines; got keys: %v", metricKeys0407(mets))
	assert.InDelta(t, 3.0, lines, 1e-9, "output_lines must equal 3")

	// AC3: error absent or empty.
	if errVal, exists := cand["error"]; exists {
		assert.Empty(t, errVal, "candidates[0].error must be absent or empty")
	}
}

// metricKeys0407 returns the keys of a metrics map for diagnostic messages.
func metricKeys0407(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
