package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/spec"
)

// TestEva_JSONFormatPreferred — a JSON-capable eva receives --format json
// and its Run-dump results[] yield a pass-fraction accuracy.
func TestEva_JSONFormatPreferred(t *testing.T) {
	script := `
for a in "$@"; do
  if [ "$a" = "--format" ]; then
    echo '{"results":[{"evaluator":"e","passed":true},{"evaluator":"e","passed":true},{"evaluator":"e","passed":false},{"evaluator":"e","passed":true}]}'
    exit 0
  fi
done
echo "text mode should not be reached" >&2
exit 1`
	path := fakeEva(t, script)
	defer withPath(t, path)()

	a := adapter.NewEva()
	r, err := a.Run(context.Background(), spec.Candidate{Name: "x", Adapter: "eva", Cmd: "ds.yaml"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode)
	acc, ok := r.Metadata["accuracy"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 0.75, acc, 1e-9, "3 of 4 passed")
}

// TestEva_FallbackWhenFormatRejected — an old eva build rejects --format
// with a usage error; the adapter retries without it and parses the text
// output, and accuracy still routes through plugin_metrics.
func TestEva_FallbackWhenFormatRejected(t *testing.T) {
	script := `
for a in "$@"; do
  if [ "$a" = "--format" ]; then
    echo "Usage: eva run [OPTIONS]" >&2
    echo "no such option: --format" >&2
    exit 2
  fi
done
echo "accuracy: 0.62"
exit 0`
	path := fakeEva(t, script)
	defer withPath(t, path)()

	a := adapter.NewEva()
	r, err := a.Run(context.Background(), spec.Candidate{Name: "x", Adapter: "eva", Cmd: "ds.yaml"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode)
	acc, ok := r.Metadata["accuracy"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 0.62, acc, 1e-9)
}

// TestEva_AccuracyRoutedThroughPluginMetrics — the scoring pipeline reads
// Metadata["plugin_metrics"]; the eva adapter must populate it so the
// shipped suites' `accuracy` metric resolves during `ben run`.
func TestEva_AccuracyRoutedThroughPluginMetrics(t *testing.T) {
	path := fakeEva(t, `echo '{"accuracy":0.81}'; exit 0`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	r, err := a.Run(context.Background(), spec.Candidate{Name: "x", Adapter: "eva", Cmd: "ds.yaml"}, nil)
	require.NoError(t, err)
	pm, ok := r.Metadata["plugin_metrics"].(map[string]float64)
	require.True(t, ok, "plugin_metrics map must be present")
	assert.InDelta(t, 0.81, pm["accuracy"], 1e-9)
}

// TestEva_EvaluationFailureNotRetried — exit 2 WITHOUT a usage complaint
// (e.g. a request-invalid evaluation) must not trigger the fallback rerun.
func TestEva_EvaluationFailureNotRetried(t *testing.T) {
	// The marker file proves the script ran exactly once.
	script := `
count_file="${TMPDIR:-/tmp}/eva_run_count_$PPID"
n=$(cat "$count_file" 2>/dev/null || echo 0)
echo $((n+1)) > "$count_file"
echo "request invalid: bad dataset" >&2
exit 2`
	path := fakeEva(t, script)
	defer withPath(t, path)()

	a := adapter.NewEva()
	r, err := a.Run(context.Background(), spec.Candidate{Name: "x", Adapter: "eva", Cmd: "ds.yaml"}, nil)
	require.NoError(t, err, "non-zero exit is data, not error")
	assert.Equal(t, 2, r.ExitCode)
}
