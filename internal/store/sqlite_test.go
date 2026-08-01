package store_test

import (
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/store"
)

func sampleRun(id string) *schema.RunResult {
	now := time.Now().UTC()
	return &schema.RunResult{
		RunID:   id,
		Profile: "nvme-random-oltp",
		Status:  "completed",
		Mock:    true,
		Results: schema.Results{IOPS: 100000, ThroughputMBps: 400},
		Timestamps: schema.Timestamps{
			StartedAt:   now,
			CompletedAt: now.Add(time.Minute),
		},
	}
}

func TestStoreSaveGetList(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	run := sampleRun("run-1")
	if err := s.Save(run); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("run-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Results.IOPS != 100000 {
		t.Fatalf("iops=%v", got.Results.IOPS)
	}

	run2 := sampleRun("run-2")
	run2.Results.IOPS = 200000
	if err := s.Save(run2); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d", len(list))
	}
}

func TestStoreBaseline(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Save(sampleRun("base-1")); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBaseline("nvme-random-oltp", "/dev/nvme0n1", "base-1"); err != nil {
		t.Fatal(err)
	}
	rec, err := s.GetBaseline("nvme-random-oltp", "/dev/nvme0n1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.RunID != "base-1" {
		t.Fatalf("run_id=%s", rec.RunID)
	}
}

func TestStoreHardware(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.SaveHardware("host1", `{"cpus":8}`); err != nil {
		t.Fatal(err)
	}
	rec, err := s.GetHardware("host1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.SnapshotJSON == "" {
		t.Fatal("empty snapshot")
	}
}
