// Package stories contains e2e tests that validate named user stories.
package stories_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUS_BEN_0403_ReporterPluginProtocol validates the reporter plugin stdio
// JSON protocol (US-BEN-0403).
//
// ACs tested:
//  1. Compile minimal reporter binary and place as "ben-reporter-stub" on PATH.
//  2. `ben run --format stub` exits 0.
//  3. stdout contains the reporter plugin's output (not ben's default table).
//  4. Remove binary; re-run --format stub → exits non-zero.
//  5. stderr from the failing run contains "not found" or a reporter error.
func TestUS_BEN_0403_ReporterPluginProtocol(t *testing.T) {
	ben := buildBen(t)
	binDir := t.TempDir()

	// Locate the existing json2 testdata as the stub source.
	// buildBen uses cmd.Dir = "../../.." (repo root); mirror that here.
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	stubSrc := filepath.Join(repoRoot, "tests", "e2e", "testdata", "ben-reporter-json2")

	stubName := "ben-reporter-stub"
	if runtime.GOOS == "windows" {
		stubName += ".exe"
	}
	stubBin := filepath.Join(binDir, stubName)

	// Build the stub binary.
	buildCmd := exec.Command("go", "build", "-o", stubBin, ".")
	buildCmd.Dir = stubSrc
	buildOut, buildErr := buildCmd.CombinedOutput()
	require.NoError(t, buildErr, "build stub failed: %s", buildOut)

	origPath := os.Getenv("PATH")
	augmentedPath := binDir + string(os.PathListSeparator) + origPath

	runArgs := []string{
		"run",
		"--task", "echo hello",
		"--candidates", "a=cli=echo a",
		"--metric", "latency_ms",
		"--scorer", "raw",
		"--format", "stub",
	}

	// AC2 + AC3: binary present → exit 0, stdout contains plugin output.
	t.Run("binary_present_exits_0", func(t *testing.T) {
		cmd := exec.Command(ben, runArgs...)
		cmd.Env = append(os.Environ(),
			"XDG_DATA_HOME="+t.TempDir(),
			"PATH="+augmentedPath,
		)
		out, err := cmd.Output()
		require.NoError(t, err, "ben run --format stub failed; stderr: %s", func() string {
			if ee, ok := err.(*exec.ExitError); ok {
				return string(ee.Stderr)
			}
			return ""
		}())

		// AC3: plugin writes "PLUGIN_REPORTER_OK"; must not be ben's default table.
		assert.Contains(t, string(out), "PLUGIN_REPORTER_OK",
			"expected reporter plugin sentinel in stdout")
		assert.NotContains(t, string(out), "Rank",
			"expected no default table output when plugin reporter is active")
	})

	// AC4 + AC5: remove binary → exit non-zero, stderr has error message.
	t.Run("binary_absent_exits_nonzero", func(t *testing.T) {
		require.NoError(t, os.Remove(stubBin), "remove stub binary")

		cmd := exec.Command(ben, runArgs...)
		cmd.Env = append(os.Environ(),
			"XDG_DATA_HOME="+t.TempDir(),
			"PATH="+augmentedPath,
		)
		out, runErr := cmd.Output()
		require.Error(t, runErr, "expected ben run to fail when reporter binary is absent; stdout: %s", out)

		var stderr string
		if ee, ok := runErr.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}

		// AC5: error message must reference the missing reporter.
		combined := stderr + string(out)
		hasNotFound := assert.True(t,
			containsAny(combined, "not found", "stub", "reporter"),
			"expected error referencing missing reporter; got stderr=%q stdout=%q", stderr, out,
		)
		_ = hasNotFound
	})
}

// containsAny returns true if s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 {
			found := false
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					found = true
					break
				}
			}
			if found {
				return true
			}
		}
	}
	return false
}
