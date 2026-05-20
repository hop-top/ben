package e2e_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resolvedPath mirrors kit's kitcliconfig.ResolvedPath JSON shape. We
// re-declare it locally so this test does not depend on the kit module's
// public test types.
type resolvedPath struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	Scope  string `json:"scope"`
	Exists bool   `json:"exists"`
}

// TestConfigPaths_NewConvention asserts that `ben config paths --format
// json` returns exactly the project/user/system paths agreed on for the
// .ben/config.yaml convention (Copilot review on PR #1, T-0088).
//
// project: ./.ben/config.yaml
// user:    $XDG_CONFIG_HOME/ben/config.yaml
// system:  /etc/ben/config.yaml
func TestConfigPaths_NewConvention(t *testing.T) {
	ben := buildBen(t)

	// macOS resolves /var/folders/... to /private/var/folders/... via a
	// symlink; the binary's filepath.Join(cwd, ...) returns the resolved
	// form while t.TempDir() returns the unresolved one. Normalize both
	// sides up front so equality holds.
	cwd, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	xdgConfig, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	cmd := exec.Command(ben, "config", "paths", "--format", "json", "--no-hints", "--quiet")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+xdgConfig,
		"HOME="+t.TempDir(),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	require.NoError(t, err, "ben config paths failed: stderr=%s", stderr.String())

	var chain []resolvedPath
	require.NoError(t, json.Unmarshal(out, &chain), "raw stdout: %s", out)

	byScope := map[string]resolvedPath{}
	for _, p := range chain {
		byScope[p.Scope] = p
	}

	proj, ok := byScope["project"]
	require.True(t, ok, "project entry missing; got %+v", chain)
	assert.Equal(t, filepath.Join(cwd, ".ben", "config.yaml"), proj.Path)

	user, ok := byScope["user"]
	require.True(t, ok, "user entry missing; got %+v", chain)
	assert.Equal(t, filepath.Join(xdgConfig, "ben", "config.yaml"), user.Path)

	sys, ok := byScope["system"]
	require.True(t, ok, "system entry missing; got %+v", chain)
	assert.Equal(t, "/etc/ben/config.yaml", sys.Path)

	// Sanity: no stale ben.yaml entries leak through.
	for _, p := range chain {
		assert.NotContains(t, p.Path, "ben.yaml", "stale ben.yaml path in chain: %s", p.Path)
	}
}
