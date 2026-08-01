package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pratham-vishk/stratabench/internal/runstate"
)

func (s *Server) handleRunStream(w http.ResponseWriter, r *http.Request, runID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			done, err := s.writeStreamEvent(w, flusher, runID)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
				flusher.Flush()
				return
			}
			if done {
				return
			}
		}
	}
}

func (s *Server) writeStreamEvent(w http.ResponseWriter, flusher http.Flusher, runID string) (done bool, err error) {
	if p, ok := runstate.Get(runID); ok {
		b, err := json.Marshal(p)
		if err != nil {
			return false, err
		}
		fmt.Fprintf(w, "event: progress\ndata: %s\n\n", b)
		flusher.Flush()
		return false, nil
	}

	run, err := s.Svc.Store.Get(runID)
	if err != nil {
		return false, err
	}
	payload := map[string]any{
		"run_id":  runID,
		"status":  run.Status,
		"profile": run.Profile,
		"iops":    run.Results.IOPS,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	event := "progress"
	if run.Status == "completed" || run.Status == "failed" {
		event = "done"
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	flusher.Flush()
	return run.Status == "completed" || run.Status == "failed", nil
}
