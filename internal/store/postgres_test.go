package store_test

import (
	"os"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/store"
)

func TestOpenPostgres(t *testing.T) {
	dsn := os.Getenv("STRATABENCH_DATABASE_URL")
	if dsn == "" {
		t.Skip("STRATABENCH_DATABASE_URL not set")
	}
	s, err := store.OpenPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().UTC()
	run := &schema.RunResult{
		RunID:   "pg-test-1",
		Profile: "nvme-random-oltp",
		Status:  "completed",
		Timestamps: schema.Timestamps{StartedAt: now},
		Results: schema.Results{IOPS: 1000},
	}
	if err := s.Save(run); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("pg-test-1")
	if err != nil || got.Results.IOPS != 1000 {
		t.Fatalf("get err=%v iops=%v", err, got.Results.IOPS)
	}
}

func TestOpenDefaultSQLite(t *testing.T) {
	s, err := store.OpenDefault(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
}
