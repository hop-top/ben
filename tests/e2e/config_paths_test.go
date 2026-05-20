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

// runConfigPaths invokes `ben config paths --format json` from cwd with
// the given KIT_INVOKED_AS value (empty string = unset) and returns the
// decoded chain keyed by scope.
func runConfigPaths(t *testing.T, invokedAs string) (cwd, xdgConfig string, byScope map[string]resolvedPath) {
	t.Helper()
	ben := buildBen(t)

	// macOS resolves /var/folders/... to /private/var/folders/... via a
	// symlink; the binary's filepath.Join(cwd, ...) returns the resolved
	// form while t.TempDir() returns the unresolved one. Normalize both
	// sides up front so equality holds.
	var err error
	cwd, err = filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	xdgConfig, err = filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	cmd := exec.Command(ben, "config", "paths", "--format", "json", "--no-hints", "--quiet")
	cmd.Dir = cwd
	env := append(os.Environ(),
		"XDG_CONFIG_HOME="+xdgConfig,
		"HOME="+t.TempDir(),
	)
	if invokedAs != "" {
		env = append(env, "KIT_INVOKED_AS="+invokedAs)
	}
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	require.NoError(t, err, "ben config paths failed: stderr=%s", stderr.String())

	var chain []resolvedPath
	require.NoError(t, json.Unmarshal(out, &chain), "raw stdout: %s", out)

	byScope = map[string]resolvedPath{}
	for _, p := range chain {
		byScope[p.Scope] = p
	}
	return cwd, xdgConfig, byScope
}

// TestConfigPaths_Standalone asserts that `ben config paths --format
// json` returns exactly the project/user/system paths agreed on for the
// .ben/config.yaml convention (Copilot review on PR #1, T-0088).
//
// project: ./.ben/config.yaml
// user:    $XDG_CONFIG_HOME/ben/config.yaml
// system:  /etc/ben/config.yaml
func TestConfigPaths_Standalone(t *testing.T) {
	cwd, xdgConfig, byScope := runConfigPaths(t, "")

	proj, ok := byScope["project"]
	require.True(t, ok, "project entry missing")
	assert.Equal(t, filepath.Join(cwd, ".ben", "config.yaml"), proj.Path)

	user, ok := byScope["user"]
	require.True(t, ok, "user entry missing")
	assert.Equal(t, filepath.Join(xdgConfig, "ben", "config.yaml"), user.Path)

	sys, ok := byScope["system"]
	require.True(t, ok, "system entry missing")
	assert.Equal(t, "/etc/ben/config.yaml", sys.Path)
}

// TestConfigPaths_InvokedUnderHop asserts the project layer switches to
// .hop/ben.yaml when KIT_INVOKED_AS=hop is set by the caller (T-0091).
func TestConfigPaths_InvokedUnderHop(t *testing.T) {
	cwd, _, byScope := runConfigPaths(t, "hop")
	proj, ok := byScope["project"]
	require.True(t, ok, "project entry missing")
	assert.Equal(t, filepath.Join(cwd, ".hop", "ben.yaml"), proj.Path)
}

// TestConfigPaths_InvokedUnderTlc asserts the project layer switches to
// .tlc/ben.yaml when KIT_INVOKED_AS=tlc is set by the caller (T-0091).
func TestConfigPaths_InvokedUnderTlc(t *testing.T) {
	cwd, _, byScope := runConfigPaths(t, "tlc")
	proj, ok := byScope["project"]
	require.True(t, ok, "project entry missing")
	assert.Equal(t, filepath.Join(cwd, ".tlc", "ben.yaml"), proj.Path)
}

// TestConfigPaths_UnknownInvokedAs falls back to the standalone .ben/
// path for any KIT_INVOKED_AS value not in the known set.
func TestConfigPaths_UnknownInvokedAs(t *testing.T) {
	cwd, _, byScope := runConfigPaths(t, "unknown-tool")
	proj, ok := byScope["project"]
	require.True(t, ok, "project entry missing")
	assert.Equal(t, filepath.Join(cwd, ".ben", "config.yaml"), proj.Path)
}
