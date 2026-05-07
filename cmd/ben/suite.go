package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"hop.top/ben/internal/spec"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/core/xdg"
)

// SuiteSummary is used for table/json/yaml listing output.
type SuiteSummary struct {
	Name        string `json:"name"        yaml:"name"        table:"Name"`
	Description string `json:"description" yaml:"description" table:"Description"`
	Version     int    `json:"version"     yaml:"version"     table:"Version"`
	Candidates  int    `json:"candidates"  yaml:"candidates"  table:"#Candidates"`
}

func suiteCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suite",
		Short: "Manage and inspect benchmark suites",
	}
	cmd.AddCommand(suiteListCmd(v))
	cmd.AddCommand(suiteShowCmd(v))
	return cmd
}

func suiteListCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all known benchmark suites",
		Annotations: map[string]string{
			"kit/side-effect": "read",
			"kit/idempotent":  "yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := suiteScanDirs()
			if err != nil {
				return err
			}
			specs, err := loadAllSuites(dirs)
			if err != nil {
				return err
			}
			summaries := make([]SuiteSummary, 0, len(specs))
			for _, s := range specs {
				summaries = append(summaries, SuiteSummary{
					Name:        s.Name,
					Description: s.Description,
					Version:     s.Version,
					Candidates:  len(s.Candidates),
				})
			}
			format := v.GetString("format")
			if format == "" {
				format = output.Table
			}
			return output.Render(os.Stdout, format, summaries)
		},
	}
}

func suiteShowCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show details of a benchmark suite",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"kit/side-effect": "read",
			"kit/idempotent":  "yes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dirs, err := suiteScanDirs()
			if err != nil {
				return err
			}
			s, err := findSuite(dirs, name)
			if err != nil {
				return err
			}
			if s == nil {
				return output.NotFoundError(fmt.Sprintf("suite %q not found", name))
			}
			format := v.GetString("format")
			switch format {
			case "json", "yaml":
				return output.Render(os.Stdout, format, s)
			default:
				// Human-readable multi-section display.
				return printSuiteHuman(os.Stdout, s)
			}
		},
	}
}

// suiteScanDirs returns the ordered list of directories to scan for suite YAML files:
// 1. XDG data dir (~/.local/share/ben/suites)
// 2. Project-local (.ben/suites)
func suiteScanDirs() ([]string, error) {
	var dirs []string

	// XDG global suites directory.
	dataDir, err := xdg.DataDir("ben")
	if err == nil {
		dirs = append(dirs, filepath.Join(dataDir, "suites"))
	}

	// Project-local suites directory.
	dirs = append(dirs, filepath.Join(".ben", "suites"))

	return dirs, nil
}

// loadAllSuites scans each directory for *.yaml files and loads them as specs.
// Files that fail to parse are silently skipped (corrupt/non-suite YAML).
func loadAllSuites(dirs []string) ([]*spec.Spec, error) {
	seen := map[string]bool{}
	var result []*spec.Spec

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("suite scan %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			path := filepath.Join(dir, e.Name())
			s, err := spec.Load(path)
			if err != nil {
				continue // skip invalid files
			}
			if seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			result = append(result, s)
		}
	}
	return result, nil
}

// printSuiteHuman prints a multi-section human-readable view of a spec.
func printSuiteHuman(w io.Writer, s *spec.Spec) error {
	fmt.Fprintf(w, "Name:        %s\n", s.Name)
	if s.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", s.Description)
	}
	fmt.Fprintf(w, "Version:     %d\n", s.Version)
	fmt.Fprintf(w, "Task:        %s\n", s.Task.Prompt)
	if len(s.Task.Input) > 0 {
		fmt.Fprintf(w, "Input:\n")
		for k, v := range s.Task.Input {
			fmt.Fprintf(w, "  %s: %s\n", k, v)
		}
	}
	if len(s.Candidates) > 0 {
		fmt.Fprintf(w, "Candidates:\n")
		for _, c := range s.Candidates {
			fmt.Fprintf(w, "  - name: %s  adapter: %s", c.Name, c.Adapter)
			if c.Cmd != "" {
				fmt.Fprintf(w, "  cmd: %s", c.Cmd)
			}
			fmt.Fprintln(w)
		}
	}
	if len(s.Metrics) > 0 {
		fmt.Fprintf(w, "Metrics:     %s\n", strings.Join(s.Metrics, ", "))
	}
	fmt.Fprintf(w, "Scorer:      %s\n", s.Scorer.Strategy)
	return nil
}

// findSuite looks for a suite by name across all scan directories.
// Returns nil, nil if not found.
func findSuite(dirs []string, name string) (*spec.Spec, error) {
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("suite scan %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			path := filepath.Join(dir, e.Name())
			s, err := spec.Load(path)
			if err != nil {
				continue
			}
			if s.Name == name {
				return s, nil
			}
		}
	}
	return nil, nil
}
