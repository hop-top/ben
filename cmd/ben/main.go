// Command ben is a general-purpose benchmarking tool.
// It answers "which approach is better, and by how much?" for any measurable task.
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	speccli "hop.top/kit/go/ai/toolspec/cli"
	"hop.top/kit/go/console/cli"
	kitcliconfig "hop.top/kit/go/console/cli/config"
	kitlog "hop.top/kit/go/console/log"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/core/xdg"
)

const (
	groupExecute  = "execute"
	groupResults  = "results"
	groupCatalog  = "catalog"
	groupRegistry = "registry"

	// schemaVersion is the ben CLI capability schema version exposed by
	// `ben spec --version`. Bumped per kit/console/cli §13.2: MAJOR for
	// removals/renames, MINOR for additive changes. Distinct from the
	// binary version in cli.Config.Version.
	schemaVersion = "1.0"
)

// commandGroups maps a top-level command name to its kit group ID.
// MANAGEMENT (config, version, upgrade, completion) is handled by kit.
var commandGroups = map[string]string{
	"run":      groupExecute,
	"list":     groupResults,
	"show":     groupResults,
	"compare":  groupResults,
	"suite":    groupCatalog,
	"registry": groupRegistry,
	"config":   "management",
	"spec":     "management",
}

func main() {
	root := cli.New(cli.Config{
		Name:    "ben",
		Version: "0.1.0",
		Short:   "benchmarking tool — answers 'which approach is better, and by how much?'",
		Help: cli.HelpConfig{
			Groups: []cli.GroupConfig{
				{ID: groupExecute, Title: "EXECUTE"},
				{ID: groupResults, Title: "RESULTS"},
				{ID: groupCatalog, Title: "CATALOG"},
				{ID: groupRegistry, Title: "REGISTRY"},
			},
		},
		EnforceValidate: true,
	}, cli.WithStatus(cli.StatusConfig{}))

	// Wire kit's slog-compatible logger as the slog default so every
	// `slog.Info/Warn/Error/Debug` call across ben respects --quiet,
	// --no-color, and -V verbosity.
	slog.SetDefault(slog.New(kitlog.New(root.Viper)))

	// Compose ben's config loader after kit's existing PersistentPreRunE
	// chain (chdir → identity → peer init).
	prev := root.Cmd.PersistentPreRunE
	loader := makeConfigLoader(root)
	root.Cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if prev != nil {
			if err := prev(cmd, args); err != nil {
				return err
			}
		}
		return loader(cmd, args)
	}

	root.Cmd.AddCommand(
		runCmd(root.Viper),
		listCmd(root.Viper),
		showCmd(root.Viper),
		compareCmd(root.Viper),
		suiteCmd(root.Viper),
		registryCmd(root.Viper),
		configCmd(root),
	)

	registerHints(root)
	speccli.RegisterSpecCommand(root, schemaVersion)
	applyCommandGroups(root.Cmd)
	// Eagerly install cobra's `completion` subcommand so we can stamp
	// kit annotations on its auto-generated leaves before kit's
	// EnforceValidate runs inside Execute(). InitDefaultCompletionCmd is
	// idempotent.
	root.Cmd.InitDefaultCompletionCmd()
	annotateUnannotatedLeaves(root.Cmd)

	if err := root.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}

// annotateUnannotatedLeaves walks the command tree and stamps any leaf
// missing kit/side-effect or kit/idempotent annotations with read/yes
// defaults. This covers cobra-supplied leaves (completion bash/fish/...)
// and kit-supplied leaves (config path/paths) that ben does not own.
func annotateUnannotatedLeaves(root *cobra.Command) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if !c.HasSubCommands() {
			if c.Annotations == nil {
				c.Annotations = map[string]string{}
			}
			if _, ok := c.Annotations["kit/side-effect"]; !ok {
				c.Annotations["kit/side-effect"] = "read"
			}
			if _, ok := c.Annotations["kit/idempotent"]; !ok {
				c.Annotations["kit/idempotent"] = "yes"
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

// configCmd is the parent `ben config` command. kit/cli/config provides
// `path` and `paths` subcommands; ben supplies the resolver that walks
// the precedence chain.
func configCmd(_ *cli.Root) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect ben configuration",
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			"kit/side-effect": "read",
			"kit/idempotent":  "yes",
		},
	}
	kitcliconfig.RegisterPathSubcommands(cmd, "ben", kitcliconfig.WithResolver(benConfigResolver))
	return cmd
}

// benConfigResolver returns ben's config precedence chain (highest first):
// project → user → system. The --config flag short-circuits this chain
// at runtime; kit's `config path` reports the highest existing entry.
func benConfigResolver(cwd string) []kitcliconfig.ResolvedPath {
	out := []kitcliconfig.ResolvedPath{}

	// Project layer.
	for _, name := range []string{".ben.yaml", filepath.Join(".ben", "ben.yaml")} {
		p := filepath.Join(cwd, name)
		out = append(out, kitcliconfig.ResolvedPath{
			Path:   p,
			Source: "file",
			Scope:  "project",
			Exists: fileExists(p),
		})
	}

	// User layer.
	if dir, err := xdg.ConfigDir("ben"); err == nil {
		p := filepath.Join(dir, "ben.yaml")
		out = append(out, kitcliconfig.ResolvedPath{
			Path:   p,
			Source: "file",
			Scope:  "user",
			Exists: fileExists(p),
		})
	}

	// System layer.
	out = append(out, kitcliconfig.ResolvedPath{
		Path:   "/etc/ben/ben.yaml",
		Source: "file",
		Scope:  "system",
		Exists: fileExists("/etc/ben/ben.yaml"),
	})

	return out
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// registerHints wires per-command next-step suggestions into kit's
// hint registry. Hints are emitted after primary output and silenced
// by --no-hints, --quiet, or non-TTY stdout.
func registerHints(root *cli.Root) {
	if root.Hints == nil {
		return
	}
	root.Hints.Register("run", output.Hint{
		Message: "Use `ben list` to see recent runs, or `ben show <run-id>` for details.",
	})
	root.Hints.Register("list", output.Hint{
		Message: "Use `ben show <run-id>` for details, or `ben compare <a> <b>` to diff two runs.",
	})
	root.Hints.Register("registry pull", output.Hint{
		Message: "Use `ben list` to see what was pulled.",
	})
}

// applyCommandGroups assigns each top-level command its GroupID from the
// commandGroups map. Centralised so re-grouping is one map edit, not N
// file edits.
func applyCommandGroups(rootCmd *cobra.Command) {
	for _, c := range rootCmd.Commands() {
		if g, ok := commandGroups[c.Name()]; ok {
			c.GroupID = g
		}
	}
}

// makeConfigLoader returns a PreRunE that loads a ben.yaml config file
// into v. It honours kit's -c/--config global (StringArray): bare-path
// tokens layer extra config files on top of discovered defaults;
// key=value tokens apply scalar overrides. When no -c paths are given
// it discovers default locations: project (.ben.yaml or .ben/ben.yaml) →
// user ($XDG_CONFIG_HOME/ben/ben.yaml) → system (/etc/ben/ben.yaml).
func makeConfigLoader(root *cli.Root) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if root == nil || root.Viper == nil {
			return nil
		}
		v := root.Viper
		paths, overrides, err := root.ConfigArgs()
		if err != nil {
			return err
		}
		if len(paths) > 0 {
			// -c <path> tokens: use the first as the primary config file
			// and merge any additional ones on top.
			v.SetConfigFile(paths[0])
			if err := v.ReadInConfig(); err != nil {
				return err
			}
			for _, extra := range paths[1:] {
				v.SetConfigFile(extra)
				if err := v.MergeInConfig(); err != nil {
					return err
				}
			}
		} else {
			v.SetConfigName("ben")
			v.SetConfigType("yaml")
			v.AddConfigPath(".")
			v.AddConfigPath(".ben")
			if dir, err := xdg.ConfigDir("ben"); err == nil {
				v.AddConfigPath(dir)
			}
			v.AddConfigPath("/etc/ben")
			_ = v.ReadInConfig() // missing config is not an error
		}
		// Apply key=value overrides on top of any loaded file(s).
		for k, val := range overrides {
			v.Set(k, val)
		}
		return nil
	}
}
