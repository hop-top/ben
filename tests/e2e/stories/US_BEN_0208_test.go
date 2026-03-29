// Package stories contains e2e tests keyed to individual user stories.
package stories_test

// US-BEN-0208 — Observe exit_code and output_size metrics captured for CLI run.
//
// Story: docs/stories/US-BEN-0208-exit-code-output-size-metrics.md
//
// Known bugs surfaced by this test (none affecting assertions here; documented for reference):
//   - US-BEN-0111: winner emits "" instead of null in some scorer paths (not triggered here,
//     run.go correctly sets winner=nil for raw mode).
//   - US-BEN-0110: score/rank emit 0 instead of null for raw mode (Score/Rank pointers are
//     only set when strategy != "raw", so raw mode correctly serialises them as null).

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUS_BEN_0208_ExitCodeOutputSize(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	// Run: two candidates via bare command syntax (no name=adapter=cmd prefix).
	// "echo hello world" → exit 0, output "hello world\n" (12 bytes)
	// "false"            → exit 1, output "" (0 bytes)
	//
	// StringSliceVar splits on comma, so passing both as a single --candidates value
	// with comma separator is equivalent to two separate --candidates flags.
	cmd := exec.Command(ben,
		"run",
		"--task", "produce output",
		"--candidates", "echo hello world,false",
		"--metric", "exit_code,output_size",
		"--scorer", "raw",
		"--format", "json",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

	out, err := cmd.Output()

	// AC: command exits 0.
	require.NoError(t, err, "ben run exited non-zero: %s", out)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result), "output is not valid JSON: %s", out)

	// AC: scorer.strategy == "raw".
	scorerRaw, ok := result["scorer"]
	require.True(t, ok, "result.scorer field missing")
	scorerMap, ok := scorerRaw.(map[string]any)
	require.True(t, ok, "result.scorer is not an object")
	assert.Equal(t, "raw", scorerMap["strategy"], "scorer.strategy must be \"raw\"")

	// AC: winner == null (JSON null → Go nil).
	assert.Nil(t, result["winner"], "winner must be JSON null for raw scorer")

	// AC: candidates present.
	cands, ok := result["candidates"].([]any)
	require.True(t, ok, "result.candidates is not an array")
	require.Len(t, cands, 2, "expected 2 candidates")

	// Index candidates by name for targeted assertions.
	byName := make(map[string]map[string]any, 2)
	for _, raw := range cands {
		c, ok := raw.(map[string]any)
		require.True(t, ok, "candidate entry is not an object")
		name, _ := c["name"].(string)
		require.NotEmpty(t, name, "candidate has no name")
		byName[name] = c
	}

	// --- "echo hello world" assertions ---
	echo, found := byName["echo hello world"]
	require.True(t, found, "candidate \"echo hello world\" not found in results; got names: %v", candidateNames(cands))

	echoMetrics, ok := echo["metrics"].(map[string]any)
	require.True(t, ok, "\"echo hello world\": metrics is not an object")

	// AC: exit_code == 0.
	echoExitCode, hasEC := echoMetrics["exit_code"]
	require.True(t, hasEC, "\"echo hello world\": metrics.exit_code missing")
	assert.Equal(t, float64(0), echoExitCode, "\"echo hello world\": metrics.exit_code must be 0")

	// AC: output_size == 12 (len("hello world\n")).
	echoOutputSize, hasOS := echoMetrics["output_size"]
	require.True(t, hasOS, "\"echo hello world\": metrics.output_size missing")
	assert.Equal(t, float64(12), echoOutputSize,
		"\"echo hello world\": metrics.output_size must be 12 (len(\"hello world\\n\"))")

	// score and rank must be null for raw scorer.
	assert.Nil(t, echo["score"], "\"echo hello world\": score must be null for raw scorer")
	assert.Nil(t, echo["rank"], "\"echo hello world\": rank must be null for raw scorer")

	// --- "false" assertions ---
	falseCand, found := byName["false"]
	require.True(t, found, "candidate \"false\" not found in results; got names: %v", candidateNames(cands))

	falseMetrics, ok := falseCand["metrics"].(map[string]any)
	require.True(t, ok, "\"false\": metrics is not an object")

	// AC: exit_code == 1.
	falseExitCode, hasEC2 := falseMetrics["exit_code"]
	require.True(t, hasEC2, "\"false\": metrics.exit_code missing")
	assert.Equal(t, float64(1), falseExitCode, "\"false\": metrics.exit_code must be 1")

	// AC: output_size == 0.
	falseOutputSize, hasOS2 := falseMetrics["output_size"]
	require.True(t, hasOS2, "\"false\": metrics.output_size missing")
	assert.Equal(t, float64(0), falseOutputSize, "\"false\": metrics.output_size must be 0")

	// score and rank must be null for raw scorer.
	assert.Nil(t, falseCand["score"], "\"false\": score must be null for raw scorer")
	assert.Nil(t, falseCand["rank"], "\"false\": rank must be null for raw scorer")
}

// candidateNames returns a slice of candidate name strings for error messages.
func candidateNames(cands []any) []string {
	names := make([]string, 0, len(cands))
	for _, raw := range cands {
		if c, ok := raw.(map[string]any); ok {
			if name, ok := c["name"].(string); ok {
				names = append(names, name)
			}
		}
	}
	return names
}
