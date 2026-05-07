// Package storage persists Run results to a local SQLite database via kit/sqlstore.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"hop.top/ben/internal/run"
	"hop.top/kit/go/storage/sqlstore"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS runs (
  id        TEXT PRIMARY KEY,
  suite     TEXT NOT NULL,
  timestamp TEXT NOT NULL,
  result    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS registry (
  run_id        TEXT PRIMARY KEY,
  suite         TEXT NOT NULL,
  suite_version INTEGER NOT NULL,
  timestamp     TEXT NOT NULL,
  pushed_at     TEXT,
  remote_id     TEXT
);
`

// RegistryEntry is a row in the local registry index.
type RegistryEntry struct {
	RunID        string
	Suite        string
	SuiteVersion int
	Timestamp    time.Time
	PushedAt     *time.Time
	RemoteID     string
}

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

// IndexRun inserts a run into the local registry index (pushed_at=NULL, remote_id=NULL).
// Ignores duplicates via ON CONFLICT DO NOTHING.
func (s *Store) IndexRun(ctx context.Context, r *run.Run) error {
	_, err := s.kv.DB().ExecContext(ctx,
		`INSERT INTO registry (run_id, suite, suite_version, timestamp, pushed_at, remote_id)
		 VALUES (?, ?, ?, ?, NULL, NULL)
		 ON CONFLICT(run_id) DO NOTHING`,
		r.RunID,
		r.Suite,
		r.SuiteVersion,
		r.Timestamp.UTC().Format("2006-01-02T15:04:05Z07:00"),
	)
	if err != nil {
		return fmt.Errorf("index run: %w", err)
	}
	return nil
}

// ListRegistry returns up to limit registry entries for suite (pass "" for all),
// ordered by timestamp DESC.
func (s *Store) ListRegistry(ctx context.Context, suite string, limit int) ([]*RegistryEntry, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if suite == "" {
		rows, err = s.kv.DB().QueryContext(ctx,
			`SELECT run_id, suite, suite_version, timestamp, pushed_at, remote_id
			 FROM registry ORDER BY timestamp DESC LIMIT ?`, limit)
	} else {
		rows, err = s.kv.DB().QueryContext(ctx,
			`SELECT run_id, suite, suite_version, timestamp, pushed_at, remote_id
			 FROM registry WHERE suite = ? ORDER BY timestamp DESC LIMIT ?`,
			suite, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list registry: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*RegistryEntry
	for rows.Next() {
		var (
			e         RegistryEntry
			tsStr     string
			pushedStr sql.NullString
			remoteStr sql.NullString
		)
		if err := rows.Scan(&e.RunID, &e.Suite, &e.SuiteVersion, &tsStr, &pushedStr, &remoteStr); err != nil {
			return nil, fmt.Errorf("scan registry row: %w", err)
		}
		e.RemoteID = remoteStr.String
		ts, err := time.Parse("2006-01-02T15:04:05Z07:00", tsStr)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp: %w", err)
		}
		e.Timestamp = ts
		if pushedStr.Valid && pushedStr.String != "" {
			pt, err := time.Parse("2006-01-02T15:04:05Z07:00", pushedStr.String)
			if err != nil {
				return nil, fmt.Errorf("parse pushed_at: %w", err)
			}
			e.PushedAt = &pt
		}
		results = append(results, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registry rows error: %w", err)
	}
	return results, nil
}

// MarkPushed sets pushed_at to now and records the remote_id for the given run.
func (s *Store) MarkPushed(ctx context.Context, runID, remoteID string) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z07:00")
	res, err := s.kv.DB().ExecContext(ctx,
		`UPDATE registry SET pushed_at = ?, remote_id = ? WHERE run_id = ?`,
		now, remoteID, runID,
	)
	if err != nil {
		return fmt.Errorf("mark pushed: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark pushed rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("mark pushed: run %q not in registry index", runID)
	}
	return nil
}
