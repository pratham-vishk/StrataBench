// Sample report generator — run: go run ./examples/sample-report
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/export"
	"github.com/pratham-vishk/stratabench/internal/importsbk"
	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func main() {
	outDir := filepath.Join("examples", "sample-report", "output")
	_ = os.MkdirAll(outDir, 0o755)
	htmlPath := filepath.Join(outDir, "sample-result.html")
	xlsxPath := filepath.Join(outDir, "sample-result.xlsx")
	jsonPath := filepath.Join(outDir, "sample-result.json")

	run, summary, source := loadSampleRun()
	opts := report.Options{
		Summary: summary,
		Insights: []analyst.Insight{
			{Severity: "info", Message: fmt.Sprintf("Sample source: %s", source)},
			{Severity: "info", Message: fmt.Sprintf("%d interval buckets · %d percentile points · full node-compare chart suite", len(run.Results.Intervals), len(run.Results.Percentiles))},
		},
	}
	if len(run.Clients) > 0 {
		opts.Insights = append(opts.Insights, analyst.Insight{
			Severity: "warning", Message: "client-03 p99.9 slightly above cluster median on storage-node-02.",
		})
	}

	if err := report.WriteHTMLWithOptions(run, opts, htmlPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := export.WriteExcel(run, xlsxPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := export.WriteJSON(run, jsonPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("One sample result (all charts + sheets):")
	fmt.Println("  HTML: ", htmlPath)
	fmt.Println("  Excel:", xlsxPath)
	fmt.Println("  JSON: ", jsonPath)
	fmt.Printf("  Run:   %s | %s | %.0f IOPS | %d intervals\n",
		run.RunID, run.Profile, run.Results.IOPS, len(run.Results.Intervals))
}

func loadSampleRun() (*schema.RunResult, string, string) {
	sbkPath := filepath.Join(".tmp", "sbk-charts", "samples", "charts", "sbk-file-read.csv")
	if _, err := os.Stat(sbkPath); err == nil {
		runs, err := importsbk.ParseCSV(sbkPath)
		if err == nil && len(runs) > 0 {
			run := runs[0]
			run.RunID = "sample-file-read"
			return run,
				"File-read benchmark — 12 interval buckets, 27 percentiles, and full latency histogram on one page.",
				"imported benchmark data"
		}
	}
	run := syntheticRun()
	return run,
		"Synthetic distributed NVMe OLTP — 3 clients, 2 storage nodes, mock intervals + full percentile set.",
		"synthetic"
}

func syntheticRun() *schema.RunResult {
	now := time.Now().UTC()
	run := &schema.RunResult{
		SchemaVersion: schema.SchemaVersion,
		RunID:         "sample-multi-node-001",
		Profile:       "nvme-random-oltp",
		Layer:         "block",
		Engine:        "fio",
		Status:        "completed",
		Mock:          true,
		Topology:      "distributed",
		Validation:    schema.ValidationResult{Passed: true, RulesChecked: []string{"mock"}},
		Target: schema.Target{
			Type: "block", Device: "/dev/nvme0n1", Host: "storage-node-01.dell.lab",
		},
		Workload: schema.Workload{
			Pattern: "random", BlockSize: "4k", ReadWriteMix: 70,
			QueueDepth: 32, Threads: 8, DurationSec: 600, RampTimeSec: 60, DirectIO: true,
		},
		Results: schema.Results{
			IOPS: 248500, ReadIOPS: 174000, WriteIOPS: 74500, ThroughputMBps: 969.1,
			OpsPerSec: 248500, CPUPercent: 42.3, TotalOperations: 149100000,
			LatencyUS: schema.LatencyUS{
				Min: 18.2, Mean: 142.5, P50: 118.0, P75: 165.4, P90: 210.8,
				P95: 285.6, P99: 412.3, P999: 890.1, P9999: 1840.0, Max: 3200.0,
			},
		},
		Clients: []schema.ClientResult{
			{Host: "client-01.dell.lab", Target: "storage-node-01.dell.lab:/dev/nvme0n1",
				Results: results(83500, 325.2, lat(128, 175, 245, 285, 445, 980))},
			{Host: "client-02.dell.lab", Target: "storage-node-01.dell.lab:/dev/nvme0n1",
				Results: results(82100, 319.8, lat(122, 168, 238, 275, 428, 920))},
			{Host: "client-03.dell.lab", Target: "storage-node-02.dell.lab:/dev/nvme0n1",
				Results: results(82900, 323.1, lat(125, 172, 252, 290, 438, 950))},
		},
		Targets: []schema.TargetResult{
			{Target: "storage-node-01.dell.lab", Results: results(165600, 646.9, lat(108, 148, 198, 240, 380, 820))},
			{Target: "storage-node-02.dell.lab", Results: results(82900, 323.1, lat(112, 155, 205, 248, 395, 880))},
		},
		Timestamps: schema.Timestamps{StartedAt: now.Add(-12 * time.Minute), CompletedAt: now},
	}
	run.Results.Percentiles = samplePercentiles(run.Results.LatencyUS)
	run.Results.PercentileCounts = samplePercentileCounts(run.Results.TotalOperations)
	run.Results.Intervals = sampleIntervals(now, 600, run.Results.IOPS, run.Results.ThroughputMBps, run.Results.LatencyUS.P50)
	// Full percentile curves per client for node-compare charts.
	for i := range run.Clients {
		run.Clients[i].Results.Percentiles = samplePercentiles(run.Clients[i].Results.LatencyUS)
	}
	for i := range run.Targets {
		run.Targets[i].Results.Percentiles = samplePercentiles(run.Targets[i].Results.LatencyUS)
	}
	return run
}

func lat(p50, p75, p90, p95, p99, p999 float64) schema.LatencyUS {
	return schema.LatencyUS{
		Min: p50 * 0.35, Mean: p50 * 1.12, P50: p50, P75: p75, P90: p90,
		P95: p95, P99: p99, P999: p999, P9999: p999 * 1.9, Max: p999 * 3.2,
	}
}

func results(iops, mbps float64, lat schema.LatencyUS) schema.Results {
	return schema.Results{
		IOPS: iops, ReadIOPS: iops * 0.7, WriteIOPS: iops * 0.3,
		ThroughputMBps: mbps, OpsPerSec: iops,
		LatencyUS: lat, CPUPercent: 38 + (iops / 10000),
	}
}

func samplePercentiles(lat schema.LatencyUS) map[string]float64 {
	m := map[string]float64{}
	for _, l := range metrics.StandardPercentileLabels {
		switch {
		case l == "p50":
			m[l] = lat.P50
		case strings.HasPrefix(l, "p99"):
			m[l] = lat.P999 * (0.6 + float64(len(l))*0.02)
		default:
			m[l] = lat.P50 * 0.9
		}
	}
	return m
}

func samplePercentileCounts(total int64) map[string]int64 {
	m := map[string]int64{}
	for _, l := range metrics.StandardPercentileLabels {
		m[l] = total / 100
	}
	return m
}

func sampleIntervals(start time.Time, durationSec int, iops, mbps, p50 float64) []schema.IntervalSample {
	buckets := durationSec / 5
	if buckets > 24 {
		buckets = 24
	}
	var out []schema.IntervalSample
	for i := 0; i < buckets; i++ {
		j := 0.92 + float64(i%5)*0.02
		out = append(out, schema.IntervalSample{
			Seq: i + 1, Timestamp: start.Add(time.Duration(i*5) * time.Second), ElapsedSec: 5,
			IOPS: iops * j, ReadIOPS: iops * j * 0.7, WriteIOPS: iops * j * 0.3,
			ThroughputMBps: mbps * j, ReadMBps: mbps * j * 0.7, WriteMBps: mbps * j * 0.3,
			AvgLatencyUS: p50 * j, MinLatencyUS: p50 * 0.4, MaxLatencyUS: p50 * 3,
			Percentiles: samplePercentiles(schema.LatencyUS{P50: p50 * j, P99: p50 * j * 2, P999: p50 * j * 4}),
		})
	}
	return out
}
