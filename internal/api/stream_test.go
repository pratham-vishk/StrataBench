package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestHandleRunStream(t *testing.T) {
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

	s := &Server{Svc: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+runID+"/stream", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		s.handleRunStream(rec, req, runID)
		close(done)
	}()

	select {
	case <-done:
		body := rec.Body.String()
		if !strings.Contains(body, "event:") {
			t.Fatalf("expected SSE events, got %q", body)
		}
		if !strings.Contains(body, "event: interval") {
			t.Fatalf("expected interval SSE events, got %q", body)
		}
	case <-time.After(45 * time.Second):
		t.Fatal("stream did not complete")
	}
}
