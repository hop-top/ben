package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/reporter"
	"hop.top/ben/internal/storage"
	"hop.top/kit/go/console/output"
)

func showCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <run-id>",
		Short: "Show details of a single run by id",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"kit/side-effect": "read",
			"kit/idempotent":  "yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return showRun(cmd.Context(), v, args[0])
		},
	}
}

func showRun(ctx context.Context, v *viper.Viper, runID string) error {
	dataDir, err := resolveDataDir()
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
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

	format := v.GetString("format")
	switch format {
	case output.JSON, output.YAML:
		return output.Render(os.Stdout, format, r)
	default:
		rep, repErr := reporter.New(output.Table)
		if repErr != nil {
			return repErr
		}
		return rep.Report(os.Stdout, r)
	}
}
