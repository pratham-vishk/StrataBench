package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestStartAsyncRun(t *testing.T) {
	dataDir := t.TempDir()
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	p := &profile.Profile{Name: "nvme-random-oltp", Engine: "fio", Layer: "block"}
	runID, err := svc.StartAsyncRun(context.Background(), orchestrator.RunOptions{
		Profile:      p,
		Target:       "/dev/null",
		Mock:         true,
		SkipValidate: true,
		DataDir:      dataDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		run, err := svc.Store.Get(runID)
		if err != nil {
			t.Fatal(err)
		}
		if run.Status == "completed" {
			if run.Results.IOPS <= 0 {
				t.Fatalf("completed without results: %+v", run)
			}
			return
		}
		if run.Status == "failed" {
			t.Fatalf("run failed: %+v", run.Validation.Errors)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timeout waiting for async run")
}
