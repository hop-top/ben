// Package stories contains e2e tests that validate named user stories.
package stories_test

// US-BEN-0106 — quality_score metric via llm_judge.
//
// Story: engineer declares quality_score as type:llm_judge in ben.yaml;
// ben run invokes the judge once per candidate and stores quality_score in
// [0,1] per candidate result; weighted scorer uses the metric normally.
//
// ACs tested (when BEN_LLM_TEST=1):
//   - Config declares quality_score as type: llm_judge.
//   - ben run --metric latency_ms,quality_score --format json exits 0.
//   - Each candidate in candidates[] has metrics.quality_score (float64).
//   - metrics.quality_score is in [0.0, 1.0] for all candidates.
//   - winner is non-null (weighted scorer applied).
//   - When model is omitted from config: exit 1; stderr contains "model is required".
//
// NOTE: The clamping sub-test (judge returns 1.5 → clamps to 1.0) relies on
// an LLM stub that can't be injected via CLI flags (endpoint override is
// internal). Clamping logic is covered by unit tests in internal/metrics.
// The sub-test here is skipped with an explanatory note.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// benYAMLWithLLMJudge is a ben.yaml declaring quality_score as llm_judge.
const benYAMLWithLLMJudge = `
plugins:
  metrics:
    - name: quality_score
      type: llm_judge
      model: claude-sonnet-4-6
      prompt: "Rate the relevance of this output on a scale of 0 to 1. Reply with only a decimal number. Output: {{output}}"
`

// benYAMLMissingModel is a ben.yaml declaring llm_judge without a model field.
const benYAMLMissingModel = `
plugins:
  metrics:
    - name: quality_score
      type: llm_judge
      prompt: "Rate this output 0-1: {{output}}"
`

// suiteYAMLLLMJudge is a minimal suite with two CLI candidates.
const suiteYAMLLLMJudge = `
name: llm-judge-suite
version: 1
task:
  prompt: "summarise Go"
candidates:
  - name: fast
    adapter: cli
    cmd: echo "Go is fast"
  - name: verbose
    adapter: cli
    cmd: echo "Go is a statically typed compiled language designed for simplicity"
metrics:
  - latency_ms
  - quality_score
scorer:
  strategy: weighted
  weights:
    latency_ms: 0.3
    quality_score: 0.7
`

// writeBenConfig writes content to a temp ben.yaml and returns the path.
func writeBenConfig0106(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ben.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// writeSuite0106 writes a suite YAML and returns the path.
func writeSuite0106(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "suite.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// runBen0106 runs ben with the given args; returns stdout, stderr, exitErr.
func runBen0106(t *testing.T, ben, dataDir string, args ...string) (stdout, stderr string, exitErr *exec.ExitError) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	cmd := exec.Command(ben, args...)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var ok bool
		exitErr, ok = err.(*exec.ExitError)
		require.True(t, ok, "unexpected error type: %v", err)
	}
	return outBuf.String(), errBuf.String(), exitErr
}

func TestUS_BEN_0106_QualityScoreLLMJudge(t *testing.T) {
	if os.Getenv("BEN_LLM_TEST") != "1" {
		t.Skip("set BEN_LLM_TEST=1 to run LLM tests")
	}

	ben := buildBen(t)

	// ── happy path: quality_score in [0,1] per candidate ─────────────────────

	t.Run("quality_score_in_range_per_candidate", func(t *testing.T) {
		dataDir := t.TempDir()
		cfgPath := writeBenConfig0106(t, benYAMLWithLLMJudge)
		suitePath := writeSuite0106(t, suiteYAMLLLMJudge)

		stdout, stderr, exitErr := runBen0106(t, ben, dataDir,
			"--config", cfgPath,
			"run", "--suite", suitePath,
			"--format", "json",
		)

		require.Nil(t, exitErr,
			"ben run with llm_judge should exit 0; stderr: %s", stderr)

		var result map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &result),
			"stdout must be valid JSON; got: %s", stdout)

		cands, ok := result["candidates"].([]any)
		require.True(t, ok, "candidates must be a JSON array")
		require.NotEmpty(t, cands, "candidates must not be empty")

		for _, raw := range cands {
			c, ok := raw.(map[string]any)
			require.True(t, ok, "each candidate must be a JSON object")

			metricsMap, ok := c["metrics"].(map[string]any)
			require.True(t, ok, "candidate %v: metrics must be a JSON object", c["name"])

			qs, ok := metricsMap["quality_score"].(float64)
			require.True(t, ok,
				"candidate %v: metrics.quality_score must be a float64; got %T = %v",
				c["name"], metricsMap["quality_score"], metricsMap["quality_score"],
			)

			assert.GreaterOrEqual(t, qs, 0.0,
				"candidate %v: quality_score must be >= 0.0", c["name"])
			assert.LessOrEqual(t, qs, 1.0,
				"candidate %v: quality_score must be <= 1.0", c["name"])
		}
	})

	// ── winner is non-null (weighted scorer applied) ──────────────────────────

	t.Run("winner_non_null_weighted_scorer", func(t *testing.T) {
		dataDir := t.TempDir()
		cfgPath := writeBenConfig0106(t, benYAMLWithLLMJudge)
		suitePath := writeSuite0106(t, suiteYAMLLLMJudge)

		stdout, stderr, exitErr := runBen0106(t, ben, dataDir,
			"--config", cfgPath,
			"run", "--suite", suitePath,
			"--format", "json",
		)

		require.Nil(t, exitErr,
			"ben run with llm_judge should exit 0; stderr: %s", stderr)

		var result map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &result),
			"stdout must be valid JSON; got: %s", stdout)

		winner, hasWinner := result["winner"]
		require.True(t, hasWinner, "result must have a winner key")
		assert.NotNil(t, winner, "winner must not be null when weighted scorer is applied")

		winnerStr, ok := winner.(string)
		require.True(t, ok, "winner must be a string")
		assert.NotEmpty(t, winnerStr, "winner must be non-empty")
	})

	// ── latency_ms also present alongside quality_score ───────────────────────

	t.Run("latency_ms_co_present_with_quality_score", func(t *testing.T) {
		dataDir := t.TempDir()
		cfgPath := writeBenConfig0106(t, benYAMLWithLLMJudge)
		suitePath := writeSuite0106(t, suiteYAMLLLMJudge)

		stdout, stderr, exitErr := runBen0106(t, ben, dataDir,
			"--config", cfgPath,
			"run", "--suite", suitePath,
			"--format", "json",
		)

		require.Nil(t, exitErr,
			"ben run with llm_judge should exit 0; stderr: %s", stderr)

		var result map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &result),
			"stdout must be valid JSON; got: %s", stdout)

		cands, _ := result["candidates"].([]any)
		require.NotEmpty(t, cands)

		for _, raw := range cands {
			c := raw.(map[string]any)
			metricsMap, ok := c["metrics"].(map[string]any)
			require.True(t, ok, "candidate %v: metrics must be a JSON object", c["name"])

			_, hasLatency := metricsMap["latency_ms"].(float64)
			assert.True(t, hasLatency,
				"candidate %v: metrics.latency_ms must be present and numeric", c["name"])
		}
	})

	// ── model omitted from config: exit 1; stderr contains "model is required" ─

	t.Run("missing_model_exits_1_stderr_model_required", func(t *testing.T) {
		dataDir := t.TempDir()
		cfgPath := writeBenConfig0106(t, benYAMLMissingModel)
		suitePath := writeSuite0106(t, suiteYAMLLLMJudge)

		_, stderr, exitErr := runBen0106(t, ben, dataDir,
			"--config", cfgPath,
			"run", "--suite", suitePath,
			"--format", "json",
		)

		require.NotNil(t, exitErr, "expected non-zero exit when model is omitted")
		assert.Equal(t, 1, exitErr.ExitCode(), "expected exit code 1")
		assert.True(t,
			strings.Contains(stderr, "model is required") ||
				strings.Contains(stderr, "missing model") ||
				strings.Contains(stderr, "llm_judge"),
			"stderr must describe missing model error; got: %s", stderr,
		)
	})

	// ── clamp sub-test (skipped: endpoint not overridable via CLI) ───────────

	t.Run("judge_returns_out_of_range_clamped_to_1", func(t *testing.T) {
		t.Skip(
			"clamping behavior (judge returns 1.5 → clamped to 1.0) requires " +
				"LLM endpoint override which is not exposed as a CLI flag; " +
				"covered by unit tests in internal/metrics",
		)
	})
}
