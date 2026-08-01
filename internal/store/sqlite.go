package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS runs (
  run_id TEXT PRIMARY KEY,
  profile TEXT NOT NULL,
  status TEXT NOT NULL,
  mock INTEGER NOT NULL DEFAULT 0,
  result_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_runs_created ON runs(created_at DESC);

CREATE TABLE IF NOT EXISTS baselines (
  profile TEXT NOT NULL,
  target_key TEXT NOT NULL,
  run_id TEXT NOT NULL,
  set_at TEXT NOT NULL,
  PRIMARY KEY (profile, target_key)
);

CREATE TABLE IF NOT EXISTS hardware_inventory (
  host_id TEXT PRIMARY KEY,
  snapshot_json TEXT NOT NULL,
  collected_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS smart_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  host_id TEXT NOT NULL,
  device TEXT NOT NULL,
  reading_json TEXT NOT NULL,
  collected_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_smart_collected ON smart_history(collected_at DESC);
`)
	return err
}

type HardwareRecord struct {
	HostID       string
	SnapshotJSON string
	CollectedAt  string
}

func (s *Store) SaveHardware(hostID, snapshotJSON string) error {
	_, err := s.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO hardware_inventory (host_id, snapshot_json, collected_at) VALUES (?, ?, ?)`,
		hostID, snapshotJSON, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) ListHardware() ([]HardwareRecord, error) {
	rows, err := s.db.QueryContext(context.Background(),
		`SELECT host_id, snapshot_json, collected_at FROM hardware_inventory ORDER BY collected_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HardwareRecord
	for rows.Next() {
		var rec HardwareRecord
		if err := rows.Scan(&rec.HostID, &rec.SnapshotJSON, &rec.CollectedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

type SMARTRecord struct {
	HostID      string
	Device      string
	ReadingJSON string
	CollectedAt string
}

func (s *Store) SaveSMART(hostID, device, readingJSON string) error {
	_, err := s.db.ExecContext(context.Background(),
		`INSERT INTO smart_history (host_id, device, reading_json, collected_at) VALUES (?, ?, ?, ?)`,
		hostID, device, readingJSON, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) ListSMART(limit int) ([]SMARTRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(context.Background(),
		`SELECT host_id, device, reading_json, collected_at FROM smart_history ORDER BY collected_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SMARTRecord
	for rows.Next() {
		var rec SMARTRecord
		if err := rows.Scan(&rec.HostID, &rec.Device, &rec.ReadingJSON, &rec.CollectedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) ListSince(since time.Time, limit int) ([]schema.RunResult, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.QueryContext(context.Background(),
		`SELECT result_json FROM runs WHERE created_at >= ? ORDER BY created_at DESC LIMIT ?`,
		since.UTC().Format(time.RFC3339), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schema.RunResult
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var run schema.RunResult
		if err := json.Unmarshal([]byte(raw), &run); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func (s *Store) GetHardware(hostID string) (*HardwareRecord, error) {
	var rec HardwareRecord
	err := s.db.QueryRowContext(context.Background(),
		`SELECT host_id, snapshot_json, collected_at FROM hardware_inventory WHERE host_id = ?`, hostID,
	).Scan(&rec.HostID, &rec.SnapshotJSON, &rec.CollectedAt)
	if err != nil {
		return nil, fmt.Errorf("hardware %s: %w", hostID, err)
	}
	return &rec, nil
}

type BaselineRecord struct {
	Profile   string
	TargetKey string
	RunID     string
	SetAt     string
}

func (s *Store) SetBaseline(profile, targetKey, runID string) error {
	_, err := s.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO baselines (profile, target_key, run_id, set_at) VALUES (?, ?, ?, ?)`,
		profile, targetKey, runID, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetBaseline(profile, targetKey string) (*BaselineRecord, error) {
	var rec BaselineRecord
	err := s.db.QueryRowContext(context.Background(),
		`SELECT profile, target_key, run_id, set_at FROM baselines WHERE profile = ? AND target_key = ?`,
		profile, targetKey,
	).Scan(&rec.Profile, &rec.TargetKey, &rec.RunID, &rec.SetAt)
	if err != nil {
		return nil, fmt.Errorf("baseline %s/%s: %w", profile, targetKey, err)
	}
	return &rec, nil
}

func (s *Store) ListBaselines() ([]BaselineRecord, error) {
	rows, err := s.db.QueryContext(context.Background(),
		`SELECT profile, target_key, run_id, set_at FROM baselines ORDER BY profile, target_key`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BaselineRecord
	for rows.Next() {
		var rec BaselineRecord
		if err := rows.Scan(&rec.Profile, &rec.TargetKey, &rec.RunID, &rec.SetAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) DeleteBaseline(profile, targetKey string) error {
	_, err := s.db.ExecContext(context.Background(),
		`DELETE FROM baselines WHERE profile = ? AND target_key = ?`,
		profile, targetKey,
	)
	return err
}

func (s *Store) Save(run *schema.RunResult) error {
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	mock := 0
	if run.Mock {
		mock = 1
	}
	_, err = s.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO runs (run_id, profile, status, mock, result_json, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		run.RunID, run.Profile, run.Status, mock, string(data), run.Timestamps.StartedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) Get(runID string) (*schema.RunResult, error) {
	var raw string
	err := s.db.QueryRowContext(context.Background(), `SELECT result_json FROM runs WHERE run_id = ?`, runID).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("run %s: %w", runID, err)
	}
	var run schema.RunResult
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Store) List(limit int) ([]schema.RunResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT result_json FROM runs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schema.RunResult
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var run schema.RunResult
		if err := json.Unmarshal([]byte(raw), &run); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}
