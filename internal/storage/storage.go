// Package storage persists Run results to a local SQLite database via kit/sqlstore.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"hop.top/ben/internal/run"
	"hop.top/kit/sqlstore"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS runs (
  id        TEXT PRIMARY KEY,
  suite     TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  result    TEXT NOT NULL
);
`

// Store wraps kit/sqlstore for Run persistence.
type Store struct {
	kv *sqlstore.Store
}

// Open creates (or opens) the SQLite database at dataDir/ben.db.
// dataDir is typically xdg.DataDir("ben") resolved by the caller.
func Open(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "ben.db")
	kv, err := sqlstore.Open(dbPath, sqlstore.Options{
		MigrateSQL: migrateSQL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage open: %w", err)
	}
	return &Store{kv: kv}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.kv.Close() }

// Save upserts a Run into both the runs table (for List queries) and the kv
// table (for fast Get-by-ID lookups).
func (s *Store) Save(ctx context.Context, r *run.Run) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}
	_, err = s.kv.DB().ExecContext(ctx,
		`INSERT INTO runs (id, suite, timestamp, result) VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET suite=excluded.suite,
		   timestamp=excluded.timestamp, result=excluded.result`,
		r.RunID,
		r.Suite,
		r.Timestamp.UTC().Format("2006-01-02T15:04:05Z07:00"),
		string(data),
	)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	// Also store in kv table for O(1) Get by ID.
	if err := s.kv.Put(ctx, r.RunID, r); err != nil {
		return fmt.Errorf("kv put run: %w", err)
	}
	return nil
}

// Get retrieves a Run by ID. Returns an error wrapping sql.ErrNoRows if not found.
func (s *Store) Get(ctx context.Context, id string) (*run.Run, error) {
	var r run.Run
	found, err := s.kv.Get(ctx, id, &r)
	if err != nil {
		return nil, fmt.Errorf("kv get run: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("run %q: %w", id, sql.ErrNoRows)
	}
	return &r, nil
}

// List returns up to limit runs for the given suite, ordered by timestamp DESC.
// Pass suite="" to list across all suites.
func (s *Store) List(ctx context.Context, suite string, limit int) ([]*run.Run, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if suite == "" {
		rows, err = s.kv.DB().QueryContext(ctx,
			`SELECT result FROM runs ORDER BY timestamp DESC LIMIT ?`, limit)
	} else {
		rows, err = s.kv.DB().QueryContext(ctx,
			`SELECT result FROM runs WHERE suite = ? ORDER BY timestamp DESC LIMIT ?`,
			suite, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*run.Run
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		var r run.Run
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, fmt.Errorf("unmarshal run: %w", err)
		}
		results = append(results, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return results, nil
}

// ErrNotFound is a sentinel for callers that want to distinguish missing IDs.
var ErrNotFound = errors.New("run not found")
