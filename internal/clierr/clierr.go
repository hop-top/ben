// Package clierr exposes ben-specific structured error helpers that
// satisfy kit's AsCLIError() interface. Errors returned from a cobra
// RunE are picked up by kit's wrapRunE middleware (kit/cli §6.4) and
// rendered to stderr as an output.Error envelope when --format is
// json|yaml. Plain mode keeps a human-readable shape.
//
// kit ships the cross-tool codes (OK, GENERIC, USAGE, NOT_FOUND,
// CONFLICT, UNAUTHORIZED, ...) in hop.top/kit/go/console/output. This
// package adds the ben-specific subtype codes the conventions doc
// (§6.4 + §8.1) calls out:
//
//	BEN_NO_RUN   — referenced run-id not present in the local store.
//	             Maps to exit 3 (the NOT_FOUND family) so scripts can
//	             still grep "rc=3" the same way.
//	BEN_NO_SUITE — referenced suite name not found in any scan dir.
//	             Same exit mapping rationale.
//
// Adopters call NoRun / NoSuite from a RunE; everything else is the
// middleware's job.
package clierr

import (
	"fmt"

	"hop.top/kit/go/console/output"
)

// Ben-specific subtype codes. Suffix the §8.1 numeric exit code in the
// trailing comment to keep the table grep-able alongside kit's own
// CodeNotFound / CodeConflict constants.
const (
	CodeBenNoRun   = "BEN_NO_RUN"   // exit 3 — NOT_FOUND family
	CodeBenNoSuite = "BEN_NO_SUITE" // exit 3 — NOT_FOUND family
)

// NoRun returns an *output.Error for a missing run-id. The Code field
// uses the ben-specific subtype so agents can disambiguate "no such
// run" from generic NOT_FOUND noise, while the ExitCode stays 3 so
// shell scripts can keep grepping the cross-tool table.
func NoRun(runID string) *output.Error {
	return &output.Error{
		Code:         CodeBenNoRun,
		Message:      fmt.Sprintf("run %q not found", runID),
		SuggestedFix: "list available runs with `ben list`",
		ExitCode:     3,
	}
}

// NoSuite returns an *output.Error for a missing suite. Same mapping
// rationale as NoRun.
func NoSuite(suite string) *output.Error {
	return &output.Error{
		Code:         CodeBenNoSuite,
		Message:      fmt.Sprintf("suite %q not found", suite),
		SuggestedFix: "list available suites with `ben suite list`",
		ExitCode:     3,
	}
}
