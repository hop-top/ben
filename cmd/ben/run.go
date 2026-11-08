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
	"hop.top/ben/internal/bench"
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
		repeat     int
		warmup     int
		aggregate  string
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
			rf := repeatFlags{aggregate: aggregate}
			if cmd.Flags().Changed("repeat") {
				rf.repeat = &repeat
			}
			if cmd.Flags().Changed("warmup") {
				rf.warmup = &warmup
			}
			if cli.IsDryRun(cmd) {
				return runDryRun(cmd, v, suitePath, taskDesc, candidates, metricList, scorerStr)
			}
			return runBenchmark(cmd.Context(), v, suitePath, taskDesc, candidates, metricList, scorerStr, inputKV, rf)
		},
	}

	f := cmd.Flags()
	f.StringVar(&suitePath, "suite", "", "Path to suite YAML spec file (mutually exclusive with --task)")
	f.StringVar(&taskDesc, "task", "", "Inline task description (mutually exclusive with --suite)")
	f.StringSliceVar(&candidates, "candidates", nil, "Candidates: name=adapter=cmd,... (adapter defaults to cli)")
	f.StringSliceVar(&metricList, "metric", nil, "Metrics to collect (e.g. latency_ms,exit_code,cpu_ms,max_rss_kb)")
	f.StringVar(&scorerStr, "scorer", "raw", "Scoring strategy: single:<metric>, weighted:<m>=<w>,..., raw")
	f.IntVar(&repeat, "repeat", 1, "Execute each candidate N times; metrics report the aggregate (mean by default), metric_stats the spread")
	f.IntVar(&warmup, "warmup", 0, "Discarded warmup runs per candidate (overrides suite warmup)")
	f.StringVar(&aggregate, "aggregate", "", "Value kept per metric across repeats: mean (default), median, p95")
	f.StringToStringVar(&inputKV, "input", nil, "Input key=value pairs passed to candidates")

	return cmd
}

// repeatFlags holds CLI overrides for suite repetition settings; nil/empty
// means the flag was not given.
type repeatFlags struct {
	repeat, warmup *int
	aggregate      string
}

// apply overrides s with any flags that were set and validates the result.
func (r repeatFlags) apply(s *spec.Spec) error {
	if r.repeat != nil {
		s.Repeat = *r.repeat
	}
	if r.warmup != nil {
		s.Warmup = *r.warmup
	}
	if r.aggregate != "" {
		s.Aggregate = r.aggregate
	}
	if s.Repeat < 0 || s.Warmup < 0 {
		return fmt.Errorf("--repeat and --warmup must be >= 0")
	}
	switch s.Aggregate {
	case "", "median", "mean", "p95":
		return nil
	}
	return fmt.Errorf("--aggregate must be median, mean or p95, got %q", s.Aggregate)
}

// runBenchmark is the main logic; separated for testability.
func runBenchmark(
	ctx context.Context,
	v *viper.Viper,
	suitePath, taskDesc string,
	candidates, metricList []string,
	scorerStr string,
	inputKV map[string]string,
	rf repeatFlags,
) error {
	if rf.repeat != nil && *rf.repeat < 1 {
		return output.UsageError("--repeat must be >= 1")
	}
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

	if err := rf.apply(s); err != nil {
		return err
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
	trialsByName := make(map[string][]map[string]float64, len(s.Candidates))
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

		oc := bench.Measure(ctx, adpt, c, s.Task.Input, metricsToCollect, bench.Options{
			Repeat: s.Repeat, Warmup: s.Warmup, Aggregate: s.Aggregate,
		})
		cr := scorer.CandidateResult{
			Name:      c.Name,
			Metrics:   oc.Metrics,
			RawOutput: oc.Output,
			Err:       oc.Err,
		}
		trialsByName[c.Name] = oc.Trials
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

	// 4c. Validate the scored metric actually exists somewhere (BUG-3):
	// a bogus "single:<metric>" must be an error, not a silent all-zero
	// ranking.
	if m, ok := scorer.SingleMetric(s.Scorer.Strategy); ok {
		found := false
		for _, cr := range scorerInputs {
			if _, exists := cr.Metrics[m]; exists {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("scorer: metric %q not in run (collected: %s)", m, strings.Join(metricsToCollect, ", "))
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
			Name:        sr.Name,
			Metrics:     sr.Metrics,
			RawOutput:   sr.RawOutput,
			Trials:      trialsByName[sr.Name],
			MetricStats: run.ComputeStats(trialsByName[sr.Name]),
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
		Warmup:     s.Warmup,
		Aggregate:  s.Aggregate,
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
