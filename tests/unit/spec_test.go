package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/spec"
)

func TestLoad_Valid(t *testing.T) {
	s, err := spec.Load("testdata/valid.yaml")
	require.NoError(t, err)
	assert.Equal(t, "test-suite", s.Name)
	assert.Equal(t, 1, s.Version)
	assert.Len(t, s.Candidates, 2)
	assert.Equal(t, "xray", s.Candidates[0].Name)
	assert.Equal(t, "cli", s.Candidates[0].Adapter)
	assert.Equal(t, "weighted", s.Scorer.Strategy)
	assert.InDelta(t, 0.5, s.Scorer.Weights["latency_ms"], 1e-9)
	assert.Equal(t, "./testdata/sample-repo", s.Task.Input["repo"])
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := spec.Load("testdata/nonexistent.yaml")
	assert.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := spec.Load("testdata/invalid.yaml")
	assert.Error(t, err)
}

func TestLoad_MissingName(t *testing.T) {
	_, err := spec.Load("testdata/missing_name.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestLoad_MissingCandidates(t *testing.T) {
	_, err := spec.Load("testdata/missing_candidates.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "candidates")
}

func TestTemplate(t *testing.T) {
	tests := []struct {
		name  string
		tmpl  string
		input map[string]string
		want  string
	}{
		{
			name:  "single var",
			tmpl:  "xray --path {{input.repo}}",
			input: map[string]string{"repo": "/tmp/myrepo"},
			want:  "xray --path /tmp/myrepo",
		},
		{
			name:  "multiple vars",
			tmpl:  "search {{input.prompt}} in {{input.repo}}",
			input: map[string]string{"prompt": "handlers", "repo": "/src"},
			want:  "search handlers in /src",
		},
		{
			name:  "no vars",
			tmpl:  "echo hello",
			input: map[string]string{},
			want:  "echo hello",
		},
		{
			name:  "missing var stays",
			tmpl:  "cmd {{input.missing}}",
			input: map[string]string{},
			want:  "cmd {{input.missing}}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := spec.Template(tc.tmpl, tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFromFlags_ToSpec(t *testing.T) {
	f := spec.FromFlags{
		Task:       "find handlers",
		Candidates: []string{"xray,cli,xray explore", "grep,cli,grep -r Handler"},
		Metrics:    []string{"latency_ms", "exit_code"},
		Scorer:     "weighted:latency_ms=0.7,exit_code=0.3",
		Input:      map[string]string{"repo": "/src"},
	}
	s, err := f.ToSpec()
	require.NoError(t, err)
	assert.Equal(t, "inline", s.Name)
	assert.Equal(t, "find handlers", s.Task.Prompt)
	assert.Equal(t, "/src", s.Task.Input["repo"])
	assert.Len(t, s.Candidates, 2)
	assert.Equal(t, "xray", s.Candidates[0].Name)
	assert.Equal(t, "cli", s.Candidates[0].Adapter)
	assert.Equal(t, "xray explore", s.Candidates[0].Cmd)
	assert.Equal(t, "weighted", s.Scorer.Strategy)
	assert.InDelta(t, 0.7, s.Scorer.Weights["latency_ms"], 1e-9)
	assert.InDelta(t, 0.3, s.Scorer.Weights["exit_code"], 1e-9)
}

func TestFromFlags_ToSpec_DefaultAdapter(t *testing.T) {
	f := spec.FromFlags{
		Task:       "run something",
		Candidates: []string{"grep"},
	}
	s, err := f.ToSpec()
	require.NoError(t, err)
	assert.Equal(t, "grep", s.Candidates[0].Name)
	assert.Equal(t, "cli", s.Candidates[0].Adapter)
	assert.Equal(t, "grep", s.Candidates[0].Cmd)
}

func TestFromFlags_ToSpec_NoCandidates(t *testing.T) {
	f := spec.FromFlags{Task: "test"}
	_, err := f.ToSpec()
	assert.Error(t, err)
}
