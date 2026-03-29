package unit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/plugin"
)

func makeFakeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	// Write a minimal shell script.
	err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755)
	require.NoError(t, err)
	return p
}

func withPATH(t *testing.T, dir string) {
	t.Helper()
	orig := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+orig)
}

func TestDiscoverAdapters(t *testing.T) {
	tmp := t.TempDir()
	makeFakeExecutable(t, tmp, "ben-adapter-foo")
	withPATH(t, tmp)

	names := plugin.DiscoverAdapters()
	assert.Contains(t, names, "foo", "should discover ben-adapter-foo as 'foo'")
}

func TestDiscoverAdapters_NotReporter(t *testing.T) {
	tmp := t.TempDir()
	makeFakeExecutable(t, tmp, "ben-reporter-bar")
	withPATH(t, tmp)

	names := plugin.DiscoverAdapters()
	for _, n := range names {
		assert.NotEqual(t, "bar", n, "reporter should not appear in adapter list")
	}
}

func TestDiscoverReporters(t *testing.T) {
	tmp := t.TempDir()
	makeFakeExecutable(t, tmp, "ben-reporter-bar")
	withPATH(t, tmp)

	names := plugin.DiscoverReporters()
	assert.Contains(t, names, "bar", "should discover ben-reporter-bar as 'bar'")
}

func TestDiscoverReporters_NotAdapter(t *testing.T) {
	tmp := t.TempDir()
	makeFakeExecutable(t, tmp, "ben-adapter-baz")
	withPATH(t, tmp)

	names := plugin.DiscoverReporters()
	for _, n := range names {
		assert.NotEqual(t, "baz", n, "adapter should not appear in reporter list")
	}
}

func TestDiscoverAdapters_EmptyPrefix(t *testing.T) {
	tmp := t.TempDir()
	// A file named exactly "ben-adapter-" (empty suffix) should be ignored.
	p := filepath.Join(tmp, "ben-adapter-")
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755))
	withPATH(t, tmp)

	names := plugin.DiscoverAdapters()
	for _, n := range names {
		assert.NotEmpty(t, n)
	}
}
