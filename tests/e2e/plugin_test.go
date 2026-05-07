package e2e_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildPlugin compiles a Go program in srcDir to binDir/name and returns the
// full path. Compilation is done once per test via t.TempDir isolation.
func buildPlugin(t *testing.T, srcDir, binDir, name string) string {
	t.Helper()
	bin := filepath.Join(binDir, name)
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", bin, ".")
	cmd.Dir = srcDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build %s failed: %s", name, out)
	return bin
}

// testdataDir returns the absolute path to tests/e2e/testdata.
func testdataDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "tests", "e2e", "testdata"))
	require.NoError(t, err)
	return abs
}

func TestPluginAdapter_Echo(t *testing.T) {
	ben := buildBen(t)
	binDir := t.TempDir()
	dataDir := t.TempDir()

	td := testdataDir(t)
	buildPlugin(t, filepath.Join(td, "ben-adapter-echo"), binDir, "ben-adapter-echo")

	// Prepend binDir so ben discovers ben-adapter-echo.
	origPath := os.Getenv("PATH")
	augmentedPath := binDir + string(os.PathListSeparator) + origPath

	// Suite YAML referencing the echo adapter.
	suiteYAML := `
name: echo-plugin-suite
version: 1
task:
  prompt: "test"
candidates:
  - name: echocandidate
    adapter: echo
metrics:
  - latency_ms
scorer:
  strategy: raw
`
	suiteFile := filepath.Join(t.TempDir(), "suite.yaml")
	require.NoError(t, os.WriteFile(suiteFile, []byte(suiteYAML), 0o644))

	cmd := exec.Command(ben, "run", "--suite", suiteFile, "--format", "json")
	cmd.Env = append(os.Environ(),
		"XDG_DATA_HOME="+dataDir,
		"PATH="+augmentedPath,
	)

	out, err := cmd.Output()
	require.NoError(t, err, "ben run failed: %s\nstderr: %s", out, func() string {
		if ee, ok := err.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())

	assert.Contains(t, string(out), "echocandidate")
	assert.Contains(t, string(out), "echo-plugin-output")
}

func TestPluginAdapter_CustomMetricRequested(t *testing.T) {
	ben := buildBen(t)
	binDir := t.TempDir()
	dataDir := t.TempDir()

	td := testdataDir(t)
	buildPlugin(t, filepath.Join(td, "ben-adapter-echo"), binDir, "ben-adapter-echo")

	origPath := os.Getenv("PATH")
	augmentedPath := binDir + string(os.PathListSeparator) + origPath

	suiteYAML := `
name: echo-plugin-custom-metric-suite
version: 1
task:
  prompt: "test"
candidates:
  - name: echocandidate
    adapter: echo
metrics:
  - items_count
scorer:
  strategy: raw
`
	suiteFile := filepath.Join(t.TempDir(), "suite.yaml")
	require.NoError(t, os.WriteFile(suiteFile, []byte(suiteYAML), 0o644))

	cmd := exec.Command(ben, "run", "--suite", suiteFile, "--format", "json")
	cmd.Env = append(os.Environ(),
		"XDG_DATA_HOME="+dataDir,
		"PATH="+augmentedPath,
	)

	out, err := cmd.Output()
	require.NoError(t, err, "ben run failed: %s\nstderr: %s", out, func() string {
		if ee, ok := err.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result), "output is not valid JSON: %s", out)

	cands, ok := result["candidates"].([]any)
	require.True(t, ok)
	require.Len(t, cands, 1)

	candidate, ok := cands[0].(map[string]any)
	require.True(t, ok)
	metricsMap, ok := candidate["metrics"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 7.0, metricsMap["items_count"])
}

func TestPluginReporter_JSON2(t *testing.T) {
	ben := buildBen(t)
	binDir := t.TempDir()
	dataDir := t.TempDir()

	td := testdataDir(t)
	buildPlugin(t, filepath.Join(td, "ben-reporter-json2"), binDir, "ben-reporter-json2")

	origPath := os.Getenv("PATH")
	augmentedPath := binDir + string(os.PathListSeparator) + origPath

	cmd := exec.Command(ben,
		"run",
		"--task", "echo hi",
		"--candidates", "a=cli=echo a",
		"--metric", "latency_ms",
		"--scorer", "raw",
		"--format", "json2",
	)
	cmd.Env = append(os.Environ(),
		"XDG_DATA_HOME="+dataDir,
		"PATH="+augmentedPath,
	)

	out, err := cmd.Output()
	require.NoError(t, err, "ben run failed: stderr=%s", func() string {
		if ee, ok := err.(*exec.ExitError); ok {
			return string(ee.Stderr)
		}
		return ""
	}())

	assert.Contains(t, string(out), "PLUGIN_REPORTER_OK")
}
