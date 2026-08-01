package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func OpenPostgres(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &Store{db: db, dialect: "postgres"}
	if err := s.migratePostgres(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migratePostgres() error {
	_, err := s.db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS runs (
  run_id TEXT PRIMARY KEY,
  profile TEXT NOT NULL,
  status TEXT NOT NULL,
  mock BOOLEAN NOT NULL DEFAULT FALSE,
  result_json TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_runs_created ON runs(created_at DESC);

CREATE TABLE IF NOT EXISTS baselines (
  profile TEXT NOT NULL,
  target_key TEXT NOT NULL,
  run_id TEXT NOT NULL,
  set_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (profile, target_key)
);

CREATE TABLE IF NOT EXISTS hardware_inventory (
  host_id TEXT PRIMARY KEY,
  snapshot_json TEXT NOT NULL,
  collected_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS smart_history (
  id BIGSERIAL PRIMARY KEY,
  host_id TEXT NOT NULL,
  device TEXT NOT NULL,
  reading_json TEXT NOT NULL,
  collected_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_smart_collected ON smart_history(collected_at DESC);
`)
	return err
}

func (s *Store) savePostgres(run *schema.RunResult) error {
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), `
INSERT INTO runs (run_id, profile, status, mock, result_json, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (run_id) DO UPDATE SET
  profile = EXCLUDED.profile,
  status = EXCLUDED.status,
  mock = EXCLUDED.mock,
  result_json = EXCLUDED.result_json,
  created_at = EXCLUDED.created_at`,
		run.RunID, run.Profile, run.Status, run.Mock, string(data), run.Timestamps.StartedAt.UTC(),
	)
	return err
}

func (s *Store) saveHardwarePostgres(hostID, snapshotJSON string) error {
	_, err := s.db.ExecContext(context.Background(), `
INSERT INTO hardware_inventory (host_id, snapshot_json, collected_at)
VALUES ($1, $2, $3)
ON CONFLICT (host_id) DO UPDATE SET
  snapshot_json = EXCLUDED.snapshot_json,
  collected_at = EXCLUDED.collected_at`,
		hostID, snapshotJSON, time.Now().UTC(),
	)
	return err
}

func (s *Store) setBaselinePostgres(profile, targetKey, runID string) error {
	_, err := s.db.ExecContext(context.Background(), `
INSERT INTO baselines (profile, target_key, run_id, set_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (profile, target_key) DO UPDATE SET
  run_id = EXCLUDED.run_id,
  set_at = EXCLUDED.set_at`,
		profile, targetKey, runID, time.Now().UTC(),
	)
	return err
}
