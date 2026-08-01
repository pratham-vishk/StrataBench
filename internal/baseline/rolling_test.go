package baseline_test

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestReferenceFromHistory(t *testing.T) {
	history := []*schema.RunResult{
		{RunID: "a", Profile: "ssd-random-4k", Target: schema.Target{Device: "test"}, Results: schema.Results{IOPS: 100000, LatencyUS: schema.LatencyUS{P99: 300}}},
		{RunID: "b", Profile: "ssd-random-4k", Target: schema.Target{Device: "test"}, Results: schema.Results{IOPS: 120000, LatencyUS: schema.LatencyUS{P99: 250}}},
	}
	ref := baseline.ReferenceFromHistory(history, "ssd-random-4k", "test", "")
	if ref == nil || ref.IOPS != 120000 {
		t.Fatalf("ref=%+v", ref)
	}
}

func TestCheckAgainstReference(t *testing.T) {
	current := &schema.RunResult{
		Results: schema.Results{IOPS: 85000, LatencyUS: schema.LatencyUS{P99: 200}},
	}
	ref := &baseline.ReferenceStats{IOPS: 100000, P99: 180}
	alerts := baseline.CheckAgainstReference(current, ref, 10, 15)
	if len(alerts) != 1 || alerts[0].Metric != "iops" {
		t.Fatalf("alerts=%+v", alerts)
	}
}
