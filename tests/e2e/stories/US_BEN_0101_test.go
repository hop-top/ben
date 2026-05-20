// Package stories contains e2e tests that validate named user stories.
package stories_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUS_BEN_0101_CompareTwoCLITools validates US-BEN-0101:
// run ben with latency_ms + quality_score metrics; both candidates have both
// metrics; winner is non-null; ranks are assigned.
//
// Requires BEN_LLM_TEST=1 and a valid ANTHROPIC_API_KEY to actually call the
// LLM judge. Without that env var the test is skipped.
func TestUS_BEN_0101_CompareTwoCLITools(t *testing.T) {
	if os.Getenv("BEN_LLM_TEST") != "1" {
		t.Skip("set BEN_LLM_TEST=1 to run LLM tests")
	}

	ben := buildBen(t)
	dataDir := t.TempDir()
	cfgPath := writeQualityScoreConfig(t)

	// Suite YAML: two CLI candidates (echo-based so they always succeed), two
	// metrics including quality_score which requires the llm_judge plugin declared
	// in the config file.
	const suiteYAML = `
name: compare-cli-tools
version: 1
task:
  prompt: "Find all HTTP handler functions"
candidates:
  - name: xray
    adapter: cli
    cmd: echo "xray result: found handler functions in routes.go"
  - name: grep
    adapter: cli
    cmd: echo "grep result: routes.go:42:func handleHTTP"
metrics:
  - latency_ms
  - quality_score
scorer:
  strategy: weighted
  weights:
    latency_ms: 0.3
    quality_score: 0.7
`
	suitePath := filepath.Join(t.TempDir(), "compare-cli-tools.yaml")
	require.NoError(t, os.WriteFile(suitePath, []byte(suiteYAML), 0o644))

	env := append(os.Environ(), "XDG_DATA_HOME="+dataDir)

	// ── AC1 + AC8: exits 0; stdout is valid JSON ──────────────────────────────

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(ben,
		"--config", cfgPath,
		"--format", "json",
		"--quiet",
		"run",
		"--suite", suitePath,
	)
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// AC1: exit 0
	require.NoError(t, cmd.Run(),
		"ben run exited non-zero; stderr: %s", stderr.String())

	// AC8: stdout is valid JSON
	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result),
		"stdout is not valid JSON: %s", stdout.String())

	// ── AC2: candidates array length == 2 ─────────────────────────────────────

	rawCandidates, ok := result["candidates"]
	require.True(t, ok, "JSON missing 'candidates' key")
	candidatesSlice, ok := rawCandidates.([]any)
	require.True(t, ok, "'candidates' must be an array, got %T", rawCandidates)
	require.Len(t, candidatesSlice, 2, "expected 2 candidates, got %d", len(candidatesSlice))

	// ── AC2 + AC5 + AC6: each candidate has latency_ms > 0 and quality_score in [0,1] ──

	for i, raw := range candidatesSlice {
		c, ok := raw.(map[string]any)
		require.True(t, ok, "candidate[%d] is not an object", i)

		metricsRaw, ok := c["metrics"]
		require.True(t, ok, "candidate[%d] missing 'metrics' key", i)
		metricsMap, ok := metricsRaw.(map[string]any)
		require.True(t, ok, "candidate[%d].metrics is not an object, got %T", i, metricsRaw)

		// AC5: latency_ms is a non-negative integer (represented as float64
		// in JSON). Sub-millisecond runs truncate to 0 on fast runners.
		latRaw, ok := metricsMap["latency_ms"]
		require.True(t, ok, "candidate[%d] missing metric latency_ms", i)
		lat, ok := latRaw.(float64)
		require.True(t, ok, "candidate[%d].metrics.latency_ms is not a number", i)
		assert.GreaterOrEqual(t, lat, float64(0),
			"candidate[%d].metrics.latency_ms must be non-negative", i)

		// AC6: quality_score is a float in [0, 1]
		qsRaw, ok := metricsMap["quality_score"]
		require.True(t, ok, "candidate[%d] missing metric quality_score", i)
		qs, ok := qsRaw.(float64)
		require.True(t, ok, "candidate[%d].metrics.quality_score is not a number", i)
		assert.GreaterOrEqual(t, qs, float64(0),
			"candidate[%d].metrics.quality_score must be >= 0.0", i)
		assert.LessOrEqual(t, qs, float64(1),
			"candidate[%d].metrics.quality_score must be <= 1.0", i)
	}

	// ── AC3: winner is one of ["xray", "grep"] (not null, not empty) ──────────

	winnerRaw, ok := result["winner"]
	require.True(t, ok, "JSON missing 'winner' key")
	winner, ok := winnerRaw.(string)
	require.True(t, ok, "'winner' must be a string (not null), got %T", winnerRaw)
	assert.NotEmpty(t, winner, "'winner' must be non-empty")
	assert.Contains(t, []string{"xray", "grep"}, winner,
		"'winner' must be one of [xray, grep], got %q", winner)

	// ── AC4: rank 1 candidate matches winner; rank 2 has lower score ──────────

	var rank1Score, rank2Score float64
	var rank1Name string
	rank1Found, rank2Found := false, false

	for _, raw := range candidatesSlice {
		c := raw.(map[string]any)
		name, _ := c["name"].(string)

		rankRaw, hasRank := c["rank"]
		if !hasRank {
			continue
		}
		rank := int(rankRaw.(float64))

		scoreRaw, hasScore := c["score"]
		require.True(t, hasScore, "candidate %q missing 'score' key", name)
		score := scoreRaw.(float64)

		switch rank {
		case 1:
			rank1Found = true
			rank1Score = score
			rank1Name = name
		case 2:
			rank2Found = true
			rank2Score = score
		}
	}

	require.True(t, rank1Found, "no candidate has rank == 1")
	require.True(t, rank2Found, "no candidate has rank == 2")

	// AC4: rank-1 name matches winner
	assert.Equal(t, winner, rank1Name,
		"winner field %q must match rank-1 candidate %q", winner, rank1Name)

	// AC4: rank-1 score >= rank-2 score
	assert.GreaterOrEqual(t, rank1Score, rank2Score,
		"rank-1 score (%.4f) must be >= rank-2 score (%.4f)", rank1Score, rank2Score)

	// ── AC7: stderr is empty with --quiet ─────────────────────────────────────

	assert.Empty(t, stderr.Bytes(),
		"stderr must be empty when --quiet is set, got: %s", stderr.String())
}

// writeQualityScoreConfig writes a ben.yaml config file declaring the
// quality_score llm_judge plugin and returns its path.
func writeQualityScoreConfig(t *testing.T) string {
	t.Helper()
	const configYAML = `
plugins:
  metrics:
    - name: quality_score
      type: llm_judge
      model: claude-haiku-4-5
      prompt: |
        You are a strict output quality judge. Rate the following CLI output on
        how well it answers the task "Find all HTTP handler functions".
        Reply with a single float between 0.0 (useless) and 1.0 (perfect).

        Output:
        {{output}}
`
	cfgPath := filepath.Join(t.TempDir(), "ben.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(configYAML), 0o644))
	return cfgPath
}
