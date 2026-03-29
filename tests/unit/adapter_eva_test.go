package unit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/spec"
)

// fakeEva writes a shell script named "eva" into dir and returns the updated PATH
// that prepends dir so the script shadows any real eva.
func fakeEva(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "eva")
	err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o755)
	require.NoError(t, err)
	return dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

func withPath(t *testing.T, p string) func() {
	t.Helper()
	orig := os.Getenv("PATH")
	require.NoError(t, os.Setenv("PATH", p))
	return func() { _ = os.Setenv("PATH", orig) }
}

func TestEva_SuccessExitZero(t *testing.T) {
	path := fakeEva(t, `echo '{"accuracy":0.75}'; exit 0`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	c := spec.Candidate{Name: "mmlu", Adapter: "eva", Cmd: "some/dataset.yaml"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode)
	assert.Greater(t, r.DurationMs, int64(0))
	assert.NotEmpty(t, r.Output)
}

func TestEva_NonZeroExitNotError(t *testing.T) {
	path := fakeEva(t, `echo "something failed"; exit 1`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	c := spec.Candidate{Name: "fail", Adapter: "eva", Cmd: "dataset.yaml"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err, "non-zero exit must not be returned as error")
	assert.Equal(t, 1, r.ExitCode)
}

func TestEva_AccuracyFromJSON(t *testing.T) {
	path := fakeEva(t, `echo '{"accuracy":0.82,"total":100}'; exit 0`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	c := spec.Candidate{Name: "mmlu", Adapter: "eva", Cmd: "dataset.yaml"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	require.NotNil(t, r.Metadata)
	acc, ok := r.Metadata["accuracy"]
	require.True(t, ok, "accuracy key must be present")
	assert.InDelta(t, 0.82, acc.(float64), 1e-9)
}

func TestEva_AccuracyFromText(t *testing.T) {
	path := fakeEva(t, `echo "accuracy: 0.91"; exit 0`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	c := spec.Candidate{Name: "mmlu", Adapter: "eva", Cmd: "dataset.yaml"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	acc, ok := r.Metadata["accuracy"]
	require.True(t, ok)
	assert.InDelta(t, 0.91, acc.(float64), 1e-9)
}

func TestEva_AccuracyNotFound(t *testing.T) {
	path := fakeEva(t, `echo "no useful output here"; exit 0`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	c := spec.Candidate{Name: "mmlu", Adapter: "eva", Cmd: "dataset.yaml"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	acc, ok := r.Metadata["accuracy"]
	require.True(t, ok, "accuracy key must always be present")
	assert.Equal(t, 0.0, acc.(float64))
}

func TestEva_ModelPassedAsTarget(t *testing.T) {
	// Script prints args so we can verify --target is passed.
	path := fakeEva(t, `echo "args: $@"; exit 0`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	c := spec.Candidate{Name: "mmlu", Adapter: "eva", Cmd: "dataset.yaml", Model: "claude-sonnet-4-6"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	assert.Contains(t, r.Output, "--target")
	assert.Contains(t, r.Output, "claude-sonnet-4-6")
}

func TestEva_TemplateExpansion(t *testing.T) {
	path := fakeEva(t, `echo "args: $@"; exit 0`)
	defer withPath(t, path)()

	a := adapter.NewEva()
	c := spec.Candidate{Name: "mmlu", Adapter: "eva", Cmd: "{{input.dataset}}"}
	input := map[string]string{"dataset": "suites/testdata/mmlu-sample.yaml"}
	r, err := a.Run(context.Background(), c, input)
	require.NoError(t, err)
	assert.Contains(t, r.Output, "suites/testdata/mmlu-sample.yaml")
}
