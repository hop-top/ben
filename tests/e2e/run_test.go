package e2e_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildBen compiles the ben binary into t.TempDir() and returns its path.
func buildBen(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "ben")
	cmd := exec.Command("go", "build", "-o", bin, "hop.top/ben/cmd/ben")
	cmd.Dir = filepath.Join("..", "..")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)
	return bin
}

func TestRunCommand_HappyPath(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	cmd := exec.Command(ben,
		"run",
		"--task", "echo hello",
		"--candidates", "fast=cli=echo fast,slow=cli=sleep 0.05 && echo slow",
		"--metric", "latency_ms,exit_code",
		"--scorer", "single:latency_ms",
		"--format", "json",
	)
	// Point storage at temp dir so tests don't pollute user data.
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

	out, err := cmd.Output()
	require.NoError(t, err, "ben run exited non-zero: %s", out)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result), "output is not valid JSON: %s", out)

	assert.Equal(t, "fast", result["winner"])

	cands, ok := result["candidates"].([]any)
	require.True(t, ok)
	require.Len(t, cands, 2)

	for _, raw := range cands {
		c := raw.(map[string]any)
		m, ok := c["metrics"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, m, "latency_ms")
		assert.NotNil(t, c["rank"])
	}
}

func TestRunCommand_BadSuitePath_ExitsOne(t *testing.T) {
	ben := buildBen(t)

	cmd := exec.Command(ben, "run", "--suite", "/nonexistent/suite.yaml")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())

	err := cmd.Run()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
}

func TestRunCommand_TableFormat(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	cmd := exec.Command(ben,
		"run",
		"--task", "echo hi",
		"--candidates", "a=cli=echo a,b=cli=echo b",
		"--metric", "latency_ms",
		"--scorer", "single:latency_ms",
		"--format", "table",
	)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)

	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "Rank")
	assert.Contains(t, string(out), "Name")
}
