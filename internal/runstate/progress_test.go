package runstate

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestProgressLifecycle(t *testing.T) {
	id := "run-1"
	Set(Progress{RunID: id, Phase: "running", TotalAssignments: 2})
	p, ok := Get(id)
	if !ok || p.TotalAssignments != 2 {
		t.Fatalf("%+v ok=%v", p, ok)
	}
	IncrementDone(id)
	p, _ = Get(id)
	if p.CompletedAssignments != 1 {
		t.Fatalf("%+v", p)
	}
	Clear(id)
	if _, ok := Get(id); ok {
		t.Fatal("expected cleared")
	}
}

func TestRecordInterval(t *testing.T) {
	id := "run-iv"
	Set(Progress{RunID: id, Phase: "running"})
	RecordInterval(id, schema.IntervalSample{Seq: 1, IOPS: 1000, ThroughputMBps: 50})
	p, ok := Get(id)
	if !ok || p.IntervalBuckets != 1 || p.LatestInterval == nil || p.LatestInterval.IOPS != 1000 {
		t.Fatalf("%+v ok=%v", p, ok)
	}
	RecordInterval(id, schema.IntervalSample{Seq: 2, IOPS: 2000})
	p, _ = Get(id)
	if p.IntervalBuckets != 2 || p.LatestInterval.Seq != 2 {
		t.Fatalf("%+v", p)
	}
}
