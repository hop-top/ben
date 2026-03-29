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

// TestUS_BEN_0405_CIPipelineDefinedAndPasses validates US-BEN-0405:
// a CI workflow file exists, contains required CI directives, and the
// project builds cleanly.
func TestUS_BEN_0405_CIPipelineDefinedAndPasses(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")

	t.Run("ci_yml_exists", func(t *testing.T) {
		_, err := os.Stat(ciPath)
		require.NoError(t, err, ".github/workflows/ci.yml must exist at repo root")
	})

	data, err := os.ReadFile(ciPath)
	require.NoError(t, err, "must be able to read .github/workflows/ci.yml")
	content := string(data)

	t.Run("contains_go_test", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "go test"),
			"ci.yml must contain 'go test'")
	})

	t.Run("contains_go_vet_or_golangci_lint", func(t *testing.T) {
		hasVet := strings.Contains(content, "go vet")
		hasLint := strings.Contains(content, "golangci-lint")
		assert.True(t, hasVet || hasLint,
			"ci.yml must contain 'go vet' or 'golangci-lint'")
	})

	t.Run("contains_pull_request_trigger", func(t *testing.T) {
		assert.True(t, strings.Contains(content, "pull_request"),
			"ci.yml must contain 'pull_request' trigger")
	})

	t.Run("go_build_exits_zero", func(t *testing.T) {
		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = root
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "go build ./... must exit 0: %s", out)
	})
}
