package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/run"
)

// writeConfig writes a minimal ben.yaml to cfgDir and returns the file path.
func writeConfig(t *testing.T, cfgDir string, registryURL string) string {
	t.Helper()
	content := "registry:\n  url: " + registryURL + "\n"
	path := filepath.Join(cfgDir, "ben.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestRegistryPush_HappyPath(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	// Produce a real run.
	runID := runCapturingID(t, ben, dataDir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"remote-push-001"}`))
	}))
	defer srv.Close()

	cfgDir := t.TempDir()
	cfgPath := writeConfig(t, cfgDir, srv.URL)

	cmd := exec.Command(ben, "registry", "push", runID, "--config", cfgPath)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := cmd.Output()
	require.NoError(t, err, "ben registry push failed: %s", out)
	assert.Contains(t, string(out), "pushed")
	assert.Contains(t, string(out), runID)
}

func TestRegistryPull_HappyPath(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	// Build a fake run to be returned by the mock server.
	fakeRun := &run.Run{
		RunID:        ulid.Make().String(),
		Suite:        "pull-e2e-suite",
		SuiteVersion: 1,
		Timestamp:    time.Now().UTC().Truncate(time.Second),
		Scorer:       run.ScorerConfig{Strategy: "raw"},
		Candidates: []run.CandidateResult{
			{Name: "cand-a", Metrics: map[string]float64{"latency_ms": 100}, Rank: 1},
		},
		Winner:   "",
		Metadata: run.Metadata{Host: "remote-host", BenVersion: "0.1.0"},
	}

	payload, err := json.Marshal(struct {
		Runs []*run.Run `json:"runs"`
	}{Runs: []*run.Run{fakeRun}})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	cfgDir := t.TempDir()
	cfgPath := writeConfig(t, cfgDir, srv.URL)

	cmd := exec.Command(ben, "registry", "pull", "--suite", "pull-e2e-suite", "--config", cfgPath)
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
	out, err := cmd.Output()
	require.NoError(t, err, "ben registry pull failed: %s", out)
	s := string(out)
	assert.Contains(t, s, "pulled")
	assert.True(t, strings.Contains(s, "pull-e2e-suite") || strings.Contains(s, "1 runs"))
}

func TestRegistryPush_NoConfig_ExitsOne(t *testing.T) {
	ben := buildBen(t)
	dataDir := t.TempDir()

	// Run to get a real run ID.
	runID := runCapturingID(t, ben, dataDir)

	// No config means no registry.url → should error.
	cmd := exec.Command(ben, "registry", "push", runID)
	cmd.Env = append(os.Environ(),
		"XDG_DATA_HOME="+dataDir,
		// Point config to nonexistent path so no default config is loaded.
		"HOME="+t.TempDir(),
	)
	err := cmd.Run()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
}
