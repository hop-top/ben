// Command ben is a general-purpose benchmarking tool.
// It answers "which approach is better, and by how much?" for any measurable task.
package main

import (
	"os"

	"github.com/spf13/cobra"
	"hop.top/kit/cli"
)

func main() {
	root := cli.New(cli.Config{
		Name:    "ben",
		Version: "0.1.0",
		Short:   "benchmarking tool",
	})

	// --config persistent flag: override config file path.
	pf := root.Cmd.PersistentFlags()
	pf.String("config", "", "Config file path (default: $HOME/.config/ben/ben.yaml)")
	_ = root.Viper.BindPFlag("config", pf.Lookup("config"))

	// Auto-load config when flag is set or discover default locations.
	root.Cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cfgFile := root.Viper.GetString("config"); cfgFile != "" {
			root.Viper.SetConfigFile(cfgFile)
			if err := root.Viper.ReadInConfig(); err != nil {
				return err
			}
		} else {
			root.Viper.SetConfigName("ben")
			root.Viper.SetConfigType("yaml")
			root.Viper.AddConfigPath("$HOME/.config/ben")
			root.Viper.AddConfigPath(".ben")
			_ = root.Viper.ReadInConfig() // ignore missing config; not required
		}
		return nil
	}

	// Subcommands.
	root.Cmd.AddCommand(runCmd(root.Viper))
	root.Cmd.AddCommand(compareCmd(root.Viper))
	root.Cmd.AddCommand(queryCmd(root.Viper))
	root.Cmd.AddCommand(registryCmd(root.Viper))
	root.Cmd.AddCommand(suiteCmd(root.Viper))

	if err := root.Cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
