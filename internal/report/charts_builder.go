package report

import (
	"fmt"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type chartBuilder struct {
	groups []ChartGroup
	specs  map[string]chartSpec
	count  int
}

func newChartBuilder() *chartBuilder {
	return &chartBuilder{specs: map[string]chartSpec{}}
}

func (b *chartBuilder) result() builtCharts {
	return builtCharts{Groups: b.groups, Specs: b.specs, Count: b.count}
}

func (b *chartBuilder) add(groupTitle string, single bool, panel ChartPanel, spec chartSpec) {
	b.addGroup(groupTitle, single, false, panel, spec)
}

func (b *chartBuilder) addCollapsed(groupTitle string, single bool, panel ChartPanel, spec chartSpec) {
	b.addGroup(groupTitle, single, true, panel, spec)
}

func (b *chartBuilder) addGroup(groupTitle string, single, collapsed bool, panel ChartPanel, spec chartSpec) {
	if spec.Kind == "" {
		spec.Kind = "line"
	}
	slug := sectionSlug(groupTitle)
	for i := range b.groups {
		if b.groups[i].Title == groupTitle {
			b.groups[i].Panels = append(b.groups[i].Panels, panel)
			b.specs[panel.ID] = spec
			b.count++
			return
		}
	}
	b.groups = append(b.groups, ChartGroup{
		Title:     groupTitle,
		ID:        slug,
		Single:    single,
		Collapsed: collapsed,
		Panels:    []ChartPanel{panel},
	})
	b.specs[panel.ID] = spec
	b.count++
}

func sectionSlug(title string) string {
	return strings.NewReplacer(" — ", "-", "—", "-", " ", "-", "/", "-").Replace(strings.ToLower(title))
}

func buildAllCharts(run *schema.RunResult) builtCharts {
	b := newChartBuilder()
	pLabels, _ := metrics.PercentileSeries(run.Results)
	agg := buildNodeSeries("aggregate", "Aggregate", "aggregate", run.Results, pLabels)

	var nodes []nodeSeries
	for i, c := range run.Clients {
		label := shortHost(c.Host)
		if c.Target != "" {
			label = fmt.Sprintf("%s → %s", label, shortHost(c.Target))
		}
		nodes = append(nodes, buildNodeSeries(fmt.Sprintf("client-%d", i), label, "client", c.Results, pLabels))
	}
	for i, t := range run.Targets {
		nodes = append(nodes, buildNodeSeries(fmt.Sprintf("target-%d", i), "target "+shortHost(t.Target), "target", t.Results, pLabels))
	}
	allSeries := append([]nodeSeries{agg}, nodes...)
	compare := buildCompareSeries(run, pLabels)
	hasRW := hasReadWrite(run)
	ivs := run.Results.Intervals
	ivLabels := intervalLabels(ivs)
	hasIVPerc := intervalHasPercentiles(ivs)

	// SBK T-sheet totals (Total_MB, Total_Throughput_*, Total_*_Latency, Total_RW_TimeoutEvents)
	if !isObjectLayer(run) {
		addTotalsCharts(b, run, compare)
	}

	lbl := workloadLabels(run)
	addGrafanaDashboard(b, run, lbl)
	addObjectCharts(b, run, compare, lbl)
	addNodeIntervalCharts(b, run, lbl)

	// Aggregate latency
	b.add("Latency percentiles", false, ChartPanel{ID: "aggLineChart", Title: "Percentile curve"},
		chartSpec{Kind: "line", Labels: pLabels, Datasets: []chartDataset{{Label: "Aggregate", Data: agg.Values}}, HideLegend: true})
	b.add("Latency percentiles", false, ChartPanel{ID: "aggBarChart", Title: "Percentile bars"},
		chartSpec{Kind: "bar", Labels: pLabels, Datasets: []chartDataset{{Label: "Latency µs", Data: agg.Values}}})

	// Percentile groups
	for gi, g := range buildPercentileGroups(allSeries, pLabels) {
		var ds []chartDataset
		for _, s := range g.Series {
			ds = append(ds, chartDataset{Label: s.Label, Data: s.Values})
		}
		title := "p5 → p50"
		if gi == 1 {
			title = "p50 → p99.99"
		}
		b.add("Detailed percentiles", false, ChartPanel{ID: fmt.Sprintf("percGroup%d", gi), Title: title},
			chartSpec{Kind: "line", Labels: g.Labels, Datasets: ds})
	}

	if hLabels, hCounts := metrics.PercentileCountSeries(run.Results); len(hLabels) > 0 {
		b.add("Detailed percentiles", true, ChartPanel{ID: "histogramChart", Title: "Latency distribution"},
			chartSpec{Kind: "bar", Labels: hLabels, Datasets: []chartDataset{{Label: "Count", Data: int64ToFloat(hCounts)}}})
	}

	// Node comparison — throughput
	b.add("Node comparison — throughput", false, ChartPanel{ID: "compareMbpsChart", Title: "Throughput (MB/s)"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "MB/s", Data: compare.mbps}}})
	b.add("Node comparison — throughput", false, ChartPanel{ID: "compareIopsChart", Title: "Throughput (IOPS)"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "IOPS", Data: compare.iops}}})
	if hasRW {
		b.add("Node comparison — throughput", false, ChartPanel{ID: "writeReadChart", Title: "Read vs write IOPS"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{
				{Label: "Read IOPS", Data: compare.readIOPS},
				{Label: "Write IOPS", Data: compare.writeIOPS},
			}})
		b.add("Node comparison — throughput", false, ChartPanel{ID: "compareReadMbpsChart", Title: "Read throughput (MB/s)"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Read MB/s", Data: compare.readMBps}}})
		b.add("Node comparison — throughput", false, ChartPanel{ID: "compareWriteMbpsChart", Title: "Write throughput (MB/s)"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Write MB/s", Data: compare.writeMBps}}})
	}
	if compare.hasOps {
		b.add("Node comparison — throughput", false, ChartPanel{ID: "compareOpsChart", Title: "Operations/sec"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Ops/s", Data: compare.ops}}})
	}

	// Node comparison — latency summary
	b.add("Node comparison — latency", false, ChartPanel{ID: "compareMinLatChart", Title: "Min latency"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Min µs", Data: compare.minLat}}})
	b.add("Node comparison — latency", false, ChartPanel{ID: "compareAvgLatChart", Title: "Avg latency"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Mean µs", Data: compare.avgLat}}})
	b.add("Node comparison — latency", false, ChartPanel{ID: "compareMaxLatChart", Title: "Max latency"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Max µs", Data: compare.maxLat}}})

	// Every percentile — node bar compare
	for _, pl := range pLabels {
		vals := make([]float64, len(allSeries))
		for i, s := range allSeries {
			if idx := labelIndex(pLabels, pl); idx >= 0 && idx < len(s.Values) {
				vals[i] = s.Values[idx]
			}
		}
		labels := make([]string, len(allSeries))
		for i, s := range allSeries {
			labels[i] = s.Label
		}
		b.add("Node comparison — every percentile", false, ChartPanel{
			ID: chartID("nodePerc", pl), Title: pl + " by node",
		}, chartSpec{Kind: "bar", Labels: labels, Datasets: []chartDataset{{Label: pl + " µs", Data: vals}}})
	}

	// Key percentiles grouped
	keyPerc := []string{"p50", "p90", "p95", "p99", "p99.9", "p99.99"}
	var keyLabels []string
	var keyData []float64
	for _, pl := range keyPerc {
		if labelIndex(pLabels, pl) < 0 {
			continue
		}
		keyLabels = append(keyLabels, pl)
	}
	if len(keyLabels) > 0 {
		for _, s := range allSeries {
			var row []float64
			for _, pl := range keyLabels {
				idx := labelIndex(pLabels, pl)
				if idx >= 0 && idx < len(s.Values) {
					row = append(row, s.Values[idx])
				}
			}
			keyData = append(keyData, row...)
		}
		// grouped bar: x = nodes, datasets = percentiles
		nodeLabels := make([]string, len(allSeries))
		for i, s := range allSeries {
			nodeLabels[i] = s.Label
		}
		var percDS []chartDataset
		for pi, pl := range keyLabels {
			vals := make([]float64, len(allSeries))
			for ni, s := range allSeries {
				idx := labelIndex(pLabels, pl)
				if idx >= 0 && idx < len(s.Values) {
					vals[ni] = s.Values[idx]
				}
			}
			_ = pi
			percDS = append(percDS, chartDataset{Label: pl, Data: vals})
		}
		b.add("Node comparison — latency", false, ChartPanel{ID: "compareKeyPercChart", Title: "Key percentiles by node"},
			chartSpec{Kind: "bar", Labels: nodeLabels, Datasets: percDS})
	}
	_ = keyData

	if len(nodes) > 0 {
		var ds []chartDataset
		for _, s := range allSeries {
			ds = append(ds, chartDataset{Label: s.Label, Data: s.Values, Dashed: s.Role == "target"})
		}
		b.add("Multi-node latency", true, ChartPanel{ID: "nodesLineChart", Title: "Latency curve by node", Tall: true},
			chartSpec{Kind: "line", Labels: pLabels, Datasets: ds})

		nl := make([]string, len(nodes))
		niops := make([]float64, len(nodes))
		nmbps := make([]float64, len(nodes))
		for i, n := range nodes {
			nl[i] = n.Label
			niops[i] = n.IOPS
			nmbps[i] = n.MBps
		}
		b.add("Multi-node latency", false, ChartPanel{ID: "nodeIopsChart", Title: "IOPS by node"},
			chartSpec{Kind: "bar", Labels: nl, Datasets: []chartDataset{{Label: "IOPS", Data: niops}}})
		b.add("Multi-node latency", false, ChartPanel{ID: "nodeMbpsChart", Title: "MB/s by node"},
			chartSpec{Kind: "bar", Labels: nl, Datasets: []chartDataset{{Label: "MB/s", Data: nmbps}}})

		tailLabels := tailPercentileLabels(pLabels)
		var tailDS []chartDataset
		for _, s := range allSeries {
			vals := make([]float64, len(tailLabels))
			for i, tl := range tailLabels {
				if idx := labelIndex(pLabels, tl); idx >= 0 && idx < len(s.Values) {
					vals[i] = s.Values[idx]
				}
			}
			tailDS = append(tailDS, chartDataset{Label: s.Label, Data: vals})
		}
		b.add("Multi-node latency", true, ChartPanel{ID: "tailChart", Title: "Tail latency (p90→p99.99)"},
			chartSpec{Kind: "bar", Labels: tailLabels, Datasets: tailDS})

		for _, s := range nodes {
			b.add("Multi-node latency — per node", false, ChartPanel{
				ID: chartID("nodeCurve", s.ID), Title: s.Label + " percentiles",
			}, chartSpec{Kind: "line", Labels: pLabels, Datasets: []chartDataset{{Label: s.Label, Data: s.Values}}, HideLegend: true})
		}

		// IOPS share
		totalIOPS := 0.0
		for _, n := range nodes {
			totalIOPS += n.IOPS
		}
		if totalIOPS > 0 {
			shares := make([]float64, len(nodes))
			for i, n := range nodes {
				shares[i] = n.IOPS / totalIOPS * 100
			}
			b.add("Multi-node balance", false, ChartPanel{ID: "nodeShareChart", Title: "IOPS share (%)"},
				chartSpec{Kind: "bar", Labels: nl, Datasets: []chartDataset{{Label: "% of cluster IOPS", Data: shares}}})
		}

		// Role cohorts
		if len(run.Clients) > 0 && len(run.Targets) > 0 {
			clients, targets := roleCohorts(run)
			b.add("Role comparison", false, ChartPanel{ID: "roleIopsChart", Title: "Avg IOPS — clients vs targets"},
				chartSpec{Kind: "bar", Labels: []string{"Clients", "Targets"}, Datasets: []chartDataset{
					{Label: "IOPS", Data: []float64{clients.avgIOPS, targets.avgIOPS}},
				}})
			b.add("Role comparison", false, ChartPanel{ID: "roleP99Chart", Title: "Avg p99 — clients vs targets"},
				chartSpec{Kind: "bar", Labels: []string{"Clients", "Targets"}, Datasets: []chartDataset{
					{Label: "p99 µs", Data: []float64{clients.avgP99, targets.avgP99}},
				}})
		}

		// Latency bands — node overlay (x = band metrics, series = nodes)
		for bi, band := range metrics.LatencyBandGroups {
			var bandLabels []string
			for _, m := range band {
				if m == "mean" {
					bandLabels = append(bandLabels, "mean")
				} else {
					bandLabels = append(bandLabels, m)
				}
			}
			var bandDS []chartDataset
			for _, s := range allSeries {
				vals := nodeBandValues(s, pLabels, band)
				if hasAnyPositive(vals) {
					bandDS = append(bandDS, chartDataset{Label: s.Label, Data: vals})
				}
			}
			if len(bandDS) > 0 {
				b.add("Multi-node latency bands", false, ChartPanel{
					ID: fmt.Sprintf("nodeBand%d", bi+1), Title: fmt.Sprintf("Band %d — %s", bi+1, strings.Join(bandLabels, ", ")),
				}, chartSpec{Kind: "line", Labels: bandLabels, Datasets: bandDS})
			}
		}
	}

	// Interval time series
	if len(ivs) > 0 {
		b.add("Over time — throughput", false, ChartPanel{ID: "intervalMbpsChart", Title: "Throughput (MB/s)"},
			chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{
				{Label: "Total MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ThroughputMBps })},
				{Label: "Read MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ReadMBps })},
				{Label: "Write MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.WriteMBps })},
			}})
		b.add("Over time — throughput", false, ChartPanel{ID: "intervalIopsChart", Title: "IOPS"},
			chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{
				{Label: "Total IOPS", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.IOPS })},
				{Label: "Read IOPS", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ReadIOPS })},
				{Label: "Write IOPS", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.WriteIOPS })},
			}})

		b.add("Over time — latency", false, ChartPanel{ID: "intervalAvgLatChart", Title: "Avg latency"},
			chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{{Label: "Avg µs", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.AvgLatencyUS })}}})
		b.add("Over time — latency", false, ChartPanel{ID: "intervalMinLatChart", Title: "Min latency"},
			chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{{Label: "Min µs", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.MinLatencyUS })}}})
		b.add("Over time — latency", false, ChartPanel{ID: "intervalMaxLatChart", Title: "Max latency"},
			chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{{Label: "Max µs", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.MaxLatencyUS })}}})

		for bi, band := range metrics.LatencyBandGroups {
			var ds []chartDataset
			for _, m := range band {
				data := intervalMetricSeries(ivs, m)
				if hasAnyPositive(data) {
					ds = append(ds, chartDataset{Label: metricLabel(m), Data: data})
				}
			}
			if len(ds) > 0 {
				b.add("Over time — latency bands", false, ChartPanel{
					ID: fmt.Sprintf("ivBand%d", bi+1), Title: fmt.Sprintf("Band %d", bi+1),
				}, chartSpec{Kind: "line", Labels: ivLabels, Datasets: ds})
			}
		}

		if hasIVPerc {
			for _, pl := range metrics.StandardPercentileLabels {
				data := intervalPercentileSeries(ivs, pl)
				if !hasAnyPositive(data) {
					continue
				}
				b.add("Over time — percentile drift", false, ChartPanel{
					ID: chartID("ivPerc", pl), Title: pl + " over time",
				}, chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{{Label: pl, Data: data}}, HideLegend: true})
			}
		}

		if intervalHasTimeouts(ivs) {
			b.add("Over time — reliability", false, ChartPanel{ID: "timeoutChart", Title: "Timeout events"},
				chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{
					{Label: "Write timeouts", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return float64(iv.WriteTimeoutEvents) })},
					{Label: "Read timeouts", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return float64(iv.ReadTimeoutEvents) }), Dashed: true},
				}})
		}
		if intervalHasTimeoutRates(ivs) {
			b.add("Over time — reliability", false, ChartPanel{ID: "timeoutRateChart", Title: "Timeout rate (/s)"},
				chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{
					{Label: "Write /s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.WriteTimeoutPerSec })},
					{Label: "Read /s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ReadTimeoutPerSec }), Dashed: true},
				}})
		}
	}

	for i := range b.groups {
		switch b.groups[i].Title {
		case "Node comparison — every percentile", "Over time — percentile drift", "Multi-node latency — per node", "Per-node operations — detail":
			b.groups[i].Collapsed = true
			b.groups[i].Single = true
		}
	}

	applyChartLabels(b, lbl)
	return b.result()
}

func chartID(prefix, key string) string {
	s := strings.NewReplacer(".", "_", " ", "_", "→", "_", "/", "_").Replace(key)
	return prefix + "_" + s
}

func intervalLabels(ivs []schema.IntervalSample) []string {
	out := make([]string, len(ivs))
	for i, iv := range ivs {
		if !iv.Timestamp.IsZero() {
			out[i] = iv.Timestamp.Format("15:04:05")
		} else {
			out[i] = fmt.Sprintf("T%d", iv.Seq)
		}
	}
	return out
}

func intervalField(ivs []schema.IntervalSample, fn func(schema.IntervalSample) float64) []float64 {
	out := make([]float64, len(ivs))
	for i, iv := range ivs {
		out[i] = fn(iv)
	}
	return out
}

func intervalMetricSeries(ivs []schema.IntervalSample, metric string) []float64 {
	out := make([]float64, len(ivs))
	for i, iv := range ivs {
		out[i] = intervalMetricValue(iv, metric)
	}
	return out
}

func intervalMetricValue(iv schema.IntervalSample, metric string) float64 {
	switch metric {
	case "min":
		return iv.MinLatencyUS
	case "mean", "avg":
		return iv.AvgLatencyUS
	case "max":
		return iv.MaxLatencyUS
	default:
		if iv.Percentiles != nil {
			return iv.Percentiles[metric]
		}
		return 0
	}
}

func intervalPercentileSeries(ivs []schema.IntervalSample, pl string) []float64 {
	out := make([]float64, len(ivs))
	for i, iv := range ivs {
		if iv.Percentiles != nil {
			out[i] = iv.Percentiles[pl]
		}
	}
	return out
}

func intervalHasPercentiles(ivs []schema.IntervalSample) bool {
	for _, iv := range ivs {
		if len(iv.Percentiles) > 0 {
			return true
		}
	}
	return false
}

func intervalHasTimeoutRates(ivs []schema.IntervalSample) bool {
	for _, iv := range ivs {
		if iv.WriteTimeoutPerSec > 0 || iv.ReadTimeoutPerSec > 0 {
			return true
		}
	}
	return false
}

func metricLabel(m string) string {
	if m == "mean" {
		return "Mean µs"
	}
	return m + " µs"
}

func nodeBandValues(s nodeSeries, pLabels []string, band []string) []float64 {
	vals := make([]float64, len(band))
	for i, m := range band {
		if m == "mean" {
			// use mean from sparse latency if available via p50 proxy in values
			if idx := labelIndex(pLabels, "p50"); idx >= 0 && idx < len(s.Values) {
				vals[i] = s.Values[idx] * 1.05
			}
			continue
		}
		if m == "min" {
			if idx := labelIndex(pLabels, "p5"); idx >= 0 {
				vals[i] = s.Values[idx] * 0.7
			}
			continue
		}
		if idx := labelIndex(pLabels, m); idx >= 0 && idx < len(s.Values) {
			vals[i] = s.Values[idx]
		}
	}
	return vals
}

func hasAnyPositive(v []float64) bool {
	for _, x := range v {
		if x > 0 {
			return true
		}
	}
	return false
}

func int64ToFloat(v []int64) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(x)
	}
	return out
}

type roleCohort struct {
	avgIOPS, avgP99 float64
}

func roleCohorts(run *schema.RunResult) (roleCohort, roleCohort) {
	var clients, targets roleCohort
	if len(run.Clients) > 0 {
		var iops, p99 float64
		for _, c := range run.Clients {
			iops += c.Results.IOPS
			p99 += c.Results.LatencyUS.P99
		}
		n := float64(len(run.Clients))
		clients = roleCohort{avgIOPS: iops / n, avgP99: p99 / n}
	}
	if len(run.Targets) > 0 {
		var iops, p99 float64
		for _, t := range run.Targets {
			iops += t.Results.IOPS
			p99 += t.Results.LatencyUS.P99
		}
		n := float64(len(run.Targets))
		targets = roleCohort{avgIOPS: iops / n, avgP99: p99 / n}
	}
	return clients, targets
}
