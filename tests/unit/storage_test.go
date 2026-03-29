package unit_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"hop.top/ben/internal/run"
	"hop.top/ben/internal/storage"
)

func newRun(suite string, suiteVersion int, ts time.Time) *run.Run {
	return &run.Run{
		RunID:        ulid.Make().String(),
		Suite:        suite,
		SuiteVersion: suiteVersion,
		Timestamp:    ts.UTC().Truncate(time.Second),
		Scorer: run.ScorerConfig{
			Strategy: "weighted",
			Weights:  map[string]float64{"latency_ms": 0.5, "quality_score": 0.5},
		},
		Candidates: []run.CandidateResult{
			{
				Name:      "xray",
				Metrics:   map[string]float64{"latency_ms": 340, "quality_score": 0.91},
				Score:     0.847,
				Rank:      1,
				RawOutput: "xray output",
			},
			{
				Name:      "grep",
				Metrics:   map[string]float64{"latency_ms": 180, "quality_score": 0.43},
				Score:     0.31,
				Rank:      2,
				RawOutput: "grep output",
			},
		},
		Winner: "xray",
		Metadata: run.Metadata{
			Host:       "testhost",
			BenVersion: "0.1.0",
		},
	}
}

func openStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSaveAndGet(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	r := newRun("codebase-indexing", 1, time.Now())
	if err := s.Save(ctx, r); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, r.RunID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.RunID != r.RunID {
		t.Errorf("RunID mismatch: got %q want %q", got.RunID, r.RunID)
	}
	if got.Suite != r.Suite {
		t.Errorf("Suite mismatch: got %q want %q", got.Suite, r.Suite)
	}
	if got.SuiteVersion != r.SuiteVersion {
		t.Errorf("SuiteVersion mismatch: got %d want %d", got.SuiteVersion, r.SuiteVersion)
	}
	if !got.Timestamp.Equal(r.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v want %v", got.Timestamp, r.Timestamp)
	}
	if got.Winner != r.Winner {
		t.Errorf("Winner mismatch: got %q want %q", got.Winner, r.Winner)
	}
	if got.Scorer.Strategy != r.Scorer.Strategy {
		t.Errorf("Scorer.Strategy mismatch: got %q want %q", got.Scorer.Strategy, r.Scorer.Strategy)
	}
	if len(got.Candidates) != len(r.Candidates) {
		t.Errorf("Candidates len mismatch: got %d want %d", len(got.Candidates), len(r.Candidates))
	}
	if got.Metadata.Host != r.Metadata.Host {
		t.Errorf("Metadata.Host mismatch: got %q want %q", got.Metadata.Host, r.Metadata.Host)
	}
}

func TestGetUnknownID(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for unknown ID, got nil")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows in error chain, got: %v", err)
	}
}

func TestListBySuiteOrdered(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	base := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)

	r1 := newRun("my-suite", 1, base.Add(0))
	r2 := newRun("my-suite", 1, base.Add(1*time.Minute))
	r3 := newRun("my-suite", 1, base.Add(2*time.Minute))
	r4 := newRun("other-suite", 1, base.Add(3*time.Minute))

	for _, r := range []*run.Run{r1, r2, r3, r4} {
		if err := s.Save(ctx, r); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	// List by suite — should return r3, r2, r1 (DESC).
	runs, err := s.List(ctx, "my-suite", 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
	if runs[0].RunID != r3.RunID {
		t.Errorf("first result should be r3 (newest), got %q", runs[0].RunID)
	}
	if runs[2].RunID != r1.RunID {
		t.Errorf("last result should be r1 (oldest), got %q", runs[2].RunID)
	}

	// other-suite should not appear.
	for _, r := range runs {
		if r.Suite != "my-suite" {
			t.Errorf("unexpected suite %q in results", r.Suite)
		}
	}
}

func TestListRespectLimit(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	base := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		r := newRun("limited-suite", 1, base.Add(time.Duration(i)*time.Minute))
		if err := s.Save(ctx, r); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	runs, err := s.List(ctx, "limited-suite", 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs (limit), got %d", len(runs))
	}
}

func TestSaveOverwrite(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()

	r := newRun("update-suite", 1, time.Now())
	if err := s.Save(ctx, r); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	// Mutate winner and save again (same ID).
	r.Winner = "grep"
	if err := s.Save(ctx, r); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	got, err := s.Get(ctx, r.RunID)
	if err != nil {
		t.Fatalf("Get after overwrite: %v", err)
	}
	if got.Winner != "grep" {
		t.Errorf("expected overwritten winner %q, got %q", "grep", got.Winner)
	}
}
