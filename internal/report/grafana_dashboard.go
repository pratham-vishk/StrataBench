package report

import (
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// addGrafanaDashboard adds Grafana-style operation time-series panels (requires intervals).
func addGrafanaDashboard(b *chartBuilder, run *schema.RunResult, lbl WorkloadLabels) {
	ivs := run.Results.Intervals
	if len(ivs) == 0 {
		return
	}
	labels := intervalLabels(ivs)
	ops := lbl.OpsRate

	b.add("Operations dashboard", true, ChartPanel{
		ID: "grafanaOpsOverview", Title: "Operations overview — " + ops + " & throughput", Tall: true,
	}, chartSpec{
		Kind: "line", Labels: labels, DualAxis: true,
		Datasets: []chartDataset{
			{Label: ops, Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.IOPS }), YAxisID: "y"},
			{Label: "MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ThroughputMBps }), YAxisID: "y1", Dashed: true},
		},
	})

	if hasReadWrite(run) || lbl.IsObject {
		b.add("Operations dashboard", true, ChartPanel{
			ID: "grafanaReadWrite", Title: lbl.ReadOp + " vs " + lbl.WriteOp, Tall: true,
		}, chartSpec{
			Kind: "line", Labels: labels, Area: true,
			Datasets: []chartDataset{
				{Label: lbl.WriteOp, Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.WriteIOPS }), Fill: true},
				{Label: lbl.ReadOp, Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ReadIOPS }), Fill: true},
			},
		})
	}

	b.add("Operations dashboard", true, ChartPanel{
		ID: "grafanaLatency", Title: "Latency over time (µs)", Tall: true,
	}, chartSpec{
		Kind: "line", Labels: labels,
		Datasets: []chartDataset{
			{Label: "Avg", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.AvgLatencyUS })},
			{Label: "Min", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.MinLatencyUS }), Dashed: true},
			{Label: "Max", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.MaxLatencyUS }), Dashed: true},
		},
	})

	b.add("Operations dashboard", true, ChartPanel{
		ID: "grafanaThroughputStack", Title: "Read vs write throughput (MB/s)",
	}, chartSpec{
		Kind: "line", Labels: labels, Area: true,
		Datasets: []chartDataset{
			{Label: "Read MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ReadMBps }), Fill: true},
			{Label: "Write MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.WriteMBps }), Fill: true},
			{Label: "Total MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ThroughputMBps })},
		},
	})
}
