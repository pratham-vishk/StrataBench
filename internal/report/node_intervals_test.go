package report

import (
	"strings"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestMultiNodeIntervalReport(t *testing.T) {
	ts := time.Now().UTC()
	ivs := func(iops float64) []schema.IntervalSample {
		return []schema.IntervalSample{
			{Seq: 1, Timestamp: ts, IOPS: iops, ThroughputMBps: iops / 100, AvgLatencyUS: 100, ReadIOPS: iops * 0.7, WriteIOPS: iops * 0.3},
			{Seq: 2, Timestamp: ts.Add(5 * time.Second), IOPS: iops * 1.1, ThroughputMBps: iops / 90, AvgLatencyUS: 110, ReadIOPS: iops * 0.7, WriteIOPS: iops * 0.3},
		}
	}
	pLabels := map[string]float64{"p50": 100, "p99": 200}
	run := &schema.RunResult{
		Profile: "nvme-random-oltp", Layer: "block", Engine: "mock", Topology: "pool",
		Validation: schema.ValidationResult{Passed: true},
		Workload:   schema.Workload{DurationSec: 60},
		Results: schema.Results{
			IOPS: 3000, ThroughputMBps: 30, ReadIOPS: 2100, WriteIOPS: 900,
			LatencyUS: schema.LatencyUS{P50: 100, P99: 250},
			Percentiles: pLabels, Intervals: ivs(3000),
		},
		Clients: []schema.ClientResult{
			{Host: "10.0.1.1:7777", Target: "/dev/nvme0n1", Results: schema.Results{
				IOPS: 1000, ThroughputMBps: 10, ReadIOPS: 700, WriteIOPS: 300,
				LatencyUS: schema.LatencyUS{P50: 120, P99: 400},
				Percentiles: pLabels, Intervals: ivs(1000),
			}},
			{Host: "10.0.1.2:7777", Target: "/dev/nvme0n1", Results: schema.Results{
				IOPS: 2000, ThroughputMBps: 20, ReadIOPS: 1400, WriteIOPS: 600,
				LatencyUS: schema.LatencyUS{P50: 118, P99: 410},
				Percentiles: pLabels, Intervals: ivs(2000),
			}},
		},
		Targets: []schema.TargetResult{
			{Target: "/dev/nvme0n1", Results: schema.Results{
				IOPS: 3000, ThroughputMBps: 30,
				LatencyUS: schema.LatencyUS{P50: 118, P99: 410},
				Percentiles: pLabels, Intervals: ivs(3000),
			}},
		},
	}

	cd, err := buildCardData(run, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !cd.HasIntervals {
		t.Fatal("expected HasIntervals")
	}
	if len(cd.NodeIntervalSections) < 3 {
		t.Fatalf("expected 3+ node interval sections, got %d", len(cd.NodeIntervalSections))
	}
	foundOverlay := false
	for _, g := range cd.ChartGroups {
		if g.Title == "Per-node operations — overlay" {
			foundOverlay = true
		}
	}
	if !foundOverlay {
		t.Fatal("missing per-node overlay chart group")
	}
	if !strings.Contains(string(cd.ChartsJS), "nodeOverlayIops") {
		t.Fatal("missing nodeOverlayIops in chart payload")
	}
}

func TestCollectIntervalSourcesSweep(t *testing.T) {
	run := &schema.RunResult{
		Results: schema.Results{Intervals: []schema.IntervalSample{{Seq: 1, IOPS: 100}}},
		Targets: []schema.TargetResult{
			{Target: "s1", Results: schema.Results{Intervals: []schema.IntervalSample{{Seq: 1, IOPS: 50}}}},
			{Target: "s2", Results: schema.Results{Intervals: []schema.IntervalSample{{Seq: 1, IOPS: 50}}}},
		},
	}
	src := collectIntervalSources(run)
	if len(src) != 3 {
		t.Fatalf("got %d sources, want aggregate + 2 targets", len(src))
	}
}
