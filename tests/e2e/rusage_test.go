package e2e_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand_RusageMetrics(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("rusage assertions need linux or darwin")
	}
	ben := buildBen(t)
	burn := filepath.Join(t.TempDir(), "burn")
	out, err := exec.Command("go", "build", "-buildvcs=false", "-o", burn,
		"hop.top/ben/tests/unit/testdata/burn").CombinedOutput()
	require.NoError(t, err, "build burn: %s", out)

	suite := filepath.Join(t.TempDir(), "suite.yaml")
	yamlSrc := "name: rusage\nversion: 1\ncandidates:\n" +
		"  - {name: shell, adapter: cli, cmd: '" + burn + "'}\n" +
		"  - {name: direct, adapter: cli, cmd: '" + burn + "', args: ['a b']}\n" +
		"metrics: [cpu_user_ms, cpu_sys_ms, cpu_ms, max_rss_kb]\n" +
		"scorer: {strategy: 'single:cpu_ms'}\n"
	require.NoError(t, os.WriteFile(suite, []byte(yamlSrc), 0o644))

	cmd := exec.Command(ben, "run", "--suite", suite, "--format", "json")
	cmd.Env = append(os.Environ(), "XDG_DATA_HOME="+t.TempDir())
	raw, err := cmd.Output()
	require.NoError(t, err, "%s", raw)

	var res struct {
		Candidates []struct {
			Name    string             `json:"name"`
			Metrics map[string]float64 `json:"metrics"`
		} `json:"candidates"`
	}
	require.NoError(t, json.Unmarshal(raw, &res))
	require.Len(t, res.Candidates, 2)
	for _, c := range res.Candidates {
		assert.GreaterOrEqual(t, c.Metrics["cpu_ms"], 30.0, c.Name)
		assert.GreaterOrEqual(t, c.Metrics["max_rss_kb"], 15000.0, c.Name)
		assert.Contains(t, c.Metrics, "cpu_user_ms")
		assert.Contains(t, c.Metrics, "cpu_sys_ms")
	}
}
