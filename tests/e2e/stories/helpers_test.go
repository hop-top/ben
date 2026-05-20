package stories_test

import (
	"encoding/json"
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

// unwrapData reaches into kit's §6.6 provenance envelope and returns the
// raw `data` payload. Commands like `ben list` and `ben show` now wrap
// their JSON output as {"data": ..., "_meta": {...}} via
// output.WithProvenance. Story tests that pre-date the envelope call
// unwrapData on stdout before unmarshalling the payload, so the meta
// channel stays out of the data assertions.
//
// If body has no _meta key the function returns body unchanged, so it's
// safe to wrap every JSON-output call.
func unwrapData(t *testing.T, body []byte) []byte {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
		Meta json.RawMessage `json:"_meta"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Meta) > 0 {
		return []byte(envelope.Data)
	}
	return body
}
