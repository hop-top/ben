package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSuiteYAML = `
name: my-test-suite
description: A suite for e2e testing
version: 2
task:
  prompt: "echo test"
candidates:
  - name: fast
    adapter: cli
    cmd: echo fast
  - name: slow
    adapter: cli
    cmd: sleep 0.01
metrics:
  - latency_ms
scorer:
  strategy: raw
`

// writeTempSuite writes testSuiteYAML to XDG_DATA_HOME/ben/suites/my-test-suite.yaml
// and returns the XDG_DATA_HOME dir.
func writeTempSuite(t *testing.T) string {
	t.Helper()
	xdgHome := t.TempDir()
	suiteDir := filepath.Join(xdgHome, "ben", "suites")
	require.NoError(t, os.MkdirAll(suiteDir, 0o755))
	suiteFile := filepath.Join(suiteDir, "my-test-suite.yaml")
	require.NoError(t, os.WriteFile(suiteFile, []byte(testSuiteYAML), 0o644))
	return xdgHome
}

func TestSuiteList_ShowsName(t *testing.T) {
	ben := buildBen(t)
	xdgHome := writeTempSuite(t)

	cmd := exec.Command(ben, "suite", "list")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)

	out, err := cmd.Output()
	require.NoError(t, err, "ben suite list failed: %s", func() string {
		if ee, ok := err.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())
	assert.Contains(t, string(out), "my-test-suite")
}

func TestSuiteList_JSONFormat(t *testing.T) {
	ben := buildBen(t)
	xdgHome := writeTempSuite(t)

	cmd := exec.Command(ben, "--format", "json", "suite", "list")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)

	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "my-test-suite")
	assert.Contains(t, string(out), "A suite for e2e testing")
}

func TestSuiteShow_PrintsCandidates(t *testing.T) {
	ben := buildBen(t)
	xdgHome := writeTempSuite(t)

	cmd := exec.Command(ben, "suite", "show", "my-test-suite")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)

	out, err := cmd.Output()
	require.NoError(t, err, "ben suite show failed: stderr=%s", func() string {
		if ee, ok := err.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())
	combined := string(out)
	assert.Contains(t, combined, "my-test-suite")
}

func TestSuiteShow_Nonexistent_ExitsOne(t *testing.T) {
	ben := buildBen(t)
	xdgHome := t.TempDir()

	cmd := exec.Command(ben, "suite", "show", "nonexistent-suite-xyz")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+xdgHome)

	err := cmd.Run()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
}
