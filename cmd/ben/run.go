package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/metrics"
	"hop.top/ben/internal/plugin"
	"hop.top/ben/internal/reporter"
	"hop.top/ben/internal/run"
	"hop.top/ben/internal/scorer"
	"hop.top/ben/internal/spec"
	"hop.top/ben/internal/storage"
	"hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/console/progress"
	"hop.top/kit/go/core/xdg"
)

func runCmd(v *viper.Viper) *cobra.Command {
	var (
		suitePath  string
		taskDesc   string
		candidates []string
		metricList []string
		scorerStr  string
		inputKV    map[string]string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a benchmark suite or inline task",
		Long: `Run a benchmark suite or inline task and persist the result.

Safe retry: this command is annotated kit/idempotent=no (§8.5). Each
invocation produces a new run with a fresh run-id; runs are not
deduplicated locally. Callers that need at-most-once semantics across
restarts should persist run-ids client-side or use suite-level metadata
to identify duplicates.`,
		Annotations: map[string]string{
			"kit/side-effect":    "write",
			"kit/idempotent":     "no",
			"kit/top-level-verb": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cli.IsDryRun(cmd) {
				return runDryRun(cmd, v, suitePath, taskDesc, candidates, metricList, scorerStr)
			}
			return runBenchmark(cmd.Context(), v, suitePath, taskDesc, candidates, metricList, scorerStr, inputKV)
		},
	}

	f := cmd.Flags()
	f.StringVar(&suitePath, "suite", "", "Path to suite YAML spec file (mutually exclusive with --task)")
	f.StringVar(&taskDesc, "task", "", "Inline task description (mutually exclusive with --suite)")
	f.StringSliceVar(&candidates, "candidates", nil, "Candidates: name=adapter=cmd,... (adapter defaults to cli)")
	f.StringSliceVar(&metricList, "metric", nil, "Metrics to collect (e.g. latency_ms,exit_code)")
	f.StringVar(&scorerStr, "scorer", "raw", "Scoring strategy: single:<metric>, weighted:<m>=<w>,..., raw")
	f.StringToStringVar(&inputKV, "input", nil, "Input key=value pairs passed to candidates")

	return cmd
}

// runBenchmark is the main logic; separated for testability.
func runBenchmark(
	ctx context.Context,
	v *viper.Viper,
	suitePath, taskDesc string,
	candidates, metricList []string,
	scorerStr string,
	inputKV map[string]string,
) error {
	// progress is selected by kit/cli from --progress-format /
	// --format (§6.5). Human reporter for TTY, JSONL when
	// --format json (or explicit --progress-format json), Discard
	// when --quiet.
	prog := progress.FromContext(ctx)

	// 0. Discover binary plugins on PATH.
	registry := plugin.DiscoverAll()

	// 1. Load spec.
	prog.Emit(ctx, progress.Event{Phase: "load_spec", Item: suitePath})
	var s *spec.Spec
	if suitePath != "" && taskDesc != "" {
		return fmt.Errorf("--suite and --task are mutually exclusive")
	}
	if suitePath != "" {
		loaded, err := spec.Load(suitePath)
		if err != nil {
			return fmt.Errorf("load suite: %w", err)
		}
		s = loaded
	} else {
		// Normalize candidate format: support "name=adapter=cmd" with "=" as separator.
		normalized := make([]string, len(candidates))
		for i, c := range candidates {
			// Convert "name=adapter=cmd" → "name,adapter,cmd" for spec.FromFlags.
			parts := strings.SplitN(c, "=", 3)
			normalized[i] = strings.Join(parts, ",")
		}
		ff := &spec.FromFlags{
			Task:       taskDesc,
			Candidates: normalized,
			Metrics:    metricList,
			Scorer:     scorerStr,
			Input:      inputKV,
		}
		loaded, err := ff.ToSpec()
		if err != nil {
			return fmt.Errorf("build spec: %w", err)
		}
		s = loaded
	}

	// 1b. Load config-declared metric plugins (e.g. llm_judge).
	var pluginCfgs []plugin.MetricPluginConfig
	if err := v.UnmarshalKey("plugins.metrics", &pluginCfgs); err == nil && len(pluginCfgs) > 0 {
		if err := plugin.LoadMetricPlugins(pluginCfgs); err != nil {
			return fmt.Errorf("load metric plugins: %w", err)
		}
	}

	// 2. Determine metrics to collect.
	metricsToCollect := s.Metrics
	if len(metricsToCollect) == 0 {
		// Default: collect all builtins.
		all := metrics.All()
		for name := range all {
			metricsToCollect = append(metricsToCollect, name)
		}
	}

	// 3. Build scorer.
	sc, err := scorer.Parse(s.Scorer.Strategy, s.Scorer.Weights)
	if err != nil {
		return fmt.Errorf("parse scorer: %w", err)
	}

	// 4. Run each candidate.
	cliAdapter := adapter.NewCLI()
	scorerInputs := make([]scorer.CandidateResult, 0, len(s.Candidates))
	for i, c := range s.Candidates {
		prog.Emit(ctx, progress.Event{
			Phase: "run_candidate",
			Item:  c.Name,
			Bytes: int64(i + 1),
			Total: int64(len(s.Candidates)),
		})
		var adpt adapter.Adapter
		switch c.Adapter {
		case "cli", "":
			adpt = cliAdapter
		case "llm":
			adpt = adapter.NewLLM()
		case "eva":
			adpt = adapter.NewEva()
		default:
			// Check plugin registry before erroring.
			if pluginAdapter, ok := registry.LookupAdapter(c.Adapter); ok {
				adpt = pluginAdapter
			} else {
				return fmt.Errorf("unknown adapter %q for candidate %q", c.Adapter, c.Name)
			}
		}

		res, runErr := adpt.Run(ctx, c, s.Task.Input)

		cr := scorer.CandidateResult{
			Name:      c.Name,
			Metrics:   make(map[string]float64),
			RawOutput: "",
		}
		if res != nil {
			cr.RawOutput = res.Output
			// Include plugin-reported metrics if present.
			if pm, ok := res.Metadata["plugin_metrics"]; ok {
				if pmMap, ok := pm.(map[string]float64); ok {
					for k, val := range pmMap {
						cr.Metrics[k] = val
					}
				}
			}
			for _, mname := range metricsToCollect {
				if _, exists := cr.Metrics[mname]; exists {
					continue
				}
				if m, ok := metrics.Get(mname); ok {
					cr.Metrics[mname] = m.Collect(res)
				}
			}
		}
		if runErr != nil {
			cr.Err = runErr
		}
		scorerInputs = append(scorerInputs, cr)
	}

	for _, mname := range metricsToCollect {
		if _, ok := metrics.Get(mname); ok {
			continue
		}
		for _, cr := range scorerInputs {
			if _, exists := cr.Metrics[mname]; !exists {
				return fmt.Errorf("metric %q unavailable for candidate %q", mname, cr.Name)
			}
		}
	}

	// 5. Score.
	prog.Emit(ctx, progress.Event{Phase: "score", Item: s.Scorer.Strategy})
	scored := sc.Score(scorerInputs)

	// 6. Build run.Run.
	hostname, _ := os.Hostname()
	runID := ulid.Make().String()
	candidates2 := make([]run.CandidateResult, len(scored))
	for i, sr := range scored {
		cr := run.CandidateResult{
			Name:      sr.Name,
			Metrics:   sr.Metrics,
			RawOutput: sr.RawOutput,
		}
		if s.Scorer.Strategy != "raw" {
			score := sr.Score
			rank := sr.Rank
			cr.Score = &score
			cr.Rank = &rank
		}
		if sr.Err != nil {
			cr.Error = sr.Err.Error()
		}
		candidates2[i] = cr
	}

	var winner *string
	if s.Scorer.Strategy != "raw" && len(scored) > 0 {
		for _, sr := range scored {
			if sr.Rank == 1 {
				w := sr.Name
				winner = &w
				break
			}
		}
	}

	r := &run.Run{
		RunID:        runID,
		Suite:        s.Name,
		SuiteVersion: s.Version,
		Timestamp:    time.Now().UTC(),
		Scorer: run.ScorerConfig{
			Strategy: s.Scorer.Strategy,
			Weights:  s.Scorer.Weights,
		},
		Candidates: candidates2,
		Winner:     winner,
		Metadata: run.Metadata{
			Host:       hostname,
			BenVersion: "0.1.0",
		},
	}

	// 7. Persist.
	prog.Emit(ctx, progress.Event{Phase: "persist", Item: r.RunID})
	dataDir, err := resolveDataDir()
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	store, err := storage.Open(dataDir)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() { _ = store.Close() }()
	if err := store.Save(ctx, r); err != nil {
		// Non-fatal: report continues even if persistence failed.
		slog.Warn("save run", "run_id", r.RunID, "err", err)
	}
	if err := store.IndexRun(ctx, r); err != nil {
		slog.Warn("index run", "run_id", r.RunID, "err", err)
	}

	// 8. Report.
	prog.Emit(ctx, progress.Event{Phase: "report", Item: r.RunID})
	format := v.GetString("format")

	// Check registry for binary reporter plugins before falling back to built-ins.
	if format != "" && format != "json" && format != "yaml" && format != "table" {
		if pluginRep, ok := registry.LookupReporter(format); ok {
			return pluginRep.Report(os.Stdout, r)
		}
	}

	rep, err := reporter.New(format)
	if err != nil {
		return fmt.Errorf("reporter: %w", err)
	}
	return rep.Report(os.Stdout, r)
}

// runDryRun assembles a Plan describing what `ben run` would do without
// invoking adapters or persisting results. The plan lists each
// candidate as a "create" effect on the run record. The actual run-id
// is not generated during dry-run since no run is recorded.
func runDryRun(cmd *cobra.Command, v *viper.Viper, suitePath, taskDesc string, candidates, metricList []string, scorerStr string) error {
	if suitePath != "" && taskDesc != "" {
		return output.UsageError("--suite and --task are mutually exclusive")
	}

	var s *spec.Spec
	if suitePath != "" {
		loaded, err := spec.Load(suitePath)
		if err != nil {
			return fmt.Errorf("load suite: %w", err)
		}
		s = loaded
	} else {
		normalized := make([]string, len(candidates))
		for i, c := range candidates {
			parts := strings.SplitN(c, "=", 3)
			normalized[i] = strings.Join(parts, ",")
		}
		ff := &spec.FromFlags{
			Task:       taskDesc,
			Candidates: normalized,
			Metrics:    metricList,
			Scorer:     scorerStr,
		}
		loaded, err := ff.ToSpec()
		if err != nil {
			return fmt.Errorf("build spec: %w", err)
		}
		s = loaded
	}

	effects := make([]cli.Effect, 0, len(s.Candidates)+1)
	effects = append(effects, cli.Effect{
		Kind:       "create",
		Target:     "run/<new-run-id>",
		Reversible: false,
		Detail:     fmt.Sprintf("suite=%q candidates=%d", s.Name, len(s.Candidates)),
	})
	for _, c := range s.Candidates {
		adapterName := c.Adapter
		if adapterName == "" {
			adapterName = "cli"
		}
		effects = append(effects, cli.Effect{
			Kind:       "execute",
			Target:     fmt.Sprintf("candidate/%s", c.Name),
			Reversible: true,
			Detail:     fmt.Sprintf("adapter=%s cmd=%q", adapterName, c.Cmd),
		})
	}

	plan := cli.Plan{
		Command: "ben run",
		Args: map[string]any{
			"suite":  suitePath,
			"task":   taskDesc,
			"scorer": scorerStr,
		},
		Effects:     effects,
		GeneratedAt: time.Now().UTC(),
	}

	format := v.GetString("format")
	if format == "" {
		format = output.Table
	}
	return output.RenderPlan(cmd.OutOrStdout(), format, plan)
}

// resolveDataDir checks for a project-local .ben/ dir first; falls back to xdg.DataDir.
func resolveDataDir() (string, error) {
	if fi, err := os.Stat(".ben"); err == nil && fi.IsDir() {
		return ".ben", nil
	}
	return xdg.DataDir("ben")
}
