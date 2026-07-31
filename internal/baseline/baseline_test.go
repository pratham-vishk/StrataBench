package baseline_test

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestCheckIOPSRegression(t *testing.T) {
	base := &schema.RunResult{
		Results: schema.Results{IOPS: 100000, LatencyUS: schema.LatencyUS{P99: 200}},
	}
	current := &schema.RunResult{
		Results: schema.Results{IOPS: 80000, LatencyUS: schema.LatencyUS{P99: 200}},
	}
	alerts := baseline.Check(current, base, 10, 15)
	if len(alerts) != 1 || alerts[0].Metric != "iops" {
		t.Fatalf("expected iops alert, got %+v", alerts)
	}
}

func TestCheckLatencyRegression(t *testing.T) {
	base := &schema.RunResult{
		Results: schema.Results{IOPS: 100000, LatencyUS: schema.LatencyUS{P99: 200}},
	}
	current := &schema.RunResult{
		Results: schema.Results{IOPS: 100000, LatencyUS: schema.LatencyUS{P99: 250}},
	}
	alerts := baseline.Check(current, base, 10, 15)
	if len(alerts) != 1 || alerts[0].Metric != "latency_p99" {
		t.Fatalf("expected latency alert, got %+v", alerts)
	}
}

func TestTargetKey(t *testing.T) {
	run := &schema.RunResult{
		Target: schema.Target{Device: "/dev/nvme0n1"},
	}
	if baseline.TargetKey(run) != "/dev/nvme0n1" {
		t.Fatalf("unexpected key %q", baseline.TargetKey(run))
	}
}
