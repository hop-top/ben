package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/registry"
	"hop.top/ben/internal/storage"
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return registryPush(cmd.Context(), v, args[0])
		},
	}
}

func registryPullCmd(v *viper.Viper) *cobra.Command {
	var suite string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull runs from the remote registry into local storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return registryPull(cmd.Context(), v, suite)
		},
	}
	cmd.Flags().StringVar(&suite, "suite", "", "Filter pulled runs by suite name")
	return cmd
}

func registryPush(ctx context.Context, v *viper.Viper, runID string) error {
	registryURL := v.GetString("registry.url")
	if registryURL == "" {
		return fmt.Errorf("registry.url not configured; set it in ben.yaml under registry.url")
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

	fmt.Fprintf(os.Stdout, "pushed %s → %s\n", runID, remoteID)
	return nil
}

func registryPull(ctx context.Context, v *viper.Viper, suite string) error {
	registryURL := v.GetString("registry.url")
	if registryURL == "" {
		return fmt.Errorf("registry.url not configured; set it in ben.yaml under registry.url")
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

	fmt.Fprintf(os.Stdout, "pulled %d runs for suite %q\n", len(runs), suite)
	return nil
}
