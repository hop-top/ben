// Package plugin provides binary plugin discovery and registration.
package plugin

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoverAdapters scans PATH for binaries named ben-adapter-* and returns
// the stripped names (prefix "ben-adapter-" removed).
func DiscoverAdapters() []string {
	return discoverByPrefix("ben-adapter-")
}

// DiscoverReporters scans PATH for binaries named ben-reporter-* and returns
// the stripped names (prefix "ben-reporter-" removed).
func DiscoverReporters() []string {
	return discoverByPrefix("ben-reporter-")
}

func discoverByPrefix(prefix string) []string {
	var found []string
	seen := map[string]bool{}

	pathEnv := os.Getenv("PATH")
	dirs := filepath.SplitList(pathEnv)

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			stripped := strings.TrimPrefix(name, prefix)
			if stripped == "" {
				continue
			}
			fullPath := filepath.Join(dir, name)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}
			// Must be executable.
			if info.Mode()&0o111 == 0 {
				continue
			}
			if seen[stripped] {
				continue
			}
			seen[stripped] = true
			found = append(found, stripped)
		}
	}
	return found
}
