// Package stories contains e2e tests that validate named user stories.
package stories_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUS_BEN_0404_TestSuiteGreenOnFirstRun validates that the test suite is
// hermetic and passes without any external dependencies (US-BEN-0404).
//
// ACs tested:
//  1. `go test ./...` from repo root exits 0.
//  2. `go test ./...` output contains no "FAIL" lines.
//  3. `go vet ./...` exits 0.
//  4. No test file in tests/unit/ or tests/e2e/ contains the forbidden service
//     dependency marker string.
//  5. `go build ./...` exits 0.
func TestUS_BEN_0404_TestSuiteGreenOnFirstRun(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	// AC4: static check — no test file must contain the service-dependency
	// marker. Stored as a joined literal so this file itself does not trigger
	// the check.
	forbidden := strings.Join([]string{"requires", "running", "service"}, " ")

	t.Run("no_service_dependency_string", func(t *testing.T) {
		dirs := []string{
			filepath.Join(repoRoot, "tests", "unit"),
			filepath.Join(repoRoot, "tests", "e2e"),
		}
		for _, dir := range dirs {
			walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, walkFnErr error) error {
				if walkFnErr != nil {
					return walkFnErr
				}
				if info.IsDir() || !strings.HasSuffix(info.Name(), "_test.go") {
					return nil
				}
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				assert.NotContains(t,
					string(data), forbidden,
					"test file %s contains forbidden service-dependency marker", path,
				)
				return nil
			})
			require.NoError(t, walkErr, "walking %s", dir)
		}
	})

	// AC5: go build ./... exits 0.
	t.Run("go_build", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "build", "./...")
		cmd.Dir = repoRoot
		out, buildErr := cmd.CombinedOutput()
		require.NoError(t, buildErr, "go build ./... failed:\n%s", out)
	})

	// AC3: go vet ./... exits 0.
	t.Run("go_vet", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "vet", "./...")
		cmd.Dir = repoRoot
		out, vetErr := cmd.CombinedOutput()
		require.NoError(t, vetErr, "go vet ./... failed:\n%s", out)
	})

	// AC1 + AC2: go test ./... exits 0 and produces no "FAIL" lines.
	// Use -skip to exclude this meta-test and avoid infinite recursion.
	t.Run("go_test_suite_green", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "go", "test", "./...",
			"-skip", "TestUS_BEN_0404",
			"-timeout", "3m",
		)
		cmd.Dir = repoRoot

		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		runErr := cmd.Run()
		output := buf.String()

		// AC1: exit code 0.
		require.NoError(t, runErr, "go test ./... exited non-zero:\n%s", output)

		// AC2: no FAIL lines.
		for _, line := range strings.Split(output, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "FAIL") {
				t.Errorf("go test ./... output contains FAIL line: %q", trimmed)
			}
		}
	})
}
