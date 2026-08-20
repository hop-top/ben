package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
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

// Run expands {{input.*}} in c.Cmd and runs eva against the dataset,
// preferring machine-readable output: it first invokes
// `eva run --dataset <cmd> --no-tui --format json`; eva builds that
// predate --format on dataset runs reject the flag with a usage error,
// in which case Run retries without it and emits a one-line deprecation
// note on stderr (the regex text-scrape era ends when those builds do).
// Accuracy lands in Metadata["accuracy"] AND Metadata["plugin_metrics"]
// so the run pipeline scores it like any plugin-reported metric.
// A non-zero exit code is NOT returned as an error — contract matches CLI adapter.
func (a *EvaAdapter) Run(ctx context.Context, c spec.Candidate, input map[string]string) (*Result, error) {
	dataset := spec.Template(c.Cmd, input)

	args := []string{"run", "--dataset", dataset, "--no-tui"}
	if c.Model != "" {
		args = append(args, "--target", c.Model)
	}

	var stdout, stderr bytes.Buffer
	runOnce := func(extra ...string) error {
		stdout.Reset()
		stderr.Reset()
		cmd := exec.CommandContext(ctx, "eva", append(append([]string{}, args...), extra...)...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		return cmd.Run()
	}

	start := time.Now()
	runErr := runOnce("--format", "json")
	if runErr != nil && ctx.Err() == nil && looksLikeUsageRejection(runErr, stderr.Bytes(), stdout.Bytes()) {
		fmt.Fprintln(os.Stderr, "ben: eva build predates --format json on dataset runs; falling back to text parsing (deprecated)")
		runErr = runOnce()
	}
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
	// Route accuracy through the plugin-metrics channel so `ben run`
	// scores it like any adapter-reported metric (same path binary
	// plugin adapters use).
	if acc, ok := result.Metadata["accuracy"].(float64); ok {
		result.Metadata["plugin_metric_accuracy"] = acc
		result.Metadata["plugin_metrics"] = map[string]float64{"accuracy": acc}
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

// looksLikeUsageRejection reports whether a failed eva invocation reads
// as "unknown flag" rather than an evaluation failure — the retry-without
// --format trigger. Typer/click builds print a usage error mentioning the
// offending option on stderr (sometimes stdout) and exit 2.
func looksLikeUsageRejection(runErr error, streams ...[]byte) bool {
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 2 {
		return false
	}
	for _, b := range streams {
		lower := strings.ToLower(string(b))
		if strings.Contains(lower, "--format") &&
			(strings.Contains(lower, "no such option") ||
				strings.Contains(lower, "unrecognized") ||
				strings.Contains(lower, "unexpected") ||
				strings.Contains(lower, "usage")) {
			return true
		}
	}
	return false
}

// parseAccuracy scans eva stdout for an accuracy value.
// Tries JSON first: a top-level "accuracy" field wins; failing that, a
// results[] list of {passed: bool} objects yields the pass fraction
// (the shape of eva's Run dump). Falls back to a text scan for
// "accuracy: <float>".
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
		if acc, ok := passFraction(m); ok {
			return acc, true
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

// passFraction derives accuracy from an eva Run dump: a "results" list
// whose entries carry a boolean "passed". Returns false when the shape
// does not match or the list is empty.
func passFraction(m map[string]any) (float64, bool) {
	raw, ok := m["results"]
	if !ok {
		return 0, false
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return 0, false
	}
	total, passed := 0, 0
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			return 0, false
		}
		p, ok := entry["passed"].(bool)
		if !ok {
			return 0, false
		}
		total++
		if p {
			passed++
		}
	}
	return float64(passed) / float64(total), true
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
