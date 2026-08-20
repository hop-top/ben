package unit_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/scorer"
	"hop.top/ben/internal/spec"
)

// TestShippedSuites_LoadAndParse guards the three suites the repo ships:
// each must load, carry candidates, and declare a scorer strategy that
// scorer.Parse accepts. This is the test that would have caught the
// "strategy: single" (metric-less) shape the suites shipped with.
func TestShippedSuites_LoadAndParse(t *testing.T) {
	for _, name := range []string{"gsm8k.yaml", "mmlu.yaml", "hellaswag.yaml"} {
		t.Run(name, func(t *testing.T) {
			s, err := spec.Load(filepath.Join("..", "..", "suites", name))
			require.NoError(t, err, "suite must load")
			assert.NotEmpty(t, s.Candidates, "suite must declare candidates")

			sc, err := scorer.Parse(s.Scorer.Strategy, s.Scorer.Weights)
			require.NoError(t, err, "scorer strategy must parse: %q", s.Scorer.Strategy)
			assert.NotNil(t, sc)

			// The scored metric must be one the suite collects.
			if m, ok := scorer.SingleMetric(s.Scorer.Strategy); ok {
				assert.Contains(t, s.Metrics, m,
					"scored metric %q must be in the suite's metrics list", m)
			}
		})
	}
}
