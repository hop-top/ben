// Package stories contains e2e tests keyed to individual user stories.
package stories_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// US-BEN-0206: ben suite show
//
// Story: as a user I want `ben suite show <name>` to display full details of a
// named benchmark suite, and to exit 1 with a "not found" message when the suite
// does not exist.

const showSuiteYAML = `
name: show-fixture-suite
description: Fixture suite for US-BEN-0206 e2e test
version: 3
task:
  prompt: "echo test"
candidates:
  - name: alpha
    adapter: cli
    cmd: echo alpha
  - name: beta
    adapter: cli
    cmd: echo beta
metrics:
  - latency_ms
  - exit_code
scorer:
  strategy: raw
`

// writeShowFixture writes showSuiteYAML into <tmpDir>/.ben/suites/show-fixture-suite.yaml.
// Returns tmpDir so the caller can use it as cmd.Dir.
func writeShowFixture(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	suiteDir := filepath.Join(tmpDir, ".ben", "suites")
	require.NoError(t, os.MkdirAll(suiteDir, 0o755))
	suiteFile := filepath.Join(suiteDir, "show-fixture-suite.yaml")
	require.NoError(t, os.WriteFile(suiteFile, []byte(showSuiteYAML), 0o644))
	return tmpDir
}

// runSuiteShow executes ben suite show <name> from cwd with an isolated XDG_DATA_HOME.
// Returns (stdout, stderr, exitCode, error).
func runSuiteShow(t *testing.T, ben, cwd string, args ...string) (string, string, int, error) {
	t.Helper()
	cmdArgs := append([]string{"suite", "show"}, args...)
	cmd := exec.Command(ben, cmdArgs...)
	cmd.Dir = cwd
	// Isolate global suite dir so only the project-local one is used.
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return stdout.String(), stderr.String(), ee.ExitCode(), err
		}
		return stdout.String(), stderr.String(), -1, err
	}
	return stdout.String(), stderr.String(), 0, nil
}

// TestUS_BEN_0206_SuiteShow validates US-BEN-0206 acceptance criteria.
func TestUS_BEN_0206_SuiteShow(t *testing.T) {
	ben := buildBen(t)

	t.Run("exits_0_for_existing_suite", func(t *testing.T) {
		// AC: `ben suite show <fixture-suite-name>` exits 0.
		dir := writeShowFixture(t)
		_, _, code, err := runSuiteShow(t, ben, dir, "show-fixture-suite")
		require.NoError(t, err, "ben suite show must exit 0 for existing suite")
		assert.Equal(t, 0, code)
	})

	t.Run("stdout_contains_name", func(t *testing.T) {
		// AC: stdout contains name field matching suite name.
		dir := writeShowFixture(t)
		stdout, _, _, err := runSuiteShow(t, ben, dir, "show-fixture-suite")
		require.NoError(t, err)
		assert.Contains(t, stdout, "show-fixture-suite",
			"stdout must contain suite name; got: %q", stdout)
	})

	t.Run("stdout_contains_candidates_section", func(t *testing.T) {
		// AC: stdout contains "candidates" section with at least one entry.
		dir := writeShowFixture(t)
		stdout, _, _, err := runSuiteShow(t, ben, dir, "show-fixture-suite")
		require.NoError(t, err)
		assert.Contains(t, stdout, "Candidates:",
			"stdout must contain 'Candidates:' section header; got: %q", stdout)
		// At least one candidate entry must appear (alpha or beta).
		assert.True(t,
			strings.Contains(stdout, "alpha") || strings.Contains(stdout, "beta"),
			"stdout must contain at least one candidate name; got: %q", stdout)
	})

	t.Run("stdout_contains_metrics", func(t *testing.T) {
		// AC: stdout contains "metrics" list.
		dir := writeShowFixture(t)
		stdout, _, _, err := runSuiteShow(t, ben, dir, "show-fixture-suite")
		require.NoError(t, err)
		assert.Contains(t, stdout, "Metrics:",
			"stdout must contain 'Metrics:' line; got: %q", stdout)
		assert.Contains(t, stdout, "latency_ms",
			"stdout must list metric 'latency_ms'; got: %q", stdout)
	})

	t.Run("format_yaml_exits_0_and_parses", func(t *testing.T) {
		// AC: `--format yaml` output parses as valid YAML with "name" key.
		dir := writeShowFixture(t)

		// --format is a root flag; must precede subcommand.
		cmd := exec.Command(ben, "--format", "yaml", "suite", "show", "show-fixture-suite")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())

		out, err := cmd.Output()
		require.NoError(t, err, "ben --format yaml suite show must exit 0; stderr: %s",
			func() string {
				if ee, ok := err.(*exec.ExitError); ok {
					return string(ee.Stderr)
				}
				return ""
			}())

		var parsed map[string]any
		require.NoError(t, yaml.Unmarshal(out, &parsed),
			"stdout must be valid YAML; got: %q", string(out))

		name, ok := parsed["name"].(string)
		assert.True(t, ok, "YAML output must have a string 'name' key")
		assert.Equal(t, "show-fixture-suite", name,
			"YAML 'name' must match suite name")
	})

	t.Run("nonexistent_exits_1", func(t *testing.T) {
		// AC: `ben suite show nonexistent` exits 1 and stderr matches "not found".
		emptyDir := t.TempDir()
		_, stderr, code, err := runSuiteShow(t, ben, emptyDir, "nonexistent")
		require.Error(t, err, "ben suite show nonexistent must exit non-zero")
		assert.Equal(t, 1, code, "exit code must be 1 for missing suite")
		assert.Contains(t, stderr, "not found",
			"stderr must contain 'not found'; got: %q", stderr)
	})
}
