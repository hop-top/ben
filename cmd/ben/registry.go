package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/registry"
	"hop.top/ben/internal/storage"
	"hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/core/xdg"
)

func registryCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Push or pull runs to/from a shared registry",
	}
	cmd.AddCommand(registryPushCmd(v))
	cmd.AddCommand(registryPullCmd(v))
	return cmd
}

func registryPushCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "push <run-id>",
		Short: "Push a local run to the remote registry",
		Long: `Push a local run to the remote registry.

Safe retry: this command is annotated kit/idempotent=conditional
(§8.5). Pushes are idempotent on (run-id, registry-url): re-pushing
the same run-id to the same registry resolves to the same remote-id
without creating a duplicate, so retries are safe under fixed inputs.
Pushing the same run-id to two different registries produces two
distinct remote-ids by design.`,
		Args: cobra.ExactArgs(1),
		Annotations: map[string]string{
			"kit/side-effect": "write",
			"kit/idempotent":  "conditional",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cli.IsDryRun(cmd) {
				return registryPushDryRun(cmd, v, args[0])
			}
			return registryPush(cmd.Context(), v, args[0])
		},
	}
}

func registryPullCmd(v *viper.Viper) *cobra.Command {
	var suite string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull runs from the remote registry into local storage",
		Annotations: map[string]string{
			"kit/side-effect": "write",
			"kit/idempotent":  "yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cli.IsDryRun(cmd) {
				return registryPullDryRun(cmd, v, suite)
			}
			return registryPull(cmd.Context(), v, suite)
		},
	}
	cmd.Flags().StringVar(&suite, "suite", "", "Filter pulled runs by suite name")
	return cmd
}

func registryPush(ctx context.Context, v *viper.Viper, runID string) error {
	registryURL := v.GetString("registry.url")
	if registryURL == "" {
		return output.UsageError("registry.url not configured; set it in ben.yaml under registry.url")
	}

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

	r, err := store.Get(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return output.NotFoundError(fmt.Sprintf("run %q not found", runID))
		}
		return fmt.Errorf("load run %q: %w", runID, err)
	}

	client := registry.NewRemoteClient(registryURL)
	remoteID, err := client.Push(ctx, r)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}

	if err := store.MarkPushed(ctx, runID, remoteID); err != nil {
		return fmt.Errorf("mark pushed: %w", err)
	}

	slog.Info("registry push complete", "run_id", runID, "remote_id", remoteID)
	return nil
}

func registryPull(ctx context.Context, v *viper.Viper, suite string) error {
	registryURL := v.GetString("registry.url")
	if registryURL == "" {
		return output.UsageError("registry.url not configured; set it in ben.yaml under registry.url")
	}

	dataDir, err := xdg.DataDir("ben")
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

	client := registry.NewRemoteClient(registryURL)
	runs, err := client.Pull(ctx, suite, 100)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	for _, r := range runs {
		if err := store.Save(ctx, r); err != nil {
			return fmt.Errorf("save pulled run %q: %w", r.RunID, err)
		}
		if err := store.IndexRun(ctx, r); err != nil {
			return fmt.Errorf("index pulled run %q: %w", r.RunID, err)
		}
	}

	slog.Info("registry pull complete", "suite", suite, "count", len(runs))
	return nil
}

// registryPushDryRun emits a Plan describing what `registry push <id>`
// would do, without contacting the remote registry or marking the run
// as pushed.
func registryPushDryRun(cmd *cobra.Command, v *viper.Viper, runID string) error {
	registryURL := v.GetString("registry.url")
	if registryURL == "" {
		return output.UsageError("registry.url not configured; set it in ben.yaml under registry.url")
	}
	plan := cli.Plan{
		Command: "ben registry push",
		Args:    map[string]any{"run_id": runID, "registry_url": registryURL},
		Effects: []cli.Effect{
			{
				Kind:       "create",
				Target:     fmt.Sprintf("%s/runs/<remote-id>", registryURL),
				Reversible: false,
				Detail:     fmt.Sprintf("uploads run %q to remote", runID),
			},
			{
				Kind:       "update",
				Target:     fmt.Sprintf("registry/%s", runID),
				Reversible: false,
				Detail:     "marks local registry row pushed_at + remote_id",
			},
		},
		GeneratedAt: time.Now().UTC(),
	}
	format := v.GetString("format")
	if format == "" {
		format = output.Table
	}
	return output.RenderPlan(cmd.OutOrStdout(), format, plan)
}

// registryPullDryRun emits a Plan describing what `registry pull` would
// do, without contacting the remote registry.
func registryPullDryRun(cmd *cobra.Command, v *viper.Viper, suite string) error {
	registryURL := v.GetString("registry.url")
	if registryURL == "" {
		return output.UsageError("registry.url not configured; set it in ben.yaml under registry.url")
	}
	plan := cli.Plan{
		Command: "ben registry pull",
		Args:    map[string]any{"suite": suite, "registry_url": registryURL},
		Effects: []cli.Effect{
			{
				Kind:       "fetch",
				Target:     fmt.Sprintf("%s/runs?suite=%s", registryURL, suite),
				Reversible: true,
				Detail:     "GET up to 100 runs",
			},
			{
				Kind:       "create",
				Target:     "runs/<each-fetched-id>",
				Reversible: true,
				Detail:     "stores fetched runs in local DB",
			},
		},
		GeneratedAt: time.Now().UTC(),
	}
	format := v.GetString("format")
	if format == "" {
		format = output.Table
	}
	return output.RenderPlan(cmd.OutOrStdout(), format, plan)
}
