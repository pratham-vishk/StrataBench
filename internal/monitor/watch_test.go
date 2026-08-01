package monitor_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/monitor"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestWatchRunCompleted(t *testing.T) {
	dataDir := t.TempDir()
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	now := time.Now().UTC()
	run := &schema.RunResult{
		RunID: "watch-test-1", Profile: "nvme-random-oltp", Status: "completed", Mock: true,
		Results:    schema.Results{IOPS: 1000, LatencyUS: schema.LatencyUS{P99: 100}},
		Timestamps: schema.Timestamps{StartedAt: now, CompletedAt: now},
	}
	if err := svc.Store.Save(run); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := monitor.WatchRun(ctx, svc, run.RunID, 50*time.Millisecond, &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("completed")) {
		t.Fatalf("output=%s", buf.String())
	}
}

func TestWatchRunAsync(t *testing.T) {
	dataDir := t.TempDir()
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	p := &profile.Profile{Name: "nvme-random-oltp", Engine: "fio", Layer: "block"}
	runID, err := svc.StartAsyncRun(context.Background(), orchestrator.RunOptions{
		Profile: p, Target: "/dev/null", Mock: true, SkipValidate: true, DataDir: dataDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var buf bytes.Buffer
	if err := monitor.WatchRun(ctx, svc, runID, 100*time.Millisecond, &buf); err != nil {
		t.Fatalf("watch err=%v output=%s", err, buf.String())
	}
}
