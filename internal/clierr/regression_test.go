package clierr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/ben/internal/clierr"
)

func TestRegression_PopulatesBenSpecificCode(t *testing.T) {
	err := clierr.Regression("accuracy moved -0.2 against direction max (threshold 0.05) (candidate \"c\")")
	assert.Equal(t, "BEN_REGRESSION", err.Code)
	assert.Equal(t, 4, err.ExitCode, "BEN_REGRESSION maps to the CONFLICT exit family")
	assert.Contains(t, err.Message, "accuracy")
	assert.NotEmpty(t, err.SuggestedFix)
}

func TestRegression_SatisfiesError(t *testing.T) {
	var _ error = clierr.Regression("x")
}
