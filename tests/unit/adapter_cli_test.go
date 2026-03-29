package unit_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/spec"
)

func TestCLI_ExitZero(t *testing.T) {
	a := adapter.NewCLI()
	c := spec.Candidate{Name: "echo", Adapter: "cli", Cmd: "echo hello"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode)
	assert.Contains(t, r.Output, "hello")
	assert.GreaterOrEqual(t, r.DurationMs, int64(0))
}

func TestCLI_ExitOne(t *testing.T) {
	a := adapter.NewCLI()
	c := spec.Candidate{Name: "fail", Adapter: "cli", Cmd: "exit 1"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err, "non-zero exit is not an error")
	assert.Equal(t, 1, r.ExitCode)
}

func TestCLI_ExitNonZero(t *testing.T) {
	a := adapter.NewCLI()
	c := spec.Candidate{Name: "fail42", Adapter: "cli", Cmd: "exit 42"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err, "non-zero exit is not an error")
	assert.Equal(t, 42, r.ExitCode)
}

func TestCLI_TemplateExpansion(t *testing.T) {
	a := adapter.NewCLI()
	c := spec.Candidate{
		Name:    "tmpl",
		Adapter: "cli",
		Cmd:     "echo {{input.greeting}} {{input.name}}",
	}
	input := map[string]string{"greeting": "hello", "name": "world"}
	r, err := a.Run(context.Background(), c, input)
	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode)
	assert.Contains(t, r.Output, "hello world")
}

func TestCLI_ContextCancellation(t *testing.T) {
	a := adapter.NewCLI()
	c := spec.Candidate{Name: "sleep", Adapter: "cli", Cmd: "sleep 10"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := a.Run(ctx, c, nil)
	assert.Error(t, err, "context cancellation should return error")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCLI_OutputCapture(t *testing.T) {
	a := adapter.NewCLI()
	// stdout and stderr combined
	c := spec.Candidate{
		Name:    "out",
		Adapter: "cli",
		Cmd:     "echo stdout && echo stderr >&2",
	}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	assert.True(t, strings.Contains(r.Output, "stdout"))
	assert.True(t, strings.Contains(r.Output, "stderr"))
}

func TestCLI_DurationRecorded(t *testing.T) {
	a := adapter.NewCLI()
	c := spec.Candidate{Name: "dur", Adapter: "cli", Cmd: "sleep 0.05"}
	r, err := a.Run(context.Background(), c, nil)
	require.NoError(t, err)
	// At least ~50ms; allow generous upper bound for CI
	assert.GreaterOrEqual(t, r.DurationMs, int64(40))
}
