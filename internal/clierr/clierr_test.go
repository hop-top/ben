package clierr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/ben/internal/clierr"
)

func TestNoRun_PopulatesBenSpecificCode(t *testing.T) {
	err := clierr.NoRun("01ABCD")
	assert.Equal(t, "BEN_NO_RUN", err.Code)
	assert.Equal(t, 3, err.ExitCode, "BEN_NO_RUN maps to the NOT_FOUND exit family (§8.1)")
	assert.Contains(t, err.Message, "01ABCD")
	assert.NotEmpty(t, err.SuggestedFix, "users get a discovery hint")
}

func TestNoSuite_PopulatesBenSpecificCode(t *testing.T) {
	err := clierr.NoSuite("smoke")
	assert.Equal(t, "BEN_NO_SUITE", err.Code)
	assert.Equal(t, 3, err.ExitCode)
	assert.Contains(t, err.Message, "smoke")
	assert.NotEmpty(t, err.SuggestedFix)
}

// TestErrors_SatisfyError ensures the helpers can be returned from a
// RunE without a separate AsCLIError shim — kit's wrapRunE middleware
// detects *output.Error directly via errors.As.
func TestErrors_SatisfyError(t *testing.T) {
	var _ error = clierr.NoRun("x")
	var _ error = clierr.NoSuite("x")
}
