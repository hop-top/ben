// Package stories contains e2e tests that validate named user stories.
package stories_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSingleCandidateSuite writes a suite YAML with two CLI candidates and a
// single:latency_ms scorer (i.e. not "raw") so that winner is populated.
// Returns the path to the suite YAML file.
func writeSingleCandidateSuite(t *testing.T) string {
	t.Helper()
	const suiteYAML = `
name: ci-json-test
description: Suite for US-BEN-0104 e2e test
version: 1
task:
  prompt: "echo hello"
candidates:
  - name: fast
    adapter: cli
    cmd: echo fast
  - name: slow
    adapter: cli
    cmd: echo slow
metrics:
  - latency_ms
scorer:
  strategy: "single:latency_ms"
`
	suitePath := filepath.Join(t.TempDir(), "ci-json-test.yaml")
	require.NoError(t, os.WriteFile(suitePath, []byte(suiteYAML), 0o644))
	return suitePath
}

// ansiRE matches ANSI escape sequences.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// TestUS_BEN_0104_FormatJSONInCI verifies the US-BEN-0104 acceptance criteria:
// machine-readable JSON output suitable for CI pipeline consumption.
func TestUS_BEN_0104_FormatJSONInCI(t *testing.T) {
	ben := buildBen(t)
	suitePath := writeSingleCandidateSuite(t)
	dataDir := t.TempDir()
	env := append(os.Environ(), "XDG_DATA_HOME="+dataDir)

	t.Run("happy_path_exits_zero", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(ben,
			"--format", "json",
			"--quiet",
			"run",
			"--suite", suitePath,
		)
		cmd.Env = env
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		require.NoError(t, err, "ben run exited non-zero; stderr: %s", stderr.String())
	})

	t.Run("stdout_is_valid_json", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(ben,
			"--format", "json",
			"--quiet",
			"run",
			"--suite", suitePath,
		)
		cmd.Env = env
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		require.NoError(t, cmd.Run(), "ben run failed; stderr: %s", stderr.String())

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &result),
			"stdout is not valid JSON: %s", stdout.String())
	})

	t.Run("stdout_contains_no_ansi_sequences", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(ben,
			"--format", "json",
			"--quiet",
			"run",
			"--suite", suitePath,
		)
		cmd.Env = env
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		require.NoError(t, cmd.Run(), "ben run failed; stderr: %s", stderr.String())

		assert.False(t, ansiRE.Match(stdout.Bytes()),
			"stdout contains ANSI escape sequences: %s", stdout.String())
	})

	t.Run("winner_field_present_and_non_empty", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(ben,
			"--format", "json",
			"--quiet",
			"run",
			"--suite", suitePath,
		)
		cmd.Env = env
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		require.NoError(t, cmd.Run(), "ben run failed; stderr: %s", stderr.String())

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

		winner, ok := result["winner"]
		require.True(t, ok, "top-level key 'winner' missing from JSON output")

		winnerStr, isStr := winner.(string)
		require.True(t, isStr, "'winner' must be a string, got %T", winner)
		assert.NotEmpty(t, winnerStr, "'winner' must be a non-empty string")
	})

	t.Run("stderr_empty_with_quiet_flag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(ben,
			"--format", "json",
			"--quiet",
			"run",
			"--suite", suitePath,
		)
		cmd.Env = env
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		require.NoError(t, cmd.Run(), "ben run failed; stderr: %s", stderr.String())

		assert.Empty(t, stderr.Bytes(),
			"stderr must be empty when --quiet is set, got: %s", stderr.String())
	})

	t.Run("bad_format_exits_one", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(ben,
			"--format", "bad-value",
			"run",
			"--task", "echo hi",
			"--candidates", "a=cli=echo a",
			"--metric", "latency_ms",
		)
		cmd.Env = env
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		require.Error(t, err, "expected non-zero exit for bad --format")

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.ExitCode(),
			"expected exit code 1 for bad --format")

		assert.NotEmpty(t, stderr.Bytes(),
			"stderr must be non-empty for bad --format error")
		assert.Empty(t, stdout.Bytes(),
			"stdout must be empty when --format is invalid")
	})
}
