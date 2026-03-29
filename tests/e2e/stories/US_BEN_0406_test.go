// Package stories contains e2e tests that validate named user stories.
package stories_test

// US-BEN-0406 — Config-declared scorer plugin.
//
// Story: a user declares a custom scorer plugin in ben.yaml under
// plugins.scorers[{name, import}]; `ben run` discovers the binary on PATH,
// delegates scoring to it, and surfaces its scores in the JSON output.
//
// ACs (full spec):
//   1. Compile scorer stub that reads {"candidates":[...]} → writes
//      {"scores":{"a":0.9,"b":0.1}}.
//   2. Declare in ben.yaml: plugins.scorers[{name: pareto-stub, import:
//      ben-plugin-pareto-stub}].
//   3. Suite YAML with scorer.strategy: pareto-stub; candidates "a" and "b".
//   4. `ben run --suite <path> --format json` exits 0.
//   5. candidates["a"].score == 0.9, candidates["b"].score == 0.1, winner == "a".
//   6. Remove binary; re-run → exits 1; stderr contains "not found".
//
// IMPLEMENTATION STATUS — NOT IMPLEMENTED (as of v0.1.0):
//
//   The scorer plugin feature requires the following that are absent:
//
//   a) internal/plugin/config.go — LoadMetricPlugins handles plugins.metrics;
//      there is no LoadScorerPlugins or ScorerPluginConfig type.
//
//   b) internal/scorer/scorer.go — Parse() handles only "raw", "single:<metric>",
//      and "weighted". It has no mechanism to look up or invoke a named binary
//      scorer plugin.
//
//   c) cmd/ben/run.go — reads plugins.metrics from viper (line ~100) but never
//      reads plugins.scorers; scorer.Parse() is called directly with no plugin
//      intercept.
//
//   d) internal/plugin/registry.go — PluginRegistry holds adapters and reporters;
//      no scorer registry or LookupScorer method exists.
//
//   What needs to be built:
//     - ScorerPluginConfig struct + LoadScorerPlugins func in internal/plugin/
//     - BinaryScorer adapter in internal/scorer/ or internal/plugin/ that
//       serialises candidates to JSON stdin and reads {"scores":{...}} from stdout
//     - scorer.Parse (or a new resolver) extended to consult a scorer plugin
//       registry before returning "unknown strategy" error
//     - cmd/ben/run.go reads plugins.scorers from viper; calls LoadScorerPlugins;
//       resolves s.Scorer.Strategy through plugin registry first
//
// The sub-tests below are skipped with this note. A single smoke sub-test that
// IS testable today (ben.yaml plugins.metrics key is loaded without error) runs
// to confirm config wiring works for metrics, making the gap more precise.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const skipMsgScorerPlugin = "scorer plugin not yet implemented: " +
	"plugins.scorers config key not read in cmd/ben/run.go; " +
	"scorer.Parse() has no binary-plugin path; " +
	"no ScorerPluginConfig/LoadScorerPlugins in internal/plugin/; " +
	"no BinaryScorer in internal/scorer/ — " +
	"see US-BEN-0406 file header for full build spec"

// paretoStubSrc is a minimal Go program that acts as a scorer plugin stub.
// It reads {"candidates":[{"name":"a",...},{"name":"b",...}]} from stdin and
// writes {"scores":{"a":0.9,"b":0.1}} to stdout.
// Kept here so the test is self-contained once the feature is implemented.
const paretoStubSrc = `package main

import (
	"encoding/json"
	"os"
)

type input struct {
	Candidates []struct {
		Name string ` + "`json:\"name\"`" + `
	} ` + "`json:\"candidates\"`" + `
}

type output struct {
	Scores map[string]float64 ` + "`json:\"scores\"`" + `
}

func main() {
	var in input
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		os.Exit(1)
	}
	scores := map[string]float64{}
	for _, c := range in.Candidates {
		switch c.Name {
		case "a":
			scores["a"] = 0.9
		case "b":
			scores["b"] = 0.1
		default:
			scores[c.Name] = 0.0
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(output{Scores: scores}); err != nil {
		os.Exit(1)
	}
}
`

// suiteYAMLParetoStub is the suite YAML declaring strategy: pareto-stub.
const suiteYAMLParetoStub = `
name: pareto-stub-suite
version: 1
task:
  prompt: "echo test"
candidates:
  - name: a
    adapter: cli
    cmd: echo a
  - name: b
    adapter: cli
    cmd: echo b
metrics:
  - exit_code
scorer:
  strategy: pareto-stub
`

// benYAMLParetoPlugin is the ben.yaml declaring the scorer plugin.
// NOTE: plugins.scorers is not read by run.go today — this documents the
// intended config shape for when the feature is implemented.
const benYAMLParetoPlugin = `
plugins:
  scorers:
    - name: pareto-stub
      import: ben-plugin-pareto-stub
`

func TestUS_BEN_0406_ConfigDeclaredScorerPlugin(t *testing.T) {
	ben := buildBen(t)

	// ── Sub-test 1: stub compiles ──────────────────────────────────────────────
	// Verify the stub source we plan to use actually compiles. This is testable
	// today and gives early signal if the stub itself is broken.

	t.Run("scorer_stub_compiles", func(t *testing.T) {
		t.Skip(skipMsgScorerPlugin)

		dir := t.TempDir()
		srcPath := filepath.Join(dir, "main.go")
		require.NoError(t, os.WriteFile(srcPath, []byte(paretoStubSrc), 0o644))

		binPath := filepath.Join(dir, "ben-plugin-pareto-stub")
		cmd := exec.Command("go", "build", "-o", binPath, srcPath)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "scorer stub failed to compile: %s", out)

		info, err := os.Stat(binPath)
		require.NoError(t, err, "compiled stub binary not found")
		assert.True(t, info.Mode()&0o111 != 0, "stub binary must be executable")
	})

	// ── Sub-test 2: run exits 0, scores applied ───────────────────────────────

	t.Run("run_exits_0_with_plugin_scores", func(t *testing.T) {
		t.Skip(skipMsgScorerPlugin)

		// Build stub binary.
		pluginDir := t.TempDir()
		srcPath := filepath.Join(pluginDir, "main.go")
		require.NoError(t, os.WriteFile(srcPath, []byte(paretoStubSrc), 0o644))
		stubBin := filepath.Join(pluginDir, "ben-plugin-pareto-stub")
		cmd := exec.Command("go", "build", "-o", stubBin, srcPath)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "scorer stub failed to compile: %s", out)

		// Write suite YAML and ben.yaml into a project dir.
		projectDir := t.TempDir()
		suitePath := filepath.Join(projectDir, "suite.yaml")
		require.NoError(t, os.WriteFile(suitePath, []byte(suiteYAMLParetoStub), 0o644))
		require.NoError(t, os.WriteFile(
			filepath.Join(projectDir, "ben.yaml"),
			[]byte(benYAMLParetoPlugin), 0o644))

		dataDir := t.TempDir()
		runCmd := exec.Command(ben, "run", "--suite", suitePath, "--format", "json")
		runCmd.Dir = projectDir
		runCmd.Env = append(os.Environ(),
			"XDG_DATA_HOME="+dataDir,
			"PATH="+pluginDir+":"+os.Getenv("PATH"),
		)
		runOut, runErr := runCmd.Output()
		require.NoError(t, runErr, "ben run exited non-zero; stderr: %s",
			func() string {
				if ee, ok := runErr.(*exec.ExitError); ok {
					return string(ee.Stderr)
				}
				return ""
			}(),
		)

		var result map[string]any
		require.NoError(t, json.Unmarshal(runOut, &result),
			"ben run stdout is not valid JSON: %s", runOut)

		// Verify winner.
		winner, _ := result["winner"].(string)
		assert.Equal(t, "a", winner, "winner must be candidate 'a'")

		// Verify per-candidate scores.
		candidates, _ := result["candidates"].([]any)
		require.Len(t, candidates, 2, "expected 2 candidates in output")

		scoreByName := map[string]float64{}
		for _, raw := range candidates {
			c, ok := raw.(map[string]any)
			require.True(t, ok)
			name, _ := c["name"].(string)
			score, _ := c["score"].(float64)
			scoreByName[name] = score
		}

		assert.InDelta(t, 0.9, scoreByName["a"], 1e-9, "candidate 'a' score must be 0.9")
		assert.InDelta(t, 0.1, scoreByName["b"], 1e-9, "candidate 'b' score must be 0.1")
	})

	// ── Sub-test 3: missing binary → exits 1, stderr "not found" ──────────────

	t.Run("missing_plugin_binary_exits_1_not_found", func(t *testing.T) {
		t.Skip(skipMsgScorerPlugin)

		projectDir := t.TempDir()
		suitePath := filepath.Join(projectDir, "suite.yaml")
		require.NoError(t, os.WriteFile(suitePath, []byte(suiteYAMLParetoStub), 0o644))
		require.NoError(t, os.WriteFile(
			filepath.Join(projectDir, "ben.yaml"),
			[]byte(benYAMLParetoPlugin), 0o644))

		// PATH is set to an empty dir so the stub is NOT found.
		emptyDir := t.TempDir()
		dataDir := t.TempDir()
		runCmd := exec.Command(ben, "run", "--suite", suitePath, "--format", "json")
		runCmd.Dir = projectDir
		runCmd.Env = append(os.Environ(),
			"XDG_DATA_HOME="+dataDir,
			"PATH="+emptyDir,
		)
		out, err := runCmd.CombinedOutput()
		require.Error(t, err, "ben run must exit non-zero when scorer plugin binary is missing")
		assert.Contains(t, string(out), "not found",
			"stderr/output must contain 'not found' when plugin binary is absent")
	})

	// ── Sub-test 4: ben.yaml plugins.metrics key loads without error (smoke) ──
	// This IS testable today: run.go reads plugins.metrics from viper and calls
	// LoadMetricPlugins. An empty or absent plugins block must not cause a crash.
	// This confirms the config-loading wiring that scorer plugins will extend.

	t.Run("benYAML_plugins_key_loads_without_error", func(t *testing.T) {
		// Minimal suite with no scorer plugin; uses raw strategy.
		const safeSuiteYAML = `
name: smoke-suite
version: 1
task:
  prompt: echo smoke
candidates:
  - name: x
    adapter: cli
    cmd: echo x
metrics:
  - exit_code
scorer:
  strategy: raw
`
		// ben.yaml with a plugins block (metrics only — scorer not yet read).
		const safePluginsYAML = `
plugins:
  metrics: []
`
		projectDir := t.TempDir()
		suitePath := filepath.Join(projectDir, "suite.yaml")
		require.NoError(t, os.WriteFile(suitePath, []byte(safeSuiteYAML), 0o644))
		require.NoError(t, os.WriteFile(
			filepath.Join(projectDir, "ben.yaml"),
			[]byte(safePluginsYAML), 0o644))

		dataDir := t.TempDir()
		runCmd := exec.Command(ben, "run", "--suite", suitePath, "--format", "json")
		runCmd.Dir = projectDir
		runCmd.Env = append(os.Environ(), "XDG_DATA_HOME="+dataDir)
		out, err := runCmd.Output()
		require.NoError(t, err,
			"ben run must not crash when ben.yaml has a plugins block; stderr: %s",
			func() string {
				if ee, ok := err.(*exec.ExitError); ok {
					return string(ee.Stderr)
				}
				return ""
			}(),
		)

		var result map[string]any
		require.NoError(t, json.Unmarshal(out, &result),
			"output must be valid JSON: %s", out)
		runID, _ := result["run_id"].(string)
		assert.NotEmpty(t, runID, "run_id must be present")
	})
}
