package runstate

import (
	"sync"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// Progress tracks an in-flight benchmark run.
type Progress struct {
	RunID                string                 `json:"run_id"`
	Phase                string                 `json:"phase"`
	TotalAssignments     int                    `json:"total_assignments"`
	CompletedAssignments int                    `json:"completed_assignments"`
	Profile              string                 `json:"profile,omitempty"`
	Error                string                 `json:"error,omitempty"`
	StartedAt            time.Time              `json:"started_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	LatestInterval       *schema.IntervalSample `json:"latest_interval,omitempty"`
	IntervalBuckets      int                    `json:"interval_buckets,omitempty"`
}

var active sync.Map

func Set(p Progress) {
	p.UpdatedAt = time.Now().UTC()
	active.Store(p.RunID, p)
}

func Get(runID string) (Progress, bool) {
	v, ok := active.Load(runID)
	if !ok {
		return Progress{}, false
	}
	return v.(Progress), true
}

func Clear(runID string) {
	active.Delete(runID)
}

func IncrementDone(runID string) {
	v, ok := active.Load(runID)
	if !ok {
		return
	}
	p := v.(Progress)
	p.CompletedAssignments++
	p.UpdatedAt = time.Now().UTC()
	active.Store(runID, p)
}

// RecordInterval stores the latest time-bucket sample for an in-flight run.
func RecordInterval(runID string, sample schema.IntervalSample) {
	v, ok := active.Load(runID)
	if !ok {
		return
	}
	p := v.(Progress)
	s := sample
	p.LatestInterval = &s
	p.IntervalBuckets++
	p.UpdatedAt = time.Now().UTC()
	active.Store(runID, p)
}
