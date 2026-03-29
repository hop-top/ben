package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"hop.top/ben/internal/spec"
)

// EvaAdapter implements Adapter by wrapping `eva run`.
// c.Cmd is the path to the eva dataset YAML (--dataset flag value).
// c.Model is passed as --target if non-empty.
type EvaAdapter struct{}

// NewEva returns a new EvaAdapter.
func NewEva() *EvaAdapter { return &EvaAdapter{} }

// Run expands {{input.*}} in c.Cmd, runs `eva run --dataset <cmd> --no-tui`
// (adding --target <model> if model is set), captures combined output and
// parses accuracy from the output into Metadata["accuracy"].
// A non-zero exit code is NOT returned as an error — contract matches CLI adapter.
func (a *EvaAdapter) Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error) {
	dataset := spec.Template(c.Cmd, input)

	args := []string{"run", "--dataset", dataset, "--no-tui"}
	if c.Model != "" {
		args = append(args, "--target", c.Model)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "eva", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start)

	result := &Result{
		DurationMs: elapsed.Milliseconds(),
		Metadata:   map[string]any{"accuracy": 0.0},
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

	// Parse accuracy — never error on parse failure.
	if acc, ok := parseAccuracy(stdout.Bytes()); ok {
		result.Metadata["accuracy"] = acc
	}

	if runErr != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		// Binary not found or exec error — infrastructure failure.
		return result, runErr
	}

	result.ExitCode = 0
	return result, nil
}

// parseAccuracy scans eva stdout for an accuracy value.
// Tries JSON first (field "accuracy"), then a text scan for "accuracy: <float>".
func parseAccuracy(data []byte) (float64, bool) {
	// Try JSON: look for {"accuracy": <float>} anywhere in the output.
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			break
		}
		if v, ok := m["accuracy"]; ok {
			if f, ok := toFloat64(v); ok {
				return f, true
			}
		}
	}

	// Try text scan: line containing "accuracy" followed by a number.
	for line := range strings.SplitSeq(string(data), "\n") {
		lower := strings.ToLower(line)
		idx := strings.Index(lower, "accuracy")
		if idx < 0 {
			continue
		}
		rest := lower[idx+len("accuracy"):]
		// Strip leading non-numeric chars (": ", "=", " ", etc.)
		rest = strings.TrimLeft(rest, ": =\t ")
		var f float64
		if n, err := parseFloat(rest); err == nil {
			return n, true
		} else {
			_ = f
		}
	}
	return 0, false
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		if f, err := n.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// parseFloat parses a leading float from s (stops at first non-numeric char).
func parseFloat(s string) (float64, error) {
	// Collect digits, dot, optional sign.
	end := 0
	for end < len(s) {
		ch := s[end]
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '+' || ch == 'e' || ch == 'E' {
			end++
		} else {
			break
		}
	}
	if end == 0 {
		return 0, &parseFloatError{s}
	}
	var f float64
	_, err := jsonUnmarshalFloat(s[:end], &f)
	return f, err
}

type parseFloatError struct{ s string }

func (e *parseFloatError) Error() string { return "parseFloat: no number in " + e.s }

func jsonUnmarshalFloat(s string, f *float64) (int, error) {
	return 0, json.Unmarshal([]byte(s), f)
}
