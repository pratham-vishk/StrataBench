package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestHandleValidate(t *testing.T) {
	dataDir := t.TempDir()
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	s := &Server{Svc: svc}

	body, _ := json.Marshal(validateRequest{Profile: "nvme-random-oltp", Target: "/dev/null", Mock: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleValidate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCompareRuns(t *testing.T) {
	dataDir := t.TempDir()
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	now := time.Now().UTC()
	save := func(id string, iops float64) {
		run := &schema.RunResult{
			RunID: id, Profile: "nvme-random-oltp", Status: "completed", Mock: true,
			Results:    schema.Results{IOPS: iops},
			Timestamps: schema.Timestamps{StartedAt: now, CompletedAt: now},
		}
		if err := svc.Store.Save(run); err != nil {
			t.Fatal(err)
		}
	}
	save("base", 100000)
	save("head", 110000)

	s := &Server{Svc: svc}
	body, _ := json.Marshal(compareRequest{RunIDA: "base", RunIDB: "head"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compare", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleCompare(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCompareMissingIDs(t *testing.T) {
	s := &Server{Svc: &orchestrator.Service{}}
	body := []byte(`{"run_id":"a"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compare", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleCompare(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
