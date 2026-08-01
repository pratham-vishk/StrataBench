package aggregate_test

import (
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/aggregate"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestAggregateResults(t *testing.T) {
	a := schema.Results{IOPS: 1000, ThroughputMBps: 100, LatencyUS: schema.LatencyUS{P99: 200}}
	b := schema.Results{IOPS: 2000, ThroughputMBps: 150, LatencyUS: schema.LatencyUS{P99: 300}}
	out := aggregate.Results([]schema.Results{a, b})
	if out.IOPS != 3000 {
		t.Fatalf("iops=%v", out.IOPS)
	}
	if out.ThroughputMBps != 250 {
		t.Fatalf("tp=%v", out.ThroughputMBps)
	}
	if out.LatencyUS.P99 != 300 {
		t.Fatalf("p99=%v", out.LatencyUS.P99)
	}
}

func TestMergeIntervals(t *testing.T) {
	ts := time.Now().UTC()
	a := schema.Results{Intervals: []schema.IntervalSample{
		{Seq: 1, Timestamp: ts, IOPS: 1000, ThroughputMBps: 10, AvgLatencyUS: 100, MaxLatencyUS: 200},
		{Seq: 2, IOPS: 1100, ThroughputMBps: 11, AvgLatencyUS: 110},
	}}
	b := schema.Results{Intervals: []schema.IntervalSample{
		{Seq: 1, IOPS: 500, ThroughputMBps: 5, AvgLatencyUS: 80, MaxLatencyUS: 300},
	}}
	out := aggregate.Intervals([]schema.Results{a, b})
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].IOPS != 1500 {
		t.Fatalf("iops=%v", out[0].IOPS)
	}
	if out[0].ThroughputMBps != 15 {
		t.Fatalf("mbps=%v", out[0].ThroughputMBps)
	}
	if out[0].MaxLatencyUS != 300 {
		t.Fatalf("max=%v", out[0].MaxLatencyUS)
	}
	if out[0].AvgLatencyUS != 90 {
		t.Fatalf("avg=%v", out[0].AvgLatencyUS)
	}

	merged := aggregate.Results([]schema.Results{a, b})
	if len(merged.Intervals) != 2 {
		t.Fatalf("merged intervals=%d", len(merged.Intervals))
	}
	if merged.Intervals[0].IOPS != 1500 {
		t.Fatalf("merged iops=%v", merged.Intervals[0].IOPS)
	}
}
