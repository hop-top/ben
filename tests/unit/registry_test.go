package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/run"
)

func TestIndexRun_AppearsInListRegistry(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	r := newRun("bench-suite", 3, time.Now())
	require.NoError(t, s.Save(ctx, r))
	require.NoError(t, s.IndexRun(ctx, r))

	entries, err := s.ListRegistry(ctx, "bench-suite", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, r.RunID, e.RunID)
	assert.Equal(t, "bench-suite", e.Suite)
	assert.Equal(t, 3, e.SuiteVersion)
	assert.Nil(t, e.PushedAt)
	assert.Equal(t, "", e.RemoteID)
}

func TestListRegistry_SuiteFilter(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	base := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)

	r1 := newRun("suite-a", 1, base)
	r2 := newRun("suite-a", 1, base.Add(time.Minute))
	r3 := newRun("suite-b", 1, base.Add(2*time.Minute))

	for _, r := range []*run.Run{r1, r2, r3} {
		require.NoError(t, s.Save(ctx, r))
		require.NoError(t, s.IndexRun(ctx, r))
	}

	got, err := s.ListRegistry(ctx, "suite-a", 10)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	for _, e := range got {
		assert.Equal(t, "suite-a", e.Suite)
	}
}

func TestListRegistry_Limit(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	base := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		r := newRun("limit-suite", 1, base.Add(time.Duration(i)*time.Minute))
		require.NoError(t, s.Save(ctx, r))
		require.NoError(t, s.IndexRun(ctx, r))
	}

	got, err := s.ListRegistry(ctx, "limit-suite", 2)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestMarkPushed(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	r := newRun("push-suite", 1, time.Now())
	require.NoError(t, s.Save(ctx, r))
	require.NoError(t, s.IndexRun(ctx, r))

	require.NoError(t, s.MarkPushed(ctx, r.RunID, "remote-xyz"))

	entries, err := s.ListRegistry(ctx, "push-suite", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "remote-xyz", e.RemoteID)
	require.NotNil(t, e.PushedAt)
	assert.False(t, e.PushedAt.IsZero())
}

func TestListRegistry_UnindexedRunNotPresent(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	r := newRun("hidden-suite", 1, time.Now())
	// Save but do NOT IndexRun.
	require.NoError(t, s.Save(ctx, r))

	entries, err := s.ListRegistry(ctx, "hidden-suite", 10)
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}
