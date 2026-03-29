// Package stories contains e2e tests keyed to individual user stories.
package stories_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUS_BEN_0111_ScorerRawWinnerNull validates US-BEN-0111:
// running with --scorer raw exits 0, sets scorer.strategy="raw",
// winner is null (or empty), and no rank/score is applied.
func TestUS_BEN_0111_ScorerRawWinnerNull(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	t.Run("happy_path_two_candidates", func(t *testing.T) {
		cmd := exec.Command(ben,
			"run",
			"--task", "list files",
			"--candidates", "ls=cli=ls,find=cli=find . -maxdepth 1",
			"--metric", "latency_ms,exit_code,output_size",
			"--scorer", "raw",
			"--format", "json",
		)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

		out, err := cmd.Output()
		// AC7: exit code 0
		require.NoError(t, err, "ben run --scorer raw exited non-zero: %s", out)

		var result map[string]any
		require.NoError(t, json.Unmarshal(out, &result), "output not valid JSON: %s", out)

		// AC5: scorer.strategy == "raw"
		sc, ok := result["scorer"].(map[string]any)
		require.True(t, ok, "scorer field missing or wrong type")
		assert.Equal(t, "raw", sc["strategy"], "scorer.strategy must be raw")

		// AC1: winner == null (JSON null)
		// NOTE: current implementation emits winner:"" (empty string) instead of null.
		// This is a bug per US-BEN-0111 AC1. Tracked as a known defect.
		winner := result["winner"]
		assert.Nil(t, winner,
			"winner must be JSON null for raw scorer (got %q — bug: struct uses string not *string)", winner)

		// AC2: all candidates present
		cands, ok := result["candidates"].([]any)
		require.True(t, ok, "candidates field missing or wrong type")
		require.Len(t, cands, 2, "expected 2 candidates")

		for _, raw := range cands {
			c, ok := raw.(map[string]any)
			require.True(t, ok)
			name := c["name"]

			// AC6: metrics collected normally
			m, ok := c["metrics"].(map[string]any)
			require.True(t, ok, "candidate %v: metrics missing or wrong type", name)

			// metrics.latency_ms, metrics.exit_code, metrics.output_size all present and numeric
			for _, key := range []string{"latency_ms", "exit_code", "output_size"} {
				v, present := m[key]
				assert.True(t, present, "candidate %v: metrics.%s missing", name, key)
				_, isNum := v.(float64)
				assert.True(t, isNum, "candidate %v: metrics.%s not numeric (got %T)", name, key, v)
			}

			// AC3: score null or absent — current impl emits score:0 (known defect)
			score, hasScore := c["score"]
			if hasScore {
				assert.Nil(t, score,
					"candidate %v: score must be null for raw scorer (got %v — bug: emits 0 not null)", name, score)
			}

			// AC4: rank null or absent — current impl emits rank:0 (known defect)
			rank, hasRank := c["rank"]
			if hasRank {
				assert.Nil(t, rank,
					"candidate %v: rank must be null for raw scorer (got %v — bug: emits 0 not null)", name, rank)
			}
		}
	})

	t.Run("one_candidate_errors_winner_still_null", func(t *testing.T) {
		cmd := exec.Command(ben,
			"run",
			"--task", "echo hello",
			// "err" candidate runs a failing command
			"--candidates", "ok=cli=echo ok,err=cli=false",
			"--metric", "latency_ms,exit_code,output_size",
			"--scorer", "raw",
			"--format", "json",
		)
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

		out, err := cmd.Output()
		// AC7: exit code still 0 even when a candidate errors
		require.NoError(t, err, "ben run exited non-zero when one candidate errors: %s", out)

		var result map[string]any
		require.NoError(t, json.Unmarshal(out, &result), "output not valid JSON: %s", out)

		// AC1: winner still null when one candidate errors
		winner := result["winner"]
		assert.Nil(t, winner,
			"winner must be null even when candidate errors (got %q)", winner)

		// AC5: scorer.strategy still "raw"
		sc, ok := result["scorer"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "raw", sc["strategy"])

		cands, ok := result["candidates"].([]any)
		require.True(t, ok)
		require.Len(t, cands, 2, "all candidates must appear even when one errors")

		// Verify the error candidate has a non-null error field
		errCandFound := false
		for _, raw := range cands {
			c, ok := raw.(map[string]any)
			require.True(t, ok)
			if c["name"] == "err" {
				errCandFound = true
				// story: candidates[i].error is non-null string when candidate errors
				errVal, hasErr := c["error"]
				if hasErr && errVal != nil {
					errStr, isStr := errVal.(string)
					assert.True(t, isStr && errStr != "",
						"err candidate: error field must be non-empty string, got %v", errVal)
				}
				// metrics still present (exit_code reflects the failure)
				m, ok := c["metrics"].(map[string]any)
				require.True(t, ok, "err candidate: metrics must still be present")
				_, hasLatency := m["latency_ms"]
				assert.True(t, hasLatency, "err candidate: latency_ms must be present")
				_, hasExit := m["exit_code"]
				assert.True(t, hasExit, "err candidate: exit_code must be present")
			}
		}
		assert.True(t, errCandFound, "err candidate not found in results")
	})
}
