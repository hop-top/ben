// Package unit_test asserts kit-conventions invariants that must hold
// for every release. The conventions trip-wire below is the long-lived
// guard against future drift: if any new leaf command lands without
// kit/side-effect and kit/idempotent annotations (§3.5 + §8.5), this
// test fails. It mirrors the e2e `ben spec` assertion but lives under
// tests/unit so it runs in the fast tier (no separate e2e harness
// required).
package unit_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manifestCmd mirrors the fields of toolspec.Manifest we assert on.
// Kept narrow on purpose so the test doesn't break when kit grows
// optional fields.
type manifestCmd struct {
	Path       []string `json:"path"`
	SideEffect string   `json:"side_effect"`
	Idempotent string   `json:"idempotent"`
	Hidden     bool     `json:"hidden,omitempty"`
}

type benManifest struct {
	Tool          string        `json:"tool"`
	SchemaVersion string        `json:"schema_version"`
	Commands      []manifestCmd `json:"commands"`
}

// buildBenForConventions compiles ben once per test invocation. We
// don't reuse the e2e helper because tests/unit must not depend on
// tests/e2e packages.
func buildBenForConventions(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "ben")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", bin, "hop.top/ben/cmd/ben")
	cmd.Dir = filepath.Join("..", "..")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)
	return bin
}

// TestConventions_EveryLeafIsAnnotated is the trip-wire that protects
// the kit-conventions audit (track: chore-kit-upgrade-cli-conventions).
// The capability manifest enumerates every leaf cobra command; we
// require each one to declare BOTH kit/side-effect and kit/idempotent
// in a value the conventions spec recognises (§3.5 + §8.5). Any new
// leaf landing without these tags fails this test.
func TestConventions_EveryLeafIsAnnotated(t *testing.T) {
	ben := buildBenForConventions(t)

	out, err := exec.Command(ben, "spec", "--format", "json").Output()
	require.NoError(t, err, "ben spec failed: %s", out)

	var m benManifest
	require.NoError(t, json.Unmarshal(out, &m),
		"spec output is not valid JSON: %s", out)
	require.Equal(t, "ben", m.Tool)
	require.NotEmpty(t, m.Commands, "manifest should list at least one command")

	validSideEffect := map[string]bool{
		"read":        true,
		"write":       true,
		"destructive": true,
		"interactive": true,
	}
	validIdempotent := map[string]bool{
		"yes":         true,
		"no":          true,
		"conditional": true,
	}

	for _, c := range m.Commands {
		path := strings.Join(c.Path, " ")
		assert.Truef(t, validSideEffect[c.SideEffect],
			"%s: kit/side-effect must be one of read|write|destructive|interactive (got %q)",
			path, c.SideEffect)
		assert.Truef(t, validIdempotent[c.Idempotent],
			"%s: kit/idempotent must be one of yes|no|conditional (got %q)",
			path, c.Idempotent)
	}
}

// TestConventions_SchemaVersion_Bumped guards the §13.2 evolution
// contract: ben spec --version must report the current schema and
// short-circuit (no commands array). Useful for agents capability-
// negotiating before issuing other commands.
func TestConventions_SchemaVersion_Bumped(t *testing.T) {
	ben := buildBenForConventions(t)

	out, err := exec.Command(ben, "spec", "--version").Output()
	require.NoError(t, err, "ben spec --version failed: %s", out)

	var v map[string]any
	require.NoError(t, json.Unmarshal(out, &v))
	assert.Equal(t, "1.1", v["schema_version"],
		"schema_version must be bumped when surface changes (kit/console/cli §13.2)")
	_, hasCommands := v["commands"]
	assert.False(t, hasCommands, "--version response should omit commands array")
}
