package stories_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// buildBen compiles the ben binary into t.TempDir() and returns its path.
func buildBen(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "ben")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", bin, "hop.top/ben/cmd/ben")
	cmd.Dir = filepath.Join("..", "..", "..")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)
	return bin
}
