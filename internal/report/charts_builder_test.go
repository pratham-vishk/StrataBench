package report

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestBuildAllChartsSingleNode(t *testing.T) {
	run := sbkLikeRun(false)
	built := buildAllCharts(run)
	if built.Count < 75 {
		t.Fatalf("expected 75+ charts including totals, got %d", built.Count)
	}
}

func TestBuildAllChartsMultiNode(t *testing.T) {
	run := sbkLikeRun(true)
	built := buildAllCharts(run)
	if built.Count < 90 {
		t.Fatalf("expected 90+ charts for multi-node, got %d", built.Count)
	}
}

func sbkLikeRun(multi bool) *schema.RunResult {
	p50 := 0.38
	percentiles := map[string]float64{}
	for _, l := range metrics.StandardPercentileLabels {
		percentiles[l] = p50
	}
	var ivs []schema.IntervalSample
	for i := 0; i < 12; i++ {
		ivs = append(ivs, schema.IntervalSample{
			Seq: i + 1, IOPS: 2e6, ThroughputMBps: 21, AvgLatencyUS: p50,
			MinLatencyUS: 0.25, MaxLatencyUS: 5000, Percentiles: percentiles,
		})
	}
	run := &schema.RunResult{
		Profile: "Reading", Layer: "application", Engine: "sbk",
		Validation: schema.ValidationResult{Passed: true},
		Results: schema.Results{
			IOPS: 2e6, ThroughputMBps: 21, LatencyUS: schema.LatencyUS{P50: p50, P99: 0.58},
			Percentiles: percentiles, PercentileCounts: map[string]int64{"p50": 100},
			Intervals: ivs,
		},
		Workload: schema.Workload{DurationSec: 60},
	}
	if multi {
		run.Clients = []schema.ClientResult{
			{Host: "c1", Results: schema.Results{IOPS: 800000, ThroughputMBps: 8, LatencyUS: schema.LatencyUS{P50: 120, P99: 400}, Percentiles: percentiles}},
			{Host: "c2", Results: schema.Results{IOPS: 820000, ThroughputMBps: 8.2, LatencyUS: schema.LatencyUS{P50: 118, P99: 410}, Percentiles: percentiles}},
		}
		run.Targets = []schema.TargetResult{
			{Target: "storage-01", Results: schema.Results{IOPS: 1.6e6, ThroughputMBps: 16, LatencyUS: schema.LatencyUS{P50: 95, P99: 350}, Percentiles: percentiles}},
		}
	}
	return run
}
