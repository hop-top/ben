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
	//
	// 1.0 → 1.1: additive — `ben run` emits per-phase progress events on
	// stderr; `ben list`/`show` carry an `_meta` provenance envelope under
	// --format json|yaml; root honours §8.6 delegation policy (cli.WithPolicy).
	schemaVersion = "1.1"
)

// version is set by the linker (-X main.version=<tag>) during release
// builds. Defaults to "dev" for local builds.
var version = "dev"

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
		Version: version,
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
	},
		cli.WithStatus(cli.StatusConfig{}),
		// §8.6 delegation safety: load named policies from
		// $XDG_CONFIG_HOME/ben/policies/<name>.yaml when --policy=<name>
		// is passed. --confirm and --max-ops are kit-globals registered
		// independently of this loader.
		cli.WithPolicy(cli.DefaultPolicyLoader("ben")),
	)

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
func configCmd(root *cli.Root) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect ben configuration",
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			"kit/side-effect": "read",
			"kit/idempotent":  "yes",
		},
	}
	kitcliconfig.RegisterPathSubcommands(cmd, "ben",
		kitcliconfig.WithResolver(benConfigResolver(root)))
	return cmd
}

// projectConfigPath returns the project-layer config path based on the
// caller-context signal (KIT_INVOKED_AS surfaced via root.InvokedAs()):
//
//	invokedAs == "hop" → cwd/.hop/ben.yaml      (under hop umbrella)
//	invokedAs == "tlc" → cwd/.tlc/ben.yaml      (under tlc workspace)
//	otherwise          → cwd/.ben/config.yaml   (standalone)
//
// Only one wins per invocation — kit does not allow multiple project-
// layer configs. Callers (tlc/hop/etc.) export KIT_INVOKED_AS before
// exec'ing ben.
func projectConfigPath(cwd, invokedAs string) string {
	switch invokedAs {
	case "hop":
		return filepath.Join(cwd, ".hop", "ben.yaml")
	case "tlc":
		return filepath.Join(cwd, ".tlc", "ben.yaml")
	default:
		return filepath.Join(cwd, ".ben", "config.yaml")
	}
}

// benConfigResolver returns a kit Resolver that walks ben's config
// precedence chain (highest first): project → user → system. The
// project-layer path is chosen by projectConfigPath based on
// root.InvokedAs() (KIT_INVOKED_AS env var, exported by callers like
// tlc/hop). The --config flag short-circuits this chain at runtime;
// kit's `config path` reports the highest existing entry.
func benConfigResolver(root *cli.Root) kitcliconfig.Resolver {
	return func(cwd string) []kitcliconfig.ResolvedPath {
		var invokedAs string
		if root != nil {
			invokedAs = root.InvokedAs()
		}
		out := []kitcliconfig.ResolvedPath{}

		// Project layer (caller-context-aware).
		p := projectConfigPath(cwd, invokedAs)
		out = append(out, kitcliconfig.ResolvedPath{
			Path:   p,
			Source: "file",
			Scope:  "project",
			Exists: fileExists(p),
		})

		// User layer.
		if dir, err := xdg.ConfigDir("ben"); err == nil {
			p := filepath.Join(dir, "config.yaml")
			out = append(out, kitcliconfig.ResolvedPath{
				Path:   p,
				Source: "file",
				Scope:  "user",
				Exists: fileExists(p),
			})
		}

		// System layer.
		out = append(out, kitcliconfig.ResolvedPath{
			Path:   "/etc/ben/config.yaml",
			Source: "file",
			Scope:  "system",
			Exists: fileExists("/etc/ben/config.yaml"),
		})

		return out
	}
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

// makeConfigLoader returns a PreRunE that loads a ben config file into v.
// It honours kit's -c/--config global (StringArray): bare-path tokens
// OVERRIDE the discovery chain when supplied (kit semantics — -c wins over
// any previously discovered file); key=value tokens apply scalar overrides
// on top of whatever file layer was loaded. When no -c paths are given it
// discovers default locations in precedence order (lowest first, highest
// merged last):
//
//	system  /etc/ben/config.yaml
//	user    $XDG_CONFIG_HOME/ben/config.yaml
//	project <projectConfigPath(cwd, root.InvokedAs())>
//
// The project layer is caller-context-aware: ".hop/ben.yaml" under hop
// umbrella, ".tlc/ben.yaml" under tlc, ".ben/config.yaml" standalone.
// See projectConfigPath.
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
			// -c <path> tokens override discovery entirely: use the first
			// as the primary config file and merge any additional ones on
			// top of it (later -c wins over earlier).
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
			// Walk the precedence chain lowest-to-highest, merging each
			// layer so higher layers (project) win over lower (system).
			// Missing files are not an error; parse errors are.
			candidates := []string{"/etc/ben/config.yaml"}
			if dir, derr := xdg.ConfigDir("ben"); derr == nil {
				candidates = append(candidates, filepath.Join(dir, "config.yaml"))
			}
			cwd, cwdErr := os.Getwd()
			if cwdErr == nil {
				candidates = append(candidates, projectConfigPath(cwd, root.InvokedAs()))
			}

			loaded := false
			for _, p := range candidates {
				if !fileExists(p) {
					continue
				}
				v.SetConfigFile(p)
				if !loaded {
					if err := v.ReadInConfig(); err != nil {
						return err
					}
					loaded = true
				} else {
					if err := v.MergeInConfig(); err != nil {
						return err
					}
				}
			}
		}
		// Apply key=value overrides on top of any loaded file(s).
		for k, val := range overrides {
			v.Set(k, val)
		}
		return nil
	}
}
