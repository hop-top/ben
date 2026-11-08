package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os"
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
//
// When candidate.Args is set (argv form) the program in Cmd is executed
// directly with those arguments and no shell is involved; each arg is
// template-expanded but never word-split.
//
// If candidate.Prepare is set it runs first (untimed, via sh -c) and a failure
// is returned as an error. If candidate.InputFile is set its contents are the
// command's stdin.
//
// CPU time and peak RSS come from the child's rusage after wait. Through a
// shell they cover the shell plus any descendants it waited for; in argv
// form they cover only the target (and its waited descendants). Descendants
// that are not waited (daemonised or background processes) are not counted.
// A non-zero exit code is NOT returned as an error — it goes into Result.ExitCode.
func (a *CLI) Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error) {
	expanded := spec.Template(c.Cmd, input)

	// Untimed prepare step, before every run.
	if c.Prepare != "" {
		prep := exec.CommandContext(ctx, "sh", "-c", spec.Template(c.Prepare, input))
		if out, err := prep.CombinedOutput(); err != nil {
			if ctx.Err() != nil {
				return &Result{}, ctx.Err()
			}
			return &Result{}, fmt.Errorf("prepare for %q failed: %w: %s", c.Name, err, strings.TrimSpace(string(out)))
		}
	}

	var stdout, stderr bytes.Buffer
	var cmd *exec.Cmd
	if c.Args != nil {
		args := make([]string, len(c.Args))
		for i, a := range c.Args {
			args[i] = spec.Template(a, input)
		}
		cmd = exec.CommandContext(ctx, expanded, args...)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", expanded)
	}
	if c.InputFile != "" {
		f, err := os.Open(spec.Template(c.InputFile, input))
		if err != nil {
			return &Result{}, fmt.Errorf("input_file for %q: %w", c.Name, err)
		}
		defer func() { _ = f.Close() }()
		cmd.Stdin = f
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start)

	durationMs := elapsed.Milliseconds()

	result := &Result{
		DurationMs: durationMs,
		Duration:   elapsed,
	}
	applyUsage(result, cmd.ProcessState)

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
