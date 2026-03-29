package unit_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"hop.top/ben/internal/reporter"
	"hop.top/ben/internal/run"
)

func sampleRun() *run.Run {
	return &run.Run{
		RunID:     "01HX0000000000000000000000",
		Suite:     "test-suite",
		Timestamp: time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
		Scorer: run.ScorerConfig{
			Strategy: "single:latency_ms",
		},
		Candidates: []run.CandidateResult{
			{
				Name:    "fast",
				Metrics: map[string]float64{"latency_ms": 100, "exit_code": 0},
				Score:   floatPtr(0.0),
				Rank:    intPtr(1),
			},
			{
				Name:    "slow",
				Metrics: map[string]float64{"latency_ms": 500, "exit_code": 0},
				Score:   floatPtr(1.0),
				Rank:    intPtr(2),
			},
		},
		Winner: strPtr("fast"),
		Metadata: run.Metadata{
			Host:       "testhost",
			BenVersion: "0.1.0",
		},
	}
}

func TestJSONReporter_ValidJSON(t *testing.T) {
	r, err := reporter.New("json")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = r.Report(&buf, sampleRun())
	require.NoError(t, err)

	// Must parse as valid JSON.
	var obj map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &obj))

	// winner field present.
	assert.Equal(t, "fast", obj["winner"])

	// candidates array present with rank and score.
	cands, ok := obj["candidates"].([]any)
	require.True(t, ok)
	require.Len(t, cands, 2)

	first := cands[0].(map[string]any)
	assert.Equal(t, "fast", first["name"])
	assert.NotNil(t, first["rank"])
	assert.NotNil(t, first["score"])
}

func TestYAMLReporter_ValidYAML(t *testing.T) {
	r, err := reporter.New("yaml")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = r.Report(&buf, sampleRun())
	require.NoError(t, err)

	// Must parse back to run.Run.
	var parsed run.Run
	require.NoError(t, yaml.Unmarshal(buf.Bytes(), &parsed))
	require.NotNil(t, parsed.Winner)
	assert.Equal(t, "fast", *parsed.Winner)
	assert.Equal(t, "test-suite", parsed.Suite)
	require.Len(t, parsed.Candidates, 2)
}

func TestTableReporter_ContainsCandidateNames(t *testing.T) {
	r, err := reporter.New("table")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = r.Report(&buf, sampleRun())
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "fast")
	assert.Contains(t, out, "slow")
	// Rank column header present.
	assert.Contains(t, out, "Rank")
	// Name column present.
	assert.Contains(t, out, "Name")
}

func TestTableReporter_RankColumn(t *testing.T) {
	r, err := reporter.New("table")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = r.Report(&buf, sampleRun())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// At least header + 2 data rows.
	require.GreaterOrEqual(t, len(lines), 3)
	// Header must contain "Rank".
	assert.Contains(t, lines[0], "Rank")
}

func TestUnknownFormat_Error(t *testing.T) {
	_, err := reporter.New("xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "xml")
}
