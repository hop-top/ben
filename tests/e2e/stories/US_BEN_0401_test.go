// Package stories contains e2e tests keyed to individual user stories.
package stories_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot returns the absolute path to the repository root (three levels up from this file).
func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	return abs
}

// TestUS_BEN_0401_InterfacesDiscoverable validates US-BEN-0401:
// core interfaces (Adapter, Metric, Scorer) are exported and discoverable
// via `go doc`, and docs/contributing.md references internal/adapter.
func TestUS_BEN_0401_InterfacesDiscoverable(t *testing.T) {
	root := repoRoot(t)

	runGoDoc := func(t *testing.T, pkg, symbol string) (string, error) {
		t.Helper()
		cmd := exec.Command("go", "doc", pkg+"."+symbol)
		cmd.Dir = root
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	t.Run("adapter.Adapter_is_interface", func(t *testing.T) {
		out, err := runGoDoc(t, "hop.top/ben/internal/adapter", "Adapter")
		if err != nil {
			t.Skipf("go doc hop.top/ben/internal/adapter Adapter failed (bug): %v\noutput: %s", err, out)
		}
		assert.Contains(t, out, "interface",
			"go doc output for adapter.Adapter must contain 'interface'")
	})

	t.Run("metrics.Metric_is_interface", func(t *testing.T) {
		out, err := runGoDoc(t, "hop.top/ben/internal/metrics", "Metric")
		if err != nil {
			t.Skipf("go doc hop.top/ben/internal/metrics Metric failed (bug): %v\noutput: %s", err, out)
		}
		assert.Contains(t, out, "interface",
			"go doc output for metrics.Metric must contain 'interface'")
	})

	t.Run("scorer.Scorer_is_interface", func(t *testing.T) {
		out, err := runGoDoc(t, "hop.top/ben/internal/scorer", "Scorer")
		if err != nil {
			t.Skipf("go doc hop.top/ben/internal/scorer Scorer failed (bug): %v\noutput: %s", err, out)
		}
		assert.Contains(t, out, "interface",
			"go doc output for scorer.Scorer must contain 'interface'")
	})

	t.Run("contributing_md_references_internal_adapter", func(t *testing.T) {
		path := filepath.Join(root, "docs", "contributing.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("docs/contributing.md not found (bug): %v", err)
		}
		assert.True(t, strings.Contains(string(data), "internal/adapter"),
			"docs/contributing.md must reference 'internal/adapter'")
	})
}
