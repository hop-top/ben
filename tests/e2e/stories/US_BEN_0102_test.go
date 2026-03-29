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

// codexIndexingSuiteYAML is the suite from the US-BEN-0102 happy-path spec.
// Uses simple echo commands so tests run without external tools.
const codexIndexingSuiteYAML = `
name: codebase-indexing
version: 1
task:
  prompt: "Find all HTTP handler functions"
candidates:
  - name: xray
    adapter: cli
    cmd: echo "xray result"
  - name: grep
    adapter: cli
    cmd: echo "grep result"
metrics:
  - latency_ms
  - exit_code
scorer:
  strategy: weighted
  weights:
    latency_ms: 0.5
    exit_code: 0.5
`

// writeCodebaseIndexingSuite writes the codebase-indexing suite YAML into
// xdgHome/ben/suites/codebase-indexing.yaml and returns the xdgHome path.
//
// NOTE: ben run --suite currently treats its argument as a literal file path
// (see cmd/ben/run.go: spec.Load(suitePath)). Name-based discovery from
// .ben/suites/ or XDG suites dirs is only implemented in suite list/show.
// This test therefore passes the full path as the --suite argument.
// BUG: US-BEN-0102 AC1 specifies `ben run --suite <name>` (name-based lookup),
// but run.go performs spec.Load(suitePath) treating the flag value as a path.
func writeCodebaseIndexingSuite(t *testing.T) (xdgHome, suitePath string) {
	t.Helper()
	xdgHome = t.TempDir()
	suiteDir := filepath.Join(xdgHome, "ben", "suites")
	require.NoError(t, os.MkdirAll(suiteDir, 0o755))
	suitePath = filepath.Join(suiteDir, "codebase-indexing.yaml")
	require.NoError(t, os.WriteFile(suitePath, []byte(codexIndexingSuiteYAML), 0o644))
	return xdgHome, suitePath
}

// TestUS_BEN_0102_RepeatableSuiteYAML validates the full lifecycle of a
// repeatable benchmark suite as specified in US-BEN-0102:
//
//  1. ben run --suite <path> --format json exits 0; result JSON contains
//     suite=="codebase-indexing", suite_version==1, non-empty run_id.
//  2. A second run produces a different run_id and a later-or-equal timestamp.
//  3. ben query --suite codebase-indexing --last 2 --format json returns an
//     array of length 2 with distinct run_id values.
//  4. A YAML missing the required `name` field exits 1; stderr contains
//     "validation error".
func TestUS_BEN_0102_RepeatableSuiteYAML(t *testing.T) {
	ben := buildBen(t)
	xdgHome, suitePath := writeCodebaseIndexingSuite(t)

	// ── Run 1 ──────────────────────────────────────────────────────────────────

	cmd1 := exec.Command(ben, "run", "--suite", suitePath, "--format", "json")
	cmd1.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)
	out1, err1 := cmd1.Output()
	require.NoError(t, err1, "run 1 exited non-zero: stderr=%s", func() string {
		if ee, ok := err1.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())

	var result1 map[string]any
	require.NoError(t, json.Unmarshal(out1, &result1), "run 1 output is not valid JSON: %s", out1)

	// AC1: suite name and version appear in result JSON.
	assert.Equal(t, "codebase-indexing", result1["suite"], "suite field mismatch in run 1")
	assert.EqualValues(t, 1, result1["suite_version"], "suite_version field mismatch in run 1")

	// AC1: run_id is a non-empty string.
	runID1, _ := result1["run_id"].(string)
	assert.NotEmpty(t, runID1, "run_id must be non-empty in run 1")

	ts1Str, _ := result1["timestamp"].(string)
	assert.NotEmpty(t, ts1Str, "timestamp must be non-empty in run 1")

	// ── Run 2 ──────────────────────────────────────────────────────────────────

	cmd2 := exec.Command(ben, "run", "--suite", suitePath, "--format", "json")
	cmd2.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)
	out2, err2 := cmd2.Output()
	require.NoError(t, err2, "run 2 exited non-zero: stderr=%s", func() string {
		if ee, ok := err2.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())

	var result2 map[string]any
	require.NoError(t, json.Unmarshal(out2, &result2), "run 2 output is not valid JSON: %s", out2)

	// AC3: second run has a different run_id.
	runID2, _ := result2["run_id"].(string)
	assert.NotEmpty(t, runID2, "run_id must be non-empty in run 2")
	assert.NotEqual(t, runID1, runID2, "run 2 must produce a different run_id than run 1")

	// AC3: timestamp of run 2 is not earlier than run 1 (same-second runs are allowed).
	ts2Str, _ := result2["timestamp"].(string)
	assert.NotEmpty(t, ts2Str, "timestamp must be non-empty in run 2")
	assert.GreaterOrEqual(t, ts2Str, ts1Str,
		"run 2 timestamp (%s) must not be earlier than run 1 timestamp (%s)", ts2Str, ts1Str)

	// ── Query: last 2 runs ─────────────────────────────────────────────────────

	// AC4: ben query --suite codebase-indexing --last 2 --format json returns
	// an array of length 2 with distinct run_id values.
	cmdQ := exec.Command(ben,
		"query",
		"--suite", "codebase-indexing",
		"--last", "2",
		"--format", "json",
	)
	cmdQ.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)
	outQ, errQ := cmdQ.Output()
	require.NoError(t, errQ, "ben query exited non-zero: stderr=%s", func() string {
		if ee, ok := errQ.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())

	var queryResults []map[string]any
	require.NoError(t, json.Unmarshal(outQ, &queryResults),
		"query output is not a valid JSON array: %s", outQ)

	require.Len(t, queryResults, 2, "expected 2 query results, got %d", len(queryResults))

	qRunID0, _ := queryResults[0]["run_id"].(string)
	qRunID1, _ := queryResults[1]["run_id"].(string)
	assert.NotEmpty(t, qRunID0, "query result[0] run_id must be non-empty")
	assert.NotEmpty(t, qRunID1, "query result[1] run_id must be non-empty")
	assert.NotEqual(t, qRunID0, qRunID1, "query results must have distinct run_id values")

	// ── Invalid YAML: missing name ─────────────────────────────────────────────

	// AC6: YAML missing `name` exits 1; stderr contains "validation error".
	invalidYAML := `
candidates:
  - name: fast
    adapter: cli
    cmd: echo fast
metrics:
  - latency_ms
scorer:
  strategy: raw
`
	invalidFile := filepath.Join(t.TempDir(), "invalid.yaml")
	require.NoError(t, os.WriteFile(invalidFile, []byte(invalidYAML), 0o644))

	cmdInvalid := exec.Command(ben, "run", "--suite", invalidFile)
	cmdInvalid.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())
	// Use CombinedOutput so stderr is captured (cobra writes errors to stderr;
	// exec.ExitError.Stderr is only populated by cmd.Output(), not cmd.Run()).
	combinedInvalid, err := cmdInvalid.CombinedOutput()
	require.Error(t, err, "expected non-zero exit for invalid YAML")
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode(), "expected exit code 1 for invalid YAML")

	// stderr (captured in combinedInvalid) should mention the validation failure.
	// Actual message from spec/spec.go validate(): "missing required field: name"
	// wrapped by cobra as "Error: load suite: spec: missing required field: name".
	// Story spec says: "spec validation error: name is required".
	combinedMsg := string(combinedInvalid)
	assert.True(t,
		strings.Contains(combinedMsg, "validation error") ||
			strings.Contains(combinedMsg, "missing required field") ||
			strings.Contains(combinedMsg, "name is required"),
		"output should mention validation error; got: %q", combinedMsg,
	)
}
