package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/orchestrator"
)

func TestHandleRunsAsync(t *testing.T) {
	dataDir := t.TempDir()
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	s := &Server{Svc: svc}

	body, _ := json.Marshal(runRequest{
		Profile:      "nvme-random-oltp",
		Target:       "/dev/null",
		Mock:         true,
		Async:        true,
		SkipValidate: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleRuns(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	runID := resp["run_id"]
	if runID == "" {
		t.Fatalf("resp=%v", resp)
	}

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		run, err := svc.Store.Get(runID)
		if err != nil {
			t.Fatal(err)
		}
		if run.Status == "completed" {
			return
		}
		if run.Status == "failed" {
			t.Fatalf("run failed: %v", run.Validation.Errors)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("async run did not complete")
}
