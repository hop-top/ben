package adapter

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"hop.top/ben/internal/spec"
)

// CLI implements Adapter by running a shell command defined in the candidate.
type CLI struct{}

// NewCLI returns a new CLI adapter.
func NewCLI() *CLI { return &CLI{} }

// Run expands {{input.*}} in candidate.Cmd, executes it via sh -c, captures
// combined stdout+stderr, exit code, and wall-clock duration.
// A non-zero exit code is NOT returned as an error — it goes into Result.ExitCode.
func (a *CLI) Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error) {
	expanded := spec.Template(c.Cmd, input)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "sh", "-c", expanded)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start)

	durationMs := elapsed.Milliseconds()

	result := &Result{
		DurationMs: durationMs,
	}

	// Combine stdout + stderr as output, stdout first.
	var out strings.Builder
	out.Write(stdout.Bytes())
	if stderr.Len() > 0 {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.Write(stderr.Bytes())
	}
	result.Output = out.String()

	if runErr != nil {
		if ctx.Err() != nil {
			// Context cancellation/deadline is an infrastructure error.
			return result, ctx.Err()
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		// Command not found or other exec error.
		return result, runErr
	}

	result.ExitCode = 0
	return result, nil
}
