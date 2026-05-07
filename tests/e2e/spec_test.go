package e2e_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manifestEntry mirrors the toolspec.Manifest entry shape we care
// about. Kept narrow so the test isn't coupled to fields ben doesn't
// populate.
type manifestEntry struct {
	Path       []string `json:"path"`
	SideEffect string   `json:"side_effect"`
	Idempotent string   `json:"idempotent"`
}

type manifest struct {
	Tool          string          `json:"tool"`
	SchemaVersion string          `json:"schema_version"`
	Commands      []manifestEntry `json:"commands"`
}

// TestSpecManifest_EveryLeafIsAnnotated verifies the kit conventions
// guarantee from §3.5 + §8.5: every leaf command exposes a
// kit/side-effect and kit/idempotent annotation. The capability
// manifest is the canonical surface, so we assert against `ben spec`
// directly rather than walking the cobra tree in-process — this also
// exercises the spec command end-to-end.
func TestSpecManifest_EveryLeafIsAnnotated(t *testing.T) {
	ben := buildBen(t)

	out, err := exec.Command(ben, "spec", "--format", "json").Output()
	require.NoError(t, err, "ben spec failed: %s", out)

	var m manifest
	require.NoError(t, json.Unmarshal(out, &m), "spec output is not valid JSON: %s", out)
	assert.Equal(t, "ben", m.Tool)
	assert.Equal(t, "1.0", m.SchemaVersion)
	require.NotEmpty(t, m.Commands, "manifest should list at least one command")

	validSideEffect := map[string]bool{"read": true, "write": true, "destructive": true, "interactive": true}
	validIdempotent := map[string]bool{"yes": true, "no": true, "conditional": true}

	for _, c := range m.Commands {
		path := strings.Join(c.Path, " ")
		assert.True(t, validSideEffect[c.SideEffect],
			"%s: kit/side-effect must be one of read|write|destructive|interactive (got %q)", path, c.SideEffect)
		assert.True(t, validIdempotent[c.Idempotent],
			"%s: kit/idempotent must be one of yes|no|conditional (got %q)", path, c.Idempotent)
	}
}

// TestSpecVersion_FastPath verifies `ben spec --version` short-circuits
// to the schema_version envelope without building the full manifest.
// Agents use this for capability negotiation before issuing other
// commands.
func TestSpecVersion_FastPath(t *testing.T) {
	ben := buildBen(t)

	out, err := exec.Command(ben, "spec", "--version").Output()
	require.NoError(t, err, "ben spec --version failed: %s", out)

	var v map[string]any
	require.NoError(t, json.Unmarshal(out, &v))
	assert.Equal(t, "1.0", v["schema_version"])
	// Short-circuit response should not include the commands array.
	_, hasCommands := v["commands"]
	assert.False(t, hasCommands, "--version response should omit commands array")
}
