package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/adapter"
	"hop.top/ben/internal/metrics"
	_ "hop.top/ben/internal/metrics" // register builtins
	"hop.top/ben/internal/reporter"
	"hop.top/ben/internal/run"
	"hop.top/ben/internal/scorer"
	"hop.top/ben/internal/spec"
	"hop.top/ben/internal/storage"
	"hop.top/kit/xdg"
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
	// 1. Load spec.
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

	// 2. Determine metrics to collect.
	metricsToCollect := s.Metrics
	if len(metricsToCollect) == 0 {
		// Default: collect all builtins.
		all := metrics.All()
		for name := range all {
			metricsToCollect = append(metricsToCollect, name)
		}
	}
	// Validate all requested metrics exist.
	for _, name := range metricsToCollect {
		if _, ok := metrics.Get(name); !ok {
			return fmt.Errorf("unknown metric %q", name)
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
	for _, c := range s.Candidates {
		var adpt adapter.Adapter
		switch c.Adapter {
		case "cli", "":
			adpt = cliAdapter
		default:
			return fmt.Errorf("unknown adapter %q for candidate %q", c.Adapter, c.Name)
		}

		res, runErr := adpt.Run(ctx, c, s.Task.Input)

		cr := scorer.CandidateResult{
			Name:      c.Name,
			Metrics:   make(map[string]float64),
			RawOutput: "",
		}
		if res != nil {
			cr.RawOutput = res.Output
			for _, mname := range metricsToCollect {
				m, _ := metrics.Get(mname)
				cr.Metrics[mname] = m.Collect(res)
			}
		}
		if runErr != nil {
			cr.Err = runErr
		}
		scorerInputs = append(scorerInputs, cr)
	}

	// 5. Score.
	scored := sc.Score(scorerInputs)

	// 6. Build run.Run.
	hostname, _ := os.Hostname()
	runID := ulid.Make().String()
	candidates2 := make([]run.CandidateResult, len(scored))
	for i, sr := range scored {
		cr := run.CandidateResult{
			Name:      sr.Name,
			Metrics:   sr.Metrics,
			Score:     sr.Score,
			Rank:      sr.Rank,
			RawOutput: sr.RawOutput,
		}
		if sr.Err != nil {
			cr.Error = sr.Err.Error()
		}
		candidates2[i] = cr
	}

	winner := ""
	if s.Scorer.Strategy != "raw" && len(scored) > 0 {
		for _, sr := range scored {
			if sr.Rank == 1 {
				winner = sr.Name
				break
			}
		}
	}

	r := &run.Run{
		RunID:     runID,
		Suite:     s.Name,
		SuiteVersion: s.Version,
		Timestamp: time.Now().UTC(),
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
		// Non-fatal: log to stderr but continue to report.
		fmt.Fprintf(os.Stderr, "warn: save run: %v\n", err)
	}

	// 8. Report.
	format := v.GetString("format")
	rep, err := reporter.New(format)
	if err != nil {
		return fmt.Errorf("reporter: %w", err)
	}
	return rep.Report(os.Stdout, r)
}

// resolveDataDir checks for a project-local .ben/ dir first; falls back to xdg.DataDir.
func resolveDataDir() (string, error) {
	if fi, err := os.Stat(".ben"); err == nil && fi.IsDir() {
		return ".ben", nil
	}
	return xdg.DataDir("ben")
}
