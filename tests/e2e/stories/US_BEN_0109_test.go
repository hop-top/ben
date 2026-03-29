package stories_test

// US-BEN-0109 — Full result schema validation.
//
// Story: verify that `ben run --format json` emits a complete, well-typed
// JSON object matching the documented run.Run schema.
//
// ACs tested:
//   - Stdout is valid JSON object with all required top-level keys.
//   - run_id matches ULID regex ^[0-9A-Z]{26}$.
//   - timestamp parses as time.RFC3339.
//   - scorer.strategy is non-empty.
//   - candidates is array of length >= 1.
//   - Each candidate: name (string), metrics (object), score (float64, non-null
//     with weighted scorer), rank (int, non-null), raw_output (string),
//     error absent or string.
//   - metrics object contains latency_ms, exit_code, output_size; all numbers.
//   - metadata.host is non-empty.
//   - winner is one of the candidate names (non-null with weighted scorer).
//
// Known notes:
//   - error field has omitempty; absent from JSON when empty (not JSON null).
//     Spec says "string or null" but actual behaviour omits the key. Assertion
//     is permissive: if present, must be string.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ulidRegex = regexp.MustCompile(`^[0-9A-Z]{26}$`)

func TestUS_BEN_0109_FullResultSchema(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	cmd := exec.Command(ben,
		"run",
		"--task", "List files",
		"--candidates", "ls=cli=ls,find=cli=find . -maxdepth 1",
		"--metric", "latency_ms,exit_code,output_size",
		"--scorer", "weighted:latency_ms=0.5,exit_code=0.3,output_size=0.2",
		"--format", "json",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Run(), "ben run exited non-zero (stderr: %s)", stderr.String())

	// --- top-level JSON object ---

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result),
		"stdout is not valid JSON: %s", stdout.String())

	requiredKeys := []string{
		"run_id", "suite", "suite_version", "timestamp",
		"scorer", "candidates", "winner", "metadata",
	}
	for _, k := range requiredKeys {
		assert.Contains(t, result, k, "top-level key %q missing from result", k)
	}

	// --- run_id: ULID ---

	runID, _ := result["run_id"].(string)
	assert.Regexp(t, ulidRegex, runID, "run_id %q does not match ULID regex", runID)

	// --- timestamp: RFC3339 ---

	tsRaw, _ := result["timestamp"].(string)
	require.NotEmpty(t, tsRaw, "timestamp is empty or not a string")
	_, err := time.Parse(time.RFC3339, tsRaw)
	assert.NoError(t, err, "timestamp %q does not parse as RFC3339", tsRaw)

	// --- scorer ---

	scorerRaw, ok := result["scorer"]
	require.True(t, ok, "scorer key missing")
	scorerMap, ok := scorerRaw.(map[string]any)
	require.True(t, ok, "scorer is not an object")

	strategy, _ := scorerMap["strategy"].(string)
	assert.NotEmpty(t, strategy, "scorer.strategy is empty")

	// --- candidates ---

	candsRaw, ok := result["candidates"]
	require.True(t, ok, "candidates key missing")
	cands, ok := candsRaw.([]any)
	require.True(t, ok, "candidates is not an array")
	require.GreaterOrEqual(t, len(cands), 1, "candidates array must have length >= 1")

	// Collect candidate names for winner validation.
	candNames := make(map[string]bool, len(cands))

	for _, raw := range cands {
		c, ok := raw.(map[string]any)
		require.True(t, ok, "candidate element is not an object")

		// name: string
		name, ok := c["name"].(string)
		require.True(t, ok, "candidate name is not a string: %v", c["name"])
		assert.NotEmpty(t, name, "candidate name is empty")
		candNames[name] = true

		// metrics: object with required keys, all numeric
		metricsRaw, ok := c["metrics"]
		require.True(t, ok, "candidate %q: metrics key missing", name)
		metrics, ok := metricsRaw.(map[string]any)
		require.True(t, ok, "candidate %q: metrics is not an object", name)

		for _, mk := range []string{"latency_ms", "exit_code", "output_size"} {
			v, exists := metrics[mk]
			assert.True(t, exists, "candidate %q: metrics.%s missing", name, mk)
			if exists {
				_, isNum := v.(float64)
				assert.True(t, isNum, "candidate %q: metrics.%s value %v is not a number", name, mk, v)
			}
		}

		// score: float64 (non-null with weighted scorer)
		scoreVal, scoreExists := c["score"]
		require.True(t, scoreExists, "candidate %q: score key missing", name)
		require.NotNil(t, scoreVal, "candidate %q: score is null (expected non-null with weighted scorer)", name)
		_, scoreIsFloat := scoreVal.(float64)
		assert.True(t, scoreIsFloat, "candidate %q: score %v is not float64", name, scoreVal)

		// rank: int (non-null with weighted scorer); JSON numbers arrive as float64
		rankVal, rankExists := c["rank"]
		require.True(t, rankExists, "candidate %q: rank key missing", name)
		require.NotNil(t, rankVal, "candidate %q: rank is null (expected non-null with weighted scorer)", name)
		rankFloat, rankIsFloat := rankVal.(float64)
		assert.True(t, rankIsFloat, "candidate %q: rank %v is not a number", name, rankVal)
		if rankIsFloat {
			assert.Equal(t, float64(int(rankFloat)), rankFloat,
				"candidate %q: rank %v is not an integer", name, rankFloat)
		}

		// raw_output: string (may be empty)
		rawOut, ok := c["raw_output"]
		assert.True(t, ok, "candidate %q: raw_output key missing", name)
		_, isStr := rawOut.(string)
		assert.True(t, isStr, "candidate %q: raw_output %v is not a string", name, rawOut)

		// error: if present, must be string (omitempty → absent when empty)
		if errVal, exists := c["error"]; exists {
			_, isStr := errVal.(string)
			assert.True(t, isStr, "candidate %q: error %v is not a string", name, errVal)
		}
	}

	// --- metadata.host ---

	metaRaw, ok := result["metadata"]
	require.True(t, ok, "metadata key missing")
	meta, ok := metaRaw.(map[string]any)
	require.True(t, ok, "metadata is not an object")

	host, _ := meta["host"].(string)
	assert.NotEmpty(t, host, "metadata.host is empty")

	// --- winner ---

	// winner is present in the result (key exists); with weighted scorer it is non-null.
	winnerRaw, winnerKeyExists := result["winner"]
	assert.True(t, winnerKeyExists, "winner key missing from result")
	require.NotNil(t, winnerRaw, "winner is null (expected non-null with weighted scorer)")
	winner, ok := winnerRaw.(string)
	require.True(t, ok, "winner %v is not a string", winnerRaw)
	assert.True(t, candNames[winner],
		"winner %q is not one of the candidate names %v", winner, candNames)
}
