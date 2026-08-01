package report

import (
	"fmt"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

type intervalSource struct {
	ID    string
	Label string
	Role  string
	IVs   []schema.IntervalSample
}

func collectIntervalSources(run *schema.RunResult) []intervalSource {
	var out []intervalSource
	seen := map[string]bool{}

	add := func(id, label, role string, ivs []schema.IntervalSample) {
		if len(ivs) == 0 || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, intervalSource{ID: id, Label: label, Role: role, IVs: ivs})
	}

	add("aggregate", "Aggregate", "aggregate", run.Results.Intervals)

	for i, c := range run.Clients {
		label := shortHost(c.Host)
		if c.Target != "" {
			label = fmt.Sprintf("%s → %s", label, shortHost(c.Target))
		}
		add(fmt.Sprintf("client-%d", i), label, "client", c.Results.Intervals)
	}

	// Server-side intervals when there are no per-client rows (sweep / local multi-target).
	if len(run.Clients) == 0 {
		for i, t := range run.Targets {
			add(fmt.Sprintf("target-%d", i), shortHost(t.Target), "target", t.Results.Intervals)
		}
	}

	return out
}

func runHasAnyIntervals(run *schema.RunResult) bool {
	return len(collectIntervalSources(run)) > 0
}

func addNodeIntervalCharts(b *chartBuilder, run *schema.RunResult, lbl WorkloadLabels) {
	sources := collectIntervalSources(run)
	if len(sources) == 0 {
		return
	}

	// Unified x-axis from aggregate or longest series.
	ref := sources[0].IVs
	for _, s := range sources {
		if len(s.IVs) > len(ref) {
			ref = s.IVs
		}
	}
	labels := intervalLabels(ref)
	ops := lbl.OpsRate

	if len(sources) > 1 {
		var iopsDS, mbpsDS, latDS []chartDataset
		for _, s := range sources {
			iopsDS = append(iopsDS, chartDataset{
				Label:  s.Label,
				Data:   alignIntervalField(s.IVs, len(labels), func(iv schema.IntervalSample) float64 { return iv.IOPS }),
				Dashed: s.Role == "target",
			})
			mbpsDS = append(mbpsDS, chartDataset{
				Label:  s.Label,
				Data:   alignIntervalField(s.IVs, len(labels), func(iv schema.IntervalSample) float64 { return iv.ThroughputMBps }),
				Dashed: s.Role == "target",
			})
			latDS = append(latDS, chartDataset{
				Label:  s.Label,
				Data:   alignIntervalField(s.IVs, len(labels), func(iv schema.IntervalSample) float64 { return iv.AvgLatencyUS }),
				Dashed: s.Role == "target",
			})
		}
		b.add("Per-node operations — overlay", true, ChartPanel{
			ID: "nodeOverlayIops", Title: ops + " over time (all nodes)", Tall: true,
		}, chartSpec{Kind: "line", Labels: labels, Datasets: iopsDS})
		b.add("Per-node operations — overlay", true, ChartPanel{
			ID: "nodeOverlayMbps", Title: "Throughput over time (all nodes)", Tall: true,
		}, chartSpec{Kind: "line", Labels: labels, Datasets: mbpsDS})
		b.add("Per-node operations — overlay", false, ChartPanel{
			ID: "nodeOverlayLatency", Title: "Avg latency over time (all nodes)",
		}, chartSpec{Kind: "line", Labels: labels, Datasets: latDS})

		if hasReadWrite(run) || lbl.IsObject {
			var readDS, writeDS []chartDataset
			for _, s := range sources {
				readDS = append(readDS, chartDataset{
					Label: s.Label,
					Data:  alignIntervalField(s.IVs, len(labels), func(iv schema.IntervalSample) float64 { return iv.ReadIOPS }),
				})
				writeDS = append(writeDS, chartDataset{
					Label:  s.Label,
					Data:   alignIntervalField(s.IVs, len(labels), func(iv schema.IntervalSample) float64 { return iv.WriteIOPS }),
					Dashed: true,
				})
			}
			b.add("Per-node operations — overlay", false, ChartPanel{
				ID: "nodeOverlayRead", Title: lbl.ReadOp + " over time (all nodes)",
			}, chartSpec{Kind: "line", Labels: labels, Datasets: readDS})
			b.add("Per-node operations — overlay", false, ChartPanel{
				ID: "nodeOverlayWrite", Title: lbl.WriteOp + " over time (all nodes)",
			}, chartSpec{Kind: "line", Labels: labels, Datasets: writeDS})
		}
	}

	for _, s := range sources {
		ivLabels := intervalLabels(s.IVs)
		panelPrefix := s.Label
		if s.Role == "aggregate" {
			panelPrefix = "Cluster aggregate"
		}

		b.addCollapsed("Per-node operations — detail", false, ChartPanel{
			ID: chartID("nodeGrafanaOps", s.ID), Title: panelPrefix + " — " + ops,
		}, chartSpec{
			Kind: "line", Labels: ivLabels, DualAxis: true,
			Datasets: []chartDataset{
				{Label: ops, Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return iv.IOPS }), YAxisID: "y"},
				{Label: "MB/s", Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return iv.ThroughputMBps }), YAxisID: "y1", Dashed: true},
			},
		})
		b.addCollapsed("Per-node operations — detail", false, ChartPanel{
			ID: chartID("nodeGrafanaLat", s.ID), Title: panelPrefix + " — latency",
		}, chartSpec{
			Kind: "line", Labels: ivLabels,
			Datasets: []chartDataset{
				{Label: "Avg", Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return iv.AvgLatencyUS })},
				{Label: "Min", Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return iv.MinLatencyUS }), Dashed: true},
				{Label: "Max", Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return iv.MaxLatencyUS }), Dashed: true},
			},
		})

		if hasReadWrite(run) || lbl.IsObject || ivHasReadWrite(s.IVs) {
			b.addCollapsed("Per-node operations — detail", false, ChartPanel{
				ID: chartID("nodeGrafanaRW", s.ID), Title: panelPrefix + " — " + lbl.ReadOp + " vs " + lbl.WriteOp,
			}, chartSpec{
				Kind: "line", Labels: ivLabels, Area: true,
				Datasets: []chartDataset{
					{Label: lbl.WriteOp, Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return iv.WriteIOPS }), Fill: true},
					{Label: lbl.ReadOp, Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return iv.ReadIOPS }), Fill: true},
				},
			})
		}

		if intervalHasTimeouts(s.IVs) {
			b.addCollapsed("Per-node operations — detail", false, ChartPanel{
				ID: chartID("nodeGrafanaTimeout", s.ID), Title: panelPrefix + " — timeouts",
			}, chartSpec{
				Kind: "line", Labels: ivLabels,
				Datasets: []chartDataset{
					{Label: "Write", Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return float64(iv.WriteTimeoutEvents) })},
					{Label: "Read", Data: intervalField(s.IVs, func(iv schema.IntervalSample) float64 { return float64(iv.ReadTimeoutEvents) }), Dashed: true},
				},
			})
		}
	}
}

func alignIntervalField(ivs []schema.IntervalSample, n int, fn func(schema.IntervalSample) float64) []float64 {
	out := make([]float64, n)
	for i := 0; i < n && i < len(ivs); i++ {
		out[i] = fn(ivs[i])
	}
	return out
}

func buildNodeIntervalSections(run *schema.RunResult) []NodeIntervalSection {
	var sections []NodeIntervalSection
	for _, s := range collectIntervalSources(run) {
		sections = append(sections, NodeIntervalSection{
			Label: s.Label,
			Role:  s.Role,
			Rows:  intervalRowsFromSamples(s.IVs),
		})
	}
	return sections
}

func intervalRowsFromSamples(ivs []schema.IntervalSample) []IntervalRow {
	var rows []IntervalRow
	for _, iv := range ivs {
		rows = append(rows, intervalSampleToRow(iv))
	}
	return rows
}
