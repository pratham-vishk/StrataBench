package monitor

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/runstate"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// WatchRun polls store and in-memory progress until the run completes or ctx is cancelled.
func WatchRun(ctx context.Context, svc *orchestrator.Service, runID string, interval time.Duration, w io.Writer) error {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastLine string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			line := formatStatus(svc, runID)
			if line != lastLine {
				fmt.Fprintln(w, line)
				lastLine = line
			}
			run, err := svc.Store.Get(runID)
			if err != nil {
				return err
			}
			switch run.Status {
			case "completed":
				fmt.Fprintf(w, "completed: run_id=%s iops=%.0f p99=%.0fµs\n",
					run.RunID, run.Results.IOPS, run.Results.LatencyUS.P99)
				return nil
			case "failed":
				return fmt.Errorf("run failed: %v", run.Validation.Errors)
			}
		}
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func formatStatus(svc *orchestrator.Service, runID string) string {
	if p, ok := runstate.Get(runID); ok {
		total := p.TotalAssignments
		if total <= 0 {
			total = 1
		}
		pct := float64(p.CompletedAssignments) / float64(total) * 100
		return fmt.Sprintf("[%s] %s %d/%d (%.0f%%)", shortID(runID), p.Phase, p.CompletedAssignments, total, pct)
	}
	run, err := svc.Store.Get(runID)
	if err != nil {
		return fmt.Sprintf("[%s] unknown", shortID(runID))
	}
	return fmt.Sprintf("[%s] status=%s", shortID(runID), run.Status)
}

// FormatRunSummary returns a one-line summary for a completed run.
func FormatRunSummary(run *schema.RunResult) string {
	return fmt.Sprintf("run_id=%s profile=%s status=%s iops=%.0f",
		run.RunID, run.Profile, run.Status, run.Results.IOPS)
}
