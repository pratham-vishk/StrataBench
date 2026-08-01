package samples

import (
	"path/filepath"
	"time"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// WriteMultiNodeSample writes an HTML report demonstrating full multi-node coverage.
func WriteMultiNodeSample(outDir string) error {
	if err := mkdir(outDir); err != nil {
		return err
	}
	run := multiNodeSampleRun()
	return report.WriteHTMLOnly(run, report.Options{
		Summary:  "Multi-node pool — 3 clients → 1 NVMe target. Per-node overlays, Grafana dashboards, interval tables, and node matrix.",
		Insights: analyst.Analyze(run, nil),
	}, filepath.Join(outDir, "multi-node-sample.html"))
}

func multiNodeSampleRun() *schema.RunResult {
	ts := time.Now().UTC().Add(-60 * time.Second)
	ivs := func(base float64, jitter float64) []schema.IntervalSample {
		var out []schema.IntervalSample
		for i := 0; i < 12; i++ {
			j := jitter + float64(i%3)*0.02
			out = append(out, schema.IntervalSample{
				Seq: i + 1, Timestamp: ts.Add(time.Duration(i*5) * time.Second),
				IOPS: base * j, ReadIOPS: base * j * 0.7, WriteIOPS: base * j * 0.3,
				ThroughputMBps: base * j / 100, ReadMBps: base * j * 0.7 / 100, WriteMBps: base * j * 0.3 / 100,
				AvgLatencyUS: 95 + float64(i), MinLatencyUS: 40, MaxLatencyUS: 400 + float64(i*10),
			})
		}
		return out
	}
	p := map[string]float64{"p50": 100, "p99": 350, "p99.9": 500}
	client := func(host string, iops float64) schema.ClientResult {
		return schema.ClientResult{
			Host: host, Target: "/dev/nvme0n1",
			Results: schema.Results{
				IOPS: iops, ReadIOPS: iops * 0.7, WriteIOPS: iops * 0.3,
				ThroughputMBps: iops / 100, Percentiles: p,
				LatencyUS: schema.LatencyUS{P50: 100, P99: 350, Min: 40, Max: 800, Mean: 110},
				Intervals: ivs(iops, 0.95),
				Totals: schema.TotalStats{TotalMB: iops / 100 * 60, TotalRecords: int64(iops * 60)},
			},
		}
	}
	c1, c2, c3 := client("10.0.1.1:7777", 800000), client("10.0.1.2:7777", 820000), client("10.0.1.3:7777", 790000)
	aggIOPS := c1.Results.IOPS + c2.Results.IOPS + c3.Results.IOPS
	return &schema.RunResult{
		RunID: "multi-node-sample", Profile: "nvme-random-oltp", Layer: "block", Engine: "fio",
		Topology: "pool", Mock: true, Validation: schema.ValidationResult{Passed: true},
		Target:   schema.Target{Type: "block", Device: "/dev/nvme0n1"},
		Workload: schema.Workload{DurationSec: 60, Pattern: "randrw", BlockSize: "4k", QueueDepth: 32},
		Results: schema.Results{
			IOPS: aggIOPS, ReadIOPS: aggIOPS * 0.7, WriteIOPS: aggIOPS * 0.3,
			ThroughputMBps: aggIOPS / 100, Percentiles: p,
			LatencyUS: schema.LatencyUS{P50: 100, P99: 380, Min: 40, Max: 900, Mean: 112},
			Intervals: ivs(aggIOPS, 1.0),
			Totals:    schema.TotalStats{TotalMB: aggIOPS / 100 * 60, TotalRecords: int64(aggIOPS * 60)},
		},
		Clients: []schema.ClientResult{c1, c2, c3},
		Targets: []schema.TargetResult{{
			Target: "/dev/nvme0n1",
			Results: schema.Results{
				IOPS: aggIOPS, ThroughputMBps: aggIOPS / 100, Percentiles: p,
				LatencyUS: schema.LatencyUS{P50: 98, P99: 370}, Intervals: ivs(aggIOPS, 1.0),
			},
		}},
		Timestamps: schema.Timestamps{StartedAt: ts, CompletedAt: ts.Add(60 * time.Second)},
	}
}
