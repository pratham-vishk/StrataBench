package aggregate_test

import (
	"testing"

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
